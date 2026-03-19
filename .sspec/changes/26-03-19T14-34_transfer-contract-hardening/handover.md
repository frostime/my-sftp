# Handover: transfer-contract-hardening

**Updated**: 2026-03-19T21:14+08:00

---

## Background
This change is a follow-up hardening pass for the explicit transfer grammar introduced by `unambiguous-transfer-syntax`. It keeps the new syntax, but fixes contract holes found during audit: inconsistent preserve/flatten behavior, missing collision coverage in some paths, weak CLI boundary validation, and incomplete verification closure.

## Git Baseline (Immutable)
<!-- Captured during `sspec change new` before any change files are written.
This section records the change starting point in git and must not be edited or refreshed later. -->

- Captured: before change file creation
- Repository: `H:/SrcCode/playground/mygo-sftp/my-sftp`
- Branch: `perf/改进sftp内语法`
- HEAD: `be4aceac6ff4384fa4a36302b087ae5805f07df3`
- Worktree: `clean`
- Status Snapshot: raw `git status --short --branch` output

```text
## perf/改进sftp内语法...origin/perf/改进sftp内语法
```

## Working Memory (Stable)
<!-- Curated, long-lived context. Survives context compression and session boundaries.
If something becomes obsolete, mark it as obsolete with a timestamp instead of deleting silently. -->

### Key Files
<!-- Files critical to understanding/continuing this change.
- `path/file` - what it contains, why it matters -->

- `.sspec/changes/26-03-19T01-30_unambiguous-transfer-syntax/spec.md` - previous approved design that this follow-up must preserve at the syntax level.
- `.sspec/changes/26-03-19T14-34_transfer-contract-hardening/spec.md` - new hardening design derived from audit findings.
- `.sspec/spec-docs/cli-usage.md` - stable project-level CLI contract covering transfer grammar, option rules, and shell command categories.
- `shell/shell.go` - current CLI parsing and explicit multi-source behavior gaps.
- `client/transfer.go` - best location for shared transfer-planning helpers to reduce policy drift.
- `client/download.go` - current download-side preserve/flatten logic, including glob-directory gaps.
- `client/upload.go` - current upload-side preserve/flatten logic, including glob-directory gaps.
- `completer/completer.go` - completion behavior that must stay aligned with the parser contract.

### Durable Memory (Typed, Timestamped)
<!-- Promote only facts still useful after the current batch ends.
Single/sub change preferred types: Alignment, Decision, VitalFinding, Constraint, Risk, VerificationShortcut.
Use a custom type only when none fit well; keep it short and clear.
- [2026-03-06T20:39] [Decision] Redis over Memcached because per-key TTL + persistence matter.
- [2026-03-06T20:39] [Constraint] Session Log stays append-only; real next action lives there.
Project-wide items -> ALSO append to project.md Notes. -->

- [2026-03-19T14:35+08:00] [Decision] This follow-up keeps the explicit `-d/--dir`, `--name`, and `--flatten` syntax; it is a contract-hardening change, not another syntax redesign.
- [2026-03-19T14:35+08:00] [VitalFinding] Fresh audit found unresolved correctness gaps after the previous change: explicit multi-source preserve drift, incomplete flatten handling for directory matches, and missing collision checks on some explicit branches.
- [2026-03-19T14:35+08:00] [Decision] Shared transfer-planning policy should move toward `client/transfer.go` because the same mapping rules are currently duplicated across shell/upload/download paths.
- [2026-03-19T14:35+08:00] [Constraint] Real SFTP end-to-end regression remains required before the transfer grammar can be marked finished.
- [2026-03-19T16:13+08:00] [VitalFinding] The hardening pass now validates both flatten basename collisions and final target-path collisions before execution; Windows/macOS local downloads are case-folded for collision checks, while other namespaces keep case-sensitive comparison unless later policy work changes that.
- [2026-03-19T16:13+08:00] [Decision] `collectDownloadTasks` no longer creates local directories during planning; local filesystem mutation is deferred until after the full plan validates.
- [2026-03-19T16:13+08:00] [VerificationShortcut] `go test ./... && go build ./...` is green; runtime SFTP transfer matrix is still required for final closure.
- [2026-03-19T17:32+08:00] [Decision] Target-namespace preflight now treats exact duplicate targets and file-vs-directory prefix conflicts as the same validation gate instead of separate ad hoc checks.
- [2026-03-19T17:32+08:00] [Decision] Case comparison policy is explicit: Windows/macOS local downloads compare case-insensitively; other namespaces default to case-sensitive comparison.
- [2026-03-19T17:32+08:00] [VitalFinding] Multi-source explicit directory operands must preserve their operand-relative paths just like multi-source explicit files, otherwise same-basename directories collapse into one target tree.
- [2026-03-19T17:32+08:00] [Decision] Parent-relative explicit operands preserve distinct namespaces via reserved `__my_sftp_parent__` path markers, and absolute Windows local sources use `__my_sftp_volume_<drive>__` prefixes when preserve mode needs to keep drive roots distinct.
- [2026-03-19T18:15+08:00] [Decision] Reserved preserve markers are guarded across explicit and glob multi-source preserve inputs so user-provided paths cannot collide with planner-generated marker namespaces.
- [2026-03-19T18:15+08:00] [Decision] Empty remote-directory downloads still create the local destination directory, but only after preflight validation succeeds.
- [2026-03-19T21:14+08:00] [VitalFinding] Runtime verification exposed two remaining correctness gaps after the first implementation pass: command-layer recursive transfers were not propagating `MaxDepth: -1`, and globstar overlap had to be normalized before task expansion to avoid false duplicate-path and false flatten-collision failures.
- [2026-03-19T21:14+08:00] [VerificationShortcut] Real SFTP runtime validation is now green; see `runtime-test-batches.md` for the initial broad run and `runtime-retest-minimal-batches.md` for the focused passing retest.
- [2026-03-19T21:14+08:00] [Decision] Project-level CLI usage is now documented in `.sspec/spec-docs/cli-usage.md` so the transfer grammar and shell conventions have a stable reference outside this change.

