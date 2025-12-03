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
