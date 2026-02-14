# Project Context

**Name**: my-sftp
**Description**: A modern SFTP CLI tool built with Go, providing enhanced interactive experience with auto-completion, visual transfer progress, and high-performance concurrent file transfers.
**Repo**: github.com/frostime/my-sftp

## Purpose

my-sftp 是为了替代 Windows 原生糟糕的 SFTP CLI 体验而开发的现代化工具。主要目标：

1. **提升交互体验**：支持 TAB 自动补全（命令、远程路径、本地路径）、命令历史记录
2. **增强文件传输**：可视化进度条、支持多文件并发传输、支持 glob 模式匹配
3. **跨平台兼容**：虽然主要针对 Windows，但在 Linux/macOS 上也能良好运行

## Tech Stack

- **语言**: Go 1.24+
- **核心依赖**:
  - `golang.org/x/crypto/ssh`: SSH 连接与认证
  - `github.com/pkg/sftp`: SFTP 协议实现
  - `github.com/chzyer/readline`: 交互式 shell 和自动补全
  - `github.com/schollz/progressbar/v3`: 可视化进度条
  - `github.com/bmatcuk/doublestar/v4`: Glob 模式匹配（支持 `**` 递归）
  - `github.com/kevinburke/ssh_config`: SSH config 文件解析
  - `golang.org/x/sync`: 并发控制（singleflight）

## Key Path

## Project Conventions

### Code Style

- **包组织**：功能模块化划分（client、shell、config、completer）
- **命名规范**：
  - 导出函数/类型使用大驼峰 (PascalCase)
  - 私有函数/变量使用小驼峰 (camelCase)
  - 常量使用大驼峰
- **错误处理**：使用 `fmt.Errorf` 包裹错误，提供清晰的上下文信息
- **注释**：导出的函数/类型必须有文档注释

### Architecture Patterns

- **分层架构**：
  - `main.go`: 程序入口，处理连接建立和认证
  - `shell/`: 交互式 shell 层，处理用户命令
  - `client/`: SFTP 客户端封装层，提供文件操作和传输
  - `config/`: SSH 配置解析
  - `completer/`: 自动补全逻辑

- **并发模型**：
  - 使用统一的任务执行引擎 (`executeTasks`) 避免并发嵌套
  - 使用 `singleflight` 保证目录创建操作的幂等性
  - 使用 `sync.Pool` 复用传输缓冲区，减少 GC 压力

- **缓存策略**：
  - 目录列表缓存（30秒过期）：减少重复的远程目录查询
  - Buffer Pool：复用 512KB 传输缓冲区

### Testing Strategy

- 当前项目以实用性为主，测试策略尚未完整建立
- 主要通过手动测试验证核心功能
- 建议未来添加：
  - 单元测试：配置解析、路径解析逻辑
  - 集成测试：实际 SFTP 服务器交互测试

## Domain Context

### SFTP 协议特性

- SFTP 基于 SSH 协议，需要 SSH 认证（密钥或密码）
- 支持断点续传、目录递归操作
- 路径处理：远程路径使用 Unix 风格 (`/`)，本地路径根据 OS 自适应

### SSH Config 支持

- 支持标准 SSH config 文件 (`~/.ssh/config`)
- 配置优先级：命令行参数 > SSH config > 默认值
- 支持的配置项：Host、HostName、Port、User、IdentityFile

### 文件传输优化

- **并发传输**：默认 4 个并发任务，充分利用带宽
- **进度显示**：
  - 单文件传输：显示文件级进度条
  - 多文件并发：显示总体进度条（文件数+总字节数）
- **Glob 模式**：支持 `*`、`?`、`**`（递归）等模式

## Important Constraints

1. **Go 版本要求**：需要 Go 1.24+
2. **SSH 密钥支持**：自动查找常见密钥位置（`~/.ssh/id_rsa`、`~/.ssh/id_ed25519` 等）
3. **Host Key 验证**：默认使用 `known_hosts` 验证，首次连接需用户确认
4. **路径兼容性**：需处理 Windows 和 Unix 路径差异

## External References

- [pkg/sftp 文档](https://pkg.go.dev/github.com/pkg/sftp)
- [golang.org/x/crypto/ssh 文档](https://pkg.go.dev/golang.org/x/crypto/ssh)
- [SSH Config 规范](https://man.openbsd.org/ssh_config)

## Notes
<!-- @RULE: Project-level memory. Append-only log of learnings, gotchas, preferences.
Agent appends here during @handover when a discovery is project-wide (not change-specific).
Format each entry as: `- YYYY-MM-DD: <learning>`
Prune entries that become outdated or graduate to Conventions/spec-docs. -->
