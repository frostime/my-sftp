---
name: cli-usage
description: "my-sftp 交互式 CLI 的使用规范，包括连接方式、命令分类、传输语法和路径约定"
updated: 2026-03-19
scope:
  - /main.go
  - /shell/**
  - /completer/**
  - /README.md
  - /README.zh.md
---

# CLI Usage Specification

## Overview

本文档定义 `my-sftp` 的交互式 CLI 使用规范。它关注用户在命令行中看到和依赖的行为，包括：

- 程序启动和连接方式
- 交互式 shell 中的命令分类
- `get` / `put` 的正式语法与约束
- 本地/远端路径的解释规则
- 兼容行为与错误边界

目标是让 README、shell 实现、自动补全和未来文档都遵循同一套稳定约定。

## Program Entry

启动形式：

```text
my-sftp [--version] <destination>
```

其中 `<destination>` 支持两种输入：

- SSH config alias，例如 `my-sftp myserver`
- 直接目标地址，例如 `my-sftp user@host` 或 `my-sftp user@host:2222`

基本约定：

- `--version` 只输出版本信息，不建立连接
- 未提供 `<destination>` 时，程序输出用法并退出
- 连接成功后进入交互式 shell

## Shell Model

shell 是一个状态化 REPL，始终维护两套工作目录：

- remote cwd: 当前远端工作目录，用于远端路径解析
- local cwd: 当前本地工作目录，用于本地路径解析

提示符展示 remote cwd。以下命令会改变工作目录状态：

- `cd <remote_dir>`: 修改 remote cwd
- `lcd <local_dir>`: 修改 local cwd
- `pwd`: 显示 remote cwd
- `lpwd`: 显示 local cwd

## Command Categories

### Session Commands

- `help`, `?`: 显示帮助
- `exit`, `quit`, `q`: 退出 shell

### Remote File Commands

- `ls`, `ll`, `dir`
- `cd`, `pwd`
- `get`, `download`
- `put`, `upload`
- `rm`, `del`, `delete`
- `mkdir`, `md`
- `rmdir`, `rd`
- `rename`, `mv`
- `stat`, `info`

### Local File Commands

- `lpwd`
- `lcd`
- `lls`, `ldir`
- `lmkdir`

### Command Execution Escapes

- `! <command>`: 在远端主机执行 shell 命令
- `!! <command>`: 在本地机器执行 shell 命令

约定：

- 这两类 escape 命令不走 `get` / `put` 语法解析
- 它们用于辅助检查和调试，不改变传输语法本身

## Transfer Command Grammar

正式语法：

```text
get [-r] [--flatten] [-d <local_target_dir>] [--name <filename>] [--] <remote_src>...
put [-r] [--flatten] [-d <remote_target_dir>] [--name <filename>] [--] <local_src>...
```

### Shared Option Rules

- `-d` / `--dir` 指定目标目录
- `--name` 仅允许单文件传输时使用，且值必须是纯文件名
- `--flatten` 将最终输出映射为目标目录下的 basename
- `-r` 允许目录递归和需要递归展开的 glob 场景
- `--` 终止选项解析；当 source 以 `-` 开头时必须使用

### Option Ordering

在遇到 `--` 之前，选项可出现在 source 前后：

```text
get src.txt --flatten -d out
put -d /srv/out src.txt
put src.txt -d /srv/out
```

### `--name` Constraints

`--name` 的值必须满足：

- 不是 `.`
- 不是 `..`
- 不包含 `/` 或 `\`

因此以下用法非法：

```text
put a.txt -d out --name nested/out.txt
get a.txt -d out --name ../escape.txt
```

## Source and Destination Semantics

### Single File Without Explicit Destination

- `get report.csv` -> 下载到 local cwd，文件名仍为 `report.csv`
- `put report.csv` -> 上传到 remote cwd，文件名仍为 `report.csv`

### Single File With `-d`

- 默认目标文件名是 source basename
- 若提供 `--name`，则 basename 被 `--name` 覆盖

### Multiple Explicit Sources

多源命令必须显式提供目标目录，除非命中临时兼容回退规则。

默认 preserve 模式下：

- 保留 operand-relative path，而不是只保留 basename
- 多个不同父目录下的同名文件不能静默折叠到同一目标位置

示例：

```text
put src/a.txt src/nested/b.txt -d /srv/out
get remote/src/a.txt remote/src/nested/b.txt -d out
```

### Directory Sources With `-r`

- 单个目录 source 保留其内容树到目标根下
- 多个目录 source 保留每个 source 的 operand-relative path，避免同名目录合并
- `-r` 必须真正开启无限递归；顶层-only 行为不符合 CLI 合约

### Glob Sources

glob 支持：

- `*`
- `**`
- `?`
- `[]`

默认 preserve 模式下：

- preserve path 相对于第一个 wildcard 之前的静态前缀
- 若没有静态前缀，则相对于当前工作目录
- globstar (`**`) 的重叠目录/文件 match 必须先规范化 source set，再展开任务，避免假重复

## Flatten Contract

启用 `--flatten` 后：

- 每个 resolved file entry 映射到 `target_root + basename(file)`
- duplicate basename 是硬错误
- 错误应在传输开始前发生
- 错误提示应包含可操作修复建议

典型提示：

```text
Error: duplicate basename in --flatten mode: readme.md
Hint: remove --flatten or narrow source set
```

## Parent-Relative and Reserved Marker Rules

当 preserve 模式下的 source 含有前导 `..` 时：

- 不允许通过路径清洗逃出目标根目录
- 前导 `..` 必须编码为保留命名空间 `__my_sftp_parent__`

示例：

```text
put ../logs/*.log -d backup
```

期望目标：

```text
backup/__my_sftp_parent__/logs/app.log
```

Windows 绝对本地路径在多源 preserve 模式下还应使用独立的 volume marker，以避免不同盘符根目录冲突。

## Compatibility Policy

存在一条临时兼容规则：

- 当多 source 命令未显式提供 `-d/--dir` 时，可尝试旧式 positional target fallback

约束：

- 这是兼容行为，不是推荐语法
- 命中兼容路径时应输出 deprecation warning
- 新文档和新示例应优先使用显式 `-d/--dir`

## Error Surface Expectations

CLI 错误应尽可能在执行前暴露，而不是在部分传输后才失败。重点包括：

- 多 source 缺少目标目录
- `--name` 非法
- 目录 source 未提供 `-r`
- `--flatten` duplicate basename
- preserve 模式 duplicate target path
- preserve 模式 file-vs-directory prefix conflict

## Completion Alignment

自动补全应与 CLI 语法保持一致：

- `get` / `download` 默认补全远端 source
- `put` / `upload` 默认补全本地 source
- `-d/--dir` 后切换到目标侧路径补全
- `--name` 后不做路径补全

## Expected Behaviour

本节给出一组应当被视为规范样例的具体案例。除非后续 spec 明确变更，这些例子描述的就是 CLI 的期望行为。

### A. Single File Cases

假设：

- local cwd = `C:/work`
- remote cwd = `/srv/app`

```text
put report.csv
=> 上传到 /srv/app/report.csv

get report.csv
=> 下载到 C:/work/report.csv

put report.csv -d /backup
=> 上传到 /backup/report.csv

get report.csv -d out
=> 下载到 C:/work/out/report.csv

put report.csv -d /backup --name final.csv
=> 上传到 /backup/final.csv

get report.csv -d out --name final.csv
=> 下载到 C:/work/out/final.csv
```

### B. Multi-Source Preserve Cases

```text
put src/a.txt src/nested/b.txt -d /srv/out
=> /srv/out/src/a.txt
=> /srv/out/src/nested/b.txt

get remote/src/a.txt remote/src/nested/b.txt -d out
=> out/remote/src/a.txt
=> out/remote/src/nested/b.txt
```

关键点：

- 多 source 默认保留 operand-relative path
- 不允许把多个 source 静默压平成同一层

### C. Recursive Directory Cases

```text
put -r assets -d /srv/static
=> /srv/static/logo.png
=> /srv/static/css/app.css
=> /srv/static/js/app.js

get -r logs -d backup
=> backup/access.log
=> backup/nginx/error.log
```

关键点：

- 单个目录 source 保留其内容树到目标根
- `-r` 必须是真递归，不允许只传顶层文件

### D. Multi-Directory Preserve Cases

```text
put -r app/config shared/config -d /srv/out
=> /srv/out/app/config/...
=> /srv/out/shared/config/...

get -r /srv/a/config /srv/b/config -d out
=> out/srv/a/config/...
=> out/srv/b/config/...
```

关键点：

- 多目录 source 不能因为 basename 相同而合并到同一棵目标树

### E. Glob Preserve Cases

```text
put src/**/*.go -d /srv/code
=> 保留从静态前缀 src/ 开始的相对结构
=> /srv/code/src/main.go
=> /srv/code/src/pkg/util/helper.go

