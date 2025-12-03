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

	switch cmd {
	case "cd", "ls", "ll", "dir", "rm", "del", "delete", "rmdir", "rd", "stat", "info", "get", "download":
		// 远程路径补全
		return c.completeRemotePath(currentArg), len(currentArg)
	case "put", "upload", "lcd", "lls", "ldir", "lmkdir":
		// 本地路径补全
		return c.completeLocalPath(currentArg), len(currentArg)
	default:
		return nil, 0
	}
}

// ToReadline 转换为 readline 的 AutoCompleter
func (c *Completer) ToReadline() readline.AutoCompleter {
	return readline.NewPrefixCompleter()
}

// ========================== Internal Helpers ==========================

func removePrefix(candidates [][]rune, prefix string) [][]rune {
	var results [][]rune
	for _, candidate := range candidates {
		candStr := string(candidate)
		if strings.HasPrefix(candStr, prefix) {
			results = append(results, []rune(candStr[len(prefix):]))
		} else {
			results = append(results, candidate)
		}
	}
	return results
}

// completeCommand 补全命令
func (c *Completer) completeCommand(prefix string) [][]rune {
	var matches [][]rune
	for _, cmd := range c.cmdList {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, []rune(cmd+" "))
		}
	}
	// return matches
	return removePrefix(matches, prefix)
}

// completeRemotePath 补全远程路径
func (c *Completer) completeRemotePath(prefix string) [][]rune {
	candidates := c.client.ListCompletion(prefix)
	if len(candidates) == 0 {
		return nil
	}

	// 如果只有一个匹配，返回需要补全的后缀部分
	if len(candidates) == 1 {
		// 只返回未输入的部分
		if len(candidates[0]) > len(prefix) {
			suffix := candidates[0][len(prefix):]
			return [][]rune{[]rune(suffix)}
		}
		return [][]rune{[]rune(candidates[0])}
	}

	// 多个匹配：计算公共前缀
	common := longestCommonPrefix(candidates)
	if len(common) > len(prefix) {
		// 可以补全更多内容，返回差异部分
		suffix := common[len(prefix):]
		return [][]rune{[]rune(suffix)}
	}

	// 无法进一步补全，返回所有候选供用户选择（去掉已输入的prefix）
	var matches [][]rune
	for _, candidate := range candidates {
		if len(candidate) > len(prefix) {
			suffix := candidate[len(prefix):]
			matches = append(matches, []rune(suffix))
		} else {
			matches = append(matches, []rune(candidate))
		}
	}
	return matches
}

// completeLocalPath 补全本地路径
func (c *Completer) completeLocalPath(prefix string) [][]rune {
	// 解析目录和文件名部分
	dir, partial := filepath.Split(prefix)
	searchDir := dir
	if searchDir == "" {
		searchDir = "."
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

	// 收集所有匹配的名称（不包含前缀路径）
	var candidates []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(partial)) {
			// 只保存匹配的文件/目录名
			if entry.IsDir() {
				name += string(os.PathSeparator)
			}
			candidates = append(candidates, name)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// 如果只有一个匹配，返回需要补全的部分（去掉已输入的partial）
	if len(candidates) == 1 {
		// 返回未输入的部分
		suffix := candidates[0][len(partial):]
		return [][]rune{[]rune(suffix)}
	}

	// 多个匹配：计算公共前缀
	common := longestCommonPrefix(candidates)
	if len(common) > len(partial) {
		// 可以补全更多内容，返回差异部分
		suffix := common[len(partial):]
		return [][]rune{[]rune(suffix)}
	}

	// 无法进一步补全，返回所有候选供用户选择（去掉已输入的partial）
	var matches [][]rune
	for _, candidate := range candidates {
		suffix := candidate[len(partial):]
		matches = append(matches, []rune(suffix))
	}
	return matches
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
