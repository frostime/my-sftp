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

// Upload 上传文件
func (c *Client) Upload(localPath, remotePath string) error {
	localPath = c.ResolveLocalPath(localPath)

	// 获取文件信息以创建进度条
	stat, err := os.Stat(localPath)
	if err != nil {
		return err
	}

	// 创建单文件进度条（显示文件名）
	bar := progressbar.NewOptions64(stat.Size(),
		progressbar.OptionSetDescription(fmt.Sprintf("Uploading %s (1/1 files)", filepath.Base(localPath))),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetPredictTime(true),
	)
	defer bar.Finish()
	defer fmt.Println()

	return c.UploadWithProgress(localPath, remotePath, bar)
}

// UploadWithProgress 上传文件（支持进度条）
func (c *Client) UploadWithProgress(localPath, remotePath string, globalBar *progressbar.ProgressBar) error {
	localPath = c.ResolveLocalPath(localPath)
	remotePath = c.ResolveRemotePath(remotePath)

	// 获取本地文件信息（确保文件存在）
	_, err := os.Stat(localPath)
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
	parent := path.Dir(remotePath)
	if parent != "/" && parent != "." {
		if err := c.ensureRemoteDir(parent); err != nil {
			return fmt.Errorf("create remote dir: %w", err)
		}
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
	var writer io.Writer = dstFile
	if globalBar != nil {
		writer = io.MultiWriter(dstFile, globalBar)
	}

	_, err = io.CopyBuffer(writer, srcFile, buf)
	return err
}

// UploadOptions 上传选项
type UploadOptions struct {
	Recursive    bool // 递归上传目录
	ShowProgress bool // 显示进度条
	Concurrency  int  // 并发数
	Flatten      bool // 扁平化目标路径
	MaxDepth     int  // 最大递归深度：-1=无限, 0=仅当前目录, 1=一层子目录...
}

// UploadGlob 使用 glob 模式匹配上传文件
func (c *Client) UploadGlob(pattern, remotePath string, opts *UploadOptions) (int, error) {
	return c.UploadSources([]string{pattern}, remotePath, opts)
}

// UploadSources 上传一个或多个本地 source（显式路径或 glob）
func (c *Client) UploadSources(localSources []string, remoteDir string, opts *UploadOptions) (int, error) {
	if len(localSources) == 0 {
		return 0, fmt.Errorf("missing source path")
	}

	if opts == nil {
		opts = &UploadOptions{
			ShowProgress: true,
			Concurrency:  MaxConcurrentTransfers,
			MaxDepth:     -1,
		}
	}

	remoteDir = c.ResolveRemotePath(remoteDir)

	var tasks []transferTask
	for _, source := range localSources {
		sourceTasks, err := c.collectUploadSourceTasks(source, remoteDir, opts, len(localSources))
		if err != nil {
			return 0, err
		}
		tasks = append(tasks, sourceTasks...)
	}

	if len(tasks) == 0 {
		return 0, fmt.Errorf("no files found in directory")
	}

	if opts.Flatten {
		if err := applyFlattenMapping(tasks, remoteDir); err != nil {
			return 0, err
		}
	}
	if err := validateTargetCollisions(tasks); err != nil {
		return 0, err
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

func (c *Client) collectUploadSourceTasks(source, remoteDir string, opts *UploadOptions, sourceCount int) ([]transferTask, error) {
	if sourceCount > 1 && !opts.Flatten && usesReservedPreservePrefix(source, true) {
		return nil, fmt.Errorf("source path uses reserved preserve prefix: %s", source)
	}
	if strings.ContainsAny(source, "*?[]") {
		return c.collectUploadGlobTasks(source, remoteDir, opts)
	}

	resolvedSource := c.ResolveLocalPath(source)
	stat, err := os.Stat(resolvedSource)
	if err != nil {
		return nil, err
	}

	if stat.IsDir() {
		if !opts.Recursive {
			return nil, fmt.Errorf("%s is a directory, use 'put -r' for recursive upload", source)
		}
		dirRoot := remoteDir
		if sourceCount > 1 {
			dirRoot = path.Join(remoteDir, explicitLocalFilePreservePath(source, resolvedSource))
		}
		tasks, err := c.collectUploadTasks(resolvedSource, dirRoot, opts.MaxDepth, 0)
		if err != nil {
			return nil, fmt.Errorf("collect tasks for %s: %w", source, err)
		}
		return tasks, nil
	}

	remoteFile := path.Join(remoteDir, path.Base(filepath.ToSlash(resolvedSource)))
	if sourceCount > 1 {
		remoteFile = path.Join(remoteDir, explicitLocalFilePreservePath(source, resolvedSource))
	}

	return []transferTask{{
		localPath:  resolvedSource,
		remotePath: remoteFile,
		isUpload:   true,
		size:       stat.Size(),
	}}, nil
}

func (c *Client) collectUploadGlobTasks(pattern, remotePath string, opts *UploadOptions) ([]transferTask, error) {
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
	var globBase string
	if !filepath.IsAbs(pattern) {
		fullPattern = filepath.Join(basePath, pattern)
		// 对于相对 pattern，从原始 pattern 计算 globBase 以保留目录结构
		globBase = localGlobBase(pattern)
	} else {
		globBase = localGlobBase(fullPattern)
	}
	globBaseAbs := globBase
	globBasePrefix := ""
	if !filepath.IsAbs(pattern) {
		if globBase == "/" || globBase == "\\" {
			globBase = "."
		}
		globBaseAbs = filepath.Clean(filepath.Join(basePath, globBase))
		if globBase != "." {
			globBasePrefix = filepath.ToSlash(globBase)
		}
	}

	// 使用 doublestar 支持 ** 递归匹配
	matches, err := doublestar.FilepathGlob(fullPattern)
	if err != nil {
		return nil, fmt.Errorf("glob pattern: %w", err)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no files match pattern: %s", pattern)
	}

	// 收集所有上传任务
	var tasks []transferTask
	for _, match := range matches {
		stat, err := os.Stat(match)
		if err != nil {
			return nil, fmt.Errorf("stat match %s: %w", match, err)
		}

		if stat.IsDir() {
			if !opts.Recursive {
				continue // 非递归模式跳过目录
			}
			// 递归收集目录内的文件
			mapped, relErr := filepath.Rel(globBaseAbs, match)
			if relErr != nil {
				return nil, fmt.Errorf("relative path for %s: %w", match, relErr)
			}
			mappedSlash := filepath.ToSlash(mapped)
			if mappedSlash == "." {
				mappedSlash = path.Base(filepath.ToSlash(match))
			}
			if globBasePrefix != "" {
				mappedSlash = path.Join(globBasePrefix, mappedSlash)
			}
			remoteSubDir := path.Join(remotePath, mappedSlash)
			subTasks, err := c.collectUploadTasks(match, remoteSubDir, opts.MaxDepth, 0)
			if err != nil {
				return nil, fmt.Errorf("collect tasks for %s: %w", match, err)
			}
			tasks = append(tasks, subTasks...)
		} else {
			mapped, relErr := filepath.Rel(globBaseAbs, match)
			if relErr != nil {
				return nil, fmt.Errorf("relative path for %s: %w", match, relErr)
			}
			mappedSlash := filepath.ToSlash(mapped)
			if mappedSlash == "." {
				mappedSlash = path.Base(filepath.ToSlash(match))
			}
			if globBasePrefix != "" {
				mappedSlash = path.Join(globBasePrefix, mappedSlash)
			}
			remoteFile := path.Join(remotePath, mappedSlash)
			tasks = append(tasks, transferTask{
				localPath:  match,
				remotePath: remoteFile,
				isUpload:   true,
				size:       stat.Size(),
			})
		}
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("no files to upload")
	}

	return tasks, nil
}

// UploadDir 递归上传整个目录
// 使用统一的任务收集+执行模式，避免并发嵌套
func (c *Client) UploadDir(localDir, remoteDir string, opts *UploadOptions) (int, error) {
	resolvedDir := c.ResolveLocalPath(localDir)
	stat, err := os.Stat(resolvedDir)
	if err != nil {
		return 0, fmt.Errorf("stat local dir: %w", err)
	}
	if !stat.IsDir() {
		return 0, fmt.Errorf("not a directory: %s", resolvedDir)
	}
	return c.UploadSources([]string{localDir}, remoteDir, opts)
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

func localGlobBase(pattern string) string {
	cleaned := filepath.Clean(pattern)
	parts := strings.Split(cleaned, string(filepath.Separator))
	base := make([]string, 0, len(parts))
	for i, part := range parts {
		if i == 0 && strings.Contains(part, ":") {
			base = append(base, part)
			continue
		}
		if strings.ContainsAny(part, "*?[]") {
			break
		}
		base = append(base, part)
	}
	if len(base) == 0 {
		return filepath.Dir(cleaned)
	}
	return filepath.Clean(filepath.Join(base...))
}
