package shell

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chzyer/readline"

	"my-sftp/client"
	"my-sftp/completer"
)

// Shell 交互式 Shell
type Shell struct {
	client    *client.Client
	rl        *readline.Instance
	completer *completer.Completer
}

// NewShell 创建 Shell
func NewShell(c *client.Client) *Shell {
	comp := completer.NewCompleter(c)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          c.Getwd() + " > ",
		HistoryFile:     filepath.Join(os.TempDir(), "my-sftp-history"),
		AutoComplete:    comp,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		panic(err)
	}

	return &Shell{
		client:    c,
		rl:        rl,
		completer: comp,
	}
}

// Run 运行交互式循环
func (s *Shell) Run() error {
	defer s.rl.Close()

	for {
		s.rl.SetPrompt(fmt.Sprintf("\033[32m%s\033[0m > ", s.client.Getwd()))

		line, err := s.rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				if len(line) == 0 {
					break
				}
				continue
			}
			if err == io.EOF {
				break
			}
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if err := s.executeCommand(line); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}

	return nil
}

// executeCommand 执行命令
func (s *Shell) executeCommand(line string) error {
	fields := parseCommandLine(line)
	if len(fields) == 0 {
		return nil
	}

	cmd := fields[0]
	args := fields[1:]

	switch cmd {
	case "help", "?":
		s.showHelp()
	case "exit", "quit", "q":
		fmt.Println("Goodbye!")
		os.Exit(0)
	case "pwd":
		fmt.Println(s.client.Getwd())
	case "cd":
		return s.cmdCd(args)
	case "ls", "ll", "dir":
		return s.cmdLs(args)
	case "get", "download":
		return s.cmdGet(args)
	case "put", "upload":
		return s.cmdPut(args)
	case "rm", "del", "delete":
		return s.cmdRm(args)
	case "mkdir", "md":
		return s.cmdMkdir(args)
	case "rmdir", "rd":
		return s.cmdRmdir(args)
	case "rename", "mv":
		return s.cmdRename(args)
	case "stat", "info":
		return s.cmdStat(args)
	// 本地命令
	case "lpwd":
		fmt.Println(s.client.GetLocalwd())
	case "lcd":
		return s.cmdLcd(args)
	case "lls", "ldir":
		return s.cmdLls(args)
	case "lmkdir":
		return s.cmdLmkdir(args)
	default:
		return fmt.Errorf("unknown command: %s (type 'help' for available commands)", cmd)
	}

	return nil
}

// parseCommandLine 解析命令行，支持引号包裹的参数
func parseCommandLine(line string) []string {
	var fields []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for i, r := range line {
		switch {
		case r == '"' || r == '\'':
			if !inQuote {
				// 开始引号
				inQuote = true
				quoteChar = r
			} else if r == quoteChar {
				// 结束引号
				inQuote = false
				quoteChar = 0
			} else {
				// 引号内的不同引号字符
				current.WriteRune(r)
			}
		case r == ' ' || r == '\t':
			if inQuote {
				// 引号内的空格保留
				current.WriteRune(r)
			} else if current.Len() > 0 {
				// 字段结束
				fields = append(fields, current.String())
				current.Reset()
			}
		case r == '\\':
			// 转义字符
			if i+1 < len(line) {
				next := rune(line[i+1])
				if next == '"' || next == '\'' || next == '\\' {
					// 跳过当前的反斜杠，下一个字符会被正常添加
					continue
				}
			}
			current.WriteRune(r)
		default:
			current.WriteRune(r)
		}
	}

	// 添加最后一个字段
	if current.Len() > 0 {
		fields = append(fields, current.String())
	}

	return fields
}

