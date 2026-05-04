---
name: bugfixes-and-asymmetry-fixes
status: PLANNING
change-type: single
created: 2026-05-05 01:57:15
reference:
- source: .sspec/requests/260505013145-bugs-and-oddities.md
  type: request
  note: Linked from request
---

# bugfixes-and-asymmetry-fixes

## Problem Statement

Code review found 7 issues: 2 bugs break core functionality (Windows path parsing, upload collision key), 2 behavioral asymmetries violate user expectations (empty directory upload vs download, `rmdir` = `rm -r`), 2 code smells cause maintenance drift (duplicate format functions, duplicate completer logic), and 1 by-design behavior (`globRemote` non-`**` depth limit) needs no change.

Impact: Windows users cannot paste local paths in the shell; case-sensitive remote paths get false collision errors from Windows/macOS clients; destructive `rmdir` is masked as a safe command.

## Proposed Solution

### Approach

Fix each issue surgically. Introduce a `remoteCaseSensitive` field on `Client` detected at connection time via SFTP probe (create mixed-case temp file → stat opposite case → cleanup). Fallback to case-sensitive with warning log if probe fails. `rmdir` gets its own implementation using SFTP `RemoveDirectory` (empty-only). Empty directory upload creates the remote directory and reports success. `parseCommandLine` makes `\` literal outside quotes. Two refactors (format function unification, completer dedup) are bundled as low-risk cleanups.

### Key Change

- **Fix 1: Backslash path parsing** — `parseCommandLine` treats `\` as literal outside quoted regions; inside quotes `\` only escapes `"`, `'`, `\`, and whitespace. Windows users can paste `C:\Users\file.txt` directly.
- **Fix 2: Upload collision key case sensitivity** — `targetConflictKey` and `flattenCollisionKey` use runtime-detected `remoteCaseSensitive` field instead of `runtime.GOOS`. Probe logic in `NewClient`; result logged at connection.
- **Fix 3: Empty directory upload** — `UploadSources` creates remote empty directory + prints `✓ Created empty directory <path>`, returns 0 file count. Symmetric with download.
- **Fix 4: `rmdir` empty-only semantics** — New `client.RemoveDir` using `sftpClient.RemoveDirectory`. Non-empty → error with hint: `rmdir: directory not empty: <dir> (use "rm" to remove recursively)`.
- **Refactor 5: Format function unification** — Replace `shell.formatSize` and `client.formatBytes` with single exported `client.FormatSize` (loop algorithm, `%.1f` precision).
- **Refactor 6: Completer dedup** — Extract `completeFromCandidates` shared function; remove `removePrefix`.

### Scope Summary

| File | Change |
|------|--------|
| `shell/shell.go` | Rewrite `parseCommandLine` backslash handling; replace `formatSize` calls with `client.FormatSize`; update `cmdRmdir` to call `client.RemoveDir` |
| `client/client.go` | Add `remoteCaseSensitive bool` field; add `probeRemoteCaseSensitivity()` in `NewClient`; log result |
| `client/transfer.go` | Convert `targetConflictKey`/`flattenCollisionKey` to `Client` methods using `remoteCaseSensitive`; remove `formatBytes`, replace calls with `FormatSize` |
| `client/upload.go` | Handle empty directory: propagate `emptyDirs []string` from `collectUploadTasks`; `UploadSources` creates dirs + prints confirmation |
| `client/common.go` | Add `RemoveDir(remotePath string) error`; add `FormatSize(bytes int64) string`; add `probeRemoteCaseSensitivity() bool` |
| `completer/completer.go` | Extract `completeFromCandidates`; simplify all three completion methods; remove `removePrefix` |
| `client/transfer_test.go` | Update case-related tests for `remoteCaseSensitive` field; add collision key method tests |
| `shell/shell_test.go` | Add `parseCommandLine` backslash tests (quoted/unquoted, Windows paths) |

### Design Reference

→ See [design.md](./design.md)