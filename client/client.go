package client

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pkg/sftp"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/crypto/ssh"
)

const (
	// BufferSize 传输缓冲区大小 (512KB)
	BufferSize = 512 * 1024
	// MaxConcurrentTransfers 最大并发传输数
	MaxConcurrentTransfers = 4
	// DirCacheTimeout 目录列表缓存超时时间
	DirCacheTimeout = 30 * time.Second
)

// dirCacheEntry 目录缓存条目
type dirCacheEntry struct {
	files    []os.FileInfo
	cachedAt time.Time
}

// Client SFTP 客户端封装
type Client struct {
	sshClient    *ssh.Client
	sftpClient   *sftp.Client
	workDir      string                    // 远程当前工作目录
	localWorkDir string                    // 本地当前工作目录
	dirCache     map[string]*dirCacheEntry // 目录列表缓存
	cacheMu      sync.RWMutex              // 缓存锁
}

// NewClient 创建 SFTP 客户端
func NewClient(addr string, config *ssh.ClientConfig) (*Client, error) {
	sshClient, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("ssh dial: %w", err)
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return nil, fmt.Errorf("sftp client: %w", err)
	}

	// 获取初始工作目录
	wd, err := sftpClient.Getwd()
	if err != nil {
		wd = "/"
	}

	// 获取程序启动时的本地工作目录
	localWd, err := os.Getwd()
	if err != nil {
		localWd = "."
	}

	return &Client{
		sshClient:    sshClient,
		sftpClient:   sftpClient,
		workDir:      wd,
		localWorkDir: localWd,
		dirCache:     make(map[string]*dirCacheEntry),
	}, nil
}

// Close 关闭连接
func (c *Client) Close() error {
	if c.sftpClient != nil {
		c.sftpClient.Close()
	}
	if c.sshClient != nil {
		return c.sshClient.Close()
	}
	return nil
}

// Getwd 获取远程当前工作目录
func (c *Client) Getwd() string {
	return c.workDir
}

// GetLocalwd 获取本地当前工作目录
func (c *Client) GetLocalwd() string {
	return c.localWorkDir
}

// LocalChdir 切换本地工作目录
func (c *Client) LocalChdir(dir string) error {
	targetPath := c.resolveLocalPath(dir)
	stat, err := os.Stat(targetPath)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}
	if !stat.IsDir() {
		return fmt.Errorf("not a directory: %s", targetPath)
	}
	c.localWorkDir = targetPath
	return nil
}

// LocalList 列出本地目录内容
func (c *Client) LocalList(dir string) ([]os.FileInfo, error) {
	targetPath := c.resolveLocalPath(dir)
	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return nil, err
	}
	var infos []os.FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// LocalMkdir 创建本地目录
func (c *Client) LocalMkdir(dir string) error {
	dir = c.resolveLocalPath(dir)
	return os.Mkdir(dir, 0755)
}

// Chdir 切换工作目录
func (c *Client) Chdir(dir string) error {
	targetPath := c.resolvePath(dir)
	stat, err := c.sftpClient.Stat(targetPath)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}
	if !stat.IsDir() {
		return fmt.Errorf("not a directory: %s", targetPath)
	}
	c.workDir = targetPath
	// 切换目录后清除缓存
	c.ClearDirCache()
	return nil
}

// List 列出目录内容
func (c *Client) List(dir string) ([]os.FileInfo, error) {
	targetPath := c.resolvePath(dir)

	// 检查缓存
	c.cacheMu.RLock()
	if entry, exists := c.dirCache[targetPath]; exists {
		// 检查是否过期
		if time.Since(entry.cachedAt) < DirCacheTimeout {
			c.cacheMu.RUnlock()
			return entry.files, nil
		}
	}
	c.cacheMu.RUnlock()

	// 缓存未命中或已过期，读取目录
	files, err := c.sftpClient.ReadDir(targetPath)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	c.cacheMu.Lock()
	c.dirCache[targetPath] = &dirCacheEntry{
		files:    files,
		cachedAt: time.Now(),
	}
	c.cacheMu.Unlock()

	return files, nil
}

