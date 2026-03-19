package shell

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/chzyer/readline"

	"github.com/frostime/my-sftp/client"
	"github.com/frostime/my-sftp/completer"
)

const legacyPositionalTargetCompatibility = true

type transferCLIOptions struct {
	recursive bool
	flatten   bool
	targetDir string
	rename    string
	sources   []string
}

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

// ==================== Internal ====================

// executeCommand 执行命令
func (s *Shell) executeCommand(line string) error {
	// 检查 !! 前缀（本地命令）- 必须先检查 !! 再检查 !
	if strings.HasPrefix(line, "!!") {
		cmdStr := strings.TrimSpace(strings.TrimPrefix(line, "!!"))
		if cmdStr == "" {
			return fmt.Errorf("usage: !! <local_command>")
		}
		return s.cmdExecLocal(cmdStr)
	}

	// 检查 ! 前缀（远程命令）
	if strings.HasPrefix(line, "!") {
		cmdStr := strings.TrimSpace(strings.TrimPrefix(line, "!"))
		if cmdStr == "" {
			return fmt.Errorf("usage: ! <remote_command>")
		}
		return s.cmdExecRemote(cmdStr)
	}

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
	escaped := false

	for _, r := range line {
		if escaped {
			// 上一个是反斜杠：当前字符一律当普通字符写入
			current.WriteRune(r)
			escaped = false
			continue
		}

		switch r {
		case '\\':
			// 下一个字符被转义
			escaped = true

		case '"', '\'':
			if inQuote {
				if r == quoteChar {
					// 结束当前引号
					inQuote = false
					quoteChar = 0
				} else {
					// 引号内的另一种引号，直接写入
					current.WriteRune(r)
				}
			} else {
				// 开始新的引号
				inQuote = true
				quoteChar = r
			}

		case ' ', '\t':
			if inQuote {
				current.WriteRune(r)
			} else if current.Len() > 0 {
				fields = append(fields, current.String())
				current.Reset()
			}

		default:
			current.WriteRune(r)
		}
	}

	// 末尾还有内容就收尾
	if escaped {
		// 行尾单独一个反斜杠，就把它当普通字符
		current.WriteRune('\\')
	}
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
	get [-r] [--flatten] [-d dir] [--name name] <remote|pattern>...  Download file(s) or directory from server
	put [-r] [--flatten] [-d dir] [--name name] <local|pattern>...   Upload file(s) or directory to server

    Options:
	  -r                   Recursive mode for directories
	  -d, --dir            Destination directory (local for get, remote for put)
	  --name               Rename single-file destination name
	  --flatten            Flatten multi-source structure into target root

    Examples:
	  get file.txt                           Download single file to current local dir
	  get file.txt -d downloads --name x.txt Download single file with rename
	  get **/*.go -d code                    Download recursively and preserve structure
	  get **/*.go -d code --flatten          Download recursively and flatten output
	  get -r remotedir -d localdir           Download entire directory recursively
	  put file.txt                           Upload single file to current remote dir
	  put file.txt -d /data/inbox --name x.txt Upload single file with rename
	  put **/*.go -d /srv/code               Upload recursively and preserve structure
	  put **/*.go -d /srv/code --flatten     Upload recursively and flatten output
	  put -r mydir -d /srv/remotedir         Upload entire directory recursively

  Remote File Operations:
    rm <path>             Remove file or directory
    mkdir <dir>           Create directory
    rmdir <dir>           Remove directory
    rename <old> <new>    Rename file or directory
    stat <path>           Show file information

  Shell Commands:
    ! <command>           Execute command on remote server
    !! <command>          Execute command on local machine

    Examples:
      ! tree -L 2              List remote directory tree
      ! cat config.yaml        View remote file content
      ! df -h                  Check remote disk usage
      !! dir                   List local directory (Windows)
      !! ls -la                List local directory (Linux/Mac)

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

func parseTransferCLIArgs(args []string) (*transferCLIOptions, error) {
	opts := &transferCLIOptions{}

	for i := 0; i < len(args); i++ {
		tok := args[i]
		switch tok {
		case "-r":
			opts.recursive = true
		case "--flatten":
			opts.flatten = true
		case "-d", "--dir":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("missing value for %s", tok)
			}
			opts.targetDir = args[i]
		case "--name":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("missing value for --name")
			}
			opts.rename = args[i]
		default:
			if strings.HasPrefix(tok, "-") {
				return nil, fmt.Errorf("unknown option: %s", tok)
			}
			opts.sources = append(opts.sources, tok)
		}
	}

	if len(opts.sources) == 0 {
		return nil, fmt.Errorf("missing source path")
	}

	return opts, nil
}

