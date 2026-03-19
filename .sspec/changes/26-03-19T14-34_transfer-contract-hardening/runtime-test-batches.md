# Runtime Test Batches

Purpose: reduce manual typing during real SFTP verification. Paste one whole block at a time into the `my-sftp` shell.

Assumed fixture roots based on your recorded setup:

- Local fixture root: `H:/SrcCode/playground/mygo-sftp/my-sftp/.sspec/changes/26-03-19T14-34_transfer-contract-hardening/runtime-fixture`
- Remote fixture root: `/tmp/test-mysftp/runtime-fixture`

If either path changes, only update the first `lcd` / `cd` commands in the relevant batch.

## Batch 0: Enter Fixture Roots

Paste this first.

```text
lcd "H:/SrcCode/playground/mygo-sftp/my-sftp/.sspec/changes/26-03-19T14-34_transfer-contract-hardening/runtime-fixture"
cd /tmp/test-mysftp/runtime-fixture
lpwd
pwd
```

```out
/tmp/test-mysftp/runtime-fixture > lcd "H:/SrcCode/playground/mygo-sftp/my-sftp/.sspec/changes/26-03-19T14-34_transfer-contract-hardening/runtime-fixture"
/tmp/test-mysftp/runtime-fixture > cd /tmp/test-mysftp/runtime-fixture
/tmp/test-mysftp/runtime-fixture > lpwd
H:/SrcCode/playground/mygo-sftp/my-sftp/.sspec/changes/26-03-19T14-34_transfer-contract-hardening/runtime-fixture
/tmp/test-mysftp/runtime-fixture > pwd
/tmp/test-mysftp/runtime-fixture
```

## Batch 1: Upload Positive Cases

```text
put local/a.txt
put local/a.txt -d upload-name --name renamed.txt
put local/src/a.txt local/src/nested/b.txt -d upload-multi
put -r local/dir -d upload-dir
put local/src/**/*.txt -d upload-glob
put local/dir/** -d upload-globstar -r
put local/dir/** -d upload-flatten-ok -r --flatten
put -d upload-dash -- local/dash/-report.txt
put local/src/a.txt upload-legacy
```

```out

/tmp/test-mysftp/runtime-fixture > put local/a.txt
Found 1 file(s) to upload
✓ a.txt (8 B)

✓ Uploaded 1 file(s) in 8ms
/tmp/test-mysftp/runtime-fixture > put local/a.txt -d upload-name --name renamed.txt
Uploading a.txt (1/1 files) 100% |████████████████████████████████████████|
✓ Uploaded 1 file(s) in 8ms
/tmp/test-mysftp/runtime-fixture > put local/src/a.txt local/src/nested/b.txt -d upload-multi
Found 2 file(s) to upload
✓ b.txt (13 B)
✓ a.txt (6 B)

✓ Uploaded 2 file(s) in 20ms
/tmp/test-mysftp/runtime-fixture > put -r local/dir -d upload-dir
Found 1 file(s) to upload
✓ root.txt (9 B)

✓ Uploaded 1 file(s) in 14ms
/tmp/test-mysftp/runtime-fixture > put local/src/**/*.txt -d upload-glob
Found 2 file(s) to upload
✓ a.txt (6 B)
✓ b.txt (13 B)

✓ Uploaded 2 file(s) in 41ms
/tmp/test-mysftp/runtime-fixture > put local/dir/** -d upload-globstar -r
Error: duplicate target path in transfer plan: /tmp/test-mysftp/runtime-fixture/upload-globstar/local/dir/nested/child.txt
/tmp/test-mysftp/runtime-fixture > put local/dir/** -d upload-flatten-ok -r --flatten
Error: duplicate basename in --flatten mode: child.txt
Hint: remove --flatten or narrow source set
/tmp/test-mysftp/runtime-fixture > put -d upload-dash -- local/dash/-report.txt
Found 1 file(s) to upload
✓ -report.txt (12 B)

✓ Uploaded 1 file(s) in 13ms
/tmp/test-mysftp/runtime-fixture > put local/src/a.txt upload-legacy
Warning: legacy positional target syntax is deprecated; use -d <remote_dir>
Found 1 file(s) to upload
✓ a.txt (6 B)

✓ Uploaded 1 file(s) in 22ms
```

## Batch 2: Upload Expected Failures

```text
put local/flat/** -d upload-flatten-dup -r --flatten
put local/a.txt -d invalid-name --name nested/out.txt
```

```out
/tmp/test-mysftp/runtime-fixture > put local/flat/** -d upload-flatten-dup -r --flatten
Error: duplicate basename in --flatten mode: readme.md
Hint: remove --flatten or narrow source set
/tmp/test-mysftp/runtime-fixture > put local/a.txt -d invalid-name --name nested/out.txt
Error: put: --name must be a filename without path separators
```

