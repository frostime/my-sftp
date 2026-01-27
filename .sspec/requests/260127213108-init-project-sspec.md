---
created: 2026-01-27T21:31:08
status: OPEN
attach-change: null
tldr: ""
---

# Request: init-project-sspec

## What I Want

我希望助手能仔细阅览这个项目，然后填充 sspec 关于本项目的知识

特别是关于项目的代码程序架构。

## Why

这个项目是已经存在的老项目，我现在开始接收。

但是我对 Go 和这个项目没有那么熟悉。需要 agent 帮我全面调研。

这样后续开发起来就更加方便。有了关于这个项目充分的信息知识。

## What to do

1. 浏览整个项目
2. 填充 .sspec/project.md
3. sspec spec new architecture  -- 填充整个项目的实现架构逻辑
4. sspec spec new sftp-transfer -- 说明当前项目如何实现基于 sftp 传输文件

## Additional Context

主要关注:

main.go
client/
completer/
config/
shell/
