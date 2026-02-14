---
name: sftp-transfer
description: "my-sftp 文件传输的实现细节，包括单文件传输、并发传输、Glob 模式匹配、递归目录传输的完整流程"
updated: 2026-01-27
scope:
  - /client/transfer.go
  - /client/upload.go
  - /client/download.go
  - /client/common.go
  - /shell/shell.go
---

# SFTP File Transfer Specification

## Overview

my-sftp 的文件传输系统设计目标是提供**高性能、用户友好、可靠**的文件传输体验。核心特性包括：

1. **多种传输模式**：单文件、多文件、Glob 模式、递归目录
2. **并发传输**：智能并发控制，充分利用带宽
3. **可视化进度**：自适应的进度条显示
4. **错误处理**：详细的错误收集和报告

本规范详细说明了文件传输的实现机制和工作流程。

## Core Concepts

### Transfer Task

传输任务是文件传输的基本单元：

```go
type transferTask struct {
    localPath  string // 本地文件路径
    remotePath string // 远程文件路径
    isUpload   bool   // true=上传, false=下载
    size       int64  // 文件大小，用于进度显示
}
```

**设计原则**：
- 一个 task 对应一个文件的传输
- 目录传输会被展开为多个文件级 task
- Task 创建阶段不执行实际传输（任务收集与执行分离）

### Transfer Options

统一的传输选项配置：

```go
type TransferOptions struct {
    Recursive    bool // 是否递归处理目录
    ShowProgress bool // 是否显示进度条
    Concurrency  int  // 并发传输数（1-N）
    MaxDepth     int  // 最大递归深度（-1=无限）
}
```

**默认值**：
- `Recursive`: true
- `ShowProgress`: true
- `Concurrency`: 4 (MaxConcurrentTransfers)
- `MaxDepth`: -1 (无限深度)

### Transfer Modes

my-sftp 支持以下传输模式：

| 模式 | 用户命令示例 | 实现函数 |
|------|------------|---------|
| **单文件传输** | `put file.txt` | `UploadWithProgress()` |
| **多文件传输** | `put file1.txt file2.txt` | `UploadGlob()` → `executeTasks()` |
| **Glob 模式** | `put *.txt` | `UploadGlob()` → `executeTasks()` |
| **递归目录** | `put -r ./src` | `UploadGlob()` → `collectUploadTasks()` |

## Transfer Workflow

### Phase 1: Command Parsing (Shell Layer)

用户在 shell 中输入命令，Shell 层负责解析：

```go
// 示例：put -r src/ /remote/dest
func (s *Shell) cmdPut(args []string) error {
    // 1. 解析选项
    recursive := parseFlag(args, "-r")

    // 2. 提取路径参数
    sources, destination := parseArgs(args)

    // 3. 调用 Client 层 API
    if recursive {
        return s.client.UploadGlob(source, dest, &UploadOptions{
            Recursive: true,
            ...
        })
    }
}
```

**支持的选项**：
- `-r`: 递归传输目录

**路径解析规则**：
- 最后一个参数是目标路径
- 前面所有参数是源路径（可以是多个）

### Phase 2: Task Collection (Client Layer)

#### 2.1 单文件上传

```go
func (c *Client) Upload(localPath, remotePath string) error
```

**流程**：
1. 解析本地路径 (`ResolveLocalPath`)
2. 解析远程路径 (`ResolveRemotePath`)
3. 检查本地文件是否存在
4. 如果远程路径是目录，自动使用本地文件名
5. 调用 `UploadWithProgress()` 传输

#### 2.2 Glob 模式上传

```go
func (c *Client) UploadGlob(pattern, remotePath string, opts *UploadOptions) (int, error)
```

