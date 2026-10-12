# 画板

`screens/<界面>.html` 是每个界面的画板，用浏览器打开。文件名和 `internal/ui/<界面>/` 对应。画板是近似效果，以真机运行的效果为准。

## 台账

| 界面 | 规格 | 画板 | 代码 | 状态 |
|---|---|---|---|---|
| 面板 | `docs/product.md`「托盘和面板」「拖入文件夹」 | `screens/panel.html` | `internal/ui/panel/` | 已定稿 |
| 账户窗口 | `docs/product.md`「账户窗口」 | 待画 | `internal/ui/accounts/` | 计划中 |
| 设置窗口 | `docs/product.md`「设置窗口」「密钥」「导入」 | 待画 | `internal/ui/settings/` | 计划中 |
| 关于窗口 | `docs/product.md`「关于窗口」 | 待画 | `internal/ui/about/` | 计划中 |

状态用：计划中、待评审、已定稿、已验收。定稿和验收由维护者在 PR 里确认。

## 查看状态

画板左侧有一排切换按钮：状态、深浅色和语言，点击直接切换，左右方向键依次切换状态。地址栏同步为 `?state=…&scheme=light|dark&lang=en|zh`，可以把某个状态的链接发给别人。新画板也这样做，每个状态都要能看到。

## 维护

改界面时，规格、画板、代码放在同一个 PR。

仓库里不存截图，`design/` 只放 HTML 画板和这份台账。验收用的截图贴在 PR 正文里，由 GitHub 托管，不进 git 历史。`.gitignore` 已忽略 `design/screenshots/`。
