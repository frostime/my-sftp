package client

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/schollz/progressbar/v3"
)

const (
	preserveMetaPrefix   = "__my_sftp_"
	preserveParentMarker = "__my_sftp_parent__"
)

// formatBytes 将字节数格式化为人类可读的形式
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// transferTask 表示单个传输任务
type transferTask struct {
	localPath  string // 本地文件路径
	remotePath string // 远程文件路径
	isUpload   bool   // true=上传, false=下载
	size       int64  // 文件大小，用于进度显示
}

type transferSourceEntry struct {
	path  string
	isDir bool
	size  int64
}

// TransferOptions 统一的传输选项
type TransferOptions struct {
	Recursive    bool // 递归处理目录
	ShowProgress bool // 显示进度条
	Concurrency  int  // 并发数
	MaxDepth     int  // 最大递归深度：-1=无限, 0=仅当前目录, 1=一层子目录...
}

func flattenCollisionError(base string) error {
	return fmt.Errorf("duplicate basename in --flatten mode: %s\nHint: remove --flatten or narrow source set", base)
}

func taskSourceBaseName(task transferTask) string {
	if task.isUpload {
		return filepath.Base(task.localPath)
	}
	return path.Base(task.remotePath)
}

func flattenCollisionKey(task transferTask) string {
	base := taskSourceBaseName(task)
	if !task.isUpload && (runtime.GOOS == "windows" || runtime.GOOS == "darwin") {
		return strings.ToLower(base)
	}
	return base
}

func applyFlattenMapping(tasks []transferTask, targetRoot string) error {
	seen := make(map[string]struct{}, len(tasks))
	for _, task := range tasks {
		key := flattenCollisionKey(task)
		if _, exists := seen[key]; exists {
			return flattenCollisionError(taskSourceBaseName(task))
		}
		seen[key] = struct{}{}
	}

	for i := range tasks {
		base := taskSourceBaseName(tasks[i])
		if tasks[i].isUpload {
			tasks[i].remotePath = path.Join(targetRoot, base)
			continue
		}
		tasks[i].localPath = filepath.Join(targetRoot, base)
	}

	return nil
}

func targetConflictKey(task transferTask) string {
	if task.isUpload {
		return path.Clean(task.remotePath)
	}

	key := filepath.ToSlash(filepath.Clean(task.localPath))
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		key = strings.ToLower(key)
	}
	return key
}

func validateTargetCollisions(tasks []transferTask) error {
	seen := make(map[string]string, len(tasks))
	for _, task := range tasks {
		target := targetConflictKey(task)
		if original, exists := seen[target]; exists {
			return fmt.Errorf("duplicate target path in transfer plan: %s conflicts with %s", original, taskTargetPath(task))
		}
		seen[target] = taskTargetPath(task)
	}

	for key, original := range seen {
		for ancestor := path.Dir(key); ancestor != "." && ancestor != "/" && ancestor != key; ancestor = path.Dir(ancestor) {
			if parent, exists := seen[ancestor]; exists {
				return fmt.Errorf("target path conflict in transfer plan: %s conflicts with descendant %s", parent, original)
			}
		}
	}

	return nil
}

func taskTargetPath(task transferTask) string {
	if task.isUpload {
		return path.Clean(task.remotePath)
	}
	return filepath.Clean(task.localPath)
}

func ensureLocalDirsExist(tasks []transferTask) error {
	for _, task := range tasks {
		if task.isUpload {
			continue
		}
		dir := filepath.Dir(task.localPath)
		if dir == "." || dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create local dir %s: %w", dir, err)
		}
	}
	return nil
}

func sanitizeSlashRelativePath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	p = path.Clean(p)
	if p == "" || p == "." || p == "/" {
		return ""
	}
	if strings.HasPrefix(p, "/") {
		return strings.TrimLeft(p, "/")
	}

	parentCount := 0
	for p == ".." || strings.HasPrefix(p, "../") {
		parentCount++
		p = strings.TrimPrefix(p, "../")
	}
	if p == "" || p == "." {
		if parentCount == 0 {
			return ""
		}
		p = ""
	}
	if parentCount == 0 {
		return strings.TrimPrefix(p, "./")
	}

	parts := make([]string, 0, parentCount+1)
	for i := 0; i < parentCount; i++ {
		parts = append(parts, preserveParentMarker)
	}
	if p != "" {
		parts = append(parts, strings.TrimPrefix(p, "./"))
	}
	return path.Join(parts...)
}

