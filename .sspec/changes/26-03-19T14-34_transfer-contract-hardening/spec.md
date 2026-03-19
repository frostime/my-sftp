---
name: transfer-contract-hardening
status: DONE
type: ""
change-type: single
created: 2026-03-19T14:34:46
reference:
  - source: ".sspec/changes/26-03-19T01-30_unambiguous-transfer-syntax"
    type: "prev-change"
    note: "This change is a follow-up to unambiguous-transfer-syntax. It keeps the explicit grammar, and hardens the unresolved mapping, validation, and regression gaps found during audit."
  - source: ".sspec/spec-docs/sftp-transfer.md"
    type: "doc"
    note: "Transfer engine behavior and task collection flow."
  - source: ".sspec/spec-docs/cli-usage.md"
    type: "doc"
    note: "Project-level CLI usage contract for interactive shell commands and transfer grammar."
---

<!-- @RULE: Frontmatter
status: PLANNING | DOING | REVIEW | DONE | BLOCKED
change-type: single | sub
reference?: Array<{source, type: 'request'|'root-change'|'sub-change'|'prev-change' |'doc', note?}>

Sub-change MUST link root:
reference:
  - source: ".sspec/changes/<root-change-dir>"
    type: "root-change"
    note: "Phase <n>: <phase-name>"

Single-change common reference:
reference:
  - source: ".sspec/requests/<request-file>.md"
    type: "request"
  - source: ".sspec/changes/<change-dir>"
    type: "prev-change"
    note: "This change is a follow-up to <change-name> which introduced <feature/bug>. This change addresses <issue> with that feature/bug."
-->

# transfer-contract-hardening

## A. Problem Statement
Current Situation: the previous change moved `get`/`put` toward explicit destination syntax, but the implemented contract is still not fully deterministic in several high-risk paths. Multi-source explicit transfers still collapse files into the target root in some branches, `--flatten` is not applied consistently when glob matches include directories, and collision protection is not enforced across the full resolved source set. Parser boundary handling also regressed for source names beginning with `-`, while `--name` accepts path fragments that escape the destination directory contract.

User Requirement: this follow-up must not redesign the syntax again. It must harden the approved explicit grammar into one fully specified transfer contract, close audit-reported correctness gaps, keep compatibility behavior deliberate, and make the implementation easier to reason about by removing mapping-policy drift across shell and client layers.

<!-- @RULE: Quantify impact. Format: "[metric] causing [impact]".
Simple: single paragraph. Complex: split "Current Situation" + "User Requirement". -->

## B. Proposed Solution
Keep the explicit `-d/--dir`, `--name`, and `--flatten` grammar introduced by the prior change, but promote it from a parser-level improvement into a source-set planning contract: every `get` / `put` command must first resolve a normalized source set, then derive destination paths from one shared preserve/flatten policy, validate collisions once, and only then execute transfers.

Compatibility remains limited to the temporary legacy positional-target fallback when no explicit destination option is present. New work focuses on correctness and predictability, not new user-facing knobs.

<!-- @RULE: Accepted review-stage changes belong here as formal design.
If user feedback changes the current change's scope/design and the work still belongs to this change,
update A/B directly instead of leaving the accepted change only in handover.md.
If review history matters, add `### Review Amendments` under B as part of the design. -->

### Approach
The current bugs come from one root cause: destination mapping rules are implemented in several places with slightly different assumptions. This follow-up introduces a single transfer-planning path for upload and download flows so that explicit-file, glob, and recursive-directory cases all pass through the same preserve/flatten and collision logic.

This is intentionally a hardening change, not a rewrite. We keep the explicit syntax and temporary compatibility mode from the previous spec, but tighten the boundary contract (`--`, strict `--name` validation), define preserve semantics for all source shapes, and back the behavior with targeted regression tests and a runtime scenario matrix.

### Review Amendments

- Review feedback after the first hardening round expanded collision validation from exact duplicates to full target-namespace validation: exact duplicate targets and file-vs-directory prefix conflicts now both fail in preflight.
- Multi-source explicit directory operands follow the same operand-relative preserve rule as multi-source explicit files; same-basename directories from different parents must no longer collapse into one target tree.
- Case comparison policy is now explicit: local download targets use Windows/macOS case-insensitive comparison, while other namespaces default to case-sensitive comparison.
- Parent-relative explicit operands stay within the target root by encoding leading `..` segments into reserved `__my_sftp_parent__` preserve-path markers rather than silently collapsing them.
- Runtime review feedback exposed one shell/client mismatch: `get -r` / `put -r` must propagate `MaxDepth: -1`, otherwise recursive directory transfers silently degrade to top-level-only behavior.
- Runtime review feedback also tightened the source-set rule for globstar planning: overlapping `**` directory/file matches must be normalized before recursive expansion so the same resolved file cannot create false duplicate-target or false flatten-collision errors.
- Focused runtime retest confirmed the hardened contract now holds for recursive directory transfer, globstar preserve mode, globstar flatten mode, and parent-relative glob placement under the reserved preserve marker namespace.

### Key Design
#### Interface Contract

