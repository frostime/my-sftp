---
name: architecture
description: "Module relationships, data flow map, and design decisions for my-sftp"
updated: 2026-05-05
scope:
  - /main.go
  - /shell/**
  - /client/**
  - /config/**
  - /completer/**
---

# Architecture

## Overview

Four-layer architecture: **Entry → Shell → Client → SFTP/SSH**. Supporting layers: Config (parsing), Completer (TAB).

## Module Map

```
main.go
  ├─ config.LoadSSHConfig / ParseDestination  → SSHConfig
  ├─ loadPrivateKey / createHostKeyCallback   → ssh.AuthMethod
  └─ client.NewClient(addr, sshClientConfig)  → Client
       └─ shell.NewShell(client)              → Shell.Run()

Shell (shell/shell.go)
  ├─ owns readline.Instance + completer.Completer
  ├─ routes commands → Client methods
  └─ parses CLI options (parseTransferCLIArgs) → UploadOptions / DownloadOptions

Client (client/client.go)
  ├─ wraps ssh.Client + sftp.Client
  ├─ manages: workDir, localWorkDir, dirCache, bufferPool, singleflight.Group, remoteCaseSensitive
  └─ exposes: file ops, transfer engine, path resolution, FormatSize

Completer (completer/completer.go)
  ├─ interface: ClientInterface (ListCompletion, GetLocalwd)
  └─ delegates path completion → Client
```

## Data Flows

### Upload: `put [-r] [--flatten] [-d dir] [--name name] [--] <src>...`

```
Shell.cmdPut
  → parseTransferCLIArgs          (shell/shell.go)
  → client.UploadSources          (client/upload.go)
    → collectUploadSourceTasks    per source: explicit path or glob
      → collectUploadGlobTasks    (glob: doublestar.FilepathGlob)
      → collectUploadTasks        (dir: recursive walk → tasks + emptyDirs)
    → [early return if 0 tasks + emptyDirs: ensureRemoteDir + print → done]
    → applyFlattenMapping         (if --flatten)
    → validateTargetCollisions    (pre-flight duplicate check)
    → collectRemoteDirsForUpload  → ensureRemoteDirsExist (singleflight)
    → executeTasks                (client/transfer.go)
```

### Download: `get [-r] [--flatten] [-d dir] [--name name] [--] <src>...`

```
Shell.cmdGet
  → parseTransferCLIArgs          (shell/shell.go)
  → client.DownloadSources        (client/download.go)
    → collectDownloadSourceTasks  per source: explicit path or glob
      → collectDownloadGlobTasks  (glob: globRemote → doublestar.Match)
      → collectDownloadTasks      (dir: recursive walk)
    → applyFlattenMapping / validateTargetCollisions
    → ensureLocalDirsExist
    → executeTasks                (client/transfer.go)
```

### Path Resolution

| Function | Location | Behavior |
|----------|----------|----------|
| `ResolveRemotePath` | client/common.go | relative → join with `workDir`; `~` → `sftp.Getwd()` |
| `ResolveLocalPath` | client/common.go | relative → join with `localWorkDir`; normalizes to `/` separator |
| `globRemote` | client/download.go | SFTP has no native glob → client-side walk + `doublestar.Match` |

## Design Decisions

### 1. Task-collect + unified-execute pattern

All transfers (single file, glob, recursive dir) funnel through `executeTasks()`.

**Why**: Avoids concurrent nesting (collect phase doesn't spawn goroutines). Enables pre-flight validation (collision check, directory creation). Unified progress bar and error aggregation.

### 2. singleflight for directory creation

`ensureRemoteDir` uses `singleflight.Group.Do(dir, ...)` instead of sharded mutexes.

**Why**: Multiple goroutines uploading to the same directory share one `MkdirAll` call. Simpler and more efficient than 64-shard lock array (which was the previous approach — see commented code in `client.go`).

### 3. sync.Pool for transfer buffers

512KB buffers pooled via `sync.Pool`. Get before transfer, put after.

**Why**: Each transfer needs a large buffer. Allocating per-transfer creates GC pressure under concurrent multi-file transfers.

### 4. Preserve vs Flatten path mapping

Default (preserve): source-relative path structure is maintained under target root. `--flatten`: basename-only, duplicate basenames are hard error pre-flight.

**Why**: Preserve is safe default for directory trees. Flatten is convenience for flat dumps. Collision detection happens before any transfer starts.

### 6. Remote filesystem case-sensitivity detection

`probeRemoteCaseSensitivity` in `client/common.go` detects remote FS case sensitivity at connection time by creating a temp file with mixed-case name, stat-ing the opposite case, and cleaning up. Result stored in `Client.remoteCaseSensitive` and logged to user. Used by collision key methods (`targetConflictKey`, `flattenCollisionKey`) in `client/transfer.go` to decide whether to lowercase upload targets.

**Why**: Remote Linux is case-sensitive; remote macOS is not. Using `runtime.GOOS` for upload targets was wrong — upload target is a remote path, not a local one.

### 7. CLI option parsing in Shell, not Client

`parseTransferCLIArgs` lives in `shell/shell.go`. Client receives structured `UploadOptions`/`DownloadOptions`.

**Why**: Shell owns CLI syntax. Client owns transfer logic. Clean separation prevents CLI concerns leaking into the library layer.