// Download 下载文件
func (c *Client) Download(remotePath, localPath string) error {
	return c.DownloadWithProgress(remotePath, localPath, true)
}

// DownloadWithProgress 下载文件（支持进度条）
func (c *Client) DownloadWithProgress(remotePath, localPath string, showProgress bool) error {
	remotePath = c.resolvePath(remotePath)
	localPath = c.resolveLocalPath(localPath)

	// 获取远程文件信息
	stat, err := c.sftpClient.Stat(remotePath)
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

	// 使用缓冲和进度条
	if showProgress {
		bar := progressbar.DefaultBytes(
			stat.Size(),
			fmt.Sprintf("Downloading %s", filepath.Base(remotePath)),
		)
		_, err = io.Copy(io.MultiWriter(dstFile, bar), srcFile)
		fmt.Println() // 换行
	} else {
		buf := make([]byte, BufferSize)
		_, err = io.CopyBuffer(dstFile, srcFile, buf)
	}

	return err
}

// Upload 上传文件
func (c *Client) Upload(localPath, remotePath string) error {
	return c.UploadWithProgress(localPath, remotePath, true)
}

// UploadWithProgress 上传文件（支持进度条）
func (c *Client) UploadWithProgress(localPath, remotePath string, showProgress bool) error {
	localPath = c.resolveLocalPath(localPath)
	remotePath = c.resolvePath(remotePath)

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

	// 使用缓冲和进度条
	if showProgress {
		bar := progressbar.DefaultBytes(
			stat.Size(),
			fmt.Sprintf("Uploading %s", filepath.Base(localPath)),
		)
		_, err = io.Copy(io.MultiWriter(dstFile, bar), srcFile)
		fmt.Println() // 换行
	} else {
		buf := make([]byte, BufferSize)
		_, err = io.CopyBuffer(dstFile, srcFile, buf)
	}

	return err
}

// Remove 删除文件或目录
func (c *Client) Remove(remotePath string) error {
	remotePath = c.resolvePath(remotePath)
	stat, err := c.sftpClient.Stat(remotePath)
	if err != nil {
		return err
	}

	var removeErr error
	if stat.IsDir() {
		// 递归删除目录
		removeErr = c.removeDir(remotePath)
	} else {
		removeErr = c.sftpClient.Remove(remotePath)
	}

	if removeErr == nil {
		// 清除父目录缓存
		c.invalidateDirCache(path.Dir(remotePath))
	}
	return removeErr
}

// removeDir 递归删除目录
func (c *Client) removeDir(dir string) error {
	files, err := c.sftpClient.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		fullPath := path.Join(dir, file.Name())
		if file.IsDir() {
			if err := c.removeDir(fullPath); err != nil {
				return err
			}
		} else {
			if err := c.sftpClient.Remove(fullPath); err != nil {
				return err
			}
		}
	}

	return c.sftpClient.RemoveDirectory(dir)
}

// Mkdir 创建目录
func (c *Client) Mkdir(dir string) error {
	dir = c.resolvePath(dir)
	err := c.sftpClient.Mkdir(dir)
	if err == nil {
		// 清除父目录缓存
		c.invalidateDirCache(path.Dir(dir))
	}
	return err
}

// Rename 重命名文件或目录
func (c *Client) Rename(oldPath, newPath string) error {
	oldPath = c.resolvePath(oldPath)
	newPath = c.resolvePath(newPath)
	err := c.sftpClient.Rename(oldPath, newPath)
	if err == nil {
		// 清除相关目录缓存
		c.invalidateDirCache(path.Dir(oldPath))
		c.invalidateDirCache(path.Dir(newPath))
	}
	return err
}

// Stat 获取文件信息
func (c *Client) Stat(remotePath string) (os.FileInfo, error) {
	remotePath = c.resolvePath(remotePath)
	return c.sftpClient.Stat(remotePath)
}