```text
get [-r] [--flatten] [-d <local_target_dir>] [--name <filename>] [--] <remote_src>...
put [-r] [--flatten] [-d <remote_target_dir>] [--name <filename>] [--] <local_src>...

- `-d/--dir` and `--name` may appear before or after sources until `--` is seen.
- `--` ends option parsing and is required for source operands that begin with `-`.
- `--name` accepts a filename only: no path separators, no `.` or `..`.
- legacy positional target fallback remains temporary, but only when `-d/--dir` is absent.
```

#### Source-Set Mapping Contract

Every transfer resolves to a list of file entries before execution. Each resolved file entry carries:

- source kind: explicit file, explicit directory member, or glob match
- preserve path: the path to keep when default preserve mode is active
- flatten name: the basename used by `--flatten`

Preserve-path rules:

```text
1. Single file, no destination specified
   - keep current convenience behavior from prior spec
   - `get report.csv` -> ./report.csv
   - `put report.csv` -> <remote_cwd>/report.csv

2. Single file with `-d`
   - target = target_root + basename(source)
   - `--name` may replace basename

3. Multiple explicit file sources
   - preserve the operand-relative path, not just basename
   - `get a/x.txt b/y.txt -d out` -> out/a/x.txt, out/b/y.txt
   - leading `..` segments are preserved inside the target root as reserved `__my_sftp_parent__` path markers

4. Directory sources with `-r`
   - single directory source keeps current ergonomic behavior: preserve contents under target root
   - multiple directory sources preserve each operand-relative source path to avoid namespace collapse

5. Absolute Windows local sources in multi-source preserve mode
    - preserve the source path under a reserved leading `__my_sftp_volume_<drive>__` segment so different drive roots cannot collapse into one target tree

6. Glob sources
    - preserve path relative to the static prefix before the first wildcard
    - if there is no static prefix, preserve path relative to current workdir
```

#### Flatten and Collision Contract

```text
- `--flatten` maps every resolved file entry to target_root + basename(file)
- collision validation runs on the complete resolved source set, regardless of whether entries came from
  explicit files, directory recursion, or glob expansion
- any duplicate basename is a hard error before transfer starts
- hint text remains actionable: remove `--flatten` or narrow the source set
- preserve mode also validates final target paths before any side effect; exact duplicate targets and
  file-vs-directory prefix conflicts are both hard errors
```

This means `--flatten` must behave identically for:

- explicit multi-file transfer
- explicit directory transfer
- glob transfer whose matches include directories
- mixed source sets in one command

#### Planning Split

Implementation should keep responsibilities narrow:

- `shell/shell.go`: parse CLI, enforce boundary validation, choose compatibility path
- `client/transfer.go`: host shared transfer-planning helpers used by upload/download flows
- `client/download.go` and `client/upload.go`: resolve source entries, then delegate mapping/collision policy to shared helpers
- tests: cover parser boundaries and mapping invariants directly

### Key Change
**Fix A: Fully Deterministic Source-Set Mapping** - Apply one preserve/flatten contract across explicit, glob, and recursive directory transfers.
Constraint: multi-source explicit transfers must no longer silently flatten into the destination root.

**Fix B: Flatten Means Flatten Everywhere** - Make `--flatten` rewrite every resolved file entry, including files discovered under directory matches.
Constraint: duplicate-basename validation must run on the final resolved source set before any transfer begins.

**Fix C: Harden CLI Boundary Rules** - Add `--` support for dash-leading source names and reject invalid `--name` values that are not pure filenames.
Constraint: preserve option-order flexibility from the previous change.

**Refactor D: Centralize Transfer Planning Logic** - Move destination mapping and collision policy to shared client-layer helpers so shell, upload, and download paths cannot drift.
Constraint: do not introduce a speculative framework; only extract the present shared concept needed to remove duplicated policy.

**Test E: Acceptance Matrix and State Sync** - Add focused regression coverage and rerun the end-to-end scenario matrix before considering the explicit grammar finished.
Constraint: `.sspec` status and verification notes must match actual runtime validation state.

### Scope Summary
| File | Change |
|------|--------|
| `shell/shell.go` | Harden argument parsing with `--` support, strict `--name` validation, and updated explicit multi-source orchestration |
| `client/transfer.go` | Add shared transfer-planning helpers for preserve-path derivation, flatten mapping, and collision validation |
| `client/download.go` | Route explicit/glob/directory download planning through the shared mapping contract |
| `client/upload.go` | Route explicit/glob/directory upload planning through the shared mapping contract |
| `completer/completer.go` | Keep completion aligned with the parser boundary rules, including `--` handling expectations |
| `client/*_test.go` or `shell/*_test.go` | Add regression tests for mapping and parser edge cases introduced by the explicit grammar |
| `README.md` | Clarify preserve semantics, `--` usage, and flatten guarantees |
| `README.zh.md` | Clarify preserve semantics, `--` usage, and flatten guarantees |
| `.sspec/spec-docs/sftp-transfer.md` | Align long-lived transfer behavior notes with the hardened preflight collision contract |
| `.sspec/changes/26-03-19T14-34_transfer-contract-hardening/handover.md` | Record audit-derived rationale and remaining verification steps |