## Session Log (Append-Only)
<!-- Newest entry first. Each entry is an atomic batch (one cohesive work record).

Header format:
### 2026-03-06T20:39 [work-log] <short title>

Tags are freeform but must be readable. Examples: work-log, user-feedback, argue, risk.
Any user interaction (feedback, @align, @argue) MUST start a new log entry. -->

### 2026-03-19T21:14+08:00 [work-log] close runtime verification and publish cli usage spec-doc

**Accomplished**
- Folded runtime review feedback back into `spec.md` and `tasks.md`, including the recursive-depth propagation fix and the pre-expansion globstar normalization rule.
- Marked the runtime checklist and Phase 4 state complete using the recorded broad run plus focused retest artifacts.
- Added `.sspec/spec-docs/cli-usage.md` as a project-level reference for `my-sftp` shell grammar, command categories, transfer options, and path-handling conventions.

**Next**
- Review the final doc set and decide whether to commit this change set.

**Notes** (optional)
- Runtime evidence now exists in two layers: the original broad matrix shows the discovered defects, and the minimal retest proves the fixes against the same real SFTP environment.

### 2026-03-19T21:05+08:00 [work-log] fix recursive depth propagation and globstar overlap normalization

**Accomplished**
- Restored true recursive behavior for shell-driven `get -r` / `put -r` by propagating `MaxDepth: -1` through shared command-option builders.
- Normalized glob matched source entries before recursive expansion so globstar directory/file overlap no longer turns into duplicate targets or false flatten collisions.
- Added regression tests for glob overlap normalization and shell option propagation defaults.
- Re-verified with `go test ./...`.

**Next**
- Run `runtime-retest-minimal-batches.md` against the real SFTP target.
- If green, sync checklist items and close remaining Phase 4 runtime verification work.

**Notes** (optional)
- The focused runtime retest should pay special attention to final path placement for parent-relative glob cases, not just command success.

### 2026-03-19T20:55+08:00 [work-log] diagnose runtime verification failures and prepare focused retest

**Accomplished**
- Reviewed the recorded runtime batch output and confirmed two live correctness defects remain.
- Located the recursive directory regression at `shell/shell.go`: `cmdGet` / `cmdPut` pass `Recursive` but forget `MaxDepth: -1`, so `-r` currently behaves like depth `0` and only transfers top-level files.
- Localized the remaining `**` overlap issue to glob planning normalization: the planner still validates collisions after expanding overlapping directory matches from globstar, instead of first normalizing the matched source-entry set.
- Added a separate minimal retest batch file focused only on the still-open defects and the parent-relative path guard.

**Next**
- Fix shell option propagation so recursive directory commands restore unlimited depth.
- Move globstar overlap normalization earlier in planning (before recursive expansion / final collision validation) for both upload and download paths.
- Re-run the focused retest batches, then update the main runtime checklist with pass/fail status.

**Notes** (optional)
- The parent-relative glob commands now succeed at runtime, but final path placement still needs explicit filesystem inspection in the focused retest.

### 2026-03-19T19:10+08:00 [work-log] fix glob planner overlap and parent-relative prefix escape

