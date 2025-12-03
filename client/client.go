package client

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/singleflight"
)

const (
	// BufferSize 传输缓冲区大小 (512KB)
	BufferSize = 512 * 1024
	// MaxConcurrentTransfers 最大并发传输数
	MaxConcurrentTransfers = 4
	// DirCacheTimeout 目录列表缓存超时时间
	DirCacheTimeout = 30 * time.Second
	// DirLockShards = 64 //目录锁分片数量
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
	bufferPool   *sync.Pool                // 统一的 buffer pool，减少 GC 压力
	// dirLocks       [DirLockShards]sync.Mutex // 分片锁，用于目录创建的并发控制, 引入 singleflight 后也许不需要了
	dirCreateGroup singleflight.Group // 确保同一目录只创建一次
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

// getBuffer 安全地从 buffer pool 获取缓冲区
func (c *Client) getBuffer() []byte {
	buf := c.bufferPool.Get()
	if b, ok := buf.(*[]byte); ok {
		return *b
	}
	// 后备方案：如果类型断言失败，创建新的缓冲区
	return make([]byte, BufferSize)
}

// putBuffer 将缓冲区归还到 pool
func (c *Client) putBuffer(buf []byte) {
	c.bufferPool.Put(&buf)
}

// getDirLock 通过哈希获取目录专属的分片锁
// 似乎不需要了, 因为引入 Singleflight，进入到内部代码块的已经是单线程环境了，临界区内不存在竞争
// func (c *Client) getDirLock(dir string) *sync.Mutex {
// 	h := fnv.New32a()
// 	h.Write([]byte(dir))
// 	idx := h.Sum32() % DirLockShards
// 	return &c.dirLocks[idx]
// }
