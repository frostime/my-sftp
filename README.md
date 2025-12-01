# my-sftp

一个支持**路径自动补全**的 SFTP 命令行客户端，基于 Go 语言实现。

## 核心特性

- ✅ **SSH Config 支持**：兼容原生 SSH 配置文件，使用别名快速连接
- ✅ **Tab 自动补全**：本地和远程路径均支持
- ✅ **命令历史**：上下箭头浏览历史命令
- ✅ **友好交互**：彩色提示符，清晰的错误提示
- ✅ **精简实现**：~600 行代码，易于扩展

## 快速开始

### 编译

```bash
go mod download
go build -o my-sftp
```

Windows 下：
```bash
go build -o my-sftp.exe
```

### 使用

**最简方式（使用 SSH config）：**
```bash
# 配置 ~/.ssh/config
Host eegsys
    HostName 202.114.66.94
    User eeg
    IdentityFile ~/.ssh/id_rsa
    Port 22

# 直接使用别名连接
./my-sftp eegsys
```

**传统方式：**

密码认证：
```bash
./my-sftp -host example.com -user username
# 会提示输入密码
```

密钥认证：
```bash
./my-sftp -host example.com -user username -key ~/.ssh/id_rsa
```

完整参数：
```bash
./my-sftp -host example.com -port 22 -user username -pass mypassword
```

**混合使用（命令行覆盖配置文件）：**
```bash
# 使用 config 中的配置，但覆盖端口
./my-sftp eegsys -port 8000

# 使用 config 中的主机和用户，指定不同的密钥
./my-sftp eegsys -key ~/.ssh/another_key
```

## 支持的命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `pwd` | 显示当前目录 | `pwd` |
| `cd <dir>` | 切换目录 | `cd /var/www` |
| `ls [dir]` | 列出文件 | `ls`, `ls /tmp` |
| `get <remote> [local]` | 下载文件 | `get file.txt`, `get a.log ./local.log` |
| `put <local> [remote]` | 上传文件 | `put file.txt`, `put ./a.txt /tmp/b.txt` |
| `rm <path>` | 删除文件/目录 | `rm file.txt`, `rm -r dir/` |
| `mkdir <dir>` | 创建目录 | `mkdir test` |
| `rename <old> <new>` | 重命名 | `rename old.txt new.txt` |
| `stat <path>` | 查看文件信息 | `stat file.txt` |
| `help` | 显示帮助 | `help` |
| `exit` | 退出程序 | `exit`, `quit`, `q` |

## SSH Config 配置

### 配置文件位置

- **Unix/Linux/macOS**: `~/.ssh/config`
- **Windows**: `%USERPROFILE%\.ssh\config`

### 配置示例

编辑 `~/.ssh/config` 文件：

```ssh-config
# 生产服务器
Host prod
    HostName 192.168.1.100
    User admin
    Port 22
    IdentityFile ~/.ssh/prod_rsa

# 开发服务器
Host dev
    HostName dev.example.com
    User developer
    IdentityFile ~/.ssh/dev_key

# 使用密码认证的服务器
Host legacy
    HostName 10.0.0.50
    User root
    Port 2222
    # 不指定 IdentityFile 则使用密码认证
```

### 支持的配置项

| 配置项 | 说明 | 示例 |
|--------|------|------|
| `HostName` | 实际主机地址 | `HostName 192.168.1.100` |
| `User` | 用户名 | `User admin` |
| `Port` | 端口号 | `Port 22` |
| `IdentityFile` | 私钥路径 | `IdentityFile ~/.ssh/id_rsa` |

### 优先级规则

配置优先级从高到低：
1. **命令行参数**（如 `-host`, `-user`, `-port`, `-key`）
2. **SSH config 文件**
3. **默认值**（端口 22）

示例：
```bash
# 使用 config 中的所有配置
./my-sftp prod

# 覆盖 config 中的端口
./my-sftp prod -port 8022

# 覆盖 config 中的用户和密钥
./my-sftp prod -user root -key ~/.ssh/another_key
```

## 使用技巧

### SSH Config 快速配置