## Batch 3: Download Positive Cases

```text
get remote/a.txt
get remote/a.txt -d download-name --name renamed.txt
get remote/src/a.txt remote/src/nested/b.txt -d download-multi
get -r remote/dir -d download-dir
get remote/src/**/*.txt -d download-glob
get remote/dir/** -d download-globstar -r
get remote/dir/** -d download-flatten-ok -r --flatten
get -d download-dash -- remote/dash/-report.txt
```

```out
/tmp/test-mysftp/runtime-fixture > get remote/a.txt
Found 1 file(s) to download
✓ a.txt (9 B)

✓ Downloaded 1 file(s) in 5ms
/tmp/test-mysftp/runtime-fixture > get remote/a.txt -d download-name --name renamed.txt
Downloading a.txt (1/1 files) 100% |████████████████████████████████████████|
✓ Downloaded 1 file(s) in 9ms
/tmp/test-mysftp/runtime-fixture > get remote/src/a.txt remote/src/nested/b.txt -d download-multi
Found 2 file(s) to download
✓ a.txt (13 B)
✓ b.txt (20 B)

✓ Downloaded 2 file(s) in 17ms
/tmp/test-mysftp/runtime-fixture > get -r remote/dir -d download-dir
Found 1 file(s) to download
✓ root.txt (16 B)

✓ Downloaded 1 file(s) in 12ms
/tmp/test-mysftp/runtime-fixture > get remote/src/**/*.txt -d download-glob
Found 2 file(s) to download
✓ b.txt (20 B)
✓ a.txt (13 B)

✓ Downloaded 2 file(s) in 43ms
/tmp/test-mysftp/runtime-fixture > get remote/dir/** -d download-globstar -r
Error: duplicate target path in transfer plan: h:\srccode\playground\mygo-sftp\my-sftp\.sspec\changes\26-03-19t14-34_transfer-contract-hardening\runtime-fixture\download-globstar\remote\dir\nested\child.txt
/tmp/test-mysftp/runtime-fixture > get remote/dir/** -d download-flatten-ok -r --flatten
Error: duplicate basename in --flatten mode: child.txt
Hint: remove --flatten or narrow source set
/tmp/test-mysftp/runtime-fixture > get -d download-dash -- remote/dash/-report.txt
Found 1 file(s) to download
✓ -report.txt (19 B)

✓ Downloaded 1 file(s) in 25ms
```

## Batch 4: Download Expected Failures

```text
get remote/flat/** -d download-flatten-dup -r --flatten
get remote/src/a.txt remote/src/nested/b.txt
```

```out
/tmp/test-mysftp/runtime-fixture > get remote/flat/** -d download-flatten-dup -r --flatten
Error: duplicate basename in --flatten mode: readme.md
Hint: remove --flatten or narrow source set
/tmp/test-mysftp/runtime-fixture > get remote/src/a.txt remote/src/nested/b.txt
Error: multiple get sources require destination: use -d <local_dir>
```

## Batch 5: Parent-Relative Cases

This batch switches both local and remote working directories to the fixture `workspace/` directories so `../logs/*.log` resolves against the sibling `logs/` trees.

```text
lcd "H:/SrcCode/playground/mygo-sftp/my-sftp/.sspec/changes/26-03-19T14-34_transfer-contract-hardening/runtime-fixture/workspace"
cd /tmp/test-mysftp/runtime-fixture/workspace
lpwd
pwd
put ../logs/*.log -d upload-parent-glob
get ../logs/*.log -d download-parent-glob
```

```out
/tmp/test-mysftp/runtime-fixture > lcd "H:/SrcCode/playground/mygo-sftp/my-sftp/.sspec/changes/26-03-19T14-34_transfer-contract-hardening/runtime-fixture/workspace"
/tmp/test-mysftp/runtime-fixture > cd /tmp/test-mysftp/runtime-fixture/workspace
/tmp/test-mysftp/runtime-fixture/workspace > lpwd
H:/SrcCode/playground/mygo-sftp/my-sftp/.sspec/changes/26-03-19T14-34_transfer-contract-hardening/runtime-fixture/workspace
/tmp/test-mysftp/runtime-fixture/workspace > pwd
/tmp/test-mysftp/runtime-fixture/workspace
/tmp/test-mysftp/runtime-fixture/workspace > put ../logs/*.log -d upload-parent-glob
Found 1 file(s) to upload
✓ app.log (19 B)

✓ Uploaded 1 file(s) in 11ms
/tmp/test-mysftp/runtime-fixture/workspace > get ../logs/*.log -d download-parent-glob
Found 1 file(s) to download
✓ app.log (26 B)

✓ Downloaded 1 file(s) in 43ms
```

## Batch 6: Optional Remote Inspection

Paste after upload-related batches if you want a quick remote tree check.