func (s *Shell) inferLegacyGetTarget(remotePaths []string) ([]string, string, bool) {
	if len(remotePaths) <= 1 {
		return remotePaths, "", false
	}

	lastArg := remotePaths[len(remotePaths)-1]
	resolvedLast := s.client.ResolveLocalPath(lastArg)
	if stat, err := os.Stat(resolvedLast); err == nil && stat.IsDir() {
		return remotePaths[:len(remotePaths)-1], lastArg, true
	}

	return remotePaths, "", false
}

func (s *Shell) inferLegacyPutTarget(localPaths []string) ([]string, string, bool) {
	if len(localPaths) <= 1 {
		return localPaths, "", false
	}

	lastArg := localPaths[len(localPaths)-1]
	resolvedLast := s.client.ResolveLocalPath(lastArg)
	if _, err := os.Stat(resolvedLast); os.IsNotExist(err) {
		return localPaths[:len(localPaths)-1], lastArg, true
	}

	return localPaths, "", false
}

// cmdGet 下载文件
func (s *Shell) cmdGet(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: get [-r] [--flatten] [-d <local_dir>] [--name <filename>] <remote_src>...")
	}

	opts, err := parseTransferCLIArgs(args)
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}

	remotePaths := opts.sources
	localDir := opts.targetDir
	if localDir == "" && len(remotePaths) > 1 {
		if legacyPositionalTargetCompatibility {
			var usedLegacy bool
			remotePaths, localDir, usedLegacy = s.inferLegacyGetTarget(remotePaths)
			if usedLegacy {
				fmt.Println("Warning: legacy positional target syntax is deprecated; use -d <local_dir>")
			}
		}
		if localDir == "" {
			return fmt.Errorf("multiple get sources require destination: use -d <local_dir>")
		}
	}
	if localDir == "" {
		localDir = "."
	}

	if opts.rename != "" && len(remotePaths) != 1 {
		return fmt.Errorf("--name is only valid with exactly one source file")
	}

	// 开始计时
	startTime := time.Now()
	totalCount := 0

	for _, remotePath := range remotePaths {
		// 检查是否包含 glob 模式
		hasGlob := strings.ContainsAny(remotePath, "*?[]")
		if hasGlob && opts.rename != "" {
			return fmt.Errorf("--name cannot be used with glob source: %s", remotePath)
		}

		if hasGlob {
			// Glob 模式匹配下载
			count, err := s.client.DownloadGlob(remotePath, localDir, &client.DownloadOptions{
				Recursive:    opts.recursive,
				ShowProgress: true,
				Concurrency:  client.MaxConcurrentTransfers,
				Flatten:      opts.flatten,
			})
			if err != nil {
				return err
			}
			totalCount += count
			continue
		}

		// 检查是否是目录
		stat, err := s.client.Stat(remotePath)
		if err != nil {
			return err
		}

		if stat.IsDir() {
			if opts.rename != "" {
				return fmt.Errorf("--name cannot be used with directory source: %s", remotePath)
			}
			if !opts.recursive {
				return fmt.Errorf("%s is a directory, use 'get -r' for recursive download", remotePath)
			}
			// 递归下载目录
			count, err := s.client.DownloadDir(remotePath, localDir, &client.DownloadOptions{
				Recursive:    true,
				ShowProgress: true,
				Flatten:      opts.flatten,
			})
			if err != nil {
				return err
			}
			totalCount += count
		} else {
			// 下载单个文件
			targetPath := filepath.Join(localDir, filepath.Base(remotePath))
			if len(remotePaths) == 1 && opts.targetDir == "" && opts.rename == "" {
				targetPath = filepath.Base(remotePath)
			}
			if opts.rename != "" {
				targetPath = filepath.Join(localDir, opts.rename)
			}
			if err := s.client.Download(remotePath, targetPath); err != nil {
				return err
			}
			totalCount++
		}
	}

	duration := time.Since(startTime)
	fmt.Printf("✓ Downloaded %d file(s) in %s\n", totalCount, duration.Round(time.Millisecond))
	return nil
}

