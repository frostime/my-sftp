package client

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/schollz/progressbar/v3"
)

// Upload 上传文件
func (c *Client) Upload(localPath, remotePath string) error {
	return c.UploadWithProgress(localPath, remotePath, true)
}

// UploadWithProgress 上传文件（支持进度条）
func (c *Client) UploadWithProgress(localPath, remotePath string, showProgress bool) error {
	localPath = c.ResolveLocalPath(localPath)
	remotePath = c.ResolveRemotePath(remotePath)

	// 获取本地文件信息
	stat, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("stat local: %w", err)
	}

	srcFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local: %w", err)
	}
	defer srcFile.Close()

	// 如果远程路径是目录，使用本地文件名
	if remoteStat, err := c.sftpClient.Stat(remotePath); err == nil && remoteStat.IsDir() {
		remotePath = path.Join(remotePath, filepath.Base(localPath))
	}

	dstFile, err := c.sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("create remote: %w", err)
	}
	defer dstFile.Close()

	// 统一获取 buffer（安全的类型断言）
	buf := c.getBuffer()
	defer c.putBuffer(buf)

	// 使用缓冲和进度条
	if showProgress {
		bar := progressbar.DefaultBytes(
			stat.Size(),
			fmt.Sprintf("Uploading %s", filepath.Base(localPath)),
		)
		_, err = io.CopyBuffer(io.MultiWriter(dstFile, bar), srcFile, buf)
		fmt.Println() // 换行
	} else {
		_, err = io.CopyBuffer(dstFile, srcFile, buf)
	}

	return err
}

// UploadOptions 上传选项
type UploadOptions struct {
	Recursive    bool // 递归上传目录
	ShowProgress bool // 显示进度条
	Concurrency  int  // 并发数
	MaxDepth     int  // 最大递归深度：-1=无限, 0=仅当前目录, 1=一层子目录...
}

// UploadGlob 使用 glob 模式匹配上传文件
func (c *Client) UploadGlob(pattern, remotePath string, opts *UploadOptions) (int, error) {
	if opts == nil {
		opts = &UploadOptions{
			ShowProgress: true,
			Concurrency:  MaxConcurrentTransfers,
			MaxDepth:     -1,
		}
	}

	// 解析 glob 模式
	basePath := c.localWorkDir
	fullPattern := pattern
	if !filepath.IsAbs(pattern) {
		fullPattern = filepath.Join(basePath, pattern)
	}

	// 使用 doublestar 支持 ** 递归匹配
	matches, err := doublestar.FilepathGlob(fullPattern)
	if err != nil {
		return 0, fmt.Errorf("glob pattern: %w", err)
	}

	if len(matches) == 0 {
		return 0, fmt.Errorf("no files match pattern: %s", pattern)
	}

	remotePath = c.ResolveRemotePath(remotePath)

	// 收集所有上传任务
	var tasks []transferTask
	for _, match := range matches {
		stat, err := os.Stat(match)
		if err != nil {
			continue
		}

		if stat.IsDir() {
			if !opts.Recursive {
				continue // 非递归模式跳过目录
			}
			// 递归收集目录内的文件
			remoteSubDir := path.Join(remotePath, filepath.Base(match))
			subTasks, err := c.collectUploadTasks(match, remoteSubDir, opts.MaxDepth, 0)
			if err != nil {
				return 0, fmt.Errorf("collect tasks for %s: %w", match, err)
			}
			tasks = append(tasks, subTasks...)
		} else {
			remoteFile := path.Join(remotePath, filepath.Base(match))
			tasks = append(tasks, transferTask{
				localPath:  match,
				remotePath: remoteFile,
				isUpload:   true,
				size:       stat.Size(),
			})
		}
	}

	if len(tasks) == 0 {
		return 0, fmt.Errorf("no files to upload")
	}

	fmt.Printf("Found %d file(s) to upload\n", len(tasks))

	// 确保所有远程目录存在
	dirs := c.collectRemoteDirsForUpload(tasks)
	if err := c.ensureRemoteDirsExist(dirs); err != nil {
		return 0, fmt.Errorf("create remote dirs: %w", err)
	}

	// 使用统一执行引擎
	transferOpts := &TransferOptions{
		Recursive:    opts.Recursive,
		ShowProgress: opts.ShowProgress,
		Concurrency:  opts.Concurrency,
		MaxDepth:     opts.MaxDepth,
	}
	return c.executeTasks(tasks, transferOpts)
}