1. 编辑配置文件（首次使用需创建）：
```bash
# Unix/Linux/macOS
mkdir -p ~/.ssh
chmod 700 ~/.ssh
nano ~/.ssh/config

# Windows (PowerShell)
mkdir ~\.ssh -ErrorAction SilentlyContinue
notepad ~\.ssh\config
```

2. 添加服务器配置：
```ssh-config
Host myserver
    HostName 192.168.1.100
    User admin
    IdentityFile ~/.ssh/id_rsa
```

3. 直接使用别名连接：
```bash
./my-sftp myserver
```

### 自动补全

按 **Tab** 键触发补全：

```
> cd /va[Tab]          → /var/
> cd /var/lo[Tab]      → /var/log/
> get app[Tab]         → application.conf
> put ~/Doc[Tab]       → ~/Documents/
```

### 路径规则

- **绝对路径**：以 `/` 开头，如 `/home/user/file.txt`
- **相对路径**：相对当前工作目录，如 `./data/log.txt`
- **主目录**：`cd ~` 或 `cd` 返回用户主目录

### 批量操作

部分命令支持多个参数：

```
> mkdir dir1 dir2 dir3
> rm file1.txt file2.txt
```

## 项目结构

```
my-sftp/
├── main.go               # 入口：参数解析、连接初始化
├── config/
│   └── config.go         # SSH config 文件解析
├── client/
│   └── client.go         # SFTP 操作封装（上传、下载、列表等）
├── shell/
│   └── shell.go          # 交互式命令解释器
└── completer/
    └── completer.go      # 自动补全逻辑（远程+本地路径）
```

**设计原则**：
- 单一职责：每个包只负责一个领域
- 接口抽象：`completer` 通过接口与 `client` 解耦
- 配置分离：SSH config 解析独立为 `config` 包
- 最小依赖：仅使用必需的第三方库

## 技术栈

| 组件 | 库 | 用途 |
|------|-----|------|
| SSH 连接 | `golang.org/x/crypto/ssh` | SSH 协议实现 |
| SFTP 协议 | `github.com/pkg/sftp` | SFTP 客户端 |
| 交互式输入 | `github.com/chzyer/readline` | 自动补全、历史记录 |
| SSH Config | `github.com/kevinburke/ssh_config` | 解析 SSH 配置文件 |

## 常见问题

**Q: 如何使用 SSH config 配置？**  
A: 创建 `~/.ssh/config` 文件，添加主机配置（参考 `ssh_config_example`），然后使用 `./my-sftp <alias>` 连接。

**Q: SSH config 不生效怎么办？**  
A: 检查：(1) 配置文件路径是否正确；(2) 文件权限（Unix 下应为 600）；(3) 使用 `-host` 等参数会覆盖配置文件。

**Q: 可以同时使用多个 SSH config 文件吗？**  
A: 默认只读取 `~/.ssh/config`。可通过环境变量 `SSH_CONFIG` 指定自定义路径。

**Q: 如何跳过主机密钥验证？**  
A: 已默认跳过（生产环境不推荐）。如需验证，修改 `main.go` 中的 `HostKeyCallback`。

**Q: 如何上传整个目录？**  
A: 当前不支持递归上传，需逐个上传文件。可自行扩展 `client.Upload` 方法。

**Q: 补全为何不显示隐藏文件？**  
A: 默认显示所有文件（包括 `.` 开头）。如需过滤，修改 `completer.go`。

## 扩展建议

**添加进度条：**
```go
// 在 client.Download/Upload 中集成 github.com/schollz/progressbar
bar := progressbar.DefaultBytes(fileSize, "Downloading")
io.Copy(io.MultiWriter(dstFile, bar), srcFile)
```

**支持通配符：**
```go
// 在 shell.cmdGet 中使用 filepath.Glob 匹配多个文件
matches, _ := filepath.Glob(pattern)
```

**多连接管理：**
```go
// 实现连接池，支持 `connect <name>` 切换连接
type SessionManager struct {
    sessions map[string]*Client
    active   string
}
```

## 许可证

MIT License - 自由使用和修改
