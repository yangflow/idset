# 0001 纯 Go 原生 UI

- 状态：已采纳
- 日期：2026-10-12
- 相关：[docs/product.md](../product.md)「技术基线」

## 背景

Idset 管理一台机器上的多个 Git 和 SSH 身份。这件事在 macOS、Windows、Linux 上一样存在：`~/.ssh/config`、`~/.gitconfig`、`includeIf "gitdir:"` 和 `ssh` 的行为三个平台基本一致。需要一种语言、可复现的构建、真正的原生窗口，三个平台共用一套代码。参考 [MyGo 原生 UI 文档](https://mygo.egoist.dev/docs/ui) 和[固定版本的源码](https://github.com/egoist/mygo/tree/v0.3.5)。

## 决定

用 MyGo 原生 UI，一个 Go 模块，命令行工具在 `go.mod` 里用 `tool` 固定版本。常驻在菜单栏（macOS）或通知区域（Windows、Linux）的托盘图标里。更新用原生更新插件。

## 代价

控件是 GPU 绘制的，无障碍和各平台表现要在真机上测。托盘图标不接受拖放，拖放只能放在窗口里（见 [ADR 0003](0003-drop-zone-in-tray-window.md)）。Linux 的托盘依赖 AppIndicator，没有这个库就只能开普通窗口。MyGo 版本还早，每次升级要看 API 和打包的变化。

## 什么时候重新考虑

某个必须的平台能力或无障碍要求，原生 UI 做不到。
