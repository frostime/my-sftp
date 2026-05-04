---
change: "bugfixes-and-asymmetry-fixes"
updated: 2026-05-05
---

# Tasks

## Legend
`[ ]` Todo | `[x]` Done

## Tasks

### Phase 1: Infrastructure — FormatSize + RemoveDir + remoteCaseSensitive 🚧

- [ ] **Task 1.1**: Add `FormatSize` to `client/common.go` — exported function per design §5 `client/transfer.go`
- [ ] **Task 1.2**: Replace `formatBytes(t.size)` call in `client/transfer.go` → `FormatSize(t.size)`, delete `formatBytes` func `client/transfer.go`
- [ ] **Task 1.3**: Replace `formatSize(...)` calls in `shell/shell.go` → `client.FormatSize(...)`, delete `formatSize` func `shell/shell.go`
- [ ] **Task 1.4**: Add `RemoveDir(remotePath string) error` to `client/common.go` per design §4 `client/common.go`
- [ ] **Task 1.5**: Add `remoteCaseSensitive bool` field to `Client` struct; implement `probeRemoteCaseSensitivity()` method in `client/common.go` per design §2; call it in `NewClient` and log result `client/client.go` `client/common.go`

**Verification**: `go build ./...` compiles; `go test ./client/... ./shell/...` passes

### Phase 2: Bug Fix — parseCommandLine backslash ⏳

- [ ] **Task 2.1**: Update `parseCommandLine` in `shell/shell.go` — `case '\\'` branches on `inQuote` per design §1 `shell/shell.go`
- [ ] **Task 2.2**: Add unit tests for `parseCommandLine` backslash handling: unquoted `\`, quoted `\"`, `\\`, Windows path `C:\Users\file.txt` `shell/shell_test.go`

**Verification**: `go test ./shell/... -run TestParseCommandLine` passes; `go build ./...` compiles

### Phase 3: Bug Fix — collision key case sensitivity ⏳

- [ ] **Task 3.1**: Convert `targetConflictKey` and `flattenCollisionKey` to `Client` methods using `c.remoteCaseSensitive` per design §2 `client/transfer.go`
- [ ] **Task 3.2**: Update all callers of `targetConflictKey`/`flattenCollisionKey`/`validateTargetCollisions`/`applyFlattenMapping` to use method syntax `client/transfer.go` `client/upload.go` `client/download.go`
- [ ] **Task 3.3**: Fix case-related tests in `client/transfer_test.go` — pass `remoteCaseSensitive` field or mock; add upload collision test for case-sensitive remote `client/transfer_test.go`

**Verification**: `go test ./client/...` passes; collision tests cover both case-sensitive and case-insensitive remote

### Phase 4: Design Fix — Empty directory upload ⏳

- [ ] **Task 4.1**: Change `collectUploadTasks` return signature to `([]transferTask, []string, error)` — add `emptyDirs []string` return per design §3 `client/upload.go`
- [ ] **Task 4.2**: Propagate `emptyDirs` through `collectUploadSourceTasks` → `UploadSources`; handle `len(tasks)==0 && len(emptyDirs)>0`: call `ensureRemoteDir` + print confirmation per design §3 `client/upload.go`
- [ ] **Task 4.3**: Update `DownloadDir` empty dir message for symmetry if needed `client/download.go`

**Verification**: `go build ./...` compiles; manual test: `put -r emptydir -d /remotedir` outputs `✓ Created empty directory`

### Phase 5: Design Fix — rmdir semantics ⏳

- [ ] **Task 5.1**: Replace `cmdRmdir` body — call `s.client.RemoveDir(args)` with usage error + per-dir success print per design §4 `shell/shell.go`
- [ ] **Task 5.2**: Update `showHelp` text for `rmdir` if needed `shell/shell.go`

**Verification**: `go build ./...` compiles; manual test: `rmdir nonemptydir` → error with hint; `rmdir emptydir` → success

### Phase 6: Refactor — Completer dedup ⏳

- [ ] **Task 6.1**: Add `completeFromCandidates(candidates []string, prefix string) [][]rune` to `completer/completer.go` per design §6 `completer/completer.go`
- [ ] **Task 6.2**: Rewrite `completeCommand`, `completeRemotePath`, `completeLocalPath` to use `completeFromCandidates`; remove `removePrefix` `completer/completer.go`

**Verification**: `go build ./...` compiles; tab completion still works for commands, remote paths, local paths

---

## Progress

**Overall**: 0%

| Phase | Progress | Status |
|-------|----------|--------|
| Phase 1 | 0% | ⏳ |
| Phase 2 | 0% | ⏳ |
| Phase 3 | 0% | ⏳ |
| Phase 4 | 0% | ⏳ |
| Phase 5 | 0% | ⏳ |
| Phase 6 | 0% | ⏳ |

**Recent**:
- (none yet)