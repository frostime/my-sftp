---
skill: sspec-ask
version: 3.1.0
description: Mid-execution user consultation. USE ACTIVELY to reduce errors and save cost.
---

# SSPEC Ask Skill

**USE ACTIVELY** — Guessing wastes more tokens than one ask. When in doubt, ask.

---

## When to Trigger (Critical Decision Points)

**REQUIRED use cases** - Agent MUST use sspec ask when:

1. **User explicitly requested** - User mentions ask/confirmation in their request
2. **Information missing** - Cannot proceed reliably without user clarification
   - Example: User mentions ambiguous terms without context → Ask for specific meaning
3. **Directional choice needed** - Multiple valid approaches exist (not minor tweaks)
   - Example: Component refactor could use multiple architecture styles → Ask for user preference
4. **Work completion check** - Agent believes task is done
   - Example: Code changes completed → Ask user to verify satisfaction before ending turn
5. **Repeated failures** - Multiple attempts failed, need user insight
   - Example: CLI command fails 3+ times → Ask user for environment details

**Why active use matters**: Reduces guessing, prevents directional errors, saves tokens by avoiding rework, ensures alignment with user intent.

---

## Workflow

**Step 1**: Create template
```bash
sspec ask create --name <topic>
```
Creates `.sspec/asks/<timestamp>_<topic>.py`

**Step 2**: Edit the `.py` file
- Fill `REASON` (why asking)
- Fill `QUESTION` (what to ask)
- Do NOT edit `USER_ANSWER`

**Step 3**: Execute
```bash
sspec ask prompt <path>
```
**Output**: Use's answer, and creates `.sspec/asks/<timestamp>_<name>.py` with template as follow.

**Error Case**: `sspec ask prompt` output shows that do not exists the `<timestamp>_<name>.py` file -- might becasue the file has been achived to md, check if exsits `<timestamp>_<name>.md`.

## Template Format

```python
CREATED = "<iso_timestamp>"

REASON = r"""
<why you're asking - for future reference>
"""

QUESTION = r"""
<your question here>
"""

# User can pre-fill to skip terminal prompt
USER_ANSWER = r""""""
```

---

## Example: Directional Choice

```python
REASON = r"""
Multiple valid approaches for caching layer refactor
"""

QUESTION = r"""
I've identified 3 approaches:

**A) Redis + In-Memory Fallback**
- Pros: High performance, resilient
- Cons: Operational complexity

**B) Pure In-Memory (LRU)**
- Pros: Simple, no external deps
- Cons: Lost on restart

**C) SQLite Cache**
- Pros: Persistent, zero-config
- Cons: Slower than Redis

Which aligns with project priorities?
"""
```

---

## Guidelines

| Do | Don't |
|----|-------|
| Use descriptive `--name` | Use generic names (`q1`, `ask`) |
| `--name` only contains letters and underscore | Use names (`非英文字符`, `other_symbols_like$#%*`) |
| Fill `REASON` for context | Leave `REASON` empty |
| Ask early when uncertain | Guess and risk rework |
| Provide options when applicable | Leave open-ended if choices exist |


---

## Final Record Format

```markdown
---
created: '<timestamp>'
name: <topic>
why: <reason>
---

# Ask: <topic>

## Question
<question_text>

## Answer
<answer_text>
```
