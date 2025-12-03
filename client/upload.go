package client

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/schollz/progressbar/v3"
)

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
