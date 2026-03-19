# My-SFTP

[![Go Version](https://img.shields.io/github/go-mod/go-version/frostime/my-sftp)](go.mod)
[![License](https://img.shields.io/badge/license-GPLv3-blue)](LICENSE)

[中文文档](./README.zh.md)

🚀 **A modern SFTP CLI tool built with Go.**

Say goodbye to the terrible experience of Windows native SFTP CLI. My-SFTP provides auto-completion, visual transfer progress bars, and high-performance concurrent transfer capabilities.

## ✨ Core Features

* **⚡ Enhanced Interactive Experience**: TAB auto-completion (commands, remote paths, local paths), command history.
* **📂 File Transfer**:

  * **Multiple Transfer Modes**:
    * Single file transfer
    * Specified multiple files transfer
    * Glob pattern
    * Transfer entire directories with `-r`
  * **Concurrent Transfer**: Support multi-file concurrent upload/download, fully utilizing bandwidth.
  * **Command Execution**: Execute commands remotely or locally via `! <cmd>` or `!! <cmd>`.

## 📦 Installation

If you have Go environment installed (1.24+):

```bash
go install my-sftp
```

Build from source:

```bash
cd my-sftp
go build -o my-sftp main.go
```

## 🚀 Quick Start

### Connecting to Server

My-SFTP supports multiple connection methods:

```bash
# 1. Using SSH Config alias (recommended)
# `~/.ssh/config` (Linux/Mac) or `%USERPROFILE%\.ssh\config`
my-sftp myserver

# 2. Standard format connection
my-sftp user@host
my-sftp user@192.168.1.100

# 3. Specify port
my-sftp user@host:2222
```

### Interactive Shell Commands

After entering the shell, you can use the following commands. **Tip: All paths support TAB completion.**

#### 📂 File Browsing and Navigation

| Command       | Description                     | Example                |
| :------------ | :------------------------------ | :--------------------- |
| `ls`, `ll`    | List **remote** directory contents | `ls -l /var/www`       |
| `cd`          | Change **remote** directory     | `cd /etc`              |
| `pwd`         | Show **remote** current path    |                        |
| `lls`, `ldir` | List **local** directory contents| `lls`                  |
| `lcd`         | Change **local** directory      | `lcd D:\Downloads`     |
| `lpwd`        | Show **local** current path     |                        |

#### ⬇️⬆️ File Transfer

> Supported parameters: `-r` (recursive), `-d/--dir` (destination directory), `--name` (single-file rename, filename only), `--flatten` (flatten output structure), `--` (treat following tokens as source operands)

| Command | Description           | Example                                               |
| :------ | :-------------------- | :---------------------------------------------------- |
| `get`   | Download files/directories | `get file.txt`<br>`get -r /var/log/nginx -d ./logs` |
| `put`   | Upload files/directories   | `put local.txt`<br>`put -r dist -d /var/www/html`  |

**🔥 Glob**

```bash
# Upload all txt files to a remote directory
> put *.txt -d /data/txt

# Multiple explicit files preserve their source-relative paths
> put src/a.txt src/nested/b.txt -d /srv/out

# Recursively upload all Go source files (preserve structure by default)
> put **/*.go -d /srv/src

# Flatten output structure (fails on duplicate basenames)
> put **/*.go -d /srv/src --flatten

# Download specific pattern files
> get access-*.log -d ./logs

# Use -- when a source name starts with -
> get -d ./downloads -- -report.txt
```

#### 🛠 File Operations

| Command          | Description               | Example                   |
| :--------------- | :------------------------ | :------------------------ |
| `mkdir`, `md`    | Create remote directory   | `mkdir new_folder`        |
| `rm`             | Delete remote files/dirs  | `rm old_file.txt`         |
| `rename`, `mv`   | Rename                    | `mv old.txt new.txt`      |
| `stat`           | View file details         | `stat file.txt`           |
| `lmkdir`         | Create local directory    | `lmkdir local_folder`     |

#### 🖥️ Shell Command Execution

| Command | Description                       | Example               |
| :------ | :-------------------------------- | :-------------------- |
| `!`     | Execute commands on **remote** server | `! tree -L 2`         |
| `!!`    | Execute commands on **local** machine  | `!! dir`              |

**🔥 Shell Command Examples**

```bash
# Remote command execution (IPython-style)
> ! cat /etc/os-release       # View remote system info
> ! df -h                     # View remote disk usage
> ! tree -L 2                 # View remote directory tree
> ! tail -n 100 app.log       # View remote log files

# Local command execution
> !! dir                      # Windows: List local directory
> !! ls -la                   # Linux/Mac: List local directory
> !! cat config.json          # View local file content
```

## ⚙️ Configuration Guide

My-SFTP automatically reads SSH configuration from the system.

**Configuration file path priority:**

1. Environment variable `SSH_CONFIG`
2. `~/.ssh/config` (Linux/Mac) or `%USERPROFILE%\.ssh\config` (Windows)

**Recommended configuration example (`.ssh/config`):**

```ssh
Host prod
    HostName 192.168.1.100
    User admin
    Port 2222
    IdentityFile ~/.ssh/id_ed25519
```

After configuration, simply run `my-sftp prod` to connect.
