# Runtime Retest Minimal Batches

Purpose: rerun only the still-open runtime checks after the next code fix.

Assumed fixture roots:

- Local fixture root: `H:/SrcCode/playground/mygo-sftp/my-sftp/.sspec/changes/26-03-19T14-34_transfer-contract-hardening/runtime-fixture`
- Remote fixture root: `/tmp/test-mysftp/runtime-fixture`

Use fresh target names so no cleanup is required.

## Batch 0: Enter Fixture Roots

```text
lcd "H:/SrcCode/playground/mygo-sftp/my-sftp/.sspec/changes/26-03-19T14-34_transfer-contract-hardening/runtime-fixture"
cd /tmp/test-mysftp/runtime-fixture
lpwd
pwd
```

```out
/home/zyp > lcd "H:/SrcCode/playground/mygo-sftp/my-sftp/.sspec/changes/26-03-19T14-34_transfer-contract-hardening/runtime-fixture"
/home/zyp > cd /tmp/test-mysftp/runtime-fixture
/tmp/test-mysftp/runtime-fixture > lpwd
H:/SrcCode/playground/mygo-sftp/my-sftp/.sspec/changes/26-03-19T14-34_transfer-contract-hardening/runtime-fixture
/tmp/test-mysftp/runtime-fixture > pwd
/tmp/test-mysftp/runtime-fixture
```

## Batch 1: Upload Retest

Expected after fix:
- all three commands succeed
- `retest-upload-dir` contains `root.txt` and `nested/child.txt`
- `retest-upload-globstar` contains `local/dir/root.txt` and `local/dir/nested/child.txt`
- `retest-upload-flatten-ok` contains only `root.txt` and `child.txt`

```text
put -r local/dir -d retest-upload-dir
put local/dir/** -d retest-upload-globstar -r
put local/dir/** -d retest-upload-flatten-ok -r --flatten
```

```out
/tmp/test-mysftp/runtime-fixture > put -r local/dir -d retest-upload-dir
Found 2 file(s) to upload
✓ root.txt (9 B)
✓ child.txt (10 B)

✓ Uploaded 2 file(s) in 23ms
/tmp/test-mysftp/runtime-fixture > put local/dir/** -d retest-upload-globstar -r
Found 2 file(s) to upload
✓ child.txt (10 B)
✓ root.txt (9 B)

✓ Uploaded 2 file(s) in 33ms
/tmp/test-mysftp/runtime-fixture > put local/dir/** -d retest-upload-flatten-ok -r --flatten
Found 2 file(s) to upload
✓ root.txt (9 B)
✓ child.txt (10 B)

✓ Uploaded 2 file(s) in 19ms
```

## Batch 2: Download Retest

Expected after fix:
- all three commands succeed
- `retest-download-dir` contains `root.txt` and `nested/child.txt`
- `retest-download-globstar` contains `remote/dir/root.txt` and `remote/dir/nested/child.txt`
- `retest-download-flatten-ok` contains only `root.txt` and `child.txt`

```text
get -r remote/dir -d retest-download-dir
get remote/dir/** -d retest-download-globstar -r
get remote/dir/** -d retest-download-flatten-ok -r --flatten
```

```out
/tmp/test-mysftp/runtime-fixture >
/tmp/test-mysftp/runtime-fixture > get -r remote/dir -d retest-download-dir
Found 2 file(s) to download
✓ root.txt (16 B)
✓ child.txt (17 B)

✓ Downloaded 2 file(s) in 38ms
/tmp/test-mysftp/runtime-fixture > get remote/dir/** -d retest-download-globstar -r
Found 2 file(s) to download
✓ child.txt (17 B)
✓ root.txt (16 B)

✓ Downloaded 2 file(s) in 20ms
/tmp/test-mysftp/runtime-fixture > get remote/dir/** -d retest-download-flatten-ok -r --flatten
Found 2 file(s) to download
✓ root.txt (16 B)
✓ child.txt (17 B)

✓ Downloaded 2 file(s) in 38ms
```

## Batch 3: Parent-Relative Guard Retest

Expected after fix:
- both commands succeed
- upload lands under `retest-upload-parent-glob/__my_sftp_parent__/logs/app.log`
- download lands under `retest-download-parent-glob/__my_sftp_parent__/logs/app.log`

```text
lcd "H:/SrcCode/playground/mygo-sftp/my-sftp/.sspec/changes/26-03-19T14-34_transfer-contract-hardening/runtime-fixture/workspace"
cd /tmp/test-mysftp/runtime-fixture/workspace
put ../logs/*.log -d retest-upload-parent-glob
get ../logs/*.log -d retest-download-parent-glob
```

