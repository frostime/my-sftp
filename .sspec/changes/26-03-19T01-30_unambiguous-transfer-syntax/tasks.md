---
change: "unambiguous-transfer-syntax"
updated: ""
---

# Tasks

## Legend
`[ ]` Todo | `[x]` Done

## Tasks
### Phase 1: CLI Syntax Parser and Validation (Feat A, Fix B) ✅
- [x] Implement explicit destination and rename option parsing for `get`/`put` in `shell/shell.go` (`-d/--dir`, `--name`, `--flatten`) with order-independent option handling.
- [x] Remove heuristic "last argument role inference" logic from `cmdGet`/`cmdPut` in `shell/shell.go`.
- [x] Add deterministic preflight validation and corrective error messages in `shell/shell.go` for: directory-without-`-r`, multi-source-with-`--name`, and invalid rename source shape.
**Verification**: Manual shell checks confirm parser accepts both `get a b -d out` and `get -d out a b`, and rejects invalid combinations with actionable errors.

### Phase 2: Transfer Mapping Policy (Feat C, Fix B) ✅
- [x] Implement default preserve-structure mapping for multi/glob download in `client/download.go`.
- [x] Implement default preserve-structure mapping for multi/glob upload in `client/upload.go`.
- [x] Implement explicit `--flatten` mapping path and duplicate-basename collision detection in `client/download.go` and `client/upload.go`.
- [x] Implement single-file rename target mapping (`--name`) in transfer path construction where applicable.
**Verification**: Functional transfer tests confirm preserve mode keeps relative paths; flatten mode writes basenames only and fails on duplicates.

### Phase 3: Completion and UX Surface (Refactor D) ✅
- [x] Update role-aware completion behavior in `completer/completer.go` so destination option values complete by destination role instead of command-level default.
- [x] Update command help text and examples in `shell/shell.go` to explicit syntax and scenario-based guidance.
- [x] Update transfer documentation and examples in `README.md` and `README.zh.md` to explicit destination grammar, `--flatten`, and `--name` semantics.
**Verification**: TAB completion behavior matches parser intent for source vs destination arguments; help/README examples are executable against new parser.

### Phase 4: Compatibility and Regression Validation 🚧
- [x] Add compatibility mode handling (legacy positional target syntax) and deprecation warning path in `shell/shell.go`.
- [ ] Run end-to-end manual regression for single file, multi-source, glob, directory recursive, explicit destination, rename, and flatten collision scenarios.
- [x] Record final behavior deltas and migration notes in `.sspec/changes/26-03-19T01-30_unambiguous-transfer-syntax/handover.md`.
**Verification**: Legacy syntax still works in compatibility mode with warning; explicit syntax is primary and stable across all scenario matrix cases.

### Feedback Tasks
- [ ] Incorporate any review-stage example adjustments or edge-case clarifications directly into `spec.md` and sync parser/docs accordingly.

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
Planning complete; implementation not started.

**Overall**: 92%

| Phase | Progress | Status |
|-------|----------|--------|
| Phase 1: CLI Syntax Parser and Validation | 100% | ✅ |
| Phase 2: Transfer Mapping Policy | 100% | ✅ |
| Phase 3: Completion and UX Surface | 100% | ✅ |
| Phase 4: Compatibility and Regression Validation | 67% | 🚧 |

**Recent**:
- Completed: Phase 1 parser and validation implementation in `shell/shell.go`.
- Completed: Phase 2 mapping policy updates in `client/download.go` and `client/upload.go`.
- Completed: Phase 3 completion/help/README alignment.
- Completed: Phase 4 compatibility mode and handover updates; runtime E2E transfer matrix still pending.
