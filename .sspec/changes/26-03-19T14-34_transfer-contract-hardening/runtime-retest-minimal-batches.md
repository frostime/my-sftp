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

## Batch 4: Remote Inspection

Use this after Batch 1 and Batch 3.

```text
cd /tmp/test-mysftp/runtime-fixture
!find retest-upload-dir retest-upload-globstar retest-upload-flatten-ok workspace/retest-upload-parent-glob -type f | sort
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
