# Handover: unambiguous-transfer-syntax

**Updated**: 2026-03-19T02:12+08:00

---

## Background
<!-- Write once on first session. What this change does and why (1-3 sentences).
Update only if scope fundamentally changes. Details belong in spec.md. -->

This change removes get/put destination ambiguity by replacing positional target guessing with explicit destination options, while keeping a temporary compatibility fallback. It also changes multi/glob mapping behavior to preserve structure by default and adds explicit flatten mode with collision protection.

## Git Baseline (Immutable)
<!-- Captured during `sspec change new` before any change files are written.
This section records the change starting point in git and must not be edited or refreshed later. -->

- Captured: before change file creation
- Repository: `D:/Arsenal/PlayCode/my-sftp`
- Branch: `main`
- HEAD: `d802ed07024cbdcfbc5209e389d64d4c2b6b698f`
- Worktree: `clean`
- Status Snapshot: raw `git status --short --branch` output

```text
## main...origin/main
```

## Working Memory (Stable)
<!-- Curated, long-lived context. Survives context compression and session boundaries.
If something becomes obsolete, mark it as obsolete with a timestamp instead of deleting silently. -->

### Key Files
<!-- Files critical to understanding/continuing this change.
- `path/file` - what it contains, why it matters -->

- `shell/shell.go` - transfer CLI parsing, compatibility fallback warning path, and strict validation rules.
- `client/download.go` - download glob mapping policy, remote-relative path mapping, flatten collision validation.
- `client/upload.go` - upload glob mapping policy, local-relative path mapping, flatten collision validation.
- `completer/completer.go` - role-aware completion for `-d/--dir` argument values.
- `README.md` - English command syntax and examples aligned with explicit grammar.
- `README.zh.md` - Chinese command syntax and examples aligned with explicit grammar.
- `.sspec/changes/26-03-19T01-30_unambiguous-transfer-syntax/spec.md` - approved design including Outcome Preview scenario matrix.
- `.sspec/changes/26-03-19T01-30_unambiguous-transfer-syntax/tasks.md` - phase progress and remaining validation work.

### Durable Memory (Typed, Timestamped)
 - [2026-03-19T02:01+08:00] [Decision] Multi/glob transfer defaults to preserve-structure; flatten behavior is explicit via `--flatten`.
 - [2026-03-19T02:01+08:00] [Decision] Destination targeting uses explicit `-d/--dir`; single-file rename uses `--name` and is forbidden for multi-source.
 - [2026-03-19T02:01+08:00] [Constraint] Full end-to-end transfer regression needs a reachable SFTP server; current session verified compile/build only.
 - [2026-03-19T02:12+08:00] [Risk] Round-1 audit identified critical deviations (glob preserve-structure base and flatten hint text) tracked in `round1-deviation.md`.
 - [2026-03-19T02:12+08:00] [Decision] Commit executed by explicit user instruction with known deviations documented for next round.
<!-- Promote only facts still useful after the current batch ends.
Single/sub change preferred types: Alignment, Decision, VitalFinding, Constraint, Risk, VerificationShortcut.
Use a custom type only when none fit well; keep it short and clear.
- [2026-03-06T20:39] [Decision] Redis over Memcached because per-key TTL + persistence matter.
- [2026-03-06T20:39] [Constraint] Session Log stays append-only; real next action lives there.
Project-wide items -> ALSO append to project.md Notes. -->

## Session Log (Append-Only)
<!-- Newest entry first. Each entry is an atomic batch (one cohesive work record).

Header format:
### 2026-03-06T20:39 [work-log] <short title>

Tags are freeform but must be readable. Examples: work-log, user-feedback, argue, risk.
Any user interaction (feedback, @align, @argue) MUST start a new log entry. -->

### <ISO timestamp> [tag] <short title>

**Accomplished**
- ...

**Next**
- ...

**Notes** (optional)
- ...

### 2026-03-19T02:01+08:00 [user-feedback] refine syntax direction

**Accomplished**
- User requested default preserve-structure, optional flatten, and better destination syntax semantics.
- Updated spec with explicit `-d/--dir` + `--name`, default preserve, explicit flatten, and scenario-driven Outcome Preview.
- User approved revised design gate and requested continuation.

**Next**
- Complete final regression matrix verification against a real SFTP target.
- Finalize compatibility behavior details after runtime validation.

### 2026-03-19T02:01+08:00 [work-log] implement parser mapping and docs

**Accomplished**
- Implemented explicit transfer parser in `shell/shell.go` with deterministic validation and legacy compatibility warnings.
- Implemented preserve/flatten mapping and collision checks in `client/download.go` and `client/upload.go`.
- Updated completion behavior and help/docs in `completer/completer.go`, `README.md`, and `README.zh.md`.
- Verified successful build with `go test ./...`.

**Next**
- Run end-to-end transfer scenario matrix with SFTP runtime to validate behavior-level correctness.

**Notes** (optional)
- Build is green; no unit tests exist in repository.

### 2026-03-19T02:12+08:00 [work-log] commit and deviation documentation

**Accomplished**
- Staged all changes and prepared commit per user request.
- Executed subagent audit for spec alignment, code quality, and whole-project scope review.
- Documented audit conclusions and agent judgment in `.sspec/changes/26-03-19T01-30_unambiguous-transfer-syntax/round1-deviation.md`.

**Next**
- Fix critical deviations from round-1 report.
- Run real SFTP E2E scenario matrix and close Phase 4.

**Notes** (optional)
- Audit verdict before commit: needs fixes before commit; commit proceeded by user directive.
