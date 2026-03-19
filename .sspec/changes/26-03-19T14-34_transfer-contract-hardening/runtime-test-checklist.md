# Runtime Test Checklist

**Created**: 2026-03-19 20:24:43

Change: `transfer-contract-hardening`

Purpose: manual end-to-end verification against a real SFTP target. Tick each item only after both command result and final filesystem state match expectations.

For faster execution, you can use the ready-to-paste grouped command batches in `runtime-test-batches.md`.

## Test Setup

- [ ] Prepare a clean remote sandbox, for example `/tmp/my-sftp-runtime/`
- [ ] Prepare a clean local sandbox, for example `./tmp/runtime/`
- [ ] Start the shell from a clean working directory and record the server/OS used for testing
- [ ] Keep one terminal open for `my-sftp`, and one terminal open locally/remotely to inspect actual files after each case

## Suggested Fixture

Create or verify these files before starting:

### Local Fixture Script (PowerShell)

Run this in the directory where you want the local test tree to be created. It will create a `runtime-fixture/` folder relative to the current location.

```powershell
$root = Join-Path (Get-Location) "runtime-fixture"

$dirs = @(
    "local/src/nested",
    "local/dir/nested",
    "local/flat/x",
    "local/flat/y",
    "local/dash",
    "local/logs",
    "workspace",
    "logs"
)

foreach ($dir in $dirs) {
    New-Item -ItemType Directory -Force -Path (Join-Path $root $dir) | Out-Null
}

$files = @{
    "local/a.txt" = "local a`n"
    "local/src/a.txt" = "src a`n"
    "local/src/nested/b.txt" = "src nested b`n"
    "local/dir/root.txt" = "dir root`n"
    "local/dir/nested/child.txt" = "dir child`n"
    "local/flat/x/readme.md" = "flat x readme`n"
    "local/flat/y/readme.md" = "flat y readme`n"
    "local/dash/-report.txt" = "dash report`n"
    "local/logs/app.log" = "local log`n"
    "logs/app.log" = "parent log sibling`n"
}

foreach ($entry in $files.GetEnumerator()) {
    $path = Join-Path $root $entry.Key
    Set-Content -Path $path -Value $entry.Value -NoNewline
}

Write-Host "Local runtime fixture created at: $root"
```

````ps1
# 执行结果如下
tree .
❯❯❯ tree .
Folder PATH listing for volume 文件
Volume serial number is 0000001C 0885:A00A
H:\SRCCODE\PLAYGROUND\MYGO-SFTP\MY-SFTP\.SSPEC\CHANGES\26-03-19T14-34_TRANSFER-CONTRACT-HARDENING
├───reference
└───runtime-fixture
    ├───local
    │   ├───dash
    │   ├───dir
    │   │   └───nested
    │   ├───flat
    │   │   ├───x
    │   │   └───y
    │   ├───logs
    │   └───src
    │       └───nested
    ├───logs
    └───workspace

````

### Remote Fixture Script (Bash)

Run this on the remote host in the directory where you want the remote test tree to be created. It will create a `runtime-fixture/` folder relative to the current location.

```bash
set -eu

ROOT="$(pwd)/runtime-fixture"

mkdir -p \
  "$ROOT/remote/src/nested" \
  "$ROOT/remote/dir/nested" \
  "$ROOT/remote/flat/x" \
  "$ROOT/remote/flat/y" \
  "$ROOT/remote/dash" \
  "$ROOT/remote/logs" \
  "$ROOT/workspace" \
  "$ROOT/logs"

cat > "$ROOT/remote/a.txt" <<'EOF'
remote a
EOF

cat > "$ROOT/remote/src/a.txt" <<'EOF'
remote src a
EOF

cat > "$ROOT/remote/src/nested/b.txt" <<'EOF'
remote src nested b
EOF

cat > "$ROOT/remote/dir/root.txt" <<'EOF'
remote dir root
EOF

cat > "$ROOT/remote/dir/nested/child.txt" <<'EOF'
remote dir child
EOF

cat > "$ROOT/remote/flat/x/readme.md" <<'EOF'
remote flat x readme
EOF

cat > "$ROOT/remote/flat/y/readme.md" <<'EOF'
remote flat y readme
EOF

cat > "$ROOT/remote/dash/-report.txt" <<'EOF'
remote dash report
EOF

cat > "$ROOT/remote/logs/app.log" <<'EOF'
remote log
EOF

cat > "$ROOT/logs/app.log" <<'EOF'
remote parent log sibling
EOF

