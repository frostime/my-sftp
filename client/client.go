package client

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/pkg/sftp"
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
	bufferPool   *sync.Pool                //统一的 buff，多文件下载情况下减少 GC 压力
	dirCreateMu  sync.Map                  // key: dirPath, value: *sync.Mutex
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
		bufferPool: &sync.Pool{
			New: func() interface{} {
				buf := make([]byte, BufferSize)
				return &buf
			},
		},
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

// func (c *Client) getBuffer() []byte {
// 	buf := c.bufferPool.Get()
// 	if b, ok := buf.(*[]byte); ok {
// 		return *b
// 	}
// 	// 后备方案
// 	return make([]byte, BufferSize)
// }
// bufPtr := c.bufferPool.Get().(*[]byte)  // 未检查类型断言是否成功会不会出错?
