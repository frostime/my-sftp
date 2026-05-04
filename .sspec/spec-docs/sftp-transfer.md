---
name: sftp-transfer
description: "Transfer pipeline, progress strategy, concurrency model, and edge case handling"
updated: 2026-05-05
scope:
  - /client/transfer.go
  - /client/upload.go
  - /client/download.go
  - /client/common.go
---

# SFTP Transfer

## Transfer Pipeline

```
Source resolution → Task collection → Pre-flight validation → Directory prep → Concurrent execution
```

| Stage | Function | File | Responsibility |
|-------|----------|------|---------------|
| 1. Resolve | `collectUploadSourceTasks` / `collectDownloadSourceTasks` | upload.go / download.go | Explicit path stat or glob expansion |
| 2. Collect | `collectUploadTasks` / `collectDownloadTasks` | transfer.go | Recursive dir walk → `[]transferTask` |
| 3. Validate | `applyFlattenMapping` + `validateTargetCollisions` | transfer.go | Flatten mapping, duplicate target detection |
| 4. Prep dirs | `collectRemoteDirsForUpload` + `ensureRemoteDirsExist` | transfer.go / upload.go | Singleflight-protected `MkdirAll` |
| 5. Execute | `executeTasks` | transfer.go | Semaphore-concurrent goroutines, progress bar, error aggregation |

### Options Layering

Two option types exist by design:

- **`UploadOptions` / `DownloadOptions`** (shell→client API): what the user asked for (`Recursive`, `Flatten`, `MaxDepth`, `ShowProgress`, `Concurrency`).
- **`TransferOptions`** (client internal): what `executeTasks` needs (`Recursive`, `ShowProgress`, `Concurrency`, `MaxDepth`).

`UploadSources` / `DownloadSources` convert the former to the latter. Keep this layering when adding new flags — Shell owns CLI syntax, Client owns execution.

## Progress Display

| Scenario | Mode | Source |
|----------|------|--------|
| Single file, concurrency=1 | Per-file byte progress bar | `Upload` / `Download` in upload.go / download.go |
| Multi-file, concurrency>1 | Global byte progress bar + file counter | `executeTasks` in transfer.go |

Progress adapts via `globalBar` (nil → per-file mode, non-nil → global mode).

**Multi-file display**:
- Dynamic description: `Transferring <filename> (n/N files)`
- Completion line: `✓ <filename> (<size>)` printed above progress bar
- Uses `\r\033[K` to clear and reprint

## Concurrency Model

```
executeTasks(tasks, opts)
  sem = make(chan struct{}, concurrency)   // semaphore
  for each task:
      sem <- struct{}{}                    // acquire
      go func:
          defer <-sem                      // release
          defer recover()                  // panic isolation
          UploadWithProgress / DownloadWithProgress
          → atomic successCount / mutex-guarded errs
```

| Mechanism | Purpose | Location |
|-----------|---------|----------|
| Channel semaphore | Cap concurrent goroutines at `opts.Concurrency` (default 4) | transfer.go |
| `atomic.Int32` | Lock-free success counter | transfer.go |
| `sync.Mutex` on `[]error` | Thread-safe error collection | transfer.go |
| `defer recover()` | Per-goroutine panic isolation | transfer.go |
| `singleflight.Group` | Dedup concurrent `MkdirAll` for same dir | upload.go |
| `sync.Pool` | Reuse 512KB transfer buffers | client.go |

## Edge Cases

| Case | Behavior | Code path |
|------|----------|-----------|
| Empty directory source | 0 tasks returned, remote dir still created | `collectUploadTasks` returns `[]` |
| Duplicate basename + `--flatten` | Hard error pre-flight, no transfer starts | `applyFlattenMapping` |
| Duplicate target path | Hard error pre-flight | `validateTargetCollisions` |
| Target is ancestor of another target | Hard error pre-flight | `validateTargetCollisions` ancestor check |
| Source uses `__my_sftp_` prefix | Hard error (reserved marker) | `usesReservedPreservePrefix` |
| Parent-relative `..` in source | Encoded as `__my_sftp_parent__` in target path | `sanitizeSlashRelativePath` |
| `--name` with glob source | Error: `--name` only for single file | `cmdGet` / `cmdPut` |
| `--name` with directory source | Error | `cmdGet` / `cmdPut` |
| Dir source without `-r` | Error: suggest `put -r` / `get -r` | `collectUploadSourceTasks` / `collectDownloadSourceTasks` |
| Remote path is a directory | Auto-append basename to remote path | `UploadWithProgress` |
| Network interruption | Transfer fails, error collected, others continue | `executeTasks` error aggregation |
| Upload empty directory | Error: "no files found in directory" | `UploadSources` |
| Download empty directory | Local dir created, 0 files transferred | `DownloadDir` special-cases `count == 0` |

### `--name` bypasses the pipeline

`--name` is handled in `cmdGet` / `cmdPut` **before** `DownloadSources` / `UploadSources`. It calls `client.Download()` / `client.Upload()` directly (single-file, no task collection, no `executeTasks`). Do not route `--name` through the multi-file pipeline.

### `sourceCount` semantics

`collectUploadSourceTasks` / `collectDownloadSourceTasks` receive `sourceCount = len(sources)` (the number of **operands** the user typed, **not** the number of matched files). This drives two behaviors:

- `sourceCount > 1` → enable `explicitLocal/RemoteFilePreservePath` to retain operand-relative structure
- `sourceCount > 1` → enable `usesReservedPreservePrefix` guard

A glob like `*.txt` has `sourceCount == 1` even if it matches 100 files. Its preserve structure is determined by the glob base, not `sourceCount`.

### Glob remote recursion

`globRemote` only recurses into subdirectories when the pattern contains `**`. A pattern like `dir/*/file.txt` walks only the base path (`dir/`) — it does **not** descend into `dir/sub/` even if `*` could theoretically match a subdirectory name. This is a performance guard, not a generic glob traversal.

### Glob match deduplication

`normalizeMatchedSourceEntries` does three things to glob results:

1. Sorts by depth (parent dirs first)
2. Deduplicates by cleaned path
3. **Filters out entries already covered by a selected ancestor directory**

Step 3 is why `dir/**` (which matches both `dir` and `dir/file`) does not produce duplicate tasks.

### Defensive directory creation in `UploadWithProgress`

`UploadWithProgress` calls `ensureRemoteDir(parent)` even though `UploadSources` already pre-creates all needed directories. This is intentional: `--name` routes directly to `UploadWithProgress` and skips `UploadSources`. Removing the inner call would break `--name` uploads. Do not "clean up" this redundancy.

### Windows volume marker

`sanitizeLocalVolume` encodes a Windows drive letter (e.g. `C:` → `__my_sftp_volume_c__`) only when **all** of the following are true:

- Local path (upload source or download target)
- Windows OS
- Multi-source preserve mode (`sourceCount > 1` && `!Flatten`)
- Absolute path with a drive letter

It prevents cross-volume files from colliding under the same preserve root. Single-source and flatten modes do not need it.

## Constants

| Constant | Value | Location |
|----------|-------|----------|
| `BufferSize` | 512 KB | client.go |
| `MaxConcurrentTransfers` | 4 | client.go |
| `DirCacheTimeout` | 30s | client.go |