```out
/tmp/test-mysftp/runtime-fixture > lcd "H:/SrcCode/playground/mygo-sftp/my-sftp/.sspec/changes/26-03-19T14-34_transfer-contract-hardening/runtime-fixture/workspace"
/tmp/test-mysftp/runtime-fixture > cd /tmp/test-mysftp/runtime-fixture/workspace
/tmp/test-mysftp/runtime-fixture/workspace > put ../logs/*.log -d retest-upload-parent-glob
Found 1 file(s) to upload
✓ app.log (19 B)

✓ Uploaded 1 file(s) in 46ms
/tmp/test-mysftp/runtime-fixture/workspace > get ../logs/*.log -d retest-download-parent-glob
Found 1 file(s) to download
✓ app.log (26 B)

✓ Downloaded 1 file(s) in 21ms
```

## Batch 4: Remote Inspection

Use this after Batch 1 and Batch 3.

```text
cd /tmp/test-mysftp/runtime-fixture
!find retest-upload-dir retest-upload-globstar retest-upload-flatten-ok workspace/retest-upload-parent-glob -type f | sort
```

```out
/tmp/test-mysftp/runtime-fixture/workspace > cd /tmp/test-mysftp/runtime-fixture
/tmp/test-mysftp/runtime-fixture > !find retest-upload-dir retest-upload-globstar retest-upload-flatten-ok workspace/retest-upload-parent-glob -type f | sort
[Remote] Executing: find retest-upload-dir retest-upload-globstar retest-upload-flatten-ok workspace/retest-upload-parent-glob -type f | sort
retest-upload-dir/nested/child.txt
retest-upload-dir/root.txt
retest-upload-flatten-ok/child.txt
retest-upload-flatten-ok/root.txt
retest-upload-globstar/local/dir/nested/child.txt
retest-upload-globstar/local/dir/root.txt
workspace/retest-upload-parent-glob/__my_sftp_parent__/logs/app.log
```

## Batch 5: Local Inspection

Use this after Batch 2 and Batch 3.

```text
lcd "H:/SrcCode/playground/mygo-sftp/my-sftp/.sspec/changes/26-03-19T14-34_transfer-contract-hardening/runtime-fixture"
!! tree /F retest-download-dir
!! tree /F retest-download-globstar
!! tree /F retest-download-flatten-ok
!! tree /F workspace\retest-download-parent-glob
```

```out
/tmp/test-mysftp/runtime-fixture > lcd "H:/SrcCode/playground/mygo-sftp/my-sftp/.sspec/changes/26-03-19T14-34_transfer-contract-hardening/runtime-fixture"
/tmp/test-mysftp/runtime-fixture > lpwd
H:/SrcCode/playground/mygo-sftp/my-sftp/.sspec/changes/26-03-19T14-34_transfer-contract-hardening/runtime-fixture
/tmp/test-mysftp/runtime-fixture > !! tree /F retest-download-dir
[Local] Executing: tree /F retest-download-dir
Folder PATH listing for volume 文件
Volume serial number is 0000002A 0885:A00A
H:\SRCCODE\PLAYGROUND\MYGO-SFTP\MY-SFTP\.SSPEC\CHANGES\26-03-19T14-34_TRANSFER-CONTRACT-HARDENING\RUNTIME-FIXTURE\RETEST-DOWNLOAD-DIR
│   root.txt
│
└───nested
        child.txt

/tmp/test-mysftp/runtime-fixture > !! tree /F retest-download-globstar
[Local] Executing: tree /F retest-download-globstar
Folder PATH listing for volume 文件
Volume serial number is 000000E8 0885:A00A
H:\SRCCODE\PLAYGROUND\MYGO-SFTP\MY-SFTP\.SSPEC\CHANGES\26-03-19T14-34_TRANSFER-CONTRACT-HARDENING\RUNTIME-FIXTURE\RETEST-DOWNLOAD-GLOBSTAR
└───remote
    └───dir
        │   root.txt
        │
        └───nested
                child.txt

/tmp/test-mysftp/runtime-fixture > !! tree /F retest-download-flatten-ok
[Local] Executing: tree /F retest-download-flatten-ok
Folder PATH listing for volume 文件
Volume serial number is 000000EE 0885:A00A
H:\SRCCODE\PLAYGROUND\MYGO-SFTP\MY-SFTP\.SSPEC\CHANGES\26-03-19T14-34_TRANSFER-CONTRACT-HARDENING\RUNTIME-FIXTURE\RETEST-DOWNLOAD-FLATTEN-OK
    child.txt
    root.txt

No subfolders exist

/tmp/test-mysftp/runtime-fixture > !! tree /F workspace\retest-download-parent-glob
[Local] Executing: tree /F workspace\retest-download-parent-glob
Folder PATH listing for volume 文件
Volume serial number is 0000005F 0885:A00A
H:\SRCCODE\PLAYGROUND\MYGO-SFTP\MY-SFTP\.SSPEC\CHANGES\26-03-19T14-34_TRANSFER-CONTRACT-HARDENING\RUNTIME-FIXTURE\WORKSPACE\RETEST-DOWNLOAD-PARENT-GLOB
└───__my_sftp_parent__
    └───logs
            app.log
```