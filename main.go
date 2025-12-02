package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	terminal "golang.org/x/term"

	"github.com/frostime/my-sftp/client"
	"github.com/frostime/my-sftp/config"
	"github.com/frostime/my-sftp/shell"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	// 支持 my-sftp --version
	if *showVersion {
		fmt.Printf("my-sftp version: %s\n", Version)
		fmt.Printf("Commit: %s\n", Commit)
		fmt.Printf("Built at: %s\n", Date)
		// fmt.Printf("Go version: %s\n", runtime.Version())
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

	// ==================== 解析 SSH 配置 ====================

	// 尝试解析 destination
	var sshConfig *config.SSHConfig
	var err error

	// 1. 解析目标地址
	if strings.Contains(destination, "@") {
		sshConfig, err = config.ParseDestination(destination)
		if err != nil {
			fmt.Printf("Invalid destination: %v\n", err)
			os.Exit(1)
		}
	} else {
		// 作为 SSH config 别名处理
		sshConfig, err = config.LoadSSHConfig(destination)
		if err != nil {
			fmt.Printf("Config error: %v\n", err)
			os.Exit(1)
		}
	}

	// 验证配置
	if err := sshConfig.Validate(); err != nil {
		fmt.Printf("Invalid config: %v\n", err)
		os.Exit(1)
	}

	// 2. 准备认证方法 (Key + Password)
	var authMethods []ssh.AuthMethod
	var keyFiles []string
	if sshConfig.IdentityFile != "" {
		keyFiles = append(keyFiles, sshConfig.IdentityFile)
	} else {
		keyFiles = config.FindDefaultKeys()
	}

	// 尝试加载所有可用的密钥
	for _, keyFile := range keyFiles {
		if authMethod, err := loadPrivateKey(keyFile); err == nil {
			authMethods = append(authMethods, authMethod)
		}
	}

	// Fallback: 使用密码验证
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

	// 3. 创建安全的 HostKeyCallback
	// 查找 known_hosts 文件路径
	homeDir, _ := os.UserHomeDir()
	knownHostsPath := filepath.Join(homeDir, ".ssh", "known_hosts")

	// 创建回调函数
	hostKeyCallback, err := createHostKeyCallback(knownHostsPath)
	if err != nil {
		fmt.Printf("Failed to initialize host key verification: %v\n", err)
		os.Exit(1)
	}

	// 4. 构建 ClientConfig
	sshClientConfig := &ssh.ClientConfig{
		User:            sshConfig.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		// HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := fmt.Sprintf("%s:%d", sshConfig.Host, sshConfig.Port)

	fmt.Printf("[my-sftp %s]Connecting to %s@%s...\n", Version, sshConfig.User, addr)

	// ==================== 创建 SSH 连接 ====================

	c, err := client.NewClient(addr, sshClientConfig)
	if err != nil {
		// 这里的错误可能包含 Host Key 验证失败的信息
		fmt.Printf("Connection failed: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	fmt.Println("✓ Connected successfully!")
	fmt.Println("Type 'help' for available commands, 'exit' to quit.")
	fmt.Println()

	// ==================== 启动交互式 Shell ====================
	sh := shell.NewShell(c)
	if err := sh.Run(); err != nil {
		fmt.Printf("Shell error: %v\n", err)
		os.Exit(1)
	}
}

func loadPrivateKey(keyPath string) (ssh.AuthMethod, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(signer), nil
}

// createHostKeyCallback 创建一个支持交互式确认的主机密钥回调
func createHostKeyCallback(path string) (ssh.HostKeyCallback, error) {
	// 确保文件存在，不存在则创建
	if err := ensureFileExists(path); err != nil {
		return nil, err
	}

	// 使用 ssh/knownhosts 包创建一个基础的回调
	// 它会帮我们解析文件并验证 Key 是否匹配
	callback, err := knownhosts.New(path)
	if err != nil {
		return nil, err
	}

	// 返回一个包装函数，处理 "未知主机" 的情况
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// 1. 调用基础回调进行检查
		err := callback(hostname, remote, key)

		// 如果没有错误，说明已知且匹配，通过
		if err == nil {
			return nil
		}

		// 2. 检查具体的错误类型
		// knownhosts.KeyError 表示：
		// - 可能是 Key 不匹配（严重安全警告）
		// - 可能是 Host 未知（需要询问用户）
		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) {
			// 情况 A: 这是一个已知的 Host，但 Key 不一样！(MITM 攻击风险)
			if len(keyErr.Want) > 0 {
				return fmt.Errorf("HOST KEY MISMATCH for %s! Possible MITM attack. Remote key: %s, Known key: %v",
					hostname, ssh.FingerprintSHA256(key), keyErr.Want)
			}

			// 情况 B: 这是一个未知的主机 (keyErr.Want 为空)
			// 我们需要询问用户是否信任它
			return askUserToTrustHost(path, hostname, remote, key)
		}

		// 其他系统错误
		return err
	}, nil
}

// askUserToTrustHost 询问用户是否信任主机，如果信任则写入文件
func askUserToTrustHost(path string, hostname string, remote net.Addr, key ssh.PublicKey) error {
	fmt.Printf("\nThe authenticity of host '%s' can't be established.\n", hostname)
	fmt.Printf("%s key fingerprint is %s.\n", key.Type(), ssh.FingerprintSHA256(key))
	fmt.Print("Are you sure you want to continue connecting (yes/no)? ")

	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(strings.ToLower(text))

	if text != "yes" {
		return fmt.Errorf("host key verification failed: user aborted")
	}

	// 用户同意，追加到 known_hosts 文件
	return appendToKnownHosts(path, hostname, remote, key)
}

// appendToKnownHosts 将新主机追加到 known_hosts 文件
func appendToKnownHosts(path string, hostname string, remote net.Addr, key ssh.PublicKey) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open known_hosts: %w", err)
	}
	defer f.Close()

	// 处理非标准端口的情况
	// ssh 规范：如果端口不是22，hostname 格式通常是 [host]:port
	// knownhosts.Normalize 帮助我们标准化这个格式
	normalizedHost := knownhosts.Normalize(hostname)

	// 序列化公钥
	keyBytes := key.Marshal()
	keyBase64 := base64.StdEncoding.EncodeToString(keyBytes)

	// 写入格式: host key-type key-base64
	line := fmt.Sprintf("%s %s %s\n", normalizedHost, key.Type(), keyBase64)

	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("failed to write to known_hosts: %w", err)
	}

	fmt.Printf("Warning: Permanently added '%s' (%s) to the list of known hosts.\n", hostname, key.Type())
	return nil
}

// ensureFileExists 确保文件存在，如果不存在则创建
func ensureFileExists(path string) error {
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	f.Close()
	return nil
}