func resolvedRemoteBaseName(resolvedSource string) string {
	base := path.Base(path.Clean(resolvedSource))
	if base == "/" || base == "." {
		return ""
	}
	return base
}

func resolvedLocalBaseName(resolvedSource string) string {
	base := filepath.Base(filepath.Clean(resolvedSource))
	if base == string(filepath.Separator) || base == "." {
		return ""
	}
	return base
}

func sanitizeLocalVolume(volume string) string {
	volume = strings.ReplaceAll(volume, "\\", "/")
	volume = strings.TrimSuffix(volume, ":")
	volume = strings.Trim(volume, "/")
	if volume == "" {
		return ""
	}
	return preserveMetaPrefix + "volume_" + strings.ToLower(volume) + "__"
}

func usesReservedPreservePrefix(source string, isLocal bool) bool {
	if source == "" || source == "~" {
		return false
	}

	cleaned := source
	if isLocal {
		volume := filepath.VolumeName(cleaned)
		if volume != "" {
			return false
		}
	}
	cleaned = strings.ReplaceAll(cleaned, "\\", "/")
	if strings.HasPrefix(cleaned, "~/") {
		cleaned = cleaned[2:]
	}
	cleaned = strings.TrimLeft(path.Clean(cleaned), "/")
	if cleaned == "" || cleaned == "." || cleaned == ".." {
		return false
	}
	for _, segment := range strings.Split(cleaned, "/") {
		if strings.HasPrefix(segment, preserveMetaPrefix) {
			return true
		}
	}
	return false
}

func explicitRemoteFilePreservePath(source, resolvedSource string) string {
	if strings.HasPrefix(source, "~/") {
		rel := sanitizeSlashRelativePath(source[2:])
		if rel != "" {
			return rel
		}
		return resolvedRemoteBaseName(resolvedSource)
	}
	if source == "~" {
		return resolvedRemoteBaseName(resolvedSource)
	}

	rel := sanitizeSlashRelativePath(source)
	if rel != "" {
		return rel
	}
	return resolvedRemoteBaseName(resolvedSource)
}

func explicitLocalFilePreservePath(source, resolvedSource string) string {
	if strings.HasPrefix(source, "~/") || strings.HasPrefix(source, "~\\") {
		rel := sanitizeSlashRelativePath(source[2:])
		if rel != "" {
			return rel
		}
		return resolvedLocalBaseName(resolvedSource)
	}
	if source == "~" {
		return resolvedLocalBaseName(resolvedSource)
	}

	cleaned := filepath.Clean(source)
	volume := filepath.VolumeName(cleaned)
	rel := sanitizeSlashRelativePath(filepath.ToSlash(strings.TrimPrefix(cleaned, volume)))
	volumePrefix := sanitizeLocalVolume(volume)
	if rel != "" {
		if volumePrefix != "" {
			return path.Join(volumePrefix, rel)
		}
		return rel
	}
	base := resolvedLocalBaseName(resolvedSource)
	if volumePrefix != "" {
		if base == "" {
			return volumePrefix
		}
		return path.Join(volumePrefix, base)
	}
	return base
}

func localGlobPreservePrefix(globBase, globBaseAbs string) string {
	if globBase == "" || globBase == "." || globBase == string(filepath.Separator) {
		return ""
	}
	return explicitLocalFilePreservePath(globBase, globBaseAbs)
}

func remoteGlobPreservePrefix(globBase, globBaseAbs string) string {
	if globBase == "" || globBase == "." || globBase == "/" {
		return ""
	}
	return explicitRemoteFilePreservePath(globBase, globBaseAbs)
}

func joinPreservePath(prefix, rel string) string {
	rel = strings.TrimPrefix(rel, "./")
	if rel == "." {
		rel = ""
	}
	switch {
	case prefix == "":
		return rel
	case rel == "":
		return prefix
	default:
		return path.Join(prefix, rel)
	}
}

func taskSourcePath(task transferTask) string {
	if task.isUpload {
		return filepath.Clean(task.localPath)
	}
	return path.Clean(task.remotePath)
}

func dedupeResolvedSourceTasks(tasks []transferTask) []transferTask {
	seen := make(map[string]struct{}, len(tasks))
	deduped := make([]transferTask, 0, len(tasks))
	for _, task := range tasks {
		key := taskSourcePath(task)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, task)
	}
	return deduped
}