// cmdPut 上传文件
func (s *Shell) cmdPut(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: put [-r] [--flatten] [-d <remote_dir>] [--name <filename>] <local_src>...")
	}

	opts, err := parseTransferCLIArgs(args)
	if err != nil {
		return fmt.Errorf("put: %w", err)
	}

	localPaths := opts.sources
	remoteDir := opts.targetDir
	if remoteDir == "" && len(localPaths) > 1 {
		if legacyPositionalTargetCompatibility {
			var usedLegacy bool
			localPaths, remoteDir, usedLegacy = s.inferLegacyPutTarget(localPaths)
			if usedLegacy {
				fmt.Println("Warning: legacy positional target syntax is deprecated; use -d <remote_dir>")
			}
		}
		if remoteDir == "" {
			return fmt.Errorf("multiple put sources require destination: use -d <remote_dir>")
		}
	}
	if remoteDir == "" {
		remoteDir = "."
	}

	if opts.rename != "" && len(localPaths) != 1 {
		return fmt.Errorf("--name is only valid with exactly one source file")
	}

	// 开始计时
	startTime := time.Now()
	totalCount := 0

	for _, localPath := range localPaths {
		// 检查是否包含 glob 模式
		hasGlob := strings.ContainsAny(localPath, "*?[]")
		if hasGlob && opts.rename != "" {
			return fmt.Errorf("--name cannot be used with glob source: %s", localPath)
		}

		if hasGlob {
			// Glob 模式匹配上传
			count, err := s.client.UploadGlob(localPath, remoteDir, &client.UploadOptions{
				Recursive:    opts.recursive,
				ShowProgress: true,
				Concurrency:  client.MaxConcurrentTransfers,
				Flatten:      opts.flatten,
			})
			if err != nil {
				return err
			}
			totalCount += count
			continue
		}

		// 解析本地路径（基于 localWorkDir）
		resolvedPath := s.client.ResolveLocalPath(localPath)

		// 检查本地文件/目录
		stat, err := os.Stat(resolvedPath)
		if err != nil {
			return err
		}

		if stat.IsDir() {
			if opts.rename != "" {
				return fmt.Errorf("--name cannot be used with directory source: %s", localPath)
			}
			if !opts.recursive {
				return fmt.Errorf("%s is a directory, use 'put -r' for recursive upload", localPath)
			}
			// 递归上传目录
			count, err := s.client.UploadDir(localPath, remoteDir, &client.UploadOptions{
				Recursive:    true,
				ShowProgress: true,
				Flatten:      opts.flatten,
			})
			if err != nil {
				return err
			}
			totalCount += count
		} else {
			// 上传单个文件
			targetPath := path.Join(remoteDir, filepath.Base(localPath))
			if len(localPaths) == 1 && opts.targetDir == "" && opts.rename == "" {
				targetPath = remoteDir
			}
			if opts.rename != "" {
				targetPath = path.Join(remoteDir, opts.rename)
			}
			if err := s.client.Upload(localPath, targetPath); err != nil {
				return err
			}
			totalCount++
		}
	}

	duration := time.Since(startTime)
	fmt.Printf("✓ Uploaded %d file(s) in %s\n", totalCount, duration.Round(time.Millisecond))
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

// ==================== Shell 命令执行 ====================

// cmdExecRemote 在远程服务器执行命令
func (s *Shell) cmdExecRemote(cmdStr string) error {
	fmt.Printf("[Remote] Executing: %s\n", cmdStr)
	// 直接绑定终端的 stdin/stdout/stderr，支持交互式命令
	if err := s.client.ExecuteRemote(cmdStr, os.Stdin, os.Stdout, os.Stderr); err != nil {
		return fmt.Errorf("remote command failed: %w", err)
	}
	return nil
}

// cmdExecLocal 在本地执行命令
func (s *Shell) cmdExecLocal(cmdStr string) error {
	fmt.Printf("[Local] Executing: %s\n", cmdStr)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", cmdStr)
	} else {
		cmd = exec.Command("sh", "-c", cmdStr)
	}

	// 设置工作目录为当前本地工作目录
	cmd.Dir = s.client.GetLocalwd()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("local command failed: %w", err)
	}
	return nil
}