// ListCompletion 获取路径补全候选列表
// 返回基于用户输入prefix的完整候选路径（保持prefix的格式：绝对/相对）
func (c *Client) ListCompletion(prefix string) []string {
	// 解析目录和部分文件名
	resolvedPath := c.resolvePath(prefix)
	dir, partial := path.Split(resolvedPath)
	if dir == "" {
		dir = c.workDir
	}

	files, err := c.sftpClient.ReadDir(dir)
	if err != nil {
		return nil
	}

	// 提取用户输入的目录前缀部分
	userDir, _ := path.Split(prefix)

	var matches []string
	for _, file := range files {
		name := file.Name()
		// SFTP 服务器通常是 Linux/Unix，文件系统大小写敏感
		if strings.HasPrefix(name, partial) {
			// 构建候选项:保留用户输入的路径前缀格式
			candidate := userDir + name
			if file.IsDir() {
				candidate += "/"
			}
			matches = append(matches, candidate)
		}
	}

	return matches
}

// resolvePath 解析远程路径（相对路径转绝对路径）
func (c *Client) resolvePath(p string) string {
	if p == "" {
		return c.workDir
	}
	if p == "~" {
		// 获取远程用户主目录
		if home, err := c.sftpClient.Getwd(); err == nil {
			return home
		}
		return c.workDir
	}
	if strings.HasPrefix(p, "~/") {
		if home, err := c.sftpClient.Getwd(); err == nil {
			return path.Clean(path.Join(home, p[2:]))
		}
	}
	if path.IsAbs(p) {
		return path.Clean(p)
	}
	return path.Clean(path.Join(c.workDir, p))
}

// resolveLocalPath 解析本地路径（相对路径转绝对路径）
func (c *Client) resolveLocalPath(p string) string {
	if p == "" {
		return c.localWorkDir
	}
	// 处理 ~ 前缀（用户主目录）
	if p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return c.localWorkDir
	}
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Clean(filepath.Join(home, p[2:]))
		}
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(c.localWorkDir, p))
}

// UploadOptions 上传选项
type UploadOptions struct {
	Recursive    bool // 递归上传目录
	ShowProgress bool // 显示进度条
	Concurrency  int  // 并发数
}

// UploadGlob 使用 glob 模式匹配上传文件
func (c *Client) UploadGlob(pattern, remotePath string, opts *UploadOptions) (int, error) {
	if opts == nil {
		opts = &UploadOptions{ShowProgress: true, Concurrency: MaxConcurrentTransfers}
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

	// 过滤掉目录（除非启用递归模式）
	var files []string
	for _, match := range matches {
		stat, err := os.Stat(match)
		if err != nil {
			continue
		}
		if stat.IsDir() && !opts.Recursive {
			continue
		}
		files = append(files, match)
	}

	if len(files) == 0 {
		return 0, fmt.Errorf("no files to upload")
	}

	fmt.Printf("Found %d file(s) to upload\n", len(files))

	// 确保远程目标是目录
	remotePath = c.resolvePath(remotePath)

	// 并发上传
	return c.uploadFiles(files, remotePath, opts)
}

// UploadDir 递归上传整个目录
func (c *Client) UploadDir(localDir, remoteDir string, opts *UploadOptions) (int, error) {
	if opts == nil {
		opts = &UploadOptions{ShowProgress: true, Concurrency: MaxConcurrentTransfers}
	}

	localDir = c.resolveLocalPath(localDir)
	remoteDir = c.resolvePath(remoteDir)

	// 检查本地目录
	stat, err := os.Stat(localDir)
	if err != nil {
		return 0, fmt.Errorf("stat local dir: %w", err)
	}
	if !stat.IsDir() {
		return 0, fmt.Errorf("not a directory: %s", localDir)
	}

	// 收集所有文件
	var files []string
	err = filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("walk directory: %w", err)
	}

	if len(files) == 0 {
		return 0, fmt.Errorf("no files found in directory")
	}

	fmt.Printf("Uploading directory with %d file(s)\n", len(files))

	// 创建远程目录结构
	if err := c.ensureRemoteDir(remoteDir); err != nil {
		return 0, err
	}

	// 上传所有文件
	count := 0
	for _, localFile := range files {
		// 计算相对路径
		relPath, err := filepath.Rel(localDir, localFile)
		if err != nil {
			return count, err
		}
		remotePath := path.Join(remoteDir, filepath.ToSlash(relPath))

		// 确保远程父目录存在
		remoteParent := path.Dir(remotePath)
		if err := c.ensureRemoteDir(remoteParent); err != nil {
			return count, err
		}

		// 上传文件
		if err := c.UploadWithProgress(localFile, remotePath, opts.ShowProgress); err != nil {
			return count, fmt.Errorf("upload %s: %w", relPath, err)
		}
		count++
	}

	return count, nil
}

