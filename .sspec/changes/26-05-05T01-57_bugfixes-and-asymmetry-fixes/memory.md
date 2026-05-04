# Memory: bugfixes-and-asymmetry-fixes

**Updated**: 2026-05-05T02:10

## Git Baseline (Immutable)

- Captured: before change file creation
- Repository: `D:/Arsenal/PlayCode/my-sftp`
- Branch: `main`
- HEAD: `2128679cf16315b40dbc0d38cfb378a4c2d0411c`
- Worktree: `dirty`

## State

Design gate pending. All 6 changes specified in spec.md + design.md. Next: user confirms design → proceed to plan.

## Key Files

- `shell/shell.go` — `parseCommandLine` (backslash bug), `cmdRmdir` (delegates to `cmdRm`), `formatSize` (duplicate)
- `client/transfer.go` — `targetConflictKey` + `flattenCollisionKey` (free functions → Client methods), `formatBytes` (duplicate), `validateTargetCollisions`, `applyFlattenMapping`
- `client/upload.go` — `UploadSources` empty dir error, `collectUploadTasks` return signature change
- `client/common.go` — new `RemoveDir`, `FormatSize`, `probeRemoteCaseSensitivity`
- `client/client.go` — new `remoteCaseSensitive` field, probe call in `NewClient`
- `completer/completer.go` — `completeCommand`/`completeRemotePath`/`completeLocalPath` dedup

## Knowledge

- [2026-05-05] [Insight] `sftpClient.RemoveDirectory()` = SFTP RMDIR (empty-only by protocol spec). Safe for `rmdir` command.
- [2026-05-05] [Insight] `targetConflictKey` and `flattenCollisionKey` are free functions — must become `Client` methods to access `remoteCaseSensitive` field. Callers in `upload.go`/`download.go` already have `c *Client`.
- [2026-05-05] [Gotcha] `parseCommandLine` change: outside quotes `\` becomes literal, meaning `\t` outside quotes → literal 2-char string, NOT tab. Acceptable because tab is the completion key, not a typable character.
- [2026-05-05] [Gotcha] Case-sensitivity probe: `/tmp` may not be writable. Fallback chain: `/tmp` → workDir (`sftpClient.Getwd()`) → default case-sensitive + warning log. No SSH exec dependency.
- [2026-05-05] [Gotcha] `collectUploadTasks` currently returns `([]transferTask, error)`. Needs `([]transferTask, []string, error)` to propagate `emptyDirs`. Callers in `upload.go` must propagate.
- [2026-05-05] [Constraint] `FormatSize` must be exported (`client.FormatSize`) so `shell` package can call it. Both consumers already import `client`.

## Milestones

- [2026-05-05T01:57] Clarify complete: 7 issues triaged, 5 actionable + 2 code smells + 1 by-design noop
- [2026-05-05T02:10] Design draft: spec.md + design.md written, awaiting gate