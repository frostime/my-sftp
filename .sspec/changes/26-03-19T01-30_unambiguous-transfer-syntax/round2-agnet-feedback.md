# Round 2 Agnet Feedback

## Scope
Change: 26-03-19T01-30_unambiguous-transfer-syntax
Goal: Verify Round-1 deviations and record Round-2 remediation outcomes.

## Verification Summary
Reviewer: GitHub Copilot (GPT-5.3-Codex)
Verdict: Round-1 listed deviations were valid and have been remediated in code.

### Checked and Fixed
1. Preserve-structure mapping for glob base-path derivation.
- Status: Fixed.
- Affected files: client/download.go, client/upload.go.
- Result: Relative and absolute glob patterns now use consistent preserve mapping logic, including no-static-prefix patterns.

2. Flatten collision errors missing corrective hint text.
- Status: Fixed.
- Affected files: client/download.go, client/upload.go.
- Result: Collision errors now include one-step remediation hint:
  - "Hint: remove --flatten or narrow source set"

3. Single-file no-destination get/put asymmetry.
- Status: Fixed (implementation symmetry improvement, behavior preserved).
- Affected file: shell/shell.go.
- Result: `cmdPut` branch is now intent-aligned with `cmdGet` for maintainability.

## Validation
1. Workspace diagnostics: no errors.
2. Build validation: `go build -o my-sftp.tmp.exe` succeeded.
3. Note: `go build -o my-sftp.exe` may fail if target executable is currently locked by process.

## Residual Work
1. End-to-end transfer matrix against a real SFTP target is still pending for final Phase-4 closure.
2. Spec frontmatter status update remains pending after runtime verification.