// DownloadOptions 下载选项
type DownloadOptions struct {
	Recursive    bool // 递归下载目录
	ShowProgress bool // 显示进度条
	Concurrency  int  // 并发数
}

// DownloadDir 递归下载整个目录
func (c *Client) DownloadDir(remoteDir, localDir string, opts *DownloadOptions) (int, error) {
	if opts == nil {
		opts = &DownloadOptions{ShowProgress: true, Concurrency: MaxConcurrentTransfers}
	}

	remoteDir = c.resolvePath(remoteDir)
	localDir = c.resolveLocalPath(localDir)

	// 检查远程目录
	stat, err := c.sftpClient.Stat(remoteDir)
	if err != nil {
		return 0, fmt.Errorf("stat remote dir: %w", err)
	}
	if !stat.IsDir() {
		return 0, fmt.Errorf("not a directory: %s", remoteDir)
	}

	// 收集所有文件
	type fileInfo struct {
		remotePath string
		localPath  string
	}
	var files []fileInfo

	var walk func(string, string) error
	walk = func(rPath, lPath string) error {
		entries, err := c.sftpClient.ReadDir(rPath)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			rFile := path.Join(rPath, entry.Name())
			lFile := filepath.Join(lPath, entry.Name())

			if entry.IsDir() {
				// 创建本地目录
				if err := os.MkdirAll(lFile, 0755); err != nil {
					return err
				}
				// 递归
				if err := walk(rFile, lFile); err != nil {
					return err
				}
			} else {
				files = append(files, fileInfo{rFile, lFile})
			}
		}
		return nil
	}

	// 确保本地目录存在
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return 0, fmt.Errorf("create local dir: %w", err)
	}

	if err := walk(remoteDir, localDir); err != nil {
		return 0, fmt.Errorf("walk remote directory: %w", err)
	}

	if len(files) == 0 {
		return 0, nil
	}

	fmt.Printf("Downloading directory with %d file(s)\n", len(files))

	// 下载所有文件
	count := 0
	for _, f := range files {
		if err := c.DownloadWithProgress(f.remotePath, f.localPath, opts.ShowProgress); err != nil {
			return count, fmt.Errorf("download %s: %w", f.remotePath, err)
		}
		count++
	}

	return count, nil
}

// DownloadGlob 使用 glob 模式匹配下载远程文件
func (c *Client) DownloadGlob(pattern, localPath string, opts *DownloadOptions) (int, error) {
	if opts == nil {
		opts = &DownloadOptions{ShowProgress: true, Concurrency: MaxConcurrentTransfers}
	}

	// 解析 glob 模式的基路径
	basePath := c.workDir
	fullPattern := pattern
	if !path.IsAbs(pattern) {
		fullPattern = path.Join(basePath, pattern)
	}

	// 查找匹配的远程文件
	matches, err := c.globRemote(fullPattern)
	if err != nil {
		return 0, fmt.Errorf("glob pattern: %w", err)
	}

	if len(matches) == 0 {
		return 0, fmt.Errorf("no files match pattern: %s", pattern)
	}

	// 过滤掉目录（除非启用递归模式）
	var files []string
	for _, match := range matches {
		stat, err := c.sftpClient.Stat(match)
		if err != nil {
			continue
		}
		if stat.IsDir() && !opts.Recursive {
			continue
		}
		files = append(files, match)
	}

	if len(files) == 0 {
		return 0, fmt.Errorf("no files to download")
	}

	fmt.Printf("Found %d file(s) to download\n", len(files))

	// 确保本地目标目录存在
	localPath = c.resolveLocalPath(localPath)
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return 0, fmt.Errorf("create local dir: %w", err)
	}

	// 下载文件
	return c.downloadFiles(files, localPath, opts)
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

