package client

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/schollz/progressbar/v3"
)

// transferTask 表示单个传输任务
type transferTask struct {
	localPath  string // 本地文件路径
	remotePath string // 远程文件路径
	isUpload   bool   // true=上传, false=下载
	size       int64  // 文件大小，用于进度显示
}

// TransferOptions 统一的传输选项
type TransferOptions struct {
	Recursive    bool // 递归处理目录
	ShowProgress bool // 显示进度条
	Concurrency  int  // 并发数
	MaxDepth     int  // 最大递归深度：-1=无限, 0=仅当前目录, 1=一层子目录...
}

// DefaultTransferOptions 返回默认传输选项
func DefaultTransferOptions() *TransferOptions {
	return &TransferOptions{
		Recursive:    true,
		ShowProgress: true,
		Concurrency:  MaxConcurrentTransfers,
		MaxDepth:     -1, // 默认无限深度
	}
}

// executeTasks 统一的任务执行引擎
// 这是所有并发传输的唯一入口点，避免并发嵌套问题
func (c *Client) executeTasks(tasks []transferTask, opts *TransferOptions) (int, error) {
	if len(tasks) == 0 {
		return 0, nil
	}

	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = MaxConcurrentTransfers
	}
	if concurrency > len(tasks) {
		concurrency = len(tasks)
	}

	// 并发控制信号量
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error
	var successCount int32 = 0

	// 整体进度条（仅在并发>1且需要显示进度时使用）
	var globalBar *progressbar.ProgressBar
	showGlobalProgress := opts.ShowProgress && concurrency > 1
	showFileProgress := opts.ShowProgress && concurrency == 1

	if showGlobalProgress {
		globalBar = progressbar.NewOptions(len(tasks),
			progressbar.OptionSetDescription("Transferring files"),
			progressbar.OptionShowCount(),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(40),
			progressbar.OptionClearOnFinish(),
			progressbar.OptionSetElapsedTime(true),
			progressbar.OptionSetPredictTime(true),
			progressbar.OptionShowBytes(true),
			progressbar.OptionShowTotalBytes(true),
		)
	}

	for _, task := range tasks {
		wg.Add(1)
		sem <- struct{}{} // 获取信号量

		go func(t transferTask) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			// panic 保护
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("panic during transfer %s: %v\nstack: %s",
						t.localPath, r, debug.Stack()))
					mu.Unlock()
				}
			}()

			var err error
			if t.isUpload {
				err = c.UploadWithProgress(t.localPath, t.remotePath, showFileProgress)
			} else {
				err = c.DownloadWithProgress(t.remotePath, t.localPath, showFileProgress)
			}

			if err != nil {
				mu.Lock()
				if t.isUpload {
					errs = append(errs, fmt.Errorf("upload %s: %w", t.localPath, err))
				} else {
					errs = append(errs, fmt.Errorf("download %s: %w", t.remotePath, err))
				}
				mu.Unlock()
			} else {
				atomic.AddInt32(&successCount, 1)
			}

			// 更新整体进度条
			if showGlobalProgress && globalBar != nil {
				globalBar.Add(1)
			}
		}(task)
	}

	wg.Wait()

	if showGlobalProgress && globalBar != nil {
		globalBar.Finish()
		fmt.Println() // 换行
	}

	if len(errs) > 0 {
		return int(successCount), errors.Join(errs...)
	}
	return int(successCount), nil
}

// collectDownloadTasks 收集下载任务（不执行传输）
// remoteDir: 远程目录路径
// localDir: 本地目录路径
// maxDepth: 最大递归深度，-1表示无限
// currentDepth: 当前深度（内部使用）
func (c *Client) collectDownloadTasks(remoteDir, localDir string, maxDepth, currentDepth int) ([]transferTask, error) {
	var tasks []transferTask

	entries, err := c.sftpClient.ReadDir(remoteDir)
	if err != nil {
		return nil, fmt.Errorf("read remote dir %s: %w", remoteDir, err)
	}

	for _, entry := range entries {
		remotePath := path.Join(remoteDir, entry.Name())
		localPath := filepath.Join(localDir, entry.Name())

		if entry.IsDir() {
			// 检查深度限制
			if maxDepth >= 0 && currentDepth >= maxDepth {
				continue // 超过深度限制，跳过此目录
			}

			// 确保本地目录存在
			if err := os.MkdirAll(localPath, 0755); err != nil {
				return nil, fmt.Errorf("create local dir %s: %w", localPath, err)
			}

			// 递归收集子目录任务
			subTasks, err := c.collectDownloadTasks(remotePath, localPath, maxDepth, currentDepth+1)
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, subTasks...)
		} else {
			tasks = append(tasks, transferTask{
				localPath:  localPath,
				remotePath: remotePath,
				isUpload:   false,
				size:       entry.Size(),
			})
		}
	}

	return tasks, nil
}

// collectUploadTasks 收集上传任务（不执行传输）
// localDir: 本地目录路径
// remoteDir: 远程目录路径
// maxDepth: 最大递归深度，-1表示无限
// currentDepth: 当前深度（内部使用）
func (c *Client) collectUploadTasks(localDir, remoteDir string, maxDepth, currentDepth int) ([]transferTask, error) {
	var tasks []transferTask

	entries, err := os.ReadDir(localDir)
	if err != nil {
		return nil, fmt.Errorf("read local dir %s: %w", localDir, err)
	}

	for _, entry := range entries {
		localPath := filepath.Join(localDir, entry.Name())
		remotePath := path.Join(remoteDir, entry.Name())

		if entry.IsDir() {
			// 检查深度限制
			if maxDepth >= 0 && currentDepth >= maxDepth {
				continue // 超过深度限制，跳过此目录
			}

			// 递归收集子目录任务
			subTasks, err := c.collectUploadTasks(localPath, remotePath, maxDepth, currentDepth+1)
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, subTasks...)
		} else {
			info, err := entry.Info()
			if err != nil {
				continue // 跳过无法获取信息的文件
			}
			tasks = append(tasks, transferTask{
				localPath:  localPath,
				remotePath: remotePath,
				isUpload:   true,
				size:       info.Size(),
			})
		}
	}

	return tasks, nil
}

// collectRemoteDirsForUpload 收集上传任务中需要创建的所有远程目录
// 返回按创建顺序排列的目录列表（父目录在前）
func (c *Client) collectRemoteDirsForUpload(tasks []transferTask) []string {
	dirSet := make(map[string]struct{})
	var dirs []string

	for _, task := range tasks {
		if !task.isUpload {
			continue
		}
		dir := path.Dir(task.remotePath)
		// 收集从根到目标目录的所有路径
		for dir != "/" && dir != "." {
			if _, exists := dirSet[dir]; !exists {
				dirSet[dir] = struct{}{}
				dirs = append(dirs, dir)
			}
			dir = path.Dir(dir)
		}
	}

	// 按路径长度排序，确保父目录在前
	sort.Slice(dirs, func(i, j int) bool {
		depthI := strings.Count(dirs[i], "/")
		depthJ := strings.Count(dirs[j], "/")
		if depthI != depthJ {
			return depthI < depthJ
		}
		return dirs[i] < dirs[j]
	})
	return dirs
}

// ensureRemoteDirsExist 确保所有远程目录存在
// 使用 singleflight 避免重复创建
func (c *Client) ensureRemoteDirsExist(dirs []string) error {
	for _, dir := range dirs {
		if err := c.ensureRemoteDir(dir); err != nil {
			return err
		}
	}
	return nil
}