**Accomplished**
- Fixed glob planning so recursive directory matches no longer schedule the same resolved file multiple times when `**` returns both directories and files.
- Normalized relative glob preserve prefixes through the same reserved-marker path encoding used by explicit operands, so parent-relative patterns stay inside the destination root.
- Added regression coverage for globstar dedupe, parent-relative glob prefix mapping, and shared preserve-path helpers.
- Added a user-checkable runtime verification checklist in `runtime-test-checklist.md` for real SFTP acceptance.
- Re-verified with `go test ./...`.

**Next**
- Run the real SFTP scenario matrix.
- Re-review any remaining glob preserve semantics edge cases before marking the change done.

**Notes** (optional)
- Exact duplicate explicit source operands are still left to target-collision validation; only duplicate resolved files created by glob expansion are normalized away during planning.

### 2026-03-19T14:35+08:00 [user-feedback] initiate follow-up change from prior spec

**Accomplished**
- User requested a new next-change based on the previous `spec.md`, with broader repair and optimization rather than blindly continuing the old change.
- Created `26-03-19T14-34_transfer-contract-hardening`.
- Drafted a follow-up design that keeps the explicit syntax but hardens mapping semantics, boundary validation, and regression coverage.

**Next**
- Get user review on the new design gate.
- After approval, expand `tasks.md` into an execution plan and begin implementation.

**Notes** (optional)
- The new design explicitly addresses both fresh audit findings and the earlier incomplete verification state.

### 2026-03-19T14:55+08:00 [user-feedback] design gate approved

**Accomplished**
- User locked the current `transfer-contract-hardening` design without requesting scope changes.
- Expanded `tasks.md` into four execution phases covering shared mapping helpers, boundary hardening, regression coverage, and runtime validation.

**Next**
- Start Phase 1 implementation in `client/transfer.go`, `client/download.go`, `client/upload.go`, and `shell/shell.go`.
- Keep docs/completion updates behind behavior changes so user-facing examples match final behavior.

**Notes** (optional)
- The approved design keeps syntax stable and shifts risk reduction into behavior hardening and verification.

### 2026-03-19T16:13+08:00 [work-log] implement transfer contract hardening

**Accomplished**
- Centralized transfer-planning helpers in `client/transfer.go` for preserve-path normalization, flatten remapping, duplicate-basename rejection, and final target collision validation.
- Refactored `client/download.go` and `client/upload.go` to plan explicit, glob, and recursive sources through shared batch flows, including deterministic handling of `~/...` sources.
- Hardened `shell/shell.go` parsing with `--` support and filename-only `--name` validation.
- Updated help/README text and added focused regression tests in `client/transfer_test.go` and `shell/shell_test.go`.
- Verified with `go test ./...` and `go build ./...`.

**Next**
- Run the real SFTP scenario matrix to validate runtime behavior for explicit multi-source, glob-directory flattening, rename, and compatibility fallback cases.
- Decide whether to refresh `.sspec/spec-docs/sftp-transfer.md` in this change or as an immediate follow-up, because the old duplicate-overwrite note is now stale.

**Notes** (optional)
- `DownloadDir` / `UploadDir` keep their directory-only contract even after the shared planner refactor.

### 2026-03-19T17:32+08:00 [work-log] address review feedback on target planning

**Accomplished**
- Merged exact duplicate detection and ancestor-prefix conflict detection into one preflight target-namespace validator.
- Updated explicit multi-directory preserve handling so operand-relative directory paths are kept for both upload and download planning.
- Removed the Darwin case-fold shortcut and documented the explicit case comparison policy in the active change files.
- Preserved distinct parent-relative explicit operands by encoding leading `..` segments into stable in-target markers instead of trimming them away.
- Added regression tests for prefix conflicts and directory-source preserve helpers.
- Re-verified with `go test ./...` and `go build ./...`.

**Next**
- Run the real SFTP scenario matrix.
- Update `.sspec/spec-docs/sftp-transfer.md` so the old overwrite note matches the hardened planner behavior.

**Notes** (optional)
- This batch intentionally stayed within the current change because it closes review feedback on the same transfer contract.

### 2026-03-19T18:15+08:00 [work-log] close second-round review gaps

**Accomplished**
- Refined preserve-path encoding so parent-relative and Windows-volume sources stay distinct without colliding with ordinary user paths.
- Restored empty remote-directory download behavior while keeping preflight side-effect free until validation passes.
- Synced the long-lived transfer spec-doc with the new preflight collision contract.
- Re-ran `go test ./...` and `go build ./...` and finished with a clean subagent review (`No findings`).

**Next**
- Run the real SFTP scenario matrix.
- Decide whether to squash or follow-up commit the review-driven hardening fixes after you inspect the diff.

**Notes** (optional)
- This round kept all changes inside the current SSPEC change rather than spawning another follow-up.

### <ISO timestamp> [tag] <short title>

**Accomplished**
- ...

**Next**
- ...

**Notes** (optional)
- ...
