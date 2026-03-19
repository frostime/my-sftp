package client

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/schollz/progressbar/v3"
)

// Download 下载文件
func (c *Client) Download(remotePath, localPath string) error {
	remotePath = c.ResolveRemotePath(remotePath)

	// 获取文件信息以创建进度条
	stat, err := c.sftpClient.Stat(remotePath)
	if err != nil {
		return err
	}

	// 创建单文件进度条（显示文件名）
	bar := progressbar.NewOptions64(stat.Size(),
		progressbar.OptionSetDescription(fmt.Sprintf("Downloading %s (1/1 files)", path.Base(remotePath))),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetPredictTime(true),
	)
	defer bar.Finish()
	defer fmt.Println()

	return c.DownloadWithProgress(remotePath, localPath, bar)
}

// DownloadWithProgress 下载文件（支持进度条）
func (c *Client) DownloadWithProgress(remotePath, localPath string, globalBar *progressbar.ProgressBar) error {
	remotePath = c.ResolveRemotePath(remotePath)
	localPath = c.ResolveLocalPath(localPath)

	// 获取远程文件信息（确保文件存在）
	_, err := c.sftpClient.Stat(remotePath)
	if err != nil {
		return fmt.Errorf("stat remote: %w", err)
	}

	srcFile, err := c.sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("open remote: %w", err)
	}
	defer srcFile.Close()

	// 如果本地路径是目录，使用远程文件名
	if localStat, err := os.Stat(localPath); err == nil && localStat.IsDir() {
		localPath = filepath.Join(localPath, path.Base(remotePath))
	}

	dstFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create local: %w", err)
	}
	defer dstFile.Close()

	// 统一获取 buffer（安全的类型断言）
	buf := c.getBuffer()
	defer c.putBuffer(buf)

	// 使用缓冲和进度条
	var writer io.Writer = dstFile
	if globalBar != nil {
		writer = io.MultiWriter(dstFile, globalBar)
	}

	_, err = io.CopyBuffer(writer, srcFile, buf)
	return err
}

// DownloadOptions 下载选项
type DownloadOptions struct {
	Recursive    bool // 递归下载目录
	ShowProgress bool // 显示进度条
	Concurrency  int  // 并发数
	Flatten      bool // 扁平化目标路径
	MaxDepth     int  // 最大递归深度：-1=无限, 0=仅当前目录, 1=一层子目录...
}

// DownloadDir 递归下载整个目录
// 使用统一的任务收集+执行模式，避免并发嵌套
func (c *Client) DownloadDir(remoteDir, localDir string, opts *DownloadOptions) (int, error) {
	if opts == nil {
		opts = &DownloadOptions{
			ShowProgress: true,
			Concurrency:  MaxConcurrentTransfers,
			MaxDepth:     -1,
		}
	}

	remoteDir = c.ResolveRemotePath(remoteDir)
	localDir = c.ResolveLocalPath(localDir)

	// 检查远程目录
	stat, err := c.sftpClient.Stat(remoteDir)
	if err != nil {
		return 0, fmt.Errorf("stat remote dir: %w", err)
	}
	if !stat.IsDir() {
		return 0, fmt.Errorf("not a directory: %s", remoteDir)
	}

	// 确保本地目录存在
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return 0, fmt.Errorf("create local dir: %w", err)
	}

	// 收集所有下载任务（不执行）
	tasks, err := c.collectDownloadTasks(remoteDir, localDir, opts.MaxDepth, 0)
	if err != nil {
		return 0, fmt.Errorf("collect download tasks: %w", err)
	}
	if opts.Flatten {
		if err := validateFlattenDownloadCollisions(tasks); err != nil {
			return 0, err
		}
		for i := range tasks {
			tasks[i].localPath = filepath.Join(localDir, filepath.Base(tasks[i].localPath))
		}
	}

	if len(tasks) == 0 {
		return 0, nil
	}

	fmt.Printf("Downloading directory with %d file(s)\n", len(tasks))

	// 使用统一执行引擎
	transferOpts := &TransferOptions{
		Recursive:    opts.Recursive,
		ShowProgress: opts.ShowProgress,
		Concurrency:  opts.Concurrency,
		MaxDepth:     opts.MaxDepth,
	}
	return c.executeTasks(tasks, transferOpts)
}

