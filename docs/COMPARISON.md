# 使用方式对比

## 原生 SFTP vs my-sftp

### 连接方式

| 功能 | 原生 SFTP | my-sftp |
|------|-----------|---------|
| 使用 SSH config | ✅ `sftp prod` | ✅ `my-sftp prod` |
| 密码认证 | `sftp user@host` | `my-sftp -host host -user user` |
| 密钥认证 | `sftp -i key user@host` | `my-sftp -host host -user user -key key` |
| 指定端口 | `sftp -P 2222 user@host` | `my-sftp -host host -port 2222` |

### 交互体验

| 功能 | 原生 SFTP (Windows) | my-sftp |
|------|---------------------|---------|
| 远程路径补全 | ❌ 不支持 | ✅ Tab 补全 |
| 本地路径补全 | ❌ 不支持 | ✅ Tab 补全 |
| 命令历史 | ⚠️ 有限支持 | ✅ 完整历史记录 |
| 彩色提示符 | ❌ | ✅ 绿色路径显示 |
| 批量操作 | ⚠️ 部分支持 | ✅ 支持 |

### 命令对比

| 操作 | 原生 SFTP | my-sftp |
|------|-----------|---------|
| 列出文件 | `ls` | `ls` / `ll` |
| 切换目录 | `cd path` | `cd path` (支持补全) |
| 下载文件 | `get remote [local]` | `get remote [local]` (支持补全) |
| 上传文件 | `put local [remote]` | `put local [remote]` (支持补全) |
| 删除文件 | `rm file` | `rm file` |
| 创建目录 | `mkdir dir` | `mkdir dir` |
| 查看信息 | `-` | `stat path` |
| 帮助 | `help` / `?` | `help` |

## 实际使用场景对比

### 场景 1：频繁连接多台服务器

**原生 SFTP:**
```bash
# 使用 SSH config
sftp prod

# 完全相同的体验
```

**my-sftp:**
```bash
# 使用 SSH config
my-sftp prod

# 优势：自动补全让文件操作更快
```

### 场景 2：在深层目录中导航

**原生 SFTP (Windows):**
```bash
sftp> cd /var/www/html/app/config   # 必须完整输入
sftp> cd /var/log/apache2            # 打错字？重新输入
```

**my-sftp:**
```bash
> cd /var/www/[Tab]                  # 显示 www 下的目录
> cd /var/www/html/[Tab]             # 逐级补全
> cd /var/log/apa[Tab]               # 自动补全为 apache2
```

### 场景 3：下载多个配置文件

**原生 SFTP (Windows):**
```bash
sftp> cd /etc/nginx/sites-available
sftp> ls                              # 看到文件列表
sftp> get default                     # 手动输入完整文件名
sftp> get api.conf                    # 再次手动输入
```

**my-sftp:**
```bash
> cd /etc/nginx/sites-available
> get de[Tab]                         # 补全为 default
> get api[Tab]                        # 补全为 api.conf
```

### 场景 4：上传本地文件

**原生 SFTP (Windows):**
```bash
sftp> put C:\Users\admin\Documents\config.yml    # 必须记住完整路径
```

**my-sftp:**
```bash
> put C:\Users\[Tab]                  # 补全用户名
> put C:\Users\admin\Doc[Tab]         # 补全 Documents
> put C:\Users\admin\Documents\conf[Tab]  # 补全文件名
```

## 性能对比

| 指标 | 原生 SFTP | my-sftp |
|------|-----------|---------|
| 连接速度 | 快 | 快（相同） |
| 文件传输 | 快 | 快（相同） |
| 命令响应 | 即时 | 即时 |
| 补全查询 | N/A | <100ms（网络延迟） |
| 内存占用 | ~5MB | ~10MB |

## 适用场景

### 推荐使用 my-sftp 的场景

- ✅ Windows 系统下的日常 SFTP 操作
- ✅ 需要频繁浏览深层目录结构
- ✅ 文件名较长或不容易记忆
- ✅ 需要在本地和远程路径间频繁切换
- ✅ 初学者或不熟悉命令行的用户

### 继续使用原生 SFTP 的场景

- 脚本自动化（原生工具更稳定）
- 极度资源受限的环境
- 需要使用 SFTP 的高级特性（如符号链接处理）
- Linux/macOS 系统（原生工具已有较好的补全）

## 总结

**my-sftp 的核心价值**：
- 在 Windows 平台填补了原生 SFTP 客户端的补全缺陷
- 通过 Tab 补全显著提升路径输入效率
- 保持与原生工具相同的命令习惯，学习成本低
- SSH config 兼容性确保与现有工作流无缝集成

**权衡考虑**：
- 增加了约 5MB 的内存开销
- 补全功能依赖网络查询（延迟约 50-100ms）
- 需要 Go 环境编译（提供预编译二进制可解决）
