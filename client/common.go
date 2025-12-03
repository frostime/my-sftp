package client

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

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
	targetPath := c.ResolveLocalPath(dir)
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
	targetPath := c.ResolveLocalPath(dir)
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
	dir = c.ResolveLocalPath(dir)
	return os.Mkdir(dir, 0755)
}

// Chdir 切换工作目录
func (c *Client) Chdir(dir string) error {
	targetPath := c.ResolveRemotePath(dir)
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
	targetPath := c.ResolveRemotePath(dir)

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

// Remove 删除文件或目录
func (c *Client) Remove(remotePath string) error {
	remotePath = c.ResolveRemotePath(remotePath)
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
	dir = c.ResolveRemotePath(dir)
	err := c.sftpClient.Mkdir(dir)
	if err == nil {
		// 清除父目录缓存
		c.invalidateDirCache(path.Dir(dir))
	}
	return err
}

// Rename 重命名文件或目录
func (c *Client) Rename(oldPath, newPath string) error {
	oldPath = c.ResolveRemotePath(oldPath)
	newPath = c.ResolveRemotePath(newPath)
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
	remotePath = c.ResolveRemotePath(remotePath)
	return c.sftpClient.Stat(remotePath)
}

// ListCompletion 获取路径补全候选列表
// 返回基于用户输入prefix的完整候选路径（保持prefix的格式：绝对/相对）
func (c *Client) ListCompletion(prefix string) []string {
	// 解析目录和部分文件名
	resolvedPath := c.ResolveRemotePath(prefix)
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

// ResolveRemotePath 解析远程路径（相对路径转绝对路径）
func (c *Client) ResolveRemotePath(p string) string {
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

// ResolveLocalPath 解析本地路径（相对路径转绝对路径）
func (c *Client) ResolveLocalPath(p string) string {
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

// ClearDirCache 清除所有目录缓存
func (c *Client) ClearDirCache() {
	c.cacheMu.Lock()
	c.dirCache = make(map[string]*dirCacheEntry)
	c.cacheMu.Unlock()
}

// invalidateDirCache 清除指定目录的缓存
func (c *Client) invalidateDirCache(dir string) {
	dir = c.ResolveRemotePath(dir)
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
