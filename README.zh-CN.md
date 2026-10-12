<h1 align="center">Idset</h1>

<p align="center">
  每个文件夹都用对 Git 和 SSH 身份。纯 Go 写的原生托盘应用。
</p>

<p align="center">
  <a href="https://github.com/yangflow/idset/actions/workflows/ci.yml"><img src="https://github.com/yangflow/idset/actions/workflows/ci.yml/badge.svg?branch=main" alt="CI"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/yangflow/idset" alt="License: MIT"></a>
</p>

<p align="center">
  <a href="README.md">English</a> · <b>简体中文</b>
</p>

## 为什么做 Idset

同一台机器上，工作仓库和个人仓库要用不同的提交邮箱和不同的 SSH 密钥。要配对，就得手改 `~/.ssh/config` 里的 Host 别名、`~/.gitconfig` 里的 `includeIf "gitdir:"`，还要记住每个克隆地址该填哪个别名。错一处，提交就署错了名，或者推送被拒。

Idset 把身份放在一个列表里，替你生成配置。它不改写你自己维护的文件：只生成自己的文件，再在 `~/.ssh/config` 和 `~/.gitconfig` 里各加一行 `Include`。私钥留在原处，Idset 只按路径引用。

## 状态

早期开发，版本 0.1.0，还没有发布。窗口里还是模板的计数示例。要做的行为写在 [docs/product.md](docs/product.md)，按[路线图](docs/roadmap.md)的顺序落地。

计划中：

- **托盘里的身份列表。** 点托盘图标打开面板，身份按最近使用排在前面。
- **拖入文件夹查看身份。** 把项目文件夹拖进面板，看 Git 和 SSH 在这里会用哪个身份，一键改绑到别的身份。
- **生成配置。** SSH Host 别名、`includeIf` 规则和 `insteadOf` 地址改写都从列表生成，每次修改前先备份。
- **诊断。** 用 `ssh -T` 检查每个身份。密钥被拒、被锁或缺失，或者仓库会用一个身份提交、用另一个身份推送时，告诉你下一步怎么做。
- **密钥和导入。** 生成密钥、加载到 ssh-agent，从已有的 `~/.ssh/config` 和 `~/.gitconfig` 导入身份。

目标平台：macOS 12+、Windows 10/11、带 GTK 3 的 Linux，amd64 和 arm64。Linux 的托盘图标需要 AppIndicator。

## 快速开始

从源码构建，需要 Go 1.27.1（见 `.go-version`）和 Python 3：

```bash
git clone https://github.com/yangflow/idset
cd idset
scripts/check.sh --cross   # 格式、vet、测试和六个目标的交叉编译
go tool mygo dev           # 运行桌面应用
go tool mygo build         # 打包到 build/
```

## 安全

漏洞请按 [SECURITY.md](SECURITY.md) 私下报告，不要开公开 issue。

## 文档

| 主题 | 位置 |
|---|---|
| 行为、数据模型和生成的文件 | [docs/product.md](docs/product.md) |
| 颜色、字体、间距和组件 | [docs/design-system.md](docs/design-system.md) |
| 界面和状态 | [design/README.md](design/README.md) |
| 工作顺序 | [docs/roadmap.md](docs/roadmap.md) |
| 架构决定 | [docs/adr/](docs/adr/README.md) |
| 改仓库的规矩 | [AGENTS.md](AGENTS.md) |

## 参与贡献

欢迎 issue 和 PR，先读 [CONTRIBUTING.md](CONTRIBUTING.md)。每个提交都用 `git commit -s` 签名（Developer Certificate of Origin）。

## 求助

见 [SUPPORT.md](SUPPORT.md)。问题统一走 issue，没有 Discussions，也没有付费支持。

## 许可

[MIT](LICENSE)。依赖的许可证列在 [THIRD_PARTY.md](THIRD_PARTY.md)。