**流程**：
```
1. 构建完整的 glob pattern
   pattern = "*.txt"
   basePath = c.localWorkDir
   fullPattern = filepath.Join(basePath, pattern)

2. 使用 doublestar 库匹配本地文件
   matches, err := doublestar.FilepathGlob(fullPattern)
   // 支持 *, ?, [abc], **, {a,b} 等模式

3. 遍历匹配结果
   for match in matches:
       if isDir && !opts.Recursive:
           continue  // 非递归模式跳过目录

       if isDir && opts.Recursive:
           // 递归收集目录内所有文件
           tasks += collectUploadTasks(match, remoteSubDir, maxDepth, 0)
       else:
           // 添加单文件任务
           tasks.append(transferTask{
               localPath: match,
               remotePath: remoteFile,
               isUpload: true,
               size: stat.Size()
           })

4. 预创建远程目录结构
   dirs := collectRemoteDirsForUpload(tasks)
   ensureRemoteDirsExist(dirs)

5. 执行所有任务
   executeTasks(tasks, opts)
```

**Glob 匹配示例**：
- `*.txt`: 当前目录所有 txt 文件
- `**/*.go`: 当前目录及所有子目录的 go 文件
- `src/{client,shell}/*.go`: src/client 和 src/shell 的 go 文件

#### 2.3 递归目录上传

```go
func (c *Client) collectUploadTasks(localDir, remoteDir string, maxDepth, currentDepth int) ([]transferTask, error)
```

**递归遍历算法**：
```
function collectUploadTasks(localDir, remoteDir, maxDepth, currentDepth):
    tasks = []

    // 深度检查
    if maxDepth >= 0 && currentDepth > maxDepth:
        return tasks

    // 读取本地目录
    entries := os.ReadDir(localDir)

    for entry in entries:
        localPath = filepath.Join(localDir, entry.Name())
        remotePath = path.Join(remoteDir, entry.Name())

        if entry.IsDir():
            // 递归收集子目录
            subTasks := collectUploadTasks(
                localPath, remotePath,
                maxDepth, currentDepth + 1
            )
            tasks.append(subTasks)
        else:
            // 添加文件任务
            tasks.append(transferTask{
                localPath: localPath,
                remotePath: remotePath,
                isUpload: true,
                size: entry.Size()
            })

    return tasks
```

**深度控制示例**：
- `MaxDepth = -1`: 无限深度，递归所有子目录
- `MaxDepth = 0`: 仅当前目录的文件
- `MaxDepth = 1`: 当前目录 + 一层子目录

#### 2.4 下载流程

下载流程与上传类似，但有以下差异：

**远程 Glob 实现**：
```go
func (c *Client) globRemote(pattern string) ([]string, error)
```

由于 SFTP 服务器不直接支持 glob，需要客户端实现：
```
1. 提取 pattern 的基路径（不含通配符部分）
   pattern = "/var/log/app-*.log"
   baseDir = "/var/log"

2. 递归遍历基路径

3. 对每个文件/目录，使用 doublestar.Match() 检查是否匹配

4. 返回所有匹配路径
```

**示例**：
```go
// 用户命令：get /var/log/nginx/*.log
pattern = "/var/log/nginx/*.log"
baseDir = "/var/log/nginx"

c.sftpClient.ReadDir("/var/log/nginx")
for entry in entries:
    fullPath = "/var/log/nginx/" + entry.Name()
    if doublestar.Match("*.log", entry.Name()):
        matches.append(fullPath)
```

### Phase 3: Task Execution (Unified Engine)

#### 3.1 执行引擎设计

```go
func (c *Client) executeTasks(tasks []transferTask, opts *TransferOptions) (int, error)
```

**核心职责**：
1. 并发控制
2. 进度显示
3. 错误收集
4. panic 保护

**执行流程**：
```
1. 确定并发数
   concurrency = min(opts.Concurrency, len(tasks))

2. 创建信号量和同步原语
   sem := make(chan struct{}, concurrency)
   var wg sync.WaitGroup
   var mu sync.Mutex
   var errs []error
   var successCount atomic.Int32

3. 决定进度条模式
   if concurrency > 1:
       globalBar = progressbar.New(len(tasks))  // 显示总体进度
   else:
       showFileProgress = true  // 每个文件显示进度

4. 启动 goroutines
   for task in tasks:
       wg.Add(1)
       sem <- struct{}{}  // 获取信号量

       go func(t transferTask):
           defer wg.Done()
           defer func() { <-sem }()  // 释放信号量

           // panic 保护
           defer recover()

           // 执行传输
           if t.isUpload:
               err = c.UploadWithProgress(t.localPath, t.remotePath, showFileProgress)
           else:
               err = c.DownloadWithProgress(t.remotePath, t.localPath, showFileProgress)

           // 错误处理
           if err != nil:
               mu.Lock()
               errs.append(err)
               mu.Unlock()
           else:
               successCount.Add(1)

           // 更新全局进度条
           if globalBar != nil:
               globalBar.Add(1)
       (task)

5. 等待完成
   wg.Wait()

6. 返回结果
   return successCount, errors.Join(errs...)
```