// UploadDir 递归上传整个目录
// 使用统一的任务收集+执行模式，避免并发嵌套
func (c *Client) UploadDir(localDir, remoteDir string, opts *UploadOptions) (int, error) {
	if opts == nil {
		opts = &UploadOptions{
			ShowProgress: true,
			Concurrency:  MaxConcurrentTransfers,
			MaxDepth:     -1,
		}
	}

	localDir = c.ResolveLocalPath(localDir)
	remoteDir = c.ResolveRemotePath(remoteDir)

	// 检查本地目录
	stat, err := os.Stat(localDir)
	if err != nil {
		return 0, fmt.Errorf("stat local dir: %w", err)
	}
	if !stat.IsDir() {
		return 0, fmt.Errorf("not a directory: %s", localDir)
	}

	// 收集所有上传任务（不执行）
	tasks, err := c.collectUploadTasks(localDir, remoteDir, opts.MaxDepth, 0)
	if err != nil {
		return 0, fmt.Errorf("collect upload tasks: %w", err)
	}

	if len(tasks) == 0 {
		return 0, fmt.Errorf("no files found in directory")
	}

	fmt.Printf("Uploading directory with %d file(s)\n", len(tasks))

	// 确保所有远程目录存在（包括根目录）
	dirs := c.collectRemoteDirsForUpload(tasks)
	// 添加根目录
	dirs = append([]string{remoteDir}, dirs...)
	if err := c.ensureRemoteDirsExist(dirs); err != nil {
		return 0, fmt.Errorf("create remote dirs: %w", err)
	}

	// 使用统一执行引擎
	transferOpts := &TransferOptions{
		Recursive:    opts.Recursive,
		ShowProgress: opts.ShowProgress,
		Concurrency:  opts.Concurrency,
		MaxDepth:     opts.MaxDepth,
	}
	return c.executeTasks(tasks, transferOpts)
}

// ensureRemoteDir 确保远程目录存在
// 确保同一目录只创建一次，避免并发竞争
func (c *Client) ensureRemoteDir(dir string) error {
	dir = c.ResolveRemotePath(dir)

	// 快速路径：目录已存在
	if stat, err := c.sftpClient.Stat(dir); err == nil && stat.IsDir() {
		return nil
	}

	// 使用 singleflight 确保同一目录只创建一次
	_, err, _ := c.dirCreateGroup.Do(dir, func() (interface{}, error) {
		// double check
		if stat, err := c.sftpClient.Stat(dir); err == nil && stat.IsDir() {
			return nil, nil
		}

		// 先递归创建父目录
		parent := path.Dir(dir)
		if parent != "/" && parent != "." {
			if err := c.ensureRemoteDir(parent); err != nil {
				return nil, err
			}
		}

		// 有 singleflight 保证，这里不需要额外加锁
		// mu := c.getDirLock(dir)
		// mu.Lock()
		// defer mu.Unlock()
		// if stat, err := c.sftpClient.Stat(dir); err == nil && stat.IsDir() {
		// 	return nil, nil
		// }

		if err := c.sftpClient.Mkdir(dir); err != nil {
			// 最后一次检查（防止服务器端刚巧被别人创建了）
			if stat, statErr := c.sftpClient.Stat(dir); statErr == nil && stat.IsDir() {
				return nil, nil
			}
			return nil, err
		}

		c.invalidateDirCache(parent)
		return nil, nil
	})

	return err
}
