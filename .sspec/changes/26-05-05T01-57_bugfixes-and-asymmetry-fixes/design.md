---
change: "bugfixes-and-asymmetry-fixes"
created: 2026-05-05
---

# Design: bugfixes-and-asymmetry-fixes

## 1. parseCommandLine Backslash Handling

### Behavioral Spec

```
Outside quotes:
  \ is a LITERAL character (no escape semantics)
  Input:  put C:\Users\file.txt
  Result: ["put", "C:\\Users\\file.txt"]

Inside double quotes:
  \" → literal "
  \\ → literal \
  \space → literal space (only these 3 trigger escape in quotes)
  \t  → literal "t" (NOT tab — \ only escapes " \ space)
  Any other \x → literal x (defensive fallback)
  Input:  put "C:\Program Files\"
  Result: ["put", "C:\\Program Files\\"]

Inside single quotes:
  \ is always literal (no escape)
  Input:  put 'C:\Users\file.txt'
  Result: ["put", "C:\\Users\\file.txt"]
```

### Implementation

Key change in `parseCommandLine`:

```go
case '\\':
    if inQuote {
        // Inside quotes: backslash is an escape prefix
        escaped = true
    } else {
        // Outside quotes: backslash is a literal character
        current.WriteRune(r)
    }
```

Only the `case '\\'` branch changes. All other logic remains identical.

---

## 2. Remote Filesystem Case Sensitivity Detection

### Detection Probe

```go
func (c *Client) probeRemoteCaseSensitivity() bool {
    // 1. Create temp file with mixed-case name in workDir
    // 2. Stat with opposite-case name
    // 3. Cleanup temp file
    // 4. If opposite-case stat succeeds → case-insensitive (return false)
    //    If opposite-case stat fails → case-sensitive (return true)
    // Fallback: if probe fails (no write access) → assume case-sensitive + log warning
}
```

### Client Field + Connection Log

```go
type Client struct {
    // ... existing fields ...
    remoteCaseSensitive bool  // true = case-sensitive (Linux default)
}

// In NewClient, after sftpClient init:
c.remoteCaseSensitive = c.probeRemoteCaseSensitivity()
if c.remoteCaseSensitive {
    fmt.Println("ℹ Remote filesystem: case-sensitive")
} else {
    fmt.Println("ℹ Remote filesystem: case-insensitive (case-variant filenames treated as same path)")
}
```

### Collision Key Methods

```go
func (c *Client) targetConflictKey(task transferTask) string {
    if task.isUpload {
        key := path.Clean(task.remotePath)
        if !c.remoteCaseSensitive {
            key = strings.ToLower(key)
        }
        return key
    }
    // Download: local OS determines case sensitivity
    key := filepath.ToSlash(filepath.Clean(task.localPath))
    if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
        key = strings.ToLower(key)
    }
    return key
}
```

Same pattern for `flattenCollisionKey`.

### Method Conversion

Free functions become `Client` methods:

```
targetConflictKey(task)           → c.targetConflictKey(task)
flattenCollisionKey(task)         → c.flattenCollisionKey(task)
validateTargetCollisions(tasks)   → c.validateTargetCollisions(tasks)
applyFlattenMapping(tasks, root)  → c.applyFlattenMapping(tasks, root)
```

Callers in `upload.go` / `download.go` already have `c *Client`.

---

## 3. Empty Directory Upload

### Data Flow Change

```go
func (c *Client) collectUploadTasks(...) ([]transferTask, []string, error) {
    // ... existing walk logic ...
    if len(tasks) == 0 {
        return nil, []string{resolvedRemoteDir}, nil  // emptyDirs
    }
    return tasks, nil, nil
}
```

Propagated through `collectUploadSourceTasks` → `UploadSources`.

### UploadSources Handling

```go
// In UploadSources, after collecting all tasks:
if len(tasks) == 0 && len(emptyDirs) > 0 {
    for _, dir := range emptyDirs {
        if err := c.ensureRemoteDir(dir); err != nil { return 0, err }
        fmt.Printf("✓ Created empty directory %s\n", dir)
    }
    return 0, nil
}
```

