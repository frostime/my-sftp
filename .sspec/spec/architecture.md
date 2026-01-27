---
name: architecture
description: "my-sftp 项目整体架构设计，包括模块划分、数据流转、并发模型和核心设计模式"
updated: 2026-01-27
scope:
  - /main.go
  - /shell/**
  - /client/**
  - /config/**
  - /completer/**
---

# Architecture Specification

## Overview

my-sftp 采用分层架构设计，将程序分为四个核心模块：

1. **Main Layer (main.go)**: 程序入口，负责连接初始化和认证
2. **Shell Layer (shell/)**: 交互式命令解析和执行
3. **Client Layer (client/)**: SFTP 操作封装和文件传输引擎
4. **Supporting Layers**: Config 解析、自动补全等辅助功能

**设计原则**：
- **职责分离**：每个模块职责清晰，便于维护和测试
- **并发安全**：使用统一的并发控制模式，避免竞态条件
- **性能优化**：缓存、Buffer Pool、并发传输等优化手段

## Module Breakdown

### 1. Main Layer (main.go)

**职责**：
- 解析命令行参数
- 建立 SSH/SFTP 连接
- 处理 SSH 认证（密钥 + 密码回退）
- Host Key 验证和 known_hosts 管理

**关键流程**：

```
用户输入 destination
    ↓
解析配置（SSH config 或 user@host:port）
    ↓
准备认证方法（密钥优先 → 密码回退）
    ↓
创建 SSH ClientConfig + HostKeyCallback
    ↓
建立 SSH 连接 → 创建 SFTP Client
    ↓
启动交互式 Shell
```

**认证优先级**：
1. 指定的 IdentityFile（来自 SSH config 或命令行）
2. 默认密钥位置：`~/.ssh/id_rsa`, `~/.ssh/id_ed25519`, `~/.ssh/id_dsa`
3. 密码验证（交互式输入）

### 2. Shell Layer (shell/)

**核心类型**：
```go
type Shell struct {
    client    *client.Client      // SFTP 客户端
    rl        *readline.Instance  // readline 实例
    completer *completer.Completer // 自动补全器
}
```

**职责**：
- 维护交互式 REPL 循环
- 解析用户命令（支持引号、转义字符）
- 路由命令到对应的处理函数
- 支持远程命令执行 (`!`) 和本地命令执行 (`!!`)

**命令分类**：
- **远程操作**：`ls`, `cd`, `pwd`, `get`, `put`, `rm`, `mkdir`, `rmdir`, `mv`, `stat`
- **本地操作**：`lpwd`, `lcd`, `lls`, `lmkdir`
- **系统命令**：`help`, `exit`

**命令解析特性**：
- 支持单引号/双引号包裹的参数
- 支持反斜杠转义
- 自动处理空格分隔的多个参数

### 3. Client Layer (client/)

#### 核心结构 (client.go)

```go
type Client struct {
    sshClient      *ssh.Client
    sftpClient     *sftp.Client
    workDir        string                    // 远程工作目录
    localWorkDir   string                    // 本地工作目录
    dirCache       map[string]*dirCacheEntry // 目录缓存
    cacheMu        sync.RWMutex              // 缓存锁
    bufferPool     *sync.Pool                // 传输缓冲区池
    dirCreateGroup singleflight.Group        // 目录创建去重
}
```

**常量配置**：
- `BufferSize = 512KB`：传输缓冲区大小
- `MaxConcurrentTransfers = 4`：默认最大并发数
- `DirCacheTimeout = 30s`：目录缓存过期时间

#### 文件传输架构 (transfer.go, upload.go, download.go)

**核心设计模式：任务收集 + 统一执行引擎**

```
用户命令 (get/put)
    ↓
解析参数（路径、glob 模式、-r 选项）
    ↓
收集传输任务（不执行）
  - collectUploadTasks()
  - collectDownloadTasks()
    ↓
确保目录结构存在
    ↓
executeTasks() 统一执行引擎
  - 并发控制（信号量）
  - 进度条管理
  - 错误收集
```

**任务结构**：
```go
type transferTask struct {
    localPath  string
    remotePath string
    isUpload   bool
    size       int64  // 用于进度显示
}
```

**传输选项**：
```go
type TransferOptions struct {
    Recursive    bool // 递归目录
    ShowProgress bool // 显示进度
    Concurrency  int  // 并发数
    MaxDepth     int  // 最大递归深度（-1=无限）
}
```

#### 并发模型

**1. 统一执行引擎 (executeTasks)**：
- 唯一的并发传输入口点，避免并发嵌套
- 使用信号量 (`sem chan struct{}`) 控制并发数
- 使用 `sync.WaitGroup` 等待所有任务完成
- 统一错误收集和成功计数

**2. 目录创建去重 (singleflight)**：
```go
// 保证同一目录只创建一次，即使多个 goroutine 同时请求
c.dirCreateGroup.Do(dir, func() (interface{}, error) {
    return nil, c.sftpClient.MkdirAll(dir)
})
```

**3. Buffer Pool**：
```go
bufferPool: &sync.Pool{
    New: func() interface{} {
        buf := make([]byte, BufferSize)
        return &buf
    },
}
```
- 减少 GC 压力
- 复用 512KB 传输缓冲区

**4. 进度条管理**：
- **单文件传输**（并发=1）：显示文件级进度条
- **多文件并发**（并发>1）：显示总体进度条（文件数 + 字节数）

#### 路径解析 (common.go)

**远程路径解析**：
```go
func (c *Client) ResolveRemotePath(p string) string
```
- 绝对路径：直接使用
- 相对路径：基于 `c.workDir` 解析
- 支持 `.` 和 `..` 规范化

**本地路径解析**：
```go
func (c *Client) ResolveLocalPath(p string) string
```
- 使用 `filepath.IsAbs()` 和 `filepath.Join()`
- 跨平台兼容（Windows/Unix）

#### Glob 模式支持

**本地 Glob** (upload.go)：
```go
// 使用 doublestar 库支持 ** 递归匹配
matches, err := doublestar.FilepathGlob(fullPattern)
```

**远程 Glob** (download.go, common.go)：
```go
func (c *Client) globRemote(pattern string) ([]string, error)
```
- 自实现远程 glob 匹配
- 支持 `*`, `?`, `[abc]`, `**` 等模式
- 递归遍历远程目录

### 4. Config Layer (config/)

**核心类型**：
```go
type SSHConfig struct {
    Host         string
    Port         int
    User         string
    IdentityFile string
}
```

**功能**：
1. **LoadSSHConfig(alias)**: 从 `~/.ssh/config` 加载配置
2. **ParseDestination(dest)**: 解析 `user@host:port` 格式
3. **FindDefaultKeys()**: 查找默认 SSH 密钥

**解析优先级**：
```
SSH config alias → user@host:port → 默认值
```

### 5. Completer Layer (completer/)

**核心接口**：
```go
type ClientInterface interface {
    ListCompletion(prefix string) []string
    GetLocalwd() string
}
```

**补全逻辑**：
1. **命令补全**：补全 SFTP 命令（ls, cd, get, put...）
2. **远程路径补全**：调用 `client.ListCompletion()`
3. **本地路径补全**：读取本地文件系统

**自动补全流程**：
```
用户按 TAB
    ↓
readline 调用 Completer.Do()
    ↓
解析当前命令和参数
    ↓
根据命令类型选择补全策略
  - cd/ls/get: 远程路径
  - put/lcd: 本地路径
    ↓
返回候选列表
```

## Data Flow

### 文件上传流程

```
用户: put file.txt /remote/path
    ↓
Shell 解析命令 → cmdPut(args)
    ↓
Client.UploadGlob(pattern, remotePath, opts)
    ↓
1. Glob 匹配本地文件
2. 收集 transferTask 列表
3. 确保远程目录存在（singleflight）
    ↓
executeTasks(tasks, opts)
    ↓
并发执行：
  - 每个 task 调用 UploadWithProgress()
  - 读取本地文件 → 写入远程文件
  - 使用 Buffer Pool 的缓冲区
  - 更新进度条
```

### 文件下载流程

```
用户: get *.log ./logs
    ↓
Shell 解析命令 → cmdGet(args)
    ↓
Client.DownloadGlob(pattern, localPath, opts)
    ↓
1. 远程 Glob 匹配
2. 收集 transferTask 列表
3. 确保本地目录存在
    ↓
executeTasks(tasks, opts)
    ↓
并发执行：
  - 每个 task 调用 DownloadWithProgress()
  - 读取远程文件 → 写入本地文件
  - 使用 Buffer Pool 的缓冲区
  - 更新进度条
```

### 目录递归传输流程

```
用户: put -r ./src /remote/dest
    ↓
解析 -r 选项 → opts.Recursive = true
    ↓
collectUploadTasks(localDir, remoteDir, maxDepth, currentDepth)
    ↓
递归遍历本地目录：
  - 深度优先遍历
  - 每个文件生成一个 transferTask
  - 记录相对路径信息
    ↓
批量创建远程目录结构
    ↓
executeTasks() 并发上传所有文件
```

## Concurrency Safety

### 目录缓存 (dirCache)

**问题**：多个 goroutine 可能同时读取/更新缓存

**解决方案**：
```go
c.cacheMu.RLock()
entry := c.dirCache[path]
c.cacheMu.RUnlock()

c.cacheMu.Lock()
c.dirCache[path] = newEntry
c.cacheMu.Unlock()
```

### 目录创建竞争

**问题**：并发上传时，多个文件可能需要同一个远程目录

**旧方案（已废弃）**：分片锁 (64 个 mutex)
```go
dirLocks [64]sync.Mutex
getDirLock(dir) *sync.Mutex  // 通过哈希选择锁
```

**新方案（当前）**：singleflight
```go
c.dirCreateGroup.Do(dir, func() (interface{}, error) {
    return nil, c.sftpClient.MkdirAll(dir)
})
```
- 保证同一目录只创建一次
- 等待的 goroutine 共享结果
- 更简洁、更高效

### Buffer Pool

**问题**：每个文件传输都分配 512KB 缓冲区会导致大量 GC

**解决方案**：
```go
bufferPool: &sync.Pool{
    New: func() interface{} {
        buf := make([]byte, BufferSize)
        return &buf
    },
}

// 使用
buf := c.getBuffer()
defer c.putBuffer(buf)
```

## Design Patterns

### 1. 任务收集 + 统一执行模式

**为什么不在收集阶段直接传输？**
- 避免并发嵌套（收集过程中启动 goroutine）
- 统一控制并发数和进度显示
- 便于预先创建目录结构
- 更容易处理错误和取消操作

### 2. 选项模式 (Options Pattern)

```go
type TransferOptions struct {
    Recursive    bool
    ShowProgress bool
    Concurrency  int
    MaxDepth     int
}

func DefaultTransferOptions() *TransferOptions { ... }
```

- 提供合理的默认值
- 灵活配置行为
- 便于扩展新选项

### 3. 接口抽象

```go
type ClientInterface interface {
    ListCompletion(prefix string) []string
    GetLocalwd() string
}
```

- Completer 不依赖具体的 Client 实现
- 便于测试（可以 mock）

## Performance Optimizations

1. **并发传输**：4 个并发任务，充分利用带宽
2. **Buffer Pool**：复用 512KB 缓冲区，减少 GC
3. **目录缓存**：30 秒缓存，减少重复的远程查询
4. **Singleflight**：目录创建去重，避免重复操作

## References

- [分层架构设计模式](https://en.wikipedia.org/wiki/Multitier_architecture)
- [singleflight 文档](https://pkg.go.dev/golang.org/x/sync/singleflight)
- [sync.Pool 最佳实践](https://pkg.go.dev/sync#Pool)