func normalizeMatchedSourceEntries(entries []transferSourceEntry, isLocal, recursive bool) []transferSourceEntry {
	if len(entries) == 0 {
		return nil
	}

	normalized := make([]transferSourceEntry, 0, len(entries))
	selectedDirs := make([]string, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))

	sort.Slice(entries, func(i, j int) bool {
		left := normalizedSourcePath(entries[i].path, isLocal)
		right := normalizedSourcePath(entries[j].path, isLocal)
		leftDepth := strings.Count(left, "/")
		rightDepth := strings.Count(right, "/")
		if leftDepth != rightDepth {
			return leftDepth < rightDepth
		}
		return left < right
	})

	for _, entry := range entries {
		entry.path = cleanedSourcePath(entry.path, isLocal)
		key := normalizedSourcePath(entry.path, isLocal)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		if hasSelectedAncestorDir(key, selectedDirs) {
			continue
		}

		if entry.isDir {
			if !recursive {
				continue
			}
			selectedDirs = append(selectedDirs, key)
		}

		normalized = append(normalized, entry)
	}

	return normalized
}

func cleanedSourcePath(source string, isLocal bool) string {
	if isLocal {
		return filepath.Clean(source)
	}
	return path.Clean(source)
}

func normalizedSourcePath(source string, isLocal bool) string {
	cleaned := cleanedSourcePath(source, isLocal)
	if isLocal {
		return filepath.ToSlash(cleaned)
	}
	return cleaned
}

func hasSelectedAncestorDir(source string, selectedDirs []string) bool {
	for _, dir := range selectedDirs {
		prefix := dir + "/"
		if strings.HasPrefix(source, prefix) {
			return true
		}
	}
	return false
}

// DefaultTransferOptions 返回默认传输选项
func DefaultTransferOptions() *TransferOptions {
	return &TransferOptions{
		Recursive:    true,
		ShowProgress: true,
		Concurrency:  MaxConcurrentTransfers,
		MaxDepth:     -1, // 默认无限深度
	}
}

// executeTasks 统一的任务执行引擎
// 这是所有并发传输的唯一入口点，避免并发嵌套问题
func (c *Client) executeTasks(tasks []transferTask, opts *TransferOptions) (int, error) {
	if len(tasks) == 0 {
		return 0, nil
	}

	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = MaxConcurrentTransfers
	}
	if concurrency > len(tasks) {
		concurrency = len(tasks)
	}

	// 并发控制信号量
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error
	var successCount int32 = 0

	// 计算总字节数和文件数
	totalBytes := int64(0)
	for _, task := range tasks {
		totalBytes += task.size
	}
	totalFiles := len(tasks)

	// 整体进度条（字节级 + 文件计数）
	var globalBar *progressbar.ProgressBar
	var completedFiles *atomic.Int32

	if opts.ShowProgress {
		globalBar = progressbar.NewOptions64(totalBytes,
			progressbar.OptionSetDescription(fmt.Sprintf("Transferring (0/%d files)", totalFiles)),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(40),
			progressbar.OptionSetPredictTime(true),
			progressbar.OptionClearOnFinish(),
		)
		completedFiles = &atomic.Int32{}
	}

	for _, task := range tasks {
		wg.Add(1)
		sem <- struct{}{} // 获取信号量

		go func(t transferTask) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			// panic 保护
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("panic during transfer %s: %v\nstack: %s",
						t.localPath, r, debug.Stack()))
					mu.Unlock()
				}
			}()

			// 显示当前正在传输的文件（多文件模式）
			if globalBar != nil {
				fileName := filepath.Base(t.localPath)
				if !t.isUpload {
					fileName = path.Base(t.remotePath)
				}
				count := completedFiles.Load()
				globalBar.Describe(fmt.Sprintf("Transferring %s (%d/%d files)", fileName, count, totalFiles))
			}

			var err error
			if t.isUpload {
				err = c.UploadWithProgress(t.localPath, t.remotePath, globalBar)
			} else {
				err = c.DownloadWithProgress(t.remotePath, t.localPath, globalBar)
			}

			if err != nil {
				mu.Lock()
				if t.isUpload {
					errs = append(errs, fmt.Errorf("upload %s: %w", t.localPath, err))
				} else {
					errs = append(errs, fmt.Errorf("download %s: %w", t.remotePath, err))
				}
				mu.Unlock()
			} else {
				atomic.AddInt32(&successCount, 1)
				// 文件完成后打印确认信息并更新计数
				if globalBar != nil && completedFiles != nil {
					count := completedFiles.Add(1)
					fileName := filepath.Base(t.localPath)
					if !t.isUpload {
						fileName = path.Base(t.remotePath)
					}
					// 打印完成信息
					fmt.Printf("\r\033[K✓ %s (%s)\n", fileName, formatBytes(t.size))
					globalBar.Describe(fmt.Sprintf("Transferring (%d/%d files)", count, totalFiles))
				}
			}
		}(task)
	}

	wg.Wait()

	if globalBar != nil {
		globalBar.Finish()
		fmt.Println() // 换行
	}

	if len(errs) > 0 {
		return int(successCount), errors.Join(errs...)
	}
	return int(successCount), nil
}

