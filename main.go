package main

import (
	"flag"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"

	"my-sftp/client"
	"my-sftp/config"
	"my-sftp/shell"
)

func main() {
	var (
		host     string
		port     int
		username string
		password string
		keyFile  string
	)

	flag.StringVar(&host, "host", "", "SFTP server host (or use positional argument)")
	flag.IntVar(&port, "port", 0, "SFTP server port (default: 22 or from config)")
	flag.StringVar(&username, "user", "", "Username (default: from config)")
	flag.StringVar(&password, "pass", "", "Password (leave empty for prompt)")
	flag.StringVar(&keyFile, "key", "", "Path to private key file (default: from config)")
	flag.Parse()

	// 获取位置参数作为主机别名
	args := flag.Args()
	var alias string
	if len(args) > 0 {
		alias = args[0]
	}

	// 尝试从 SSH config 加载配置
	var sshConfig *config.SSHConfig
	if alias != "" {
		var err error
		sshConfig, err = config.LoadSSHConfig(alias)
		if err != nil {
			// 如果加载失败，将别名作为主机名
			fmt.Printf("Warning: Failed to load SSH config for '%s': %v\n", alias, err)
			fmt.Println("Using alias as hostname...")
			sshConfig = &config.SSHConfig{
				Host: alias,
				Port: 22,
			}
		} else {
			fmt.Printf("Loaded config for '%s'\n", alias)
		}

		// 命令行参数覆盖配置文件
		sshConfig.Merge(host, port, username, keyFile)
	} else {
		// 没有别名，使用命令行参数
		sshConfig = &config.SSHConfig{
			Host:         host,
			Port:         port,
			User:         username,
			IdentityFile: keyFile,
		}
		if sshConfig.Port == 0 {
			sshConfig.Port = 22
		}
	}

	// 交互式输入缺失的必需参数
	if sshConfig.Host == "" {
		fmt.Print("Host: ")
		fmt.Scanln(&sshConfig.Host)
	}
	if sshConfig.User == "" {
		fmt.Print("Username: ")
		fmt.Scanln(&sshConfig.User)
	}

	// 验证配置
	if err := sshConfig.Validate(); err != nil {
		fmt.Printf("Invalid configuration: %v\n", err)
		os.Exit(1)
	}

	// 构建 SSH 配置
	sshClientConfig := &ssh.ClientConfig{
		User:            sshConfig.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应验证主机密钥
	}

	// 认证方式：优先使用密钥，其次密码
	if sshConfig.IdentityFile != "" {
		authMethod, err := loadPrivateKey(sshConfig.IdentityFile)
		if err != nil {
			fmt.Printf("Failed to load private key: %v\n", err)
			os.Exit(1)
		}
		sshClientConfig.Auth = []ssh.AuthMethod{authMethod}
	} else {
		// 密码认证
		if password == "" {
			fmt.Print("Password: ")
			pw, _ := terminal.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			password = string(pw)
		}
		sshClientConfig.Auth = []ssh.AuthMethod{ssh.Password(password)}
	}

	// 连接 SFTP 服务器
	addr := fmt.Sprintf("%s:%d", sshConfig.Host, sshConfig.Port)
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
		return nil, fmt.Errorf("read key file: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		// 如果解析失败，尝试使用密码解密
		fmt.Print("Key passphrase: ")
		passphrase, _ := terminal.ReadPassword(int(syscall.Stdin))
		fmt.Println()

		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, passphrase)
		if err != nil {
			return nil, fmt.Errorf("parse key: %w", err)
		}
	}

	return ssh.PublicKeys(signer), nil
}