// formatSize 格式化文件大小为人类可读格式
func formatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case size >= TB:
		return fmt.Sprintf("%.2f TB", float64(size)/float64(TB))
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// showHelp 显示帮助
func (s *Shell) showHelp() {
	help := `
Available commands:
  Remote Navigation:
    pwd                    Print remote working directory
    cd <dir>              Change remote directory
    ls [dir]              List remote directory contents
    ll [dir]              List with details (alias of ls)
  
  Local Navigation:
    lpwd                   Print local working directory
    lcd <dir>             Change local directory
    lls [dir]             List local directory contents
    lmkdir <dir>          Create local directory
  
  File Transfer:
    get [-r] <remote> [local]      Download file or directory from server
    put [-r] <local|pattern> [remote]  Upload file(s) or directory to server
    
    Options:
      -r                   Recursive mode for directories
    
    Examples:
      put file.txt                   Upload single file
      put *.log logs/                Upload all .log files
      put **/*.go code/              Upload all .go files recursively
      put -r mydir remotedir/        Upload entire directory
      get -r remotedir localdir/     Download entire directory
  
  Remote File Operations:
    rm <path>             Remove file or directory
    mkdir <dir>           Create directory
    rmdir <dir>           Remove directory
    rename <old> <new>    Rename file or directory
    stat <path>           Show file information
  
  Other:
    help                  Show this help
    exit/quit/q           Exit program

Features:
  ✓ Progress bar with transfer speed for all file operations
  ✓ Glob pattern matching (*, **, ?, [])
  ✓ Recursive directory upload/download
  ✓ Concurrent file transfers (up to 4 parallel)
  ✓ Buffered I/O for better performance (512KB buffer)

Tips:
  - Use TAB for auto-completion
  - Paths can be absolute (/path) or relative (./path)
  - Use ~ for home directory (both local and remote)
  - Directories in completion end with /
  - Use quotes for paths with spaces: "my folder/file.txt"
  - Use glob patterns for batch operations: *.txt, **/*.go
`
	fmt.Println(help)
}

// cmdCd 切换目录
func (s *Shell) cmdCd(args []string) error {
	dir := "~"
	if len(args) > 0 {
		dir = args[0]
	}
	return s.client.Chdir(dir)
}

// cmdLs 列出目录
func (s *Shell) cmdLs(args []string) error {
	dir := ""
	if len(args) > 0 {
		dir = args[0]
	}

	// 用户主动执行 ls 时，清除缓存以获取最新内容
	s.client.ClearDirCache()

	files, err := s.client.List(dir)
	if err != nil {
		return err
	}

	fmt.Printf("Total: %d items\n", len(files))
	for _, file := range files {
		typeChar := "-"
		if file.IsDir() {
			typeChar = "d"
		}

		fmt.Printf("%s %10s  %s  %s\n",
			typeChar,
			formatSize(file.Size()),
			file.ModTime().Format("2006-01-02 15:04:05"),
			file.Name(),
		)
	}

	return nil
}

// cmdGet 下载文件
func (s *Shell) cmdGet(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: get [-r] <remote_file> [local_path]")
	}

	// 解析参数
	recursive := false
	startIdx := 0
	if args[0] == "-r" {
		recursive = true
		startIdx = 1
		if len(args) < 2 {
			return fmt.Errorf("usage: get -r <remote_path> [local_path]")
		}
	}

	remotePath := args[startIdx]
	localPath := filepath.Base(remotePath)
	if len(args) > startIdx+1 {
		localPath = args[startIdx+1]
	}

	// 开始计时
	startTime := time.Now()

	// 检查是否是目录
	stat, err := s.client.Stat(remotePath)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		if !recursive {
			return fmt.Errorf("%s is a directory, use 'get -r' for recursive download", remotePath)
		}
		// 递归下载目录
		count, err := s.client.DownloadDir(remotePath, localPath, &client.DownloadOptions{
			Recursive:    true,
			ShowProgress: true,
		})
		if err != nil {
			return err
		}
		duration := time.Since(startTime)
		fmt.Printf("✓ Downloaded %d file(s) in %s\n", count, duration.Round(time.Millisecond))
		return nil
	}

	// 下载单个文件
	if err := s.client.Download(remotePath, localPath); err != nil {
		return err
	}

	duration := time.Since(startTime)

	// 显示文件大小和用时
	if stat, err := os.Stat(localPath); err == nil {
		fmt.Printf("✓ Downloaded: %s in %s\n", formatSize(stat.Size()), duration.Round(time.Millisecond))
	}

	return nil
}

