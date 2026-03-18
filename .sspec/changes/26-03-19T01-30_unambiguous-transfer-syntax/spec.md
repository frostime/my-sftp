---
name: unambiguous-transfer-syntax
status: PLANNING
type: ""
change-type: single
created: 2026-03-19T01:30:56
reference: null
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

# unambiguous-transfer-syntax

## A. Problem Statement
Current get/put grammar relies on implicit inference of the last argument (based on local path existence), causing nondeterministic behavior across environments. Three high-frequency scenarios are causing operational risk: (1) multi-file download target inference, (2) explicit target directory ambiguity for upload/download, and (3) path flattening during glob-based multi-transfer. This ambiguity causes user intent mismatch, accidental destination errors, and overwrite risk when duplicate basenames exist.

<!-- @RULE: Quantify impact. Format: "[metric] causing [impact]".
Simple: single paragraph. Complex: split "Current Situation" + "User Requirement". -->

## B. Proposed Solution
Define a deterministic transfer grammar where source and target roles are explicit and never inferred from local filesystem state. Replace positional target guessing with explicit target options, default to structure-preserving mapping for multi-source transfer, and provide opt-in flatten behavior with strict collision protection.

Compatibility strategy: keep legacy positional target syntax available behind a compatibility mode during transition, but default all help, completion, and examples to explicit grammar.

<!-- @RULE: Accepted review-stage changes belong here as formal design.
If user feedback changes the current change's scope/design and the work still belongs to this change,
update A/B directly instead of leaving the accepted change only in handover.md.
If review history matters, add `### Review Amendments` under B as part of the design. -->

### Approach
Use explicit option-based contracts for transfer direction:

- `get`: remote source operand(s), local destination directory via `-d` or `--dir`
- `put`: local source operand(s), remote destination directory via `-d` or `--dir`
- `--name <filename>`: single-file rename at destination (for exactly one file source)
- default multi/glob mapping preserves relative structure
- `--flatten`: opt-in flatten mapping to destination root

This approach removes hidden heuristics (`exists` / `not exists` checks for last argument), making behavior stable across machines and directories. It is preferred over "improved heuristics" because heuristics remain context-sensitive and non-verifiable by users before execution. To match user intuition, option order is flexible: destination options can appear before or after source operands.

### Key Design
#### Interface Contract

```text
# CHANGED: explicit-target grammar (default preserve)
get [-r] [--flatten] [-d <local_target_dir>] [--name <filename>] <remote_src>...
put [-r] [--flatten] [-d <remote_target_dir>] [--name <filename>] <local_src>...

# CHANGED: option placement
- `-d/--dir` and `--name` may appear before or after source operands.

# CHANGED: single-file convenience (destination optional)
get <remote_file>
put <local_file>

# NEW: strict validation
- If source count > 1, `--name` is forbidden.
- If any source is directory, `-r` is required.
- Default mapping for multi/glob is preserve-structure.
- If `--flatten` is enabled, duplicate basename across resolved source set is a hard error.
```

#### Behavioral Spec

```text
get/put command parse
  │
  ├── parse flags/options
  │     ├── destination dir option (`-d/--dir`)
  │     ├── optional single-file rename (`--name`)
  │     └── mapping mode (`default preserve` | `--flatten`)
  │
  ├── parse source operands (never reinterpret as target)
  │
  ├── validate
  │     ├── source_count > 1 && has --name -> error
  │     ├── has_directory_source && !-r -> error
  │     ├── has --name && source resolves to non-single-file -> error
  │     └── flatten mode duplicate basename -> error
  │
  ├── resolve source set (single | multi | glob)
  │
  ├── map destination path
  │     ├── preserve mode: target_root + relative_path
  │     ├── flatten mode: target_root + basename
  │     └── name mode: target_root + explicit_filename
  │
  └── execute transfer tasks via existing executeTasks pipeline
```

`get` and `put` share one mapping policy abstraction so flatten/preserve behavior is symmetrical across directions. Command completer switches to context-aware completion by argument role (source vs target option value), not command-level global assumptions.

#### Outcome Preview

```text
# 1) Single file, no destination specified
> get report.csv
Result: ./report.csv

> put report.csv
Result: <remote_cwd>/report.csv
```

```text
# 2) Single file, rename at destination
> get report.csv -d downloads --name report-2026.csv
Result: ./downloads/report-2026.csv

> put report.csv -d /data/inbox --name incoming.csv
Result: /data/inbox/incoming.csv
```

```text
# 3) Multi source with explicit destination (default preserve)
> get logs/*.log src/**/*.go -d backup
Result examples:
  ./backup/logs/app.log
  ./backup/src/cmd/main.go
```

```text
# 4) Flatten mode and collision protection
> put assets/** -d /tmp/upload --flatten
If two files resolve to same basename (for example a/readme.md and b/readme.md):
  Error: duplicate basename in --flatten mode: readme.md
  Hint: remove --flatten or narrow source set
```

```text
# 5) Option order flexibility
> get a.txt b.txt -d localdir
> get -d localdir a.txt b.txt
Result: same behavior
```

#### Migration Path

```text
Before (heuristic):
- get [-r] <remote|pattern> [local]
- put [-r] <local|pattern> [remote]
- last argument role inferred by local path existence checks

After (explicit):
- get [-r] [--flatten] [-d <local_target_dir>] [--name <filename>] <remote_src>...
- put [-r] [--flatten] [-d <remote_target_dir>] [--name <filename>] <local_src>...
- source/target roles are fixed by syntax, not runtime existence checks
- multi/glob default preserves structure; flatten is explicit and guarded
```

```text
Migration: staged compatibility
1. Introduce explicit grammar and make it primary in help/completion/docs.
2. Keep legacy positional target as compatibility mode for one transition window.
3. Legacy mode prints deprecation warning with explicit replacement command.
4. Remove legacy inference after transition window (separate future change).
Rollback: keep compatibility mode enabled; explicit grammar remains valid.
```

### Key Change
**Feat A: Explicit Destination Contract** - Replace positional-target inference with explicit `-d/--dir` destination option and single-file `--name` rename option.
Constraint: `--name` is valid only for exactly one file source.

**Fix B: Deterministic Validation and Error Surface** - Add preflight validation for multi-source target requirement, directory recursion requirement, and duplicate basename collision in flatten mode.
Constraint: error text must provide one-step corrective syntax.

**Feat C: Default Preserve + Explicit Flatten Mode** - Make preserve-structure default for multi/glob, and add explicit `--flatten` mode.
Constraint: flatten mode must fail fast on duplicate basenames.

**Refactor D: Role-Aware Completion and Docs** - Update completer and help/README examples to reflect explicit target grammar and remove heuristic mental model.
Constraint: completion behavior must match actual parser role semantics.

### Scope Summary
| File | Change |
|------|--------|
| `shell/shell.go` | Replace get/put argument parsing with explicit target-option parser and strict validation |
| `completer/completer.go` | Implement role-aware completion for target option values and source operands |
| `client/download.go` | Add preserve-structure destination mapping and flatten-collision checks |
| `client/upload.go` | Add preserve-structure destination mapping and flatten-collision checks |
| `README.md` | Update transfer syntax/help examples to explicit grammar |
| `README.zh.md` | Update transfer syntax/help examples to explicit grammar |