get logs/*.log -d backup
=> 保留从静态前缀 logs/ 开始的相对结构
=> backup/logs/access.log
=> backup/logs/error.log
```

如果 glob 在 wildcard 前没有静态前缀：

```text
put *.txt -d /srv/txt
=> 相对于当前 local cwd 解析
=> 每个匹配文件落到 /srv/txt/<basename>
```

### F. Globstar Overlap Cases

```text
put dir/** -d /srv/out -r
=> 即使 `dir/**` 同时匹配目录和目录内文件，最终也只上传每个 resolved file 一次

get remote/dir/** -d out -r
=> 即使 `remote/dir/**` 同时匹配目录和目录内文件，最终也只下载每个 resolved file 一次
```

期望结果：

- 不报假 duplicate target path
- 不因为同一文件被重复规划而报假 flatten collision

### G. Flatten Cases

```text
put src/**/*.go -d /srv/flat --flatten
=> /srv/flat/main.go
=> /srv/flat/helper.go

get reports/**/*.csv -d out --flatten
=> out/january.csv
=> out/february.csv
```

如果 basename 冲突：

```text
put a/readme.md b/readme.md -d /srv/flat --flatten
=> 失败，不开始传输

get docs/x/readme.md docs/y/readme.md -d out --flatten
=> 失败，不开始传输
```

典型错误：

```text
Error: duplicate basename in --flatten mode: readme.md
Hint: remove --flatten or narrow source set
```

### H. Parent-Relative Preserve Cases

```text
put ../logs/*.log -d backup
=> backup/__my_sftp_parent__/logs/app.log

get ../logs/*.log -d out
=> out/__my_sftp_parent__/logs/app.log
```

关键点：

- parent-relative source 不能逃出目标根目录
- `..` 必须被编码进保留命名空间，而不是被路径清洗吞掉

### I. Dash-Leading Source Cases

```text
put -d /srv/out -- -report.txt
=> 把 `-report.txt` 当作 source，而不是 option

get -d out -- -report.txt
=> 把 `-report.txt` 当作 remote source，而不是 option
```

如果不写 `--`：

```text
put -report.txt
=> 失败，按未知选项处理
```

### J. Compatibility Fallback Cases

```text
put src/a.txt /srv/out
=> 仍允许走旧式 positional target fallback
=> 同时输出 deprecation warning
```

约定：

- 这是临时兼容，不是推荐写法
- 推荐写法始终是 `put src/a.txt -d /srv/out`

### K. Explicit Error Cases

```text
get a.txt b.txt
=> 失败：multiple get sources require destination

put dir -d /srv/out
=> 失败：目录 source 必须配合 -r

put file.txt -d /srv/out --name nested/out.txt
=> 失败：--name 不能包含路径分隔符

put a.txt b.txt -d /srv/out --name merged.txt
=> 失败：--name 只能用于单文件 source
```

### L. Inspection-Oriented Examples

这类命令不改变传输规范，但可用于辅助验证：

```text
!find /srv/out -type f | sort
=> 在远端列出传输结果

!! tree /F out
=> 在本地查看下载结果树
```

它们的价值在于：

- 验证 preserve/flatten 后的最终落点
- 验证 parent-relative source 没有逃逸目标根
- 验证 globstar 场景没有重复文件

## Documentation Rules

当 README、帮助文本或后续 spec-doc 更新 CLI 示例时，应遵循：

- 优先展示显式 `-d/--dir` 语法
- 展示 `--` 处理 dash-leading source 的例子
- 展示 preserve 与 `--flatten` 的区别
- 如涉及 parent-relative source，应明确说明它们会被保留在目标根内部，而不是逃逸到目标根外
