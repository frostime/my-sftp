package shell

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"

	"my-sftp/client"
	"my-sftp/completer"
)

// Shell 交互式 Shell
type Shell struct {
	client   *client.Client
	rl       *readline.Instance
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
	fields := strings.Fields(line)
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
    get <remote> [local]  Download file from server
    put <local> [remote]  Upload file to server
  
  Remote File Operations:
    rm <path>             Remove file or directory
    mkdir <dir>           Create directory
    rmdir <dir>           Remove directory
    rename <old> <new>    Rename file or directory
    stat <path>           Show file information
  
  Other:
    help                  Show this help
    exit/quit/q           Exit program

Tips:
  - Use TAB for auto-completion
  - Paths can be absolute (/path) or relative (./path)
  - Use ~ for home directory (both local and remote)
  - Directories in completion end with /
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
		
		fmt.Printf("%s %10d  %s  %s\n",
			typeChar,
			file.Size(),
			file.ModTime().Format("2006-01-02 15:04:05"),
			file.Name(),
		)
	}

	return nil
}

// cmdGet 下载文件
func (s *Shell) cmdGet(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: get <remote_file> [local_path]")
	}

	remotePath := args[0]
	localPath := filepath.Base(remotePath)
	if len(args) > 1 {
		localPath = args[1]
	}

	fmt.Printf("Downloading %s -> %s ...\n", remotePath, localPath)
	if err := s.client.Download(remotePath, localPath); err != nil {
		return err
	}

	// 显示文件大小
	if stat, err := os.Stat(localPath); err == nil {
		fmt.Printf("Downloaded: %d bytes\n", stat.Size())
	}

	return nil
}

// cmdPut 上传文件
func (s *Shell) cmdPut(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: put <local_file> [remote_path]")
	}

	localPath := args[0]
	remotePath := filepath.Base(localPath)
	if len(args) > 1 {
		remotePath = args[1]
	}

	// 检查本地文件
	stat, err := os.Stat(localPath)
	if err != nil {
		return err
	}

	fmt.Printf("Uploading %s -> %s (%d bytes) ...\n", localPath, remotePath, stat.Size())
	if err := s.client.Upload(localPath, remotePath); err != nil {
		return err
	}

	fmt.Println("Uploaded successfully")
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
	fmt.Printf("Size:     %d bytes\n", stat.Size())
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

		fmt.Printf("%s %10d  %s  %s\n",
			typeChar,
			file.Size(),
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
