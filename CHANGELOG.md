# Changelog

## v0.10.1

### Bug Fixes

- **shell**: `parseCommandLine` now treats backslash as literal character outside quotes — Windows paths like `C:\Users\file.txt` can be pasted directly into the interactive shell
- **client**: Remote filesystem case sensitivity is detected at runtime via probe file; upload collision keys (`flattenCollisionKey`, `targetConflictKey`) use remote detection instead of local `runtime.GOOS`
- **client**: Uploading an empty directory with `put -r` now creates the remote directory instead of returning "no files found" error — symmetric with download behavior
- **shell**: `rmdir` now uses SFTP `RemoveDirectory` (empty-only) instead of delegating to recursive `rm`; error message hints user to use `rm` for recursive deletion
- **client**: `probeRemoteCaseSensitivity` cleanup: `defer` ensures probe file removal; `os.IsNotExist` distinguishes "file not found" from network/permission errors; PID suffix prevents concurrent probe interference
- **completer**: `completeFromCandidates` returns empty suffix on exact match instead of full candidate, preventing readline from duplicating already-complete input

### Refactors

- **client/shell**: Unified `formatSize` / `formatBytes` into single exported `client.FormatSize` (binary units, `%.1f` precision)
- **completer**: Extracted `completeFromCandidates` helper, replacing three duplicated completion implementations (`completeCommand`, `completeRemotePath`, `completeLocalPath`)

---

## v0.10.0

### Features

- Explicit transfer syntax with round-by-round deviation log
