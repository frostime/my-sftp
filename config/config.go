package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kevinburke/ssh_config"
)

// SSHConfig 封装 SSH 配置信息
type SSHConfig struct {
	Host         string
	Port         int
	User         string
	IdentityFile string
}

// LoadSSHConfig 从 SSH config 文件加载配置
// alias 是主机别名，如 "eegsys"
func LoadSSHConfig(alias string) (*SSHConfig, error) {
	// 查找 SSH config 文件位置
	configPath := findSSHConfigPath()
	if configPath == "" {
		return nil, fmt.Errorf("SSH config file not found")
	}

	// 打开并解析配置文件
	f, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	cfg, err := ssh_config.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// 提取配置项
	conf := &SSHConfig{}

	// HostName
	hostname, err := cfg.Get(alias, "HostName")
	if err != nil || hostname == "" {
		// 如果没有 HostName，使用别名本身
		hostname = alias
	}
	conf.Host = hostname

	// Port
	portStr, _ := cfg.Get(alias, "Port")
	if portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			conf.Port = port
		}
	}
	if conf.Port == 0 {
		conf.Port = 22 // 默认端口
	}

	// User
	user, _ := cfg.Get(alias, "User")
	conf.User = user

	// IdentityFile
	identityFile, _ := cfg.Get(alias, "IdentityFile")
	if identityFile != "" {
		// 展开 ~ 为用户主目录
		if identityFile[0] == '~' {
			home, _ := os.UserHomeDir()
			identityFile = filepath.Join(home, identityFile[1:])
		}
		conf.IdentityFile = identityFile
	}

	return conf, nil
}

// findSSHConfigPath 查找 SSH config 文件路径
func findSSHConfigPath() string {
	// 优先级：
	// 1. 环境变量指定
	// 2. ~/.ssh/config (Unix/Linux/macOS)
	// 3. %USERPROFILE%\.ssh\config (Windows)

	if configPath := os.Getenv("SSH_CONFIG"); configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Unix-like 系统
	unixPath := filepath.Join(home, ".ssh", "config")
	if _, err := os.Stat(unixPath); err == nil {
		return unixPath
	}

	// Windows 系统
	windowsPath := filepath.Join(home, ".ssh", "config")
	if _, err := os.Stat(windowsPath); err == nil {
		return windowsPath
	}

	return ""
}

// Merge 合并配置（命令行参数优先级更高）
func (c *SSHConfig) Merge(host string, port int, user string, keyFile string) {
	if host != "" {
		c.Host = host
	}
	if port != 0 {
		c.Port = port
	}
	if user != "" {
		c.User = user
	}
	if keyFile != "" {
		c.IdentityFile = keyFile
	}
}

// Validate 验证配置完整性
func (c *SSHConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.User == "" {
		return fmt.Errorf("user is required")
	}
	return nil
}

// ParseDestination 解析 user@host[:port] 格式的目标字符串
// 例如: "user@192.168.1.100" 或 "user@example.com:2222" 或 "user@[2001:db8::1]:22"
func ParseDestination(dest string) (*SSHConfig, error) {
	if dest == "" {
		return nil, fmt.Errorf("destination is empty")
	}

	// 检查是否包含 @ 符号
	if !strings.Contains(dest, "@") {
		return nil, fmt.Errorf("invalid format: expected user@host[:port]")
	}

	config := &SSHConfig{
		Port: 22, // 默认端口
	}

	// 分割 user@host[:port]
	parts := strings.SplitN(dest, "@", 2)
	config.User = parts[0]
	hostPart := parts[1]

	// 使用 net.SplitHostPort 正确处理 IPv6 地址
	host, portStr, err := net.SplitHostPort(hostPart)
	if err != nil {
		// 如果出错，可能是没有指定端口，只有主机名/IP
		// 对于 IPv6 literal 没有端口的情况 (如 "[2001:db8::1]")，需要去除方括号
		if strings.HasPrefix(hostPart, "[") && strings.HasSuffix(hostPart, "]") {
			config.Host = hostPart[1 : len(hostPart)-1]
		} else {
			config.Host = hostPart
		}
	} else {
		// 成功分离出主机和端口
		config.Host = host
		if port, err := strconv.Atoi(portStr); err == nil {
			config.Port = port
		} else {
			return nil, fmt.Errorf("invalid port number: %s", portStr)
		}
	}

	return config, nil
}

// FindDefaultKeys 查找默认的 SSH 私钥文件
// 返回存在的密钥文件路径列表
func FindDefaultKeys() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	sshDir := filepath.Join(home, ".ssh")
	defaultKeys := []string{
		"id_ed25519",
		"id_rsa",
		"id_ecdsa",
		"id_dsa",
	}

	var foundKeys []string
	for _, keyName := range defaultKeys {
		keyPath := filepath.Join(sshDir, keyName)
		if _, err := os.Stat(keyPath); err == nil {
			foundKeys = append(foundKeys, keyPath)
		}
	}

	return foundKeys
}