// collectDownloadTasks 收集下载任务（不执行传输）
// remoteDir: 远程目录路径
// localDir: 本地目录路径
// maxDepth: 最大递归深度，-1表示无限
// currentDepth: 当前深度（内部使用）
func (c *Client) collectDownloadTasks(remoteDir, localDir string, maxDepth, currentDepth int) ([]transferTask, error) {
	var tasks []transferTask

	entries, err := c.sftpClient.ReadDir(remoteDir)
	if err != nil {
		return nil, fmt.Errorf("read remote dir %s: %w", remoteDir, err)
	}

	for _, entry := range entries {
		remotePath := path.Join(remoteDir, entry.Name())
		localPath := filepath.Join(localDir, entry.Name())

		if entry.IsDir() {
			// 检查深度限制
			if maxDepth >= 0 && currentDepth >= maxDepth {
				continue // 超过深度限制，跳过此目录
			}

			// 递归收集子目录任务
			subTasks, err := c.collectDownloadTasks(remotePath, localPath, maxDepth, currentDepth+1)
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, subTasks...)
		} else {
			tasks = append(tasks, transferTask{
				localPath:  localPath,
				remotePath: remotePath,
				isUpload:   false,
				size:       entry.Size(),
			})
		}
	}

	return tasks, nil
}

// collectUploadTasks 收集上传任务（不执行传输）
// localDir: 本地目录路径
// remoteDir: 远程目录路径
// maxDepth: 最大递归深度，-1表示无限
// currentDepth: 当前深度（内部使用）
func (c *Client) collectUploadTasks(localDir, remoteDir string, maxDepth, currentDepth int) ([]transferTask, error) {
	var tasks []transferTask

	entries, err := os.ReadDir(localDir)
	if err != nil {
		return nil, fmt.Errorf("read local dir %s: %w", localDir, err)
	}

	for _, entry := range entries {
		localPath := filepath.Join(localDir, entry.Name())
		remotePath := path.Join(remoteDir, entry.Name())

		if entry.IsDir() {
			// 检查深度限制
			if maxDepth >= 0 && currentDepth >= maxDepth {
				continue // 超过深度限制，跳过此目录
			}

			// 递归收集子目录任务
			subTasks, err := c.collectUploadTasks(localPath, remotePath, maxDepth, currentDepth+1)
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, subTasks...)
		} else {
			info, err := entry.Info()
			if err != nil {
				continue // 跳过无法获取信息的文件
			}
			tasks = append(tasks, transferTask{
				localPath:  localPath,
				remotePath: remotePath,
				isUpload:   true,
				size:       info.Size(),
			})
		}
	}

	return tasks, nil
}

// collectRemoteDirsForUpload 收集上传任务中需要创建的所有远程目录
// 返回按创建顺序排列的目录列表（父目录在前）
func (c *Client) collectRemoteDirsForUpload(tasks []transferTask) []string {
	dirSet := make(map[string]struct{})
	var dirs []string

	for _, task := range tasks {
		if !task.isUpload {
			continue
		}
		dir := path.Dir(task.remotePath)
		// 收集从根到目标目录的所有路径
		for dir != "/" && dir != "." {
			if _, exists := dirSet[dir]; !exists {
				dirSet[dir] = struct{}{}
				dirs = append(dirs, dir)
			}
			dir = path.Dir(dir)
		}
	}

	// 按路径长度排序，确保父目录在前
	sort.Slice(dirs, func(i, j int) bool {
		depthI := strings.Count(dirs[i], "/")
		depthJ := strings.Count(dirs[j], "/")
		if depthI != depthJ {
			return depthI < depthJ
		}
		return dirs[i] < dirs[j]
	})
	return dirs
}

// ensureRemoteDirsExist 确保所有远程目录存在
// 使用 singleflight 避免重复创建
func (c *Client) ensureRemoteDirsExist(dirs []string) error {
	for _, dir := range dirs {
		if err := c.ensureRemoteDir(dir); err != nil {
			return err
		}
	}
	return nil
}
