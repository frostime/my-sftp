package client

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Client SFTP 客户端封装
type Client struct {
	sshClient    *ssh.Client
	sftpClient   *sftp.Client
	workDir      string // 远程当前工作目录
	localWorkDir string // 本地当前工作目录
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
	return nil
}

// List 列出目录内容
func (c *Client) List(dir string) ([]os.FileInfo, error) {
	targetPath := c.resolvePath(dir)
	return c.sftpClient.ReadDir(targetPath)
}

// Download 下载文件
func (c *Client) Download(remotePath, localPath string) error {
	remotePath = c.resolvePath(remotePath)
	localPath = c.resolveLocalPath(localPath)

	srcFile, err := c.sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("open remote: %w", err)
	}
	defer srcFile.Close()

	// 如果本地路径是目录，使用远程文件名
	if stat, err := os.Stat(localPath); err == nil && stat.IsDir() {
		localPath = filepath.Join(localPath, path.Base(remotePath))
	}

	dstFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create local: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// Upload 上传文件
func (c *Client) Upload(localPath, remotePath string) error {
	localPath = c.resolveLocalPath(localPath)
	remotePath = c.resolvePath(remotePath)

	srcFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local: %w", err)
	}
	defer srcFile.Close()

	// 如果远程路径是目录，使用本地文件名
	if stat, err := c.sftpClient.Stat(remotePath); err == nil && stat.IsDir() {
		remotePath = path.Join(remotePath, filepath.Base(localPath))
	}

	dstFile, err := c.sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("create remote: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// Remove 删除文件或目录
func (c *Client) Remove(remotePath string) error {
	remotePath = c.resolvePath(remotePath)
	stat, err := c.sftpClient.Stat(remotePath)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		// 递归删除目录
		return c.removeDir(remotePath)
	}
	return c.sftpClient.Remove(remotePath)
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
	return c.sftpClient.Mkdir(dir)
}

// Rename 重命名文件或目录
func (c *Client) Rename(oldPath, newPath string) error {
	oldPath = c.resolvePath(oldPath)
	newPath = c.resolvePath(newPath)
	return c.sftpClient.Rename(oldPath, newPath)
}

// Stat 获取文件信息
func (c *Client) Stat(remotePath string) (os.FileInfo, error) {
	remotePath = c.resolvePath(remotePath)
	return c.sftpClient.Stat(remotePath)
}

// ListCompletion 获取路径补全候选列表
func (c *Client) ListCompletion(prefix string) []string {
	dir, partial := path.Split(c.resolvePath(prefix))
	if dir == "" {
		dir = c.workDir
	}

	files, err := c.sftpClient.ReadDir(dir)
	if err != nil {
		return nil
	}

	var matches []string
	for _, file := range files {
		name := file.Name()
		if strings.HasPrefix(name, partial) {
			fullPath := path.Join(dir, name)
			// 如果是目录，添加尾部斜杠
			if file.IsDir() {
				fullPath += "/"
			}
			// 返回相对路径或绝对路径
			if strings.HasPrefix(prefix, "/") {
				matches = append(matches, fullPath)
			} else {
				rel, _ := filepath.Rel(c.workDir, fullPath)
				matches = append(matches, rel)
			}
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
