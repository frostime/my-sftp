# My-SFTP

🚀 **一个基于 Go 实现的 SFTP CLI 工具。**

My-SFTP 旨在解决 Windows 平台下原生 SFTP 体验不佳的问题（如缺乏自动补全、进度条简陋等），提供更好的交互体验。

## ✨ 核心特性

* **⚡ 交互体验升级**：支持 TAB 自动补全（命令、远程路径、本地路径）、命令历史记录。
* **📂 文件传输**：

  * **通配符支持**：支持 `*`, `?`, `[a-z]`, `**` (递归匹配) 等 Glob 模式。
  * **递归传输**：使用 `-r` 轻松上传/下载整个目录。
  * **并发传输**：支持多文件并发上传/下载，充分利用带宽。

## 📦 安装说明

如果你已安装 Go 环境 (1.23+)：

```bash
go install my-sftp
```

源码编译

```bash
cd my-sftp
go build -o my-sftp main.go
```

## 🚀 快速开始

### 连接服务器

My-SFTP 支持多种连接方式：

```bash
# 1. 使用 SSH Config 别名 (推荐)
# `~/.ssh/config` (Linux/Mac) 或 `%USERPROFILE%\.ssh\config`
my-sftp myserver

# 2. 标准格式连接
my-sftp user@host
my-sftp user@192.168.1.100

# 3. 指定端口
my-sftp user@host:2222
```

### 交互式 Shell 命令

进入 Shell 后，你可以使用以下命令。**提示：所有路径均支持 TAB 补全。**

#### 📂 文件浏览与导航

| 命令            | 说明           | 示例                 |
| :------------ | :----------- | :----------------- |
| `ls`, `ll`    | 列出**远程**目录内容 | `ls -l /var/www`   |
| `cd`          | 切换**远程**目录   | `cd /etc`          |
| `pwd`         | 显示**远程**当前路径 |                    |
| `lls`, `ldir` | 列出**本地**目录内容 | `lls`              |
| `lcd`         | 切换**本地**目录   | `lcd D:\Downloads` |
| `lpwd`        | 显示**本地**当前路径 |                    |

#### ⬇️⬆️ 文件传输

> 支持参数：`-r` (递归目录)

| 命令    | 说明      | 示例                                               |
| :---- | :------ | :----------------------------------------------- |
| `get` | 下载文件/目录 | `get file.txt`<br>`get -r /var/log/nginx ./logs` |
| `put` | 上传文件/目录 | `put local.txt`<br>`put -r dist/ /var/www/html`  |

**🔥 Glob**

```bash
# 上传所有 txt 文件
> put *.txt

# 递归上传所有 Go 源代码文件
> put **/*.go src/

# 下载特定模式的文件
> get access-*.log
```

#### 🛠 文件操作

| 命令             | 说明        | 示例                    |
| :------------- | :-------- | :-------------------- |
| `mkdir`, `md`  | 创建远程目录    | `mkdir new_folder`    |
| `rm`           | 删除远程文件/目录 | `rm old_file.txt`     |
| `rename`, `mv` | 重命名       | `mv old.txt new.txt`  |
| `stat`         | 查看文件详细信息  | `stat file.txt`       |
| `lmkdir`       | 创建本地目录    | `lmkdir local_folder` |

#### 🖥️ Shell 命令执行

| 命令   | 说明               | 示例                |
| :--- | :--------------- | :---------------- |
| `!`  | 在**远程**服务器执行命令   | `! tree -L 2`     |
| `!!` | 在**本地**机器执行命令    | `!! dir`          |

**🔥 Shell 命令示例**

```bash
# 远程命令执行（模仿 IPython 风格）
> ! cat /etc/os-release       # 查看远程系统信息
> ! df -h                     # 查看远程磁盘使用情况
> ! tree -L 2                 # 查看远程目录树
> ! tail -n 100 app.log       # 查看远程日志文件

# 本地命令执行
> !! dir                      # Windows: 列出本地目录
> !! ls -la                   # Linux/Mac: 列出本地目录
> !! cat config.json          # 查看本地文件内容
```

## ⚙️ 配置指南

My-SFTP 会自动读取系统中的 SSH 配置。

**配置文件路径优先级：**

1. 环境变量 `SSH_CONFIG`
2. `~/.ssh/config` (Linux/Mac) 或 `%USERPROFILE%\.ssh\config` (Windows)

**推荐配置示例 (`.ssh/config`)：**

```ssh
Host prod
    HostName 192.168.1.100
    User admin
    Port 2222
    IdentityFile ~/.ssh/id_ed25519
```

配置后，仅需运行 `my-sftp prod` 即可连接。