// DownloadGlob 使用 glob 模式匹配下载远程文件
func (c *Client) DownloadGlob(pattern, localPath string, opts *DownloadOptions) (int, error) {
	if opts == nil {
		opts = &DownloadOptions{
			ShowProgress: true,
			Concurrency:  MaxConcurrentTransfers,
			MaxDepth:     -1,
		}
	}

	// 解析 glob 模式的基路径
	basePath := c.workDir
	fullPattern := pattern
	var globBase string
	if !path.IsAbs(pattern) {
		fullPattern = path.Join(basePath, pattern)
		// 对于相对 pattern，从原始 pattern 计算 globBase 以保留目录结构
		globBase = remoteGlobBase(pattern)
	} else {
		globBase = remoteGlobBase(fullPattern)
	}
	globBaseAbs := globBase
	globBasePrefix := ""
	if !path.IsAbs(pattern) {
		if globBase == "/" {
			globBase = "."
		}
		globBaseAbs = path.Clean(path.Join(basePath, globBase))
		if globBase != "." {
			globBasePrefix = globBase
		}
	}

	// 查找匹配的远程文件
	matches, err := c.globRemote(fullPattern)
	if err != nil {
		return 0, fmt.Errorf("glob pattern: %w", err)
	}

	if len(matches) == 0 {
		return 0, fmt.Errorf("no files match pattern: %s", pattern)
	}

	// 确保本地目标目录存在
	localPath = c.ResolveLocalPath(localPath)
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return 0, fmt.Errorf("create local dir: %w", err)
	}

	// 收集所有下载任务
	var tasks []transferTask
	for _, match := range matches {
		stat, err := c.sftpClient.Stat(match)
		if err != nil {
			continue
		}

		if stat.IsDir() {
			if !opts.Recursive {
				continue // 非递归模式跳过目录
			}
			// 递归收集目录内的文件
			localSubDir := localPath
			if !opts.Flatten {
				rel := remoteRelativePath(globBaseAbs, match)
				if rel != "." {
					mapped := rel
					if globBasePrefix != "" {
						mapped = path.Join(globBasePrefix, rel)
					}
					localSubDir = filepath.Join(localPath, filepath.FromSlash(mapped))
				} else {
					mapped := path.Base(match)
					if globBasePrefix != "" {
						mapped = path.Join(globBasePrefix, mapped)
					}
					localSubDir = filepath.Join(localPath, filepath.FromSlash(mapped))
				}
			}
			if err := os.MkdirAll(localSubDir, 0755); err != nil {
				return 0, fmt.Errorf("create local dir %s: %w", localSubDir, err)
			}
			subTasks, err := c.collectDownloadTasks(match, localSubDir, opts.MaxDepth, 0)
			if err != nil {
				return 0, fmt.Errorf("collect tasks for %s: %w", match, err)
			}
			tasks = append(tasks, subTasks...)
		} else {
			localFile := filepath.Join(localPath, path.Base(match))
			if !opts.Flatten {
				rel := remoteRelativePath(globBaseAbs, match)
				if rel != "" && rel != "." {
					mapped := rel
					if globBasePrefix != "" {
						mapped = path.Join(globBasePrefix, rel)
					}
					localFile = filepath.Join(localPath, filepath.FromSlash(mapped))
				} else if globBasePrefix != "" {
					localFile = filepath.Join(localPath, filepath.FromSlash(path.Join(globBasePrefix, path.Base(match))))
				}
			}
			tasks = append(tasks, transferTask{
				localPath:  localFile,
				remotePath: match,
				isUpload:   false,
				size:       stat.Size(),
			})
		}
	}

	if len(tasks) == 0 {
		return 0, fmt.Errorf("no files to download")
	}
	if opts.Flatten {
		if err := validateFlattenDownloadCollisions(tasks); err != nil {
			return 0, err
		}
	}

	fmt.Printf("Found %d file(s) to download\n", len(tasks))

	// 使用统一执行引擎
	transferOpts := &TransferOptions{
		Recursive:    opts.Recursive,
		ShowProgress: opts.ShowProgress,
		Concurrency:  opts.Concurrency,
		MaxDepth:     opts.MaxDepth,
	}
	return c.executeTasks(tasks, transferOpts)
}

func remoteGlobBase(pattern string) string {
	parts := strings.Split(pattern, "/")
	base := make([]string, 0, len(parts))
	for i, part := range parts {
		if part == "" && i == 0 {
			base = append(base, "")
			continue
		}
		if strings.ContainsAny(part, "*?[]") {
			break
		}
		base = append(base, part)
	}
	if len(base) == 0 {
		return "/"
	}
	joined := strings.Join(base, "/")
	if joined == "" {
		return "/"
	}
	return path.Clean(joined)
}

func validateFlattenDownloadCollisions(tasks []transferTask) error {
	seen := make(map[string]struct{})
	for _, task := range tasks {
		base := filepath.Base(task.localPath)
		if _, exists := seen[base]; exists {
			return fmt.Errorf("duplicate basename in --flatten mode: %s\nHint: remove --flatten or narrow source set", base)
		}
		seen[base] = struct{}{}
	}
	return nil
}

func remoteRelativePath(base, target string) string {
	base = path.Clean(base)
	target = path.Clean(target)
	if target == base {
		return "."
	}
	if base == "/" {
		return strings.TrimPrefix(target, "/")
	}
	prefix := base + "/"
	if strings.HasPrefix(target, prefix) {
		return strings.TrimPrefix(target, prefix)
	}
	return path.Base(target)
}

// globRemote 在远程文件系统上执行 glob 匹配
func (c *Client) globRemote(pattern string) ([]string, error) {
	// 找到第一个包含通配符的路径段
	parts := strings.Split(pattern, "/")
	baseIdx := 0
	for i, part := range parts {
		if strings.ContainsAny(part, "*?[]") {
			baseIdx = i
			break
		}
	}

	// 基路径是通配符之前的部分
	basePath := "/"
	if baseIdx > 0 {
		basePath = strings.Join(parts[:baseIdx], "/")
		if basePath == "" {
			basePath = "/"
		}
	}

	// 收集所有远程文件
	var allFiles []string
	var walk func(string) error
	walk = func(dir string) error {
		entries, err := c.sftpClient.ReadDir(dir)
		if err != nil {
			return nil // 忽略无法访问的目录
		}

		for _, entry := range entries {
			fullPath := path.Join(dir, entry.Name())
			allFiles = append(allFiles, fullPath)
			if entry.IsDir() {
				// 只有在模式包含 ** 时才递归
				if strings.Contains(pattern, "**") {
					walk(fullPath)
				}
			}
		}
		return nil
	}

	// 从基路径开始遍历
	walk(basePath)

	// 使用 doublestar 进行匹配
	var matches []string
	for _, file := range allFiles {
		matched, err := doublestar.Match(pattern, file)
		if err != nil {
			continue
		}
		if matched {
			matches = append(matches, file)
		}
	}

	return matches, nil
}
