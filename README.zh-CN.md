# Idset

[English](README.md) | 中文

每个文件夹都用对 Git 和 SSH 身份。纯 Go 写的原生托盘应用。

[![CI](https://github.com/yangflow/idset/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/yangflow/idset/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/github/license/yangflow/idset)](LICENSE)

同一台机器上，工作仓库和个人仓库要用不同的提交邮箱和不同的 SSH 密钥。要配对，就得手改 `~/.ssh/config` 里的 Host 别名、`~/.gitconfig` 里的 `includeIf "gitdir:"`，还要记住每个克隆地址该填哪个别名。错一处，提交就署错了名，或者推送被拒。

Idset 把身份放在一个列表里，替你生成配置。它不改写你自己维护的文件：只生成自己的文件，再在 `~/.ssh/config` 和 `~/.gitconfig` 里各加一行 `Include`。私钥留在原处，Idset 只按路径引用。

## 功能

早期开发，版本 0.1.0，还没有发布。要做的行为写在 [docs/product.md](docs/product.md)，按[路线图](docs/roadmap.md)的顺序落地。

已经可用：

- **托盘里的身份列表。** 点托盘图标打开面板，列出 `~/.config/idset/accounts.json` 里的身份，最近用过的排在前面；三个及以上时可以筛选。
- **拖入文件夹查看身份。** 把项目文件夹拖进面板，或者选择一个，看 Git 在这里会用哪个身份。
- **在应用里管理身份。** 账户窗口可以新建、修改和删除身份：署名、平台、SSH 密钥和别名、各自生效的文件夹、地址改写。删除时可以连私钥一起删，但只在你勾选时才会删。
- **生成配置。** SSH Host 别名、`includeIf` 规则和 `insteadOf` 地址改写都从列表生成，每次修改后重新生成，新增身份前先备份。你自己的 `~/.ssh/config` 和 `~/.gitconfig` 里只多一个带标记的 `Include` 块，别的都不动。

计划中：

- **一键绑定。** 在面板里把文件夹绑到另一个身份，可以撤销。
- **诊断。** 用 `ssh -T` 检查每个身份。密钥被拒、被锁或缺失，或者仓库会用一个身份提交、用另一个身份推送时，告诉你下一步怎么做。
- **密钥和导入。** 生成密钥、加载到 ssh-agent，从已有的 `~/.ssh/config` 和 `~/.gitconfig` 导入身份。

目标平台：macOS 12+、Windows 10/11、带 GTK 3 的 Linux，amd64 和 arm64。Linux 的托盘图标需要 AppIndicator。

## 快速开始

还没有发布。从源码构建，需要 `.go-version` 里的 Go 版本和 Python 3：

```bash
git clone https://github.com/yangflow/idset
cd idset
go tool mygo dev           # 运行桌面应用
go tool mygo build         # 打包到 build/
```

## 使用

1. 启动 Idset，它常驻在托盘里（macOS 是菜单栏，Windows 是通知区域）。
2. 点托盘图标打开面板，选「添加身份」打开账户窗口新建一个；点面板里的身份可以修改或删除。身份保存在 `~/.config/idset/accounts.json`（Windows 上是 `%USERPROFILE%\.config\idset\accounts.json`），格式见 [docs/product.md](docs/product.md)。
3. 把项目文件夹拖进面板，或者选择一个，看 Git 在这里会用哪个身份。

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
| 工作顺序 | [docs/roadmap.md](docs/roadmap.md) |
| 架构决定（英文） | [docs/adr/](docs/adr/README.md) |

人和代理改仓库的规矩在 [AGENTS.md](AGENTS.md)。漏洞请按 [SECURITY.md](SECURITY.md) 私下报告，不要开公开 issue。

## 参与贡献

欢迎用中文或英文提 issue 和 PR，流程见 [CONTRIBUTING.md](CONTRIBUTING.md)（英文）。提交信息和 PR 标题用 [Conventional Commits](https://www.conventionalcommits.org/)，不需要 DCO 签名。

## 许可

[MIT](LICENSE)。依赖的许可证列在 [THIRD_PARTY.md](THIRD_PARTY.md)。
