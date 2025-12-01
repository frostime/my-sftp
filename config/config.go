package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

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
