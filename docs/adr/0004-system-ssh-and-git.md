# 0004 调用系统的 ssh 和 git，不在进程内重新实现

- 状态：已采纳
- 日期：2026-10-12
- 相关：[docs/product.md](../product.md)「连通性检查」「密钥」

## 背景

连通性检查、生成密钥、加载到 ssh-agent、读仓库的 `user.email` 和远程地址，都可以用 Go 库在进程内做（`golang.org/x/crypto/ssh`、go-git），也可以调用系统自带的 `ssh`、`ssh-keygen`、`ssh-add`、`git`。

## 决定

调用系统命令。检查连通性要和用户在终端里 `git push` 时走同一条路径：同一份 `~/.ssh/config`、同一个 agent、同一套 `ProxyCommand` 和 `Include`。进程内实现很难完全一致，查出来的结果会和真实情况不符。三个平台都自带这些命令：macOS 和大多数 Linux 默认安装，Windows 10/11 自带 OpenSSH 客户端，Git for Windows 也带一套。

- 每个命令带超时，参数用参数列表传，不拼 shell 字符串。
- 找不到命令时，诊断里说明缺哪个、怎么装，其他功能照常可用。

## 代价

依赖用户机器上的命令版本，输出格式要按版本兼容。Windows 上可能同时有系统 OpenSSH 和 Git 自带的 ssh，要决定用哪个（跟随 `git config core.sshCommand`，没有就用 PATH 里第一个）。

## 什么时候重新考虑

某个平台上系统命令普遍缺失，或者输出格式变化频繁到难以维护。