// downloadFiles 下载多个文件到指定目录
func (c *Client) downloadFiles(files []string, localDir string, opts *DownloadOptions) (int, error) {
	count := 0
	for _, remoteFile := range files {
		stat, err := c.sftpClient.Stat(remoteFile)
		if err != nil {
			continue
		}

		if stat.IsDir() {
			// 递归下载目录
			localSubDir := filepath.Join(localDir, path.Base(remoteFile))
			n, err := c.DownloadDir(remoteFile, localSubDir, opts)
			if err != nil {
				return count, fmt.Errorf("download dir %s: %w", remoteFile, err)
			}
			count += n
		} else {
			// 下载单个文件
			localFile := filepath.Join(localDir, path.Base(remoteFile))
			if err := c.DownloadWithProgress(remoteFile, localFile, opts.ShowProgress); err != nil {
				return count, fmt.Errorf("download %s: %w", remoteFile, err)
			}
			count++
		}
	}

	return count, nil
}

// uploadFiles 并发上传多个文件
func (c *Client) uploadFiles(files []string, remotePath string, opts *UploadOptions) (int, error) {
	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = MaxConcurrentTransfers
	}
	if concurrency > len(files) {
		concurrency = len(files)
	}

	// 创建工作池
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	count := 0

	for _, localFile := range files {
		wg.Add(1)
		sem <- struct{}{} // 获取信号量

		go func(lf string) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			// 确定远程文件名
			rf := path.Join(remotePath, filepath.Base(lf))

			// 上传
			err := c.UploadWithProgress(lf, rf, opts.ShowProgress)

			mu.Lock()
			defer mu.Unlock()
			if err != nil && firstErr == nil {
				firstErr = err
			} else if err == nil {
				count++
			}
		}(localFile)
	}

	wg.Wait()
	return count, firstErr
}

// ensureRemoteDir 确保远程目录存在
func (c *Client) ensureRemoteDir(dir string) error {
	dir = c.resolvePath(dir)

	// 检查目录是否存在
	if _, err := c.sftpClient.Stat(dir); err == nil {
		return nil
	}

	// 递归创建父目录
	parent := path.Dir(dir)
	if parent != "/" && parent != "." {
		if err := c.ensureRemoteDir(parent); err != nil {
			return err
		}
	}

	// 创建目录
	err := c.sftpClient.Mkdir(dir)
	if err == nil {
		c.invalidateDirCache(parent)
	}
	return err
}

// ClearDirCache 清除所有目录缓存
func (c *Client) ClearDirCache() {
	c.cacheMu.Lock()
	c.dirCache = make(map[string]*dirCacheEntry)
	c.cacheMu.Unlock()
}

// invalidateDirCache 清除指定目录的缓存
func (c *Client) invalidateDirCache(dir string) {
	dir = c.resolvePath(dir)
	c.cacheMu.Lock()
	delete(c.dirCache, dir)
	c.cacheMu.Unlock()
}

// ExecuteRemote 在远程服务器执行命令（交互式）
func (c *Client) ExecuteRemote(command string, stdin io.Reader, stdout, stderr io.Writer) error {
	session, err := c.sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	// 绑定 stdin/stdout/stderr 实现交互
	session.Stdin = stdin
	session.Stdout = stdout
	session.Stderr = stderr

	// 在当前工作目录执行命令
	fullCommand := fmt.Sprintf("cd %s && %s", c.workDir, command)
	return session.Run(fullCommand)
}