#### 3.2 单文件传输实现

```go
func (c *Client) UploadWithProgress(localPath, remotePath string, showProgress bool) error
```

**传输步骤**：
```
1. 解析路径
   localPath = c.ResolveLocalPath(localPath)
   remotePath = c.ResolveRemotePath(remotePath)

2. 打开本地文件
   stat, _ := os.Stat(localPath)
   srcFile, _ := os.Open(localPath)
   defer srcFile.Close()

3. 处理远程路径
   if remoteIsDir:
       remotePath = path.Join(remotePath, filepath.Base(localPath))

4. 创建远程文件
   dstFile, _ := c.sftpClient.Create(remotePath)
   defer dstFile.Close()

5. 获取缓冲区
   buf := c.getBuffer()  // 从 Pool 获取 512KB buffer
   defer c.putBuffer(buf)

6. 执行传输
   if showProgress:
       bar := progressbar.DefaultBytes(stat.Size(), "Uploading ...")
       io.CopyBuffer(io.MultiWriter(dstFile, bar), srcFile, buf)
   else:
       io.CopyBuffer(dstFile, srcFile, buf)
```

**下载流程类似**，但方向相反：
```
srcFile := c.sftpClient.Open(remotePath)
dstFile := os.Create(localPath)
io.CopyBuffer(dstFile, srcFile, buf)
```

### Phase 4: Directory Structure Preparation

#### 4.1 远程目录创建

```go
func (c *Client) ensureRemoteDirsExist(dirs []string) error
```

**问题**：并发上传时，多个 goroutine 可能同时需要同一个远程目录

**解决方案**：使用 `singleflight.Group` 去重

```go
func (c *Client) ensureRemoteDirExists(dir string) error {
    // singleflight 保证同一 dir 只创建一次
    _, err, _ := c.dirCreateGroup.Do(dir, func() (interface{}, error) {
        return nil, c.sftpClient.MkdirAll(dir)
    })
    return err
}
```

**工作原理**：
1. 第一个调用 `Do(dir, fn)` 的 goroutine 执行 `fn`
2. 其他同时调用 `Do(dir, fn)` 的 goroutine 等待
3. 所有等待的 goroutine 共享第一个的执行结果

**示例**：
```
goroutine 1: ensureRemoteDirExists("/remote/a/b")
goroutine 2: ensureRemoteDirExists("/remote/a/b")
goroutine 3: ensureRemoteDirExists("/remote/a/b")

结果：MkdirAll 只执行一次，三个 goroutine 都收到结果
```

#### 4.2 收集所需目录

```go
func (c *Client) collectRemoteDirsForUpload(tasks []transferTask) []string
```

**算法**：
```
dirs = set()

for task in tasks:
    dir = path.Dir(task.remotePath)
    dirs.add(dir)

    // 添加所有父目录
    while dir != "/" && dir != ".":
        dir = path.Dir(dir)
        dirs.add(dir)

// 排序：确保父目录在子目录前面
sort(dirs)

return dirs
```

**示例**：
```
tasks = [
    {remotePath: "/a/b/c/file1.txt"},
    {remotePath: "/a/b/d/file2.txt"}
]

collectRemoteDirsForUpload(tasks) = [
    "/a",
    "/a/b",
    "/a/b/c",
    "/a/b/d"
]
```

## Progress Display

my-sftp 提供了**统一且多维**的文件传输进度展示。无论单文件还是多文件任务，进度条均基于**字节数**，保持一致的视觉体验。

### Progress Visualization Principles

