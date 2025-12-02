package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh"
	terminal "golang.org/x/term"

	"my-sftp/client"
	"my-sftp/config"
	"my-sftp/shell"
)

// Version 项目版本号，推荐用 -ldflags 注入
var Version = "v0.3.0"

func main() {
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	// 支持 my-sftp --version
	if *showVersion {
		fmt.Printf("my-sftp version: %s\n", Version)
		os.Exit(0)
	}

	// 获取位置参数作为 destination
	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Usage: my-sftp [--version] <destination>")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  my-sftp myserver           # Use SSH config alias")
		fmt.Println("  my-sftp user@host          # Connect to host")
		fmt.Println("  my-sftp user@host:2222     # Connect to host with custom port")
		os.Exit(1)
	}

	destination := args[0]

	// 尝试解析 destination
	var sshConfig *config.SSHConfig
	var err error

	// 先尝试作为 user@host[:port] 解析
	if strings.Contains(destination, "@") {
		sshConfig, err = config.ParseDestination(destination)
		if err != nil {
			fmt.Printf("Invalid destination format: %v\n", err)
			os.Exit(1)
		}
	} else {
		// 作为 SSH config 别名处理
		sshConfig, err = config.LoadSSHConfig(destination)
		if err != nil {
			// 如果加载失败，提示错误
			fmt.Printf("Failed to load SSH config for '%s': %v\n", destination, err)
			fmt.Println("Hint: Use 'user@host' format or check your SSH config file.")
			os.Exit(1)
		}
	}

	// 验证配置
	if err := sshConfig.Validate(); err != nil {
		fmt.Printf("Invalid configuration: %v\n", err)
		os.Exit(1)
	}

	// 构建认证方法列表（按优先级顺序）
	var authMethods []ssh.AuthMethod

	// 1. 尝试从配置或默认位置加载密钥
	var keyFiles []string
	if sshConfig.IdentityFile != "" {
		keyFiles = append(keyFiles, sshConfig.IdentityFile)
	} else {
		// 查找默认密钥
		keyFiles = config.FindDefaultKeys()
	}

	// 尝试加载所有可用的密钥
	for _, keyFile := range keyFiles {
		if authMethod, err := loadPrivateKey(keyFile); err == nil {
			authMethods = append(authMethods, authMethod)
		}
	}

	// 2. 添加密码认证作为回退方案
	// 使用 ssh.PasswordCallback 让 SSH 客户端在需要时才提示输入密码
	passwordCallback := ssh.PasswordCallback(func() (string, error) {
		fmt.Printf("%s@%s's password: ", sshConfig.User, sshConfig.Host)
		pw, err := terminal.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return "", err
		}
		return string(pw), nil
	})
	authMethods = append(authMethods, passwordCallback)

	// 构建 SSH 配置
	sshClientConfig := &ssh.ClientConfig{
		User:            sshConfig.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应验证主机密钥
	}

	// 连接 SFTP 服务器
	addr := fmt.Sprintf("%s:%d", sshConfig.Host, sshConfig.Port)

	fmt.Printf("my-sftp version: %s\n", Version)
	fmt.Printf("\033[33mWARNING: Host key verification is disabled (insecure mode)\033[0m\n")
	fmt.Printf("Connecting to %s@%s...\n", sshConfig.User, addr)

	c, err := client.NewClient(addr, sshClientConfig)
	if err != nil {
		fmt.Printf("Connection failed: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	fmt.Println("✓ Connected successfully!")
	fmt.Println("Type 'help' for available commands, 'exit' to quit.")
	fmt.Println()

	// 启动交互式 Shell
	sh := shell.NewShell(c)
	if err := sh.Run(); err != nil {
		fmt.Printf("Shell error: %v\n", err)
		os.Exit(1)
	}
}

// loadPrivateKey 加载私钥文件
func loadPrivateKey(keyPath string) (ssh.AuthMethod, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		// 如果解析失败，可能需要密码短语
		// 注意：这里不提示输入密码短语，因为我们会尝试多个密钥
		// 如果真的需要密码短语，用户可以使用 ssh-agent
		return nil, err
	}

	return ssh.PublicKeys(signer), nil
}
