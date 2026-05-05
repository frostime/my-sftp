# Memory: bugfixes-and-asymmetry-fixes

**Updated**: 2026-05-05T02:20

## Git Baseline (Immutable)

- Captured: before change file creation
- Repository: `D:/Arsenal/PlayCode/my-sftp`
- Branch: `main`
- HEAD: `2128679cf16315b40dbc0d38cfb378a4c2d0411c`
- Worktree: `dirty`

## State

Implementation complete. All 6 phases done, 63/63 tests pass. Next: check spec-doc alignment, then commit spec-doc updates if needed.

## Key Files

- `client/common.go` — `FormatSize`, `RemoveDir`, `probeRemoteCaseSensitivity` (all new)
- `client/client.go` — `remoteCaseSensitive` field, probe call in `NewClient`
- `client/transfer.go` — `targetConflictKey`/`flattenCollisionKey`/`validateTargetCollisions`/`applyFlattenMapping` now Client methods; `formatBytes` deleted
- `client/upload.go` — `collectUploadTasks`/`collectUploadSourceTasks`/`collectUploadGlobTasks` return `emptyDirs`; `UploadSources` handles empty dir creation
- `shell/shell.go` — `parseCommandLine` backslash fix; `cmdRmdir` uses `RemoveDir`; `formatSize` deleted, uses `client.FormatSize`
- `completer/completer.go` — `completeFromCandidates` extracted; `removePrefix` deleted

## Knowledge

- [2026-05-05] [Insight] `sftpClient.RemoveDirectory()` = SFTP RMDIR (empty-only by protocol spec). Safe for `rmdir` command.
- [2026-05-05] [Insight] `targetConflictKey` and `flattenCollisionKey` are free functions — must become `Client` methods to access `remoteCaseSensitive` field. Callers in `upload.go`/`download.go` already have `c *Client`.
- [2026-05-05] [Gotcha] `parseCommandLine` change: outside quotes `\` becomes literal, meaning `\t` outside quotes → literal 2-char string, NOT tab. Acceptable because tab is the completion key, not a typable character.
- [2026-05-05] [Gotcha] Case-sensitivity probe: `/tmp` may not be writable. Fallback chain: `/tmp` → workDir (`sftpClient.Getwd()`) → default case-sensitive + warning log. No SSH exec dependency.
- [2026-05-05] [Gotcha] `collectUploadTasks` currently returns `([]transferTask, error)`. Needs `([]transferTask, []string, error)` to propagate `emptyDirs`. Callers in `upload.go` must propagate.
- [2026-05-05] [Constraint] `FormatSize` must be exported (`client.FormatSize`) so `shell` package can call it. Both consumers already import `client`.
- [2026-05-05] [Insight] subagent implemented correctly with `opencode-go/mimo-v2.5-pro` model + `high` thinking. `mimo-plan/mimo-2.5-pro` was rejected (not supported).

## Milestones

- [2026-05-05T01:57] Clarify complete: 7 issues triaged, 5 actionable + 2 code smells + 1 by-design noop
- [2026-05-05T02:10] Design draft: spec.md + design.md written, awaiting gate
- [2026-05-05T02:15] Plan: tasks.md written (6 phases, 14 tasks)
- [2026-05-05T02:20] Implementation: all 6 phases complete, 63/63 tests pass