1.  **字节级精度 (Byte-level Precision)**: 进度条基于总传输字节数，通过 `io.MultiWriter` 结合 `progressbar` 实现实时刷新，避免因传输大文件而导致的进度条"卡死"现象。
2.  **统一体验 (Unified UX)**: 单文件和多文件传输采用相同的进度条格式、单位显示、速度估算和 ETA。
3.  **多维反馈 (Multi-dimensional Feedback)**:
    -   **字节进度**: 已传输数据量 / 总数据量。
    -   **文件进度**: 进度条描述包含 `(n/N files)`，显示已完成文件数。
    -   **当前任务**: 动态显示正在传输的文件名。
    -   **完成清单**: 每个文件完成后，在控制台打印一条带 `✓` 标记的记录及文件大小。

### Technical Implementation

-   **核心类库**: `github.com/schollz/progressbar/v3`
-   **单文件封装**: `Upload()` 和 `Download()` 内部创建 `progressbar`。
-   **并发引擎**: `executeTasks()` 统一管理全局进度，预先遍历任务列表计算 `totalBytes`，并使用 `atomic.Int32` 安全地跟踪完成文件数。
-   **实时描述更新**: 通过 `globalBar.Describe(fmt.Sprintf(...))` 动态修改进度条左侧的文本（包含文件名和计数）。

### Display Examples

#### 1. Single File Progress

```
Downloading file.txt (1/1 files)  50.2 MB / 100.4 MB [=====>      ] 50% 2.1 MB/s
```

#### 2. Multiple Files Global Progress (Concurrent)

```
✓ file1.txt (1.2 MB)
✓ file2.txt (3.5 MB)
Transferring file3.txt (2/10 files)  526.2 MB / 2.5 GB [===>         ] 21% 15.2 MB/s ETA 2m15s
```

**更新特性**:
-   **完成清单**: 在进度条上方通过 `\r\033[K` 清除当前行并打印 `✓` 记录，随后进度条在下一行重新渲染。
-   **动态文件名**: 进度条左侧描述实时显示当前并发池中最新开始或刚完成的文件。

## Concurrency Control

### Semaphore Pattern

```go
sem := make(chan struct{}, concurrency)

for task := range tasks {
    sem <- struct{}{}  // 获取令牌（阻塞直到有空位）

    go func(t transferTask) {
        defer func() { <-sem }()  // 释放令牌

        // 执行任务...
    }(task)
}
```

**优势**：
- 精确控制并发数量
- 避免创建过多 goroutine
- 简单高效

### Error Aggregation

```go
var mu sync.Mutex
var errs []error

// 在 goroutine 中
if err != nil {
    mu.Lock()
    errs = append(errs, fmt.Errorf("upload %s: %w", localPath, err))
    mu.Unlock()
}

// 最后返回
return errors.Join(errs...)
```

**errors.Join()** (Go 1.20+)：
- 合并多个错误
- 保留所有错误信息
- 支持 `errors.Is()` 和 `errors.As()` 检查

## Performance Optimizations

### 1. Buffer Pool

**问题**：每次传输分配 512KB 缓冲区会导致频繁 GC

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
io.CopyBuffer(dst, src, buf)
```

**效果**：减少 GC 压力，提升传输性能

### 2. Directory Cache

**问题**：频繁列出远程目录很慢（网络延迟）

**解决方案**：
```go
type dirCacheEntry struct {
    files    []os.FileInfo
    cachedAt time.Time
}

dirCache map[string]*dirCacheEntry
```

**缓存策略**：
- 缓存时间：30 秒
- 切换目录时清空缓存
- 并发安全（RWMutex）

### 3. Concurrent Transfer

**默认并发数**：4

**自适应调整**：
```go
if concurrency > len(tasks) {
    concurrency = len(tasks)  // 避免创建过多 goroutine
}
```

**性能提升**：
- 小文件：减少连接建立开销
- 大文件：充分利用带宽

## Error Handling

### Panic Protection

```go
defer func() {
    if r := recover(); r != nil {
        mu.Lock()
        errs = append(errs, fmt.Errorf("panic: %v\nstack: %s", r, debug.Stack()))
        mu.Unlock()
    }
}()
```

**保护范围**：每个 goroutine 独立保护，避免一个任务 panic 影响其他任务

### Error Reporting

**成功/失败统计**：
```go
successCount, err := c.executeTasks(tasks, opts)

