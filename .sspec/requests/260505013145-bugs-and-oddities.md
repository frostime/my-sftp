---
created: 2026-05-05T01:31:45
status: OPEN
attach-change: null
tldr: "Code review发现的bug、不对称行为和代码异味，需要新AGENT调研并决策"
---

# Request: Bugs, Asymmetries, and Code Smells

## Background

对 `my-sftp` 进行 code-vs-spec 对齐审查时发现若干问题。用户已确认 #1 为已知 bug，其余需要新 AGENT 调研后给出修复方案或设计决策。

## Issues

### 1. Windows 反斜杠路径在 Shell 解析器中被破坏（已知 Bug）

**位置**: `shell/shell.go` — `parseCommandLine`

`parseCommandLine` 把 `\` 当作转义字符：`case '\': escaped = true`。用户在交互式 shell 中输入 Windows 绝对路径如 `C:\Users\file.txt` 时，反斜杠被吃掉，解析结果为 `C:Usersfile.txt`。

**影响**: Windows 用户无法直接粘贴本地路径到 shell 中执行 `put` / `lcd` 等命令。

**用户态度**: 已知 bug，需要修复。

---

### 2. `targetConflictKey` 上传时在 Windows 上错误地大小写不敏感

**位置**: `client/transfer.go` — `targetConflictKey`

```go
func targetConflictKey(task transferTask) string {
    // ...
    if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
        key = strings.ToLower(key)   // 上传的 target 是 remote Unix 路径
    }
    return key
}
```

上传的 target 是 **remote Unix 路径**（`path.Clean(task.remotePath)`），但代码根据**本地** OS 判断是否转小写。Windows 客户端上传时，`Remote/A.txt` 和 `Remote/a.txt` 会被误判为冲突，而 remote 通常是大小写敏感的 Linux 文件系统。

**问题**: 这是有意统一跨平台体验，还是 bug？

---

### 3. 上传/下载空目录行为不对称

**位置**: `client/upload.go` — `UploadSources` vs `client/download.go` — `DownloadDir`

- **Upload 空目录**: `collectUploadTasks` 返回 `[]` → `UploadSources` 报错 `"no files found in directory"`
- **Download 空目录**: `DownloadSources` 返回 `0, nil` → `DownloadDir` 检测到 `count == 0` 后**额外创建本地空目录**

**问题**: 为什么下载空目录是合法行为（创建空目录），上传空目录是硬错误？是否应该对称处理？

---

### 4. `rmdir` 语义与 Unix `rmdir` 不符

**位置**: `shell/shell.go` — `cmdRmdir`

```go
func (s *Shell) cmdRmdir(args []string) error {
    return s.cmdRm(args)   // rmdir == rm
}
```

Help 文本说 `rmdir <dir>` 是"Remove directory"，但实际调用 `client.Remove`，它会**递归删除非空目录**（等同于 `rm -r`）。Unix 的 `rmdir` 语义是"仅删除空目录"。

**问题**: 这是故意简化为 `rm -r` 的别名，还是实现不完整？是否需要区分 `rm`（文件/空目录）和 `rmdir`（仅空目录）？

---

### 5. `formatSize` 与 `formatBytes` 重复实现且不一致

**位置**: `shell/shell.go` — `formatSize` vs `client/transfer.go` — `formatBytes`

| 函数 | 算法 | 精度 | 包 |
|------|------|------|-----|
| `formatSize` | `switch` + 硬编码阈值 | `%.2f` | `shell` |
| `formatBytes` | 循环除法 | `%.1f` | `client` |

同一项目两种人类可读大小实现。`executeTasks` 用 `formatBytes` 打印完成行，`shell` 用 `formatSize` 列目录。

**问题**: 是否应该统一到一个公共函数？放在哪个包？

---

### 6. `globRemote` 非 `**` 模式的深度限制未文档化

**位置**: `client/download.go` — `globRemote`

`globRemote` 仅在 pattern 包含 `**` 时才递归遍历子目录。不含 `**` 时只遍历基路径一层。

这意味着 `get dir/*/file.txt` 不会进入 `dir/sub/file.txt`，即使 `*` 在 doublestar 语义下本可以匹配目录名。

**问题**: 这是性能优化的有意约束，还是实现缺陷？是否应该支持非 `**` 的深度递归 glob？

---

### 7. `completer.go` 三段复制粘贴的补全逻辑

**位置**: `completer/completer.go`

`completeCommand`、`completeRemotePath`、`completeLocalPath` 各自实现了几乎相同的"单候选返回 suffix / 多候选计算公共前缀"逻辑。`completeCommand` 还引入了一个 `removePrefix` 辅助函数，但另外两个不用。

**问题**: 是否应该提取公共补全逻辑？`removePrefix` 的行为（无条件去掉 prefix）是否与其他两个一致？