// cmdPut 上传文件
func (s *Shell) cmdPut(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: put [-r] <local_file|pattern> [remote_path]")
	}

	// 解析参数
	recursive := false
	startIdx := 0
	if args[0] == "-r" {
		recursive = true
		startIdx = 1
		if len(args) < 2 {
			return fmt.Errorf("usage: put -r <local_path> [remote_path]")
		}
	}

	localPath := args[startIdx]
	remotePath := "."
	if len(args) > startIdx+1 {
		remotePath = args[startIdx+1]
	}

	// 开始计时
	startTime := time.Now()

	// 检查是否包含 glob 模式
	hasGlob := strings.ContainsAny(localPath, "*?[]")

	if hasGlob {
		// Glob 模式匹配上传
		count, err := s.client.UploadGlob(localPath, remotePath, &client.UploadOptions{
			Recursive:    recursive,
			ShowProgress: true,
			Concurrency:  client.MaxConcurrentTransfers,
		})
		if err != nil {
			return err
		}
		duration := time.Since(startTime)
		fmt.Printf("✓ Uploaded %d file(s) in %s\n", count, duration.Round(time.Millisecond))
		return nil
	}

	// 检查本地文件/目录
	stat, err := os.Stat(localPath)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		if !recursive {
			return fmt.Errorf("%s is a directory, use 'put -r' for recursive upload", localPath)
		}
		// 递归上传目录
		count, err := s.client.UploadDir(localPath, remotePath, &client.UploadOptions{
			Recursive:    true,
			ShowProgress: true,
		})
		if err != nil {
			return err
		}
		duration := time.Since(startTime)
		fmt.Printf("✓ Uploaded %d file(s) in %s\n", count, duration.Round(time.Millisecond))
		return nil
	}

	// 上传单个文件
	if err := s.client.Upload(localPath, remotePath); err != nil {
		return err
	}

	duration := time.Since(startTime)
	fmt.Printf("✓ Uploaded successfully (%s) in %s\n", formatSize(stat.Size()), duration.Round(time.Millisecond))
	return nil
}

// cmdRm 删除文件或目录
func (s *Shell) cmdRm(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: rm <path>")
	}

	for _, path := range args {
		fmt.Printf("Removing %s ...\n", path)
		if err := s.client.Remove(path); err != nil {
			return err
		}
	}

	fmt.Println("Removed successfully")
	return nil
}

// cmdMkdir 创建目录
func (s *Shell) cmdMkdir(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mkdir <directory>")
	}

	for _, dir := range args {
		if err := s.client.Mkdir(dir); err != nil {
			return err
		}
		fmt.Printf("Created: %s\n", dir)
	}

	return nil
}

// cmdRmdir 删除目录
func (s *Shell) cmdRmdir(args []string) error {
	return s.cmdRm(args)
}

// cmdRename 重命名
func (s *Shell) cmdRename(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: rename <old_path> <new_path>")
	}

	if err := s.client.Rename(args[0], args[1]); err != nil {
		return err
	}

	fmt.Printf("Renamed: %s -> %s\n", args[0], args[1])
	return nil
}

// cmdStat 查看文件信息
func (s *Shell) cmdStat(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: stat <path>")
	}

	stat, err := s.client.Stat(args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Path:     %s\n", args[0])
	fmt.Printf("Type:     %s\n", s.fileType(stat))
	fmt.Printf("Size:     %s (%d bytes)\n", formatSize(stat.Size()), stat.Size())
	fmt.Printf("Modified: %s\n", stat.ModTime().Format("2006-01-02 15:04:05"))
	fmt.Printf("Mode:     %s\n", stat.Mode())

	return nil
}

// fileType 获取文件类型描述
func (s *Shell) fileType(info os.FileInfo) string {
	if info.IsDir() {
		return "Directory"
	}
	return "Regular File"
}

// ==================== 本地命令 ====================

// cmdLcd 切换本地目录
func (s *Shell) cmdLcd(args []string) error {
	dir := "~"
	if len(args) > 0 {
		dir = args[0]
	}
	return s.client.LocalChdir(dir)
}

// cmdLls 列出本地目录
func (s *Shell) cmdLls(args []string) error {
	dir := ""
	if len(args) > 0 {
		dir = args[0]
	}

	files, err := s.client.LocalList(dir)
	if err != nil {
		return err
	}

	fmt.Printf("Local: %d items\n", len(files))
	for _, file := range files {
		typeChar := "-"
		if file.IsDir() {
			typeChar = "d"
		}

		fmt.Printf("%s %10s  %s  %s\n",
			typeChar,
			formatSize(file.Size()),
			file.ModTime().Format("2006-01-02 15:04:05"),
			file.Name(),
		)
	}

	return nil
}

// cmdLmkdir 创建本地目录
func (s *Shell) cmdLmkdir(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: lmkdir <directory>")
	}

	for _, dir := range args {
		if err := s.client.LocalMkdir(dir); err != nil {
			return err
		}
		fmt.Printf("Created local: %s\n", dir)
	}

	return nil
}