printf 'Remote runtime fixture created at: %s\n' "$ROOT"
```

````bash
# 执行记录如下
zyp in 🌐 eeg-4090 in /tmp/test-mysftp 🅒 torch
❯ tree .
.
└── runtime-fixture
    ├── logs
    │   └── app.log
    ├── remote
    │   ├── a.txt
    │   ├── dash
    │   │   └── -report.txt
    │   ├── dir
    │   │   ├── nested
    │   │   │   └── child.txt
    │   │   └── root.txt
    │   ├── flat
    │   │   ├── x
    │   │   │   └── readme.md
    │   │   └── y
    │   │       └── readme.md
    │   ├── logs
    │   │   └── app.log
    │   └── src
    │       ├── a.txt
    │       └── nested
    │           └── b.txt
    └── workspace

13 directories, 10 files
````

Local:
- `local/a.txt`
- `local/src/a.txt`
- `local/src/nested/b.txt`
- `local/dir/root.txt`
- `local/dir/nested/child.txt`
- `local/flat/x/readme.md`
- `local/flat/y/readme.md`
- `local/dash/-report.txt`
- `local/logs/app.log`

Remote:
- `remote/a.txt`
- `remote/src/a.txt`
- `remote/src/nested/b.txt`
- `remote/dir/root.txt`
- `remote/dir/nested/child.txt`
- `remote/flat/x/readme.md`
- `remote/flat/y/readme.md`
- `remote/dash/-report.txt`
- `remote/logs/app.log`

## Upload Matrix

- [ ] Single file default target
  - Command: `put local/a.txt`
  - Expect: file appears under current remote dir as `a.txt`

- [ ] Single file explicit rename
  - Command: `put local/a.txt -d /tmp/my-sftp-runtime/upload-name --name renamed.txt`
  - Expect: only `/tmp/my-sftp-runtime/upload-name/renamed.txt` is created

- [ ] Explicit multi-source preserve structure
  - Command: `put local/src/a.txt local/src/nested/b.txt -d /tmp/my-sftp-runtime/upload-multi`
  - Expect: remote keeps `src/a.txt` and `src/nested/b.txt` under target root

- [ ] Recursive directory upload
  - Command: `put -r local/dir -d /tmp/my-sftp-runtime/upload-dir`
  - Expect: remote contains `root.txt` and `nested/child.txt` under target root

- [ ] Glob preserve structure
  - Command: `put local/src/**/*.txt -d /tmp/my-sftp-runtime/upload-glob`
  - Expect: remote keeps relative layout from static prefix, including `src/nested/b.txt`

- [ ] Glob with `**` does not double-plan files
  - Command: `put local/dir/** -d /tmp/my-sftp-runtime/upload-globstar -r`
  - Expect: command succeeds without duplicate-target error; each file is uploaded once

- [ ] Glob flatten success
  - Command: `put local/dir/** -d /tmp/my-sftp-runtime/upload-flatten-ok -r --flatten`
  - Expect: target contains only `root.txt` and `child.txt` at root level

- [ ] Glob flatten duplicate basename failure
  - Command: `put local/flat/** -d /tmp/my-sftp-runtime/upload-flatten-dup -r --flatten`
  - Expect: command fails before transfer with duplicate basename error and hint text

- [ ] Dash-leading source with `--`
  - Command: `put -d /tmp/my-sftp-runtime/upload-dash -- local/dash/-report.txt`
  - Expect: upload succeeds and remote filename remains `-report.txt`

- [ ] Legacy positional target compatibility
  - Command: `put local/src/a.txt /tmp/my-sftp-runtime/upload-legacy`
  - Expect: upload still works via compatibility path; note whether deprecation warning appears

## Download Matrix

- [ ] Single file default target
  - Command: `get remote/a.txt`
  - Expect: file appears in current local dir as `a.txt`

- [ ] Single file explicit rename
  - Command: `get remote/a.txt -d ./tmp/runtime/download-name --name renamed.txt`
  - Expect: only `./tmp/runtime/download-name/renamed.txt` is created

- [ ] Explicit multi-source preserve structure
  - Command: `get remote/src/a.txt remote/src/nested/b.txt -d ./tmp/runtime/download-multi`
  - Expect: local keeps `remote/src/a.txt`-style operand-relative layout under target root according to current contract

- [ ] Recursive directory download
  - Command: `get -r remote/dir -d ./tmp/runtime/download-dir`
  - Expect: local contains `root.txt` and `nested/child.txt` under target root

- [ ] Glob preserve structure
  - Command: `get remote/src/**/*.txt -d ./tmp/runtime/download-glob`
  - Expect: local keeps relative layout from static prefix, including nested files

- [ ] Glob with `**` does not double-plan files
  - Command: `get remote/dir/** -d ./tmp/runtime/download-globstar -r`
  - Expect: command succeeds without duplicate-target error; each file is downloaded once

- [ ] Glob flatten success
  - Command: `get remote/dir/** -d ./tmp/runtime/download-flatten-ok -r --flatten`
  - Expect: target contains only `root.txt` and `child.txt` at root level

- [ ] Glob flatten duplicate basename failure
  - Command: `get remote/flat/** -d ./tmp/runtime/download-flatten-dup -r --flatten`
  - Expect: command fails before transfer with duplicate basename error and hint text

- [ ] Dash-leading source with `--`
  - Command: `get -d ./tmp/runtime/download-dash -- remote/dash/-report.txt`
  - Expect: download succeeds and local filename remains `-report.txt`

## Parent-Relative and Boundary Cases

- [ ] Parent-relative upload glob stays inside target root
  - Precondition: current local working dir is `./tmp/runtime/workspace`, sibling dir `./tmp/runtime/logs/app.log` exists
  - Command: `put ../logs/*.log -d /tmp/my-sftp-runtime/upload-parent-glob`
  - Expect: remote file lands under `/tmp/my-sftp-runtime/upload-parent-glob/__my_sftp_parent__/logs/app.log`, not outside target root

- [ ] Parent-relative download glob stays inside target root
  - Precondition: current remote working dir is a directory whose sibling `../logs/app.log` exists
  - Command: `get ../logs/*.log -d ./tmp/runtime/download-parent-glob`
  - Expect: local file lands under `./tmp/runtime/download-parent-glob/__my_sftp_parent__/logs/app.log`, not outside target root

- [ ] Invalid `--name` with path separator is rejected
  - Command: `put local/a.txt -d /tmp/my-sftp-runtime/invalid-name --name nested/out.txt`
  - Expect: command fails immediately with filename-only validation error

- [ ] Multiple sources without `-d` are rejected
  - Command: `get remote/src/a.txt remote/src/nested/b.txt`
  - Expect: command fails with explicit destination requirement unless compatibility fallback intentionally applies

## Notes

- [ ] Record any mismatch, including exact command, output, and observed filesystem state
- [ ] After all applicable cases pass, use this checklist as evidence to close Phase 4 in `tasks.md`
