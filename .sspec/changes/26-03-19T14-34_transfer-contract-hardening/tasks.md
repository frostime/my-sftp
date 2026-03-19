---
change: "transfer-contract-hardening"
updated: "2026-03-19T21:14:39+08:00"
---

# Tasks

## Legend
`[ ]` Todo | `[x]` Done

## Tasks
### Phase 0: Design Gate Preparation ✅
- [x] Create follow-up change `transfer-contract-hardening` from the audited branch state.
- [x] Review prior spec, handover notes, and independent audit findings to derive the new scope.
- [x] Align with the user on the follow-up design before planning implementation work.
**Verification**: `spec.md` captures the corrective contract, scope, and rationale for the follow-up change.

### Phase 1: Shared Mapping Contract ✅
- [x] Implement shared transfer-planning helpers in `client/transfer.go` for preserve-path derivation, flatten remapping, duplicate-basename validation, and final target collision checks. (Refactor D, Fix A, Fix B)
- [x] Route explicit multi-source download planning through the shared helpers so preserve mode no longer collapses files into the destination root. (Fix A)
- [x] Route explicit multi-source upload planning through the shared helpers so preserve mode no longer collapses files into the destination root. (Fix A)
**Verification**: static review plus focused tests confirm explicit multi-source file and directory transfers preserve the intended source-relative structure.

### Phase 2: Boundary Hardening and Consistency ✅
- [x] Add `--` handling in `shell/shell.go` so source operands beginning with `-` are accepted without breaking option-order flexibility. (Fix C)
- [x] Reject invalid `--name` values that contain path separators or `.` / `..` traversal forms. (Fix C)
- [x] Make `--flatten` semantics apply consistently to files discovered under explicit directories and glob-matched directories. (Fix B)
**Verification**: parser and mapping tests cover dash-leading sources, strict rename validation, and flatten behavior across explicit/glob/directory paths.

### Phase 3: Completion, Docs, and Regression Coverage ✅
- [x] Review `completer/completer.go` against the new parser behavior and keep it unchanged because its source/destination completion split already matches the hardened `--` flow. (Fix C)
- [x] Add regression tests in `shell/*_test.go` and/or `client/*_test.go` for preserve/flatten, target-collision, and boundary scenarios. (Test E)
- [x] Refresh `README.md` and `README.zh.md` to document preserve semantics, `--` usage, and validated flatten guarantees. (Test E)
**Verification**: `go test ./...` passes and docs examples match the new contract.

### Phase 4: Runtime Validation and Change Closure ✅
- [x] Run the end-to-end transfer matrix against a real SFTP target for single file, explicit multi-source, recursive directory, glob, `--flatten`, collision failure, `--name`, and compatibility fallback scenarios. (Test E)
- [x] Sync `.sspec/changes/26-03-19T14-34_transfer-contract-hardening/handover.md` and status fields to match the actual validation state. (Test E)
**Verification**: `runtime-test-batches.md` and `runtime-retest-minimal-batches.md` capture both the initial runtime findings and the final green retest against a real SFTP target.

### Feedback Tasks ✅
- [x] Extend preflight target validation to reject both exact duplicate targets and file-vs-directory prefix conflicts before any transfer side effect.
- [x] Make multi-source explicit directory operands preserve operand-relative paths instead of collapsing same-basename directories into one target tree.
- [x] Replace the ad hoc case-fold logic with an explicit comparison policy: Windows/macOS local downloads are case-insensitive; other namespaces default to case-sensitive comparison.
- [x] Record runtime review feedback that `get -r` / `put -r` lost unlimited recursion because command-layer option builders failed to pass `MaxDepth: -1`, then fix the propagation path.
- [x] Record runtime review feedback that globstar overlap must be normalized before task expansion, then fix upload/download planning and re-verify with a focused retest.

<!-- @RULE: Organize by phases. Each task <2h, independently testable.
Phase emoji: ⏳ pending | 🚧 in progress | ✅ done

### Phase 1: <name> ⏳
- [ ] Task description `path/file.py`
- [ ] Task description `path/file.py`
**Verification**: <how to verify this phase>

### Feedback Tasks
Use this section for review/feedback tasks that still belong to the current change.
If accepted feedback changes scope/design, update `spec.md` first, then add the execution work here.
If the work should become a new follow-up or replacement change, do not put it here unless the user has first approved that direction via `@align`.
-->

---

## Progress
Implementation, review-driven hardening, tests, user-facing docs, and real SFTP runtime validation are all complete.

**Overall**: 100%

| Phase | Progress | Status |
|-------|----------|--------|
| Phase 0: Design Gate Preparation | 100% | ✅ |
| Phase 1: Shared Mapping Contract | 100% | ✅ |
| Phase 2: Boundary Hardening and Consistency | 100% | ✅ |
| Phase 3: Completion, Docs, and Regression Coverage | 100% | ✅ |
| Phase 4: Runtime Validation and Change Closure | 100% | ✅ |

**Recent**:
- Runtime review feedback is now fully folded back into `spec.md`, `tasks.md`, and the final green retest artifacts.
