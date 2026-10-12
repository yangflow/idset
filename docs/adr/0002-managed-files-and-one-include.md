# 0002 只写托管文件，用户配置里只加一行 Include

- 状态：已采纳
- 日期：2026-10-12
- 相关：[docs/product.md](../product.md)「生成配置」

## 背景

用户的 `~/.ssh/config` 和 `~/.gitconfig` 往往是手写的，有注释、有别的工具的配置，有的放在 dotfiles 仓库里。直接改写这两个文件，一旦出错会弄坏用户的整个 SSH 和 Git 环境，而且和 dotfiles 管理工具冲突。

## 决定

- Idset 只拥有自己的目录（各平台路径见 product.md），所有身份配置都生成到这个目录下的托管文件里，每次从 `accounts.json` 整体重新生成。
- 用户的 `~/.ssh/config` 和 `~/.gitconfig` 里只插入一个带标记的块，块里只有一行 `Include`（Git 是 `[include] path = ...`）。插入是幂等的，第二次插入不会重复。
- 移除时只删标记块，不动文件其余内容，也不删 `accounts.json` 和私钥。
- 私钥只按路径引用，Idset 不读取、不复制、不上传私钥内容。

## 代价

`Include` 在 `~/.ssh/config` 里必须放在最前面，才能对所有 `Host` 生效，所以标记块总是插到文件开头。用户手写的同名 `Host` 或 `includeIf` 可能和托管文件冲突，要靠诊断提示，不能自动修。

## 什么时候重新考虑

`ssh` 或 `git` 改变 `Include` 的语义，或者某个平台上的 `ssh` 不支持 `Include`。
