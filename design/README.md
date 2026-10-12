# 画板

`screens/<界面>.html` 是每个界面的画板，用浏览器打开。文件名和 `internal/ui/<界面>/` 对应。画板是近似效果，以真机运行的效果为准。

## 台账

| 界面 | 规格 | 画板 | 代码 | 状态 |
|---|---|---|---|---|
| Example | `docs/product.md` 当前界面 | `screens/example.html` | `internal/ui/example/` | 模板示例，产品第一个界面落地时删掉 |
| 面板 | `docs/product.md`「托盘和面板」「拖入文件夹」 | 待画 | `internal/ui/panel/` | 计划中 |
| 账户窗口 | `docs/product.md`「账户窗口」 | 待画 | `internal/ui/accounts/` | 计划中 |
| 设置窗口 | `docs/product.md`「设置窗口」「密钥」「导入」 | 待画 | `internal/ui/settings/` | 计划中 |
| 关于窗口 | `docs/product.md`「关于窗口」 | 待画 | `internal/ui/about/` | 计划中 |

状态用：计划中、待评审、已定稿、已验收。定稿和验收由维护者在 PR 里确认。

## 查看状态

`example.html` 支持 `?state=zero|positive|negative` 和 `?scheme=light|dark`。新画板也用 URL 参数切状态和深浅色，每个状态都要能看到。

## 维护

改界面时，规格、画板、代码放在同一个 PR。

仓库里不存截图，`design/` 只放 HTML 画板和这份台账。验收用的截图贴在 PR 正文里，由 GitHub 托管，不进 git 历史。`.gitignore` 已忽略 `design/screenshots/`。