### Output

```
> put -r emptydir -d /srv/backup
✓ Created empty directory /srv/backup/emptydir
Transferred 0 files, 1 directory created
```

---

## 4. rmdir Empty-Only Semantics

### New Client Method

```go
func (c *Client) RemoveDir(remotePath string) error {
    remotePath = c.ResolveRemotePath(remotePath)
    stat, err := c.sftpClient.Stat(remotePath)
    if err != nil { return fmt.Errorf("stat: %w", err) }
    if !stat.IsDir() { return fmt.Errorf("not a directory: %s", remotePath) }
    err = c.sftpClient.RemoveDirectory(remotePath)
    if err != nil {
        return fmt.Errorf("rmdir: directory not empty: %s (use \"rm\" to remove recursively)", remotePath)
    }
    c.invalidateDirCache(path.Dir(remotePath))
    return nil
}
```

`sftpClient.RemoveDirectory` = SFTP `RMDIR` (empty-only by protocol).

### Shell Change

```go
func (s *Shell) cmdRmdir(args []string) error {
    if len(args) < 1 { return fmt.Errorf("usage: rmdir <dir>") }
    for _, dir := range args {
        if err := s.client.RemoveDir(dir); err != nil { return err }
        fmt.Printf("Removed directory: %s\n", dir)
    }
    return nil
}
```

---

## 5. Format Function Unification

```go
// FormatSize formats bytes into human-readable form (binary units, 1 decimal).
func FormatSize(bytes int64) string {
    const unit = 1024
    if bytes < unit { return fmt.Sprintf("%d B", bytes) }
    div, exp := int64(unit), 0
    for n := bytes / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
```

Location: `client/common.go`. Exported so `shell` can call `client.FormatSize()`.

Migration:
| Old | New | File |
|-----|-----|------|
| `formatSize(file.Size())` | `client.FormatSize(file.Size())` | `shell/shell.go` |
| `formatBytes(t.size)` | `FormatSize(t.size)` | `client/transfer.go` |

---

## 6. Completer Dedup

### Shared Function

```go
func completeFromCandidates(candidates []string, prefix string) [][]rune {
    if len(candidates) == 0 { return nil }
    if len(candidates) == 1 {
        if len(candidates[0]) > len(prefix) {
            return [][]rune{[]rune(candidates[0][len(prefix):])}
        }
        return [][]rune{[]rune(candidates[0])}
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
```

### Usage

```go
func (c *Completer) completeCommand(prefix string) [][]rune {
    var candidates []string
    for _, cmd := range c.cmdList {
        if strings.HasPrefix(cmd, prefix) { candidates = append(candidates, cmd+" ") }
    }
    return completeFromCandidates(candidates, prefix)
}

func (c *Completer) completeRemotePath(prefix string) [][]rune {
    candidates := c.client.ListCompletion(prefix)
    return completeFromCandidates(candidates, prefix)
}
```

`completeLocalPath`: existing dir/partial resolution produces `[]string candidates`, then `return completeFromCandidates(candidates, prefix)`.

Remove `removePrefix` — subsumed by `completeFromCandidates`.

---

## Outcome Preview

| Scenario | Before | After |
|----------|--------|-------|
| Windows: `put C:\Users\file.txt` | Error: parsed as `C:Usersfile.txt` | Works: literal backslash |
| Windows→Linux upload `A.txt` + `a.txt` | False collision error | Both upload (case-sensitive remote) |
| macOS→macOS upload `A.txt` + `a.txt` | No collision → silent overwrite | Collision detected |
| `put -r emptydir -d /out` | Error: "no files found" | `✓ Created empty directory /out/emptydir` |
| `rmdir nonemptydir` | Recursively deletes | Error with `rm` hint |
| `ls` size column | `1.23 MB` (shell) vs `1.2 MB` (transfer) | Both `1.2 MB` via `FormatSize` |