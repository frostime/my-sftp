# Round 1 Deviation Report

## Scope
Change: 26-03-19T01-30_unambiguous-transfer-syntax
Goal: Record first-round audit deviations after initial implementation.

## Audit Summary
Reviewer: Explore subagent
Verdict: Needs fixes before commit

### Critical
1. Preserve-structure mapping for glob is not fully aligned with spec examples in some path-base cases.
- Affected areas: client/download.go, client/upload.go
- Risk: directory hierarchy may be partially flattened unexpectedly.

2. Flatten collision errors do not include corrective hint text mandated by spec.
- Affected areas: client/download.go, client/upload.go
- Risk: poor remediation guidance and spec mismatch.

### High
1. Single-file no-destination behavior between get and put is internally asymmetric.
- Affected area: shell/shell.go
- Risk: maintainability and edge-case behavior divergence.

### Medium/Low (non-blocking)
1. Validation ordering and minor UX/document formatting consistency points.
2. Spec frontmatter status still PLANNING while implementation progressed.

## Agent Judgment
1. The implemented direction is correct and mostly consistent with approved design.
2. Critical deviations are real and should be fixed in next round before final release-quality acceptance.
3. Commit is performed per explicit user instruction, with known deviations documented.

## Next Round Fix Plan
1. Normalize glob preserve-structure base-path derivation and verify against spec scenario matrix.
2. Add flatten collision hint text with one-step remediation guidance.
3. Reconcile get/put single-file destination semantics for consistency.
4. Run full E2E transfer matrix with real SFTP target and close Phase 4.