```text
cd /tmp/test-mysftp/runtime-fixture
!tree upload-name upload-multi upload-dir upload-glob upload-globstar upload-flatten-ok upload-dash upload-legacy upload-parent-glob
```

```out
# 执行失败
/tmp/test-mysftp/runtime-fixture > !tree upload-name upload-multi upload-dir upload-glob upload-globstar upload-flatten-ok upload-dash upload-legacy upload-parent-glob
[Remote] Executing: tree upload-name upload-multi upload-dir upload-glob upload-globstar upload-flatten-ok upload-dash upload-legacy upload-parent-glob
Error: remote command failed: Process exited with status 139 from signal SEGV
```

## Batch 7: Optional Local Inspection

Paste after download-related batches if you want a quick local tree check.

```text
lcd "H:/SrcCode/playground/mygo-sftp/my-sftp/.sspec/changes/26-03-19T14-34_transfer-contract-hardening/runtime-fixture"
!! tree download-name
!! tree download-multi
!! tree download-dir
!! tree download-glob
!! tree download-globstar
!! tree download-flatten-ok
!! tree download-dash
!! tree download-parent-glob
```

```out
/tmp/test-mysftp/runtime-fixture > lcd "H:/SrcCode/playground/mygo-sftp/my-sftp/.sspec/changes/26-03-19T14-34_transfer-contract-hardening/runtime-fixture"
/tmp/test-mysftp/runtime-fixture > !! tree download-name
[Local] Executing: tree download-name
Folder PATH listing for volume 文件
Volume serial number is 0885-A00A
H:\SRCCODE\PLAYGROUND\MYGO-SFTP\MY-SFTP\.SSPEC\CHANGES\26-03-19T14-34_TRANSFER-CONTRACT-HARDENING\RUNTIME-FIXTURE\DOWNLOAD-NAME
No subfolders exist

/tmp/test-mysftp/runtime-fixture > !! tree download-multi
[Local] Executing: tree download-multi
Folder PATH listing for volume 文件
Volume serial number is 00000055 0885:A00A
H:\SRCCODE\PLAYGROUND\MYGO-SFTP\MY-SFTP\.SSPEC\CHANGES\26-03-19T14-34_TRANSFER-CONTRACT-HARDENING\RUNTIME-FIXTURE\DOWNLOAD-MULTI
└───remote
    └───src
        └───nested
/tmp/test-mysftp/runtime-fixture > !! tree download-dir
[Local] Executing: tree download-dir
Folder PATH listing for volume 文件
Volume serial number is 000000BD 0885:A00A
H:\SRCCODE\PLAYGROUND\MYGO-SFTP\MY-SFTP\.SSPEC\CHANGES\26-03-19T14-34_TRANSFER-CONTRACT-HARDENING\RUNTIME-FIXTURE\DOWNLOAD-DIR
No subfolders exist

/tmp/test-mysftp/runtime-fixture > !! tree download-glob
[Local] Executing: tree download-glob
Folder PATH listing for volume 文件
Volume serial number is 0000009D 0885:A00A
H:\SRCCODE\PLAYGROUND\MYGO-SFTP\MY-SFTP\.SSPEC\CHANGES\26-03-19T14-34_TRANSFER-CONTRACT-HARDENING\RUNTIME-FIXTURE\DOWNLOAD-GLOB
└───remote
    └───src
        └───nested
/tmp/test-mysftp/runtime-fixture > !! tree download-globstar
[Local] Executing: tree download-globstar
Folder PATH listing for volume 文件
Volume serial number is 000000D7 0885:A00A
H:\SRCCODE\PLAYGROUND\MYGO-SFTP\MY-SFTP\.SSPEC\CHANGES\26-03-19T14-34_TRANSFER-CONTRACT-HARDENING\RUNTIME-FIXTURE\DOWNLOAD-GLOBSTAR
No subfolders exist

/tmp/test-mysftp/runtime-fixture > !! tree download-flatten-ok
[Local] Executing: tree download-flatten-ok
Folder PATH listing for volume 文件
Volume serial number is 000000EA 0885:A00A
H:\SRCCODE\PLAYGROUND\MYGO-SFTP\MY-SFTP\.SSPEC\CHANGES\26-03-19T14-34_TRANSFER-CONTRACT-HARDENING\RUNTIME-FIXTURE\DOWNLOAD-FLATTEN-OK
No subfolders exist

/tmp/test-mysftp/runtime-fixture > !! tree download-dash
[Local] Executing: tree download-dash
Folder PATH listing for volume 文件
Volume serial number is 000000AD 0885:A00A
H:\SRCCODE\PLAYGROUND\MYGO-SFTP\MY-SFTP\.SSPEC\CHANGES\26-03-19T14-34_TRANSFER-CONTRACT-HARDENING\RUNTIME-FIXTURE\DOWNLOAD-DASH
No subfolders exist
```
