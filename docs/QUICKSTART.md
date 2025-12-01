# 快速开始指南

## 1. 编译项目

```bash
cd my-sftp
go mod download
go build -o my-sftp
```

Windows 用户：
```bash
go build -o my-sftp.exe
```

## 2. 配置 SSH Config（推荐）

### 创建配置文件

**Unix/Linux/macOS:**
```bash
mkdir -p ~/.ssh
chmod 700 ~/.ssh
nano ~/.ssh/config
```

**Windows (PowerShell):**
```powershell
mkdir ~\.ssh -ErrorAction SilentlyContinue
notepad ~\.ssh\config
```

### 添加服务器配置

复制以下内容到配置文件（根据实际情况修改）：

```ssh-config
# 示例：生产服务器
Host prod
    HostName 192.168.1.100
    User admin
    IdentityFile ~/.ssh/id_rsa
    Port 22

# 示例：开发服务器
Host dev
    HostName dev.example.com
    User developer
    IdentityFile ~/.ssh/dev_key
```

### 使用别名连接

```bash
./my-sftp prod
```

就这么简单！程序会自动读取配置文件中的所有参数。

## 3. 传统连接方式

如果不使用 SSH config，可以手动指定参数：

**密码认证:**
```bash
./my-sftp -host 192.168.1.100 -user admin
# 会提示输入密码
```

**密钥认证:**
```bash
./my-sftp -host 192.168.1.100 -user admin -key ~/.ssh/id_rsa
```

## 4. 基本命令演示

连接成功后，你会看到提示符：

```
/home/admin > 
```

### 浏览文件

```bash
# 列出当前目录
> ls

# 切换目录（支持 Tab 补全）
> cd /var/www

# 查看当前位置
> pwd
```

### 上传/下载

```bash
# 下载文件
> get application.log

# 上传文件
> put local_file.txt

# 指定目标路径
> get /var/log/app.log ./local.log
> put ./data.csv /tmp/data.csv
```

### 文件管理

```bash
# 创建目录
> mkdir backup

# 删除文件
> rm old_file.txt

# 重命名
> rename old.txt new.txt
```

### 其他

```bash
# 查看帮助
> help

# 退出
> exit
```

## 5. 高级技巧

### Tab 自动补全

在输入命令时按 Tab 键：

```bash
> cd /va[Tab]       # 自动补全为 /var/
> get app[Tab]      # 列出所有以 app 开头的文件
```

### 命令历史

- 上箭头 ↑：查看上一条命令
- 下箭头 ↓：查看下一条命令
- Ctrl+R：搜索历史命令

### 覆盖配置文件参数

即使使用了 SSH config，仍可通过命令行参数覆盖：

```bash
# 使用 prod 配置，但改用另一个密钥
./my-sftp prod -key ~/.ssh/another_key

# 使用 prod 配置，但连接到不同端口
./my-sftp prod -port 8022
```

## 6. 常见问题排查

### 连接失败

1. 检查主机地址和端口是否正确
2. 确认用户名和认证方式（密码/密钥）
3. 如果使用密钥，确保密钥文件权限正确（Unix 下应为 600）

```bash
chmod 600 ~/.ssh/id_rsa
```

### SSH config 不生效

1. 确认文件路径：`~/.ssh/config`
2. 检查文件权限（Unix 下应为 600）

```bash
chmod 600 ~/.ssh/config
```

3. 验证配置语法是否正确（不要有多余的空格或制表符）

### 自动补全不工作

1. 确保已正确连接到服务器
2. 检查是否有权限访问要补全的目录
3. 尝试在命令后加空格再按 Tab

## 7. 获取帮助

- 查看内置帮助：`help` 命令
- 阅读 README.md 获取详细文档
- 检查 ssh_config_example 文件获取配置示例

祝使用愉快！