if err != nil {
    fmt.Printf("Completed with errors: %d/%d successful\n", successCount, len(tasks))
    fmt.Printf("Errors:\n%v\n", err)
}
```

**输出示例**：
```
Completed with errors: 38/42 successful
Errors:
upload file1.txt: permission denied
upload file2.txt: disk full
upload file3.txt: connection timeout
upload file4.txt: no such file or directory
```

## Edge Cases

### Case 1: Empty Directory

**场景**：`put -r empty_dir/ /remote/`

**处理**：
```
collectUploadTasks(empty_dir) → tasks = []
executeTasks([]) → return (0, nil)
```

**结果**：不传输任何文件，但远程目录会被创建

### Case 2: Duplicate Files

**场景**：`put file.txt file.txt /remote/`

**处理**：
- Glob 去重在收集阶段完成
- 第二次上传会覆盖第一次

### Case 3: Symbolic Links

**当前行为**：
- 上传：跟随符号链接（传输目标文件）
- 下载：不处理符号链接

**未来改进**：添加 `--no-dereference` 选项保留符号链接

### Case 4: Permission Errors

**场景**：远程目录只读

**处理**：
```
ensureRemoteDirExists() → err: permission denied
→ executeTasks() 提前返回错误
→ 不执行任何传输
```

### Case 5: Network Interruption

**当前行为**：传输失败，返回错误

**未来改进**：
- 添加重试机制
- 支持断点续传

## Examples

### Example 1: Simple Upload

```bash
> put local.txt
Uploading local.txt  1.2 MB / 1.2 MB [============] 100% 2.3 MB/s
```

**内部流程**：
1. `cmdPut(["local.txt"])`
2. `UploadGlob("local.txt", ".", opts)`
3. Glob 匹配到 1 个文件
4. `executeTasks([task], opts)`
5. `UploadWithProgress("local.txt", "/remote/workdir/local.txt", true)`

### Example 2: Glob Upload

```bash
> put *.log /var/log/backup/
Found 15 file(s) to upload
Transferring files 15/15 [============] 100% 5.2 MB/s
```

**内部流程**：
1. `UploadGlob("*.log", "/var/log/backup/", opts)`
2. `doublestar.FilepathGlob("*.log")` → 15 个文件
3. 收集 15 个 transferTask
4. 创建远程目录 `/var/log/backup/`
5. 并发执行 15 个任务（并发数=4）

### Example 3: Recursive Upload

```bash
> put -r src/ /remote/project/
Found 127 file(s) to upload
Transferring files 127/127 [============] 100% 3.8 MB/s
```

**内部流程**：
1. `cmdPut(["-r", "src/", "/remote/project/"])`
2. `UploadGlob("src/", "/remote/project/", {Recursive: true})`
3. `collectUploadTasks("src/", "/remote/project/", -1, 0)` → 127 个任务
4. `collectRemoteDirsForUpload()` → 收集所有目录
5. `ensureRemoteDirsExist()` → 批量创建目录
6. `executeTasks(127 tasks, opts)` → 并发传输

### Example 4: Download with Pattern

```bash
> get /var/log/app-202601*.log ./logs/
Found 31 file(s) to download
Transferring files 31/31 [============] 100% 4.5 MB/s
```

**内部流程**：
1. `DownloadGlob("/var/log/app-202601*.log", "./logs/", opts)`
2. `globRemote("/var/log/app-202601*.log")` → 遍历 `/var/log/`，匹配 31 个文件
3. 收集 31 个 transferTask
4. 创建本地目录 `./logs/`
5. 并发下载 31 个文件

## References

- [pkg/sftp 文档](https://pkg.go.dev/github.com/pkg/sftp)
- [doublestar Glob 模式](https://pkg.go.dev/github.com/bmatcuk/doublestar/v4)
- [singleflight 使用指南](https://pkg.go.dev/golang.org/x/sync/singleflight)
- [progressbar 库](https://pkg.go.dev/github.com/schollz/progressbar/v3)
