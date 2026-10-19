# Idset

[English](README.md) | 中文

每个文件夹都用对 Git 和 SSH 身份。纯 Go 写的原生托盘应用。

[![CI](https://github.com/yangflow/idset/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/yangflow/idset/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/github/license/yangflow/idset)](LICENSE)

同一台机器上，工作仓库和个人仓库要用不同的提交邮箱和不同的 SSH 密钥。要配对，就得手改 `~/.ssh/config` 里的 Host 别名、`~/.gitconfig` 里的 `includeIf "gitdir:"`，还要记住每个克隆地址该填哪个别名。错一处，提交就署错了名，或者推送被拒。

Idset 把身份放在一个列表里，替你生成配置。它不改写你自己维护的文件：只生成自己的文件，再在 `~/.ssh/config` 和 `~/.gitconfig` 里各加一行 `Include`。私钥留在原处，Idset 只按路径引用。

## 功能

早期开发，0.1.0 是第一个发布版。要做的行为写在 [docs/product.md](docs/product.md)，按[里程碑](https://github.com/yangflow/idset/milestones)的顺序，一个 issue 一个 issue 地落地。

已经可用：

- **托盘里的身份列表。** 点托盘图标打开面板，列出 `~/.config/idset/accounts.json` 里的身份，最近用过的排在前面；三个及以上时可以筛选。
- **拖入文件夹查看身份。** 把项目文件夹拖进面板，或者选择一个，看 Git 在这里会用哪个身份。
- **核对仓库和身份。** 匹配之后，卡片会指出仓库自己的 `user.email` 缺失、属于另一个身份或不属于任何身份，或者 `core.sshCommand` 指定了别的密钥，也就是提交和推送不会以这个身份进行。
- **在结果卡片上绑定。** 在拖入文件夹的结果卡片上，把文件夹绑到某个身份、解绑、改绑到另一个身份，或者处理多个身份覆盖同一个文件夹的情况。绑定、改绑和「这里只用」之后 5 秒内可以撤销。
- **文件夹改名或挪位置后跟着改。** 拖入的文件夹没有匹配，但像是某个身份还指向的文件夹改了名或挪了位置时，面板会提示把旧路径改成新路径，也可以保留旧路径。
- **在应用里管理身份。** 账户窗口可以新建、修改和删除身份：署名、平台、SSH 密钥和别名、各自生效的文件夹、地址改写。删除时可以连私钥一起删，但只在你勾选时才会删。
- **生成配置。** SSH Host 别名、`includeIf` 规则和 `insteadOf` 地址改写都从列表生成，每次修改后重新生成，新增身份前先备份。你自己的 `~/.ssh/config` 和 `~/.gitconfig` 里只多一个带标记的 `Include` 块，别的都不动。
- **克隆和改地址用 SSH 别名。** 仓库的远程地址是 HTTPS 时，卡片复制把它改成这个身份 SSH 别名的命令。克隆助手把 `owner/repo`、仓库地址或 `git clone` 命令变成这个身份的克隆命令。
- **设置、备份和恢复。** 设置窗口可以选界面语言、设置开机启动，列出备份供恢复或删除（`accounts.json` 读不出来时也能用），并移除 `Include` 块，同时保留你的账户。
- **诊断。** 在后台用 `ssh -T` 检查每个身份，结果是身份行上的一个圆点。密钥被拒、被锁或缺失，主机连不上，或者没装 `ssh` 时，告诉你下一步怎么做；列表下方的汇总和托盘图标上的角标显示最严重的问题。
- **密钥。** 设置窗口可以生成 SSH 密钥、复制公钥、加载到 ssh-agent，并把它交给一个身份，或用它新建身份。私钥缺失的问题条可以直接打开它，并按那个身份填好。
- **导入。** 设置窗口读取你已有的 `~/.ssh/config` 和 Git 配置，列出其中的身份，添加你勾选的那些。你的文件只读不写。

目标平台：macOS 12+、Windows 10/11、带 GTK 3 的 Linux，amd64 和 arm64。Linux 的托盘图标需要 AppIndicator。

发布版由版本 tag 构建并发布（见 [docs/release.md](docs/release.md)），已安装的副本从发布版自动更新。macOS 应用经过签名和公证。Windows 安装包暂未签名，第一次启动时 SmartScreen 会警告。Debian 包需要 `libayatana-appindicator3-1`。

## 快速开始

从[最新发布版](https://github.com/yangflow/idset/releases/latest)下载对应平台的安装包。
也可以从源码构建，需要 `.go-version` 里的 Go 版本和 Python 3：

```bash
git clone https://github.com/yangflow/idset
cd idset
go tool mygo dev           # 运行桌面应用
go tool mygo build         # 打包到 build/
```

## 使用

1. 启动 Idset，它常驻在托盘里（macOS 是菜单栏，Windows 是通知区域）。
2. 点托盘图标打开面板，选「添加身份」打开账户窗口新建一个；点面板里的身份可以修改或删除。身份保存在 `~/.config/idset/accounts.json`（Windows 上是 `%USERPROFILE%\.config\idset\accounts.json`），格式见 [docs/product.md](docs/product.md)。
3. 把项目文件夹拖进面板，或者选择一个，看 Git 在这里会用哪个身份；不对的话直接在卡片上改绑。

## 开发

```bash
scripts/check.sh --cross          # gofmt、go mod tidy、vet、测试、脚本测试、六个目标的交叉编译
go test -race ./...
python3 scripts/scan_secrets.py   # 扫描全部历史里的密钥
```

| 主题 | 位置 |
|---|---|
| 行为、数据模型和生成的文件 | [docs/product.md](docs/product.md) |
| 设计语言：布局、颜色、字体、组件、无障碍（英文） | [docs/design.md](docs/design.md) |
| 界面画板 | [design/](design/README.md) |
| 工作顺序 | [里程碑](https://github.com/yangflow/idset/milestones)和 [issue](https://github.com/yangflow/idset/issues) |
| 架构决定（英文） | [docs/adr/](docs/adr/README.md) |

人和代理改仓库的规矩在 [AGENTS.md](AGENTS.md)。漏洞请按 [SECURITY.md](SECURITY.md) 私下报告，不要开公开 issue。

## 参与贡献

欢迎用中文或英文提 issue 和 PR，流程见 [CONTRIBUTING.md](CONTRIBUTING.md)（英文）。提交信息和 PR 标题用 [Conventional Commits](https://www.conventionalcommits.org/)，不需要 DCO 签名。

## 许可

[MIT](LICENSE)。依赖的许可证列在 [THIRD_PARTY.md](THIRD_PARTY.md)。
