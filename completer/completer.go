package completer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
)

// ClientInterface 定义 SFTP 客户端必需的接口
type ClientInterface interface {
	ListCompletion(prefix string) []string
	GetLocalwd() string
}

// Completer 自动补全器
type Completer struct {
	client  ClientInterface
	cmdList []string // 命令列表
}

// NewCompleter 创建补全器
func NewCompleter(client ClientInterface) *Completer {
	return &Completer{
		client: client,
		cmdList: []string{
			"help", "exit", "quit", "q",
			"ls", "ll", "dir",
			"cd", "pwd",
			"get", "download",
			"put", "upload",
			"rm", "del", "delete",
			"mkdir", "md",
			"rmdir", "rd",
			"rename", "mv",
			"stat", "info",
			// 本地命令
			"lpwd", "lcd", "lls", "ldir", "lmkdir",
		},
	}
}

// Do 执行自动补全
// readline 会用返回的候选项替换最后 length 个字符
func (c *Completer) Do(line []rune, pos int) (newLine [][]rune, length int) {
	text := string(line[:pos])
	fields := strings.Fields(text)

	// 空输入：补全命令
	if len(fields) == 0 {
		return c.completeCommand(""), 0
	}

	// 只有命令且未输入空格，补全命令
	if len(fields) == 1 && !strings.HasSuffix(text, " ") {
		return c.completeCommand(fields[0]), len(fields[0])
	}

	// 获取当前正在输入的参数（可能为空）
	var currentArg string
	if strings.HasSuffix(text, " ") {
		// 刚输入完空格，开始新参数
		currentArg = ""
	} else {
		// 正在输入参数
		currentArg = fields[len(fields)-1]
	}

	cmd := fields[0]
	hasTrailingSpace := strings.HasSuffix(text, " ")
	optExpectValue := ""
	if hasTrailingSpace {
		if len(fields) > 1 {
			prev := fields[len(fields)-1]
			if prev == "-d" || prev == "--dir" || prev == "--name" {
				optExpectValue = prev
			}
		}
	} else {
		if len(fields) > 2 {
			prev := fields[len(fields)-2]
			if prev == "-d" || prev == "--dir" || prev == "--name" {
				optExpectValue = prev
			}
		}
	}

	switch cmd {
	case "cd", "ls", "ll", "dir", "rm", "del", "delete", "rmdir", "rd", "stat", "info":
		// 远程路径补全
		return c.completeRemotePath(currentArg), len(currentArg)
	case "lcd", "lls", "ldir", "lmkdir":
		// 本地路径补全
		return c.completeLocalPath(currentArg), len(currentArg)
	case "get", "download":
		switch optExpectValue {
		case "-d", "--dir":
			return c.completeLocalPath(currentArg), len(currentArg)
		case "--name":
			return nil, 0
		default:
			return c.completeRemotePath(currentArg), len(currentArg)
		}
	case "put", "upload":
		switch optExpectValue {
		case "-d", "--dir":
			return c.completeRemotePath(currentArg), len(currentArg)
		case "--name":
			return nil, 0
		default:
			return c.completeLocalPath(currentArg), len(currentArg)
		}
	default:
		return nil, 0
	}
}

// ToReadline 转换为 readline 的 AutoCompleter
func (c *Completer) ToReadline() readline.AutoCompleter {
	return readline.NewPrefixCompleter()
}

// ========================== Internal Helpers ==========================

// completeFromCandidates computes completion suffixes from a list of candidates.
func completeFromCandidates(candidates []string, prefix string) [][]rune {
	if len(candidates) == 0 {
		return nil
	}
	if len(candidates) == 1 {
		if len(candidates[0]) > len(prefix) {
			return [][]rune{[]rune(candidates[0][len(prefix):])}
		}
		return [][]rune{[]rune("")}
	}
	common := longestCommonPrefix(candidates)
	if len(common) > len(prefix) {
		return [][]rune{[]rune(common[len(prefix):])}
	}
	var results [][]rune
	for _, candidate := range candidates {
		if len(candidate) > len(prefix) {
			results = append(results, []rune(candidate[len(prefix):]))
		} else {
			results = append(results, []rune(candidate))
		}
	}
	return results
}

// completeCommand 补全命令
func (c *Completer) completeCommand(prefix string) [][]rune {
	var candidates []string
	for _, cmd := range c.cmdList {
		if strings.HasPrefix(cmd, prefix) {
			candidates = append(candidates, cmd+" ")
		}
	}
	return completeFromCandidates(candidates, prefix)
}

// completeRemotePath 补全远程路径
func (c *Completer) completeRemotePath(prefix string) [][]rune {
	candidates := c.client.ListCompletion(prefix)
	return completeFromCandidates(candidates, prefix)
}

// completeLocalPath 补全本地路径
func (c *Completer) completeLocalPath(prefix string) [][]rune {
	// 解析目录和文件名部分
	dir, partial := filepath.Split(prefix)
	searchDir := dir
	if searchDir == "" {
		// 使用 SFTP shell 的本地工作目录，而不是进程当前目录
		searchDir = c.client.GetLocalwd()
	}

	// 处理 ~ 前缀
	if strings.HasPrefix(prefix, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			if prefix == "~" {
				searchDir = home
				partial = ""
				dir = "~" + string(os.PathSeparator)
			} else if strings.HasPrefix(prefix, "~/") || strings.HasPrefix(prefix, "~\\") {
				searchDir = filepath.Join(home, dir[2:])
			}
		}
	}

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return nil
	}

	// 收集所有匹配的名称
	var candidates []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(partial)) {
			if entry.IsDir() {
				name += "/"
			}
			candidates = append(candidates, name)
		}
	}

	return completeFromCandidates(candidates, partial)
}

// longestCommonPrefix 计算字符串列表的最长公共前缀
func longestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	// 以第一个字符串为基准
	prefix := strs[0]
	for i := 1; i < len(strs); i++ {
		for len(prefix) > 0 && !strings.HasPrefix(strs[i], prefix) {
			prefix = prefix[:len(prefix)-1]
		}
		if prefix == "" {
			break
		}
	}
	return prefix
}
