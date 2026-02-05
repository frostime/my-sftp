---
skill: sspec
version: 6.0.0
description: Quality standards and workflows for SSPEC changes. Covers single/multi-change assessment, document standards, reference field usage, and edge cases. Consult when starting changes, handling complex scenarios, or unsure about quality standards.
---

# SSPEC Skill

**When to consult**:
- Starting a new change (single vs multi-change decision)
- Writing spec.md / tasks.md / handover.md
- Using reference field or reference/ directories
- Handling edge cases (blockers, rejection, multi-change coordination)
- Unsure about quality standards or workflows

**Note**: Core protocol is in AGENTS.md. This SKILL provides depth and practical guidance.

## Change Complexity Assessment

**FIRST DECISION**: Single change or multi-change?

### Single Change (default)

Use when work is:
- **Time**: Completable in <1 week
- **Scope**: Modifies <10 files or focuses on one subsystem
- **Tasks**: <20 actionable tasks
- **Risk**: Low, changes are reversible

**Examples**: Bug fix, add single API endpoint, refactor one module, update docs.

### Multi-Change (complex projects)

Use when work requires:
- **Time**: >1 week estimated
- **Scope**: Touches multiple subsystems or >15 files
- **Tasks**: >30 tasks, or phases with distinct milestones
- **Risk**: High, requires staged rollout or extensive testing

**Pattern**: Create **root change** for coordination + multiple **sub-changes** for execution.

**When uncertain**: Use `sspec ask` to consult user on splitting approach.

---

## Multi-Change Management

For complex features requiring staged delivery or multiple subsystems.

### Structure

```
Root change (coordinator):
  - Changes/<root-name>/
    ├── spec.md         # Overall vision, phases overview
    ├── tasks.md        # High-level milestones
    ├── reference/      # Shared design docs, architecture
    └── script/         # Shared migration scripts

Sub-changes (execution):
  - Changes/<sub-name-1>/, Changes/<sub-name-2>/, ...
    ├── spec.md         # Focused scope for this sub-change
    │   └── reference: [{source: "changes/<root-name>", type: "sup-change"}]
    ├── tasks.md        # Specific tasks for this phase
    └── handover.md     # Sub-change progress
```

### Workflow

1. **Create root change**: Design overall approach, break into phases
2. **Create first sub-change**: Implement Phase 1, link to root via `reference`
3. **Complete → Archive → Next**: Archive sub-change, create next sub-change
4. **Root stays active**: Until all sub-changes complete
5. **Final archive**: Archive root change when everything done

### Example

**Root**: `26-02-05T14-00_auth-system-overhaul`
- Spec.md: "Replace basic auth with JWT + LDAP integration"
- Tasks.md: "Phase 1: JWT impl → Phase 2: LDAP → Phase 3: Migrate users"
- Reference/: `design.md`, `api-migration-plan.md`

**Sub-1**: `26-02-10T09-00_jwt-token-implementation`
- Spec.md frontmatter: `reference: [{source: "changes/26-02-05T14-00_auth-system-overhaul", type: "sup-change"}]`
- Tasks.md: 12 tasks focused on JWT

---

## Single Change Specification

Standards for individual changes (whether standalone or sub-change in multi-change).

### spec.md Quality Standards

| Section | Requirement | ❌ Fail | ✅ Pass |
|---------|-------------|---------|---------|
| A. Problem | Quantified impact | "Need to refactor" | "Auth takes 5s → 12% conversion drop" |
| B. Solution | Approach + rationale | "Use caching" | "JWT + Redis: DB→memory, <100ms target" |
| C. Implementation | File-level tasks | "Modify auth files" | "`src/auth/jwt.py` — create refresh_token()" |
| D. Blockers | Dated, actionable | "Waiting on DevOps" | "Blocker (01-27): Need Redis host:port" |

**Section C format**:
```markdown
### Phase 1: Infrastructure
- `src/cache/redis.py` — create, connection pool setup
- `requirements.txt` — modify, add redis>=4.0

### Phase 2: Core Logic
- `src/auth/jwt.py` — create, token generation/validation
- `src/auth/middleware.py` — modify, add cache-first lookup

### Risks & Dependencies
- Redis server required (coordinate with DevOps)
```

### tasks.md Structure

| Criterion | Standard |
|-----------|----------|
| Granularity | Each task <2h, independently testable |
| Verification | Each phase has explicit pass criteria |
| Progress tracking | Update after completing EACH task |

**Format**:
```markdown
### Phase 1: Infrastructure ✅
- [x] Add redis dependency to `requirements.txt`
- [x] Create connection pool in `src/cache/redis.py`
**Verification**: `pytest tests/test_cache.py` all pass

### Phase 2: Core Logic 🚧
- [x] Implement cache-first lookup in `src/auth/middleware.py`
- [ ] Create token refresh in `src/auth/jwt.py`
**Verification**: Auth endpoint responds in <100ms
```

### handover.md Essentials

| Field | Purpose | Bad Example | Good Example |
|-------|---------|-------------|--------------|
| Background | One-sentence overview | "Doing auth" | "JWT+Redis cache to reduce auth from 5s to <1s" |
| Accomplished | What's done this session | "Made progress" | "Phase 1 complete: redis pool + middleware integration" |
| Next Steps | 1-3 specific file actions | "Continue" | "1. Code jwt.py:refresh_token() 2. Add token expiry tests" |
| Conventions | Patterns discovered | (empty) | "Cache key format: `auth:{user_id}`, TTL: 900s" |

**Quality test**: New Agent can resume in <30 seconds?

### reference/ Directory (Optional)

Use for **complex changes** needing design iteration before implementation.

| Use Case | File Example |
|----------|--------------|
| Architecture exploration | `design-draft.md` |
| API alternatives comparison | `api-options.md` |
| Research notes | `research.md` |
| Diagrams | `architecture.mmd`, `dataflow.png` |

**Workflow**:
1. PLANNING: Draft designs in reference/
2. Iterate via `sspec ask` for user feedback
3. Finalize: Distill into spec.md Sections A/B/C
4. Keep reference/ for record, or discard if no longer needed

**When to use**: Architectural decisions, multiple valid approaches, user needs to review design.

**Skip for**: Simple bug fixes, well-understood features.

### script/ Directory (Optional)

One-off scripts for this change: migrations, test data generators, analysis tools.

**Lifecycle**: Created in DOING, may promote to project-level if reusable, otherwise archived with change.

---

## Frontmatter Reference Field

### Purpose
Track relationships: request → change, sub-change ↔ root change, change → spec-doc.

```yaml
reference:
  - source: "requests/26-02-05T14-00_add-auth.md"  # Relative to .sspec/
    type: "request"  # 'request' | 'sub-change' | 'sup-change' | 'doc'
    note: "Original feature proposal"  # Optional
```

### Auto-populated
- `sspec request link <req> <chg>`: Updates both request and change
- `sspec change new --from <req>`: Creates change with request reference

### Manual usage
- Sub-change → Root: `{source: "changes/<root>", type: "sup-change"}`
- Root → Sub: `{source: "changes/<sub>", type: "sub-change"}`
- Change → Spec-doc: `{source: "spec-docs/api-contract.md", type: "doc"}`

---

## Edge Cases & Workflows

### Partial Blockers

Some tasks blocked, others can proceed.

**Decision tree**:
```
IF blocked tasks are dependencies for remaining:
    → Status = BLOCKED, document in spec.md Section D
ELSE IF blocked tasks are non-critical:
    → Continue other tasks, move blocked to end, document in spec.md D
ELSE:
    → Consider splitting into two changes
```

### REVIEW Across Sessions

- Keep status = REVIEW
- Update handover.md: "Awaiting user review since <date>"
- Can start other changes meanwhile
- Next session: Prompt user for review result first

### Mid-Flight Rejection (@argue)

User says "this isn't right" during DOING.

**Protocol**:
1. **STOP** immediately, don't continue current task
2. **Clarify** rejection scope:
   - Implementation detail → Update tasks.md only
   - Design decision → Revise spec.md Section B + tasks.md
   - Requirement itself → Revise spec.md Section A, mark PIVOT in Section D
3. **Re-plan**: If scope changed significantly, consider DOING → PLANNING
4. **Wait**: Get explicit user approval before resuming work

### Multiple Active Changes

**Guideline**: Limit to ≤2 changes in DOING status simultaneously.

**Context switching**:
1. Run `@handover` on current change
2. Run `@change <other-name>` to switch
3. Read `<other-name>/handover.md` before taking action

### Design Iteration Loop

When spec.md keeps getting revised in PLANNING:

**Pattern**:
1. Save current spec.md → `reference/spec-v1.md` (version it)
2. Create `reference/design-exploration.md` for brainstorming
3. Iterate with user via `sspec ask`
4. Once stable: Write final clean version to spec.md

---

## Anti-Patterns to Avoid

| Bad Practice | Why It Fails | Correct Approach |
|--------------|--------------|------------------|
| Skip @handover | Next session wastes time re-discovering context | **ALWAYS** `@handover` before ending session |
| Mark `[x]` without testing | False sense of progress, bugs hidden | Task is done = coded **AND** verified |
| No file paths in spec.md C | Agent guesses, may modify wrong files | List specific file paths for each task |
| Stay DOING when blocked | Waste time on workarounds | Change to BLOCKED immediately, document blocker |
| Skip REVIEW status | User doesn't validate, wrong direction | DOING → REVIEW → (user approves) → DONE |
| Batch progress updates | Lose track of what's actually done | Update tasks.md after **each** task completion |
| Over-use reference/ | Time wasted on docs for simple changes | Reserve for genuinely complex design |
| Forget reference field | Lost traceability to requests/root changes | Use CLI auto-link or add manually |

---

## Checklists

### Starting New Change

- [ ] Assessed: Single vs multi-change?
- [ ] If multi-change: Created root change first?
- [ ] Spec.md Section A: Problem quantified with metrics?
- [ ] Spec.md Section B: Solution approach + rationale stated?
- [ ] Spec.md Section C: File-level task breakdown provided?
- [ ] Tasks.md: Each task <2h, has verification criteria?
- [ ] Handover.md: Initial context documented?
- [ ] Reference field: Linked to originating request (if applicable)?

### Before Transitioning to REVIEW

- [ ] All tasks marked `[x]` in tasks.md?
- [ ] All phase verification criteria met?
- [ ] Handover.md reflects completion?
- [ ] Spec.md Section D: No undocumented blockers?
- [ ] Code tested and passing?

### Before @handover (End of Session)

- [ ] Handover.md: Updated "Accomplished" section?
- [ ] Handover.md: "Next Steps" clear (1-3 file-level actions)?
- [ ] Handover.md: "Conventions" updated if new patterns found?
- [ ] Tasks.md: Progress percentage updated?
- [ ] Spec.md: Status accurate (PLANNING/DOING/BLOCKED/REVIEW)?
