# Design tokens

The values Idset's UI uses, in one place. Spacing and type are units of the MyGo theme (`c.Theme()`), so the code
reads the theme and never copies the numbers. Color is Idset's own glass theme: a `ui.Theme` built once from the
values below and given to every window, together with each window's material. Identity colors and corner radii of
cards are Idset's own as well. The rules behind them are in [docs/design.md](../docs/design.md).

## Spacing and size

The unit is `Theme.Spacing`: 4 DIPs by default (3 compact, 5 roomy). In code, `t.Space(n)` is `n` units and
`t.Rem(n)` is `n` times the theme font size (13 on macOS, 14 elsewhere, scaled by the desktop's text size).
Mockups show the 4 DIP unit and 13 px text.

| Token | Code | Value at defaults (macOS / elsewhere) | Used for |
|---|---|---|---|
| space-0.25 | `t.Space(0.25)` | 1 | between a name and its second line |
| space-0.5 | `t.Space(0.5)` | 2 | group heading inset; lines of the read-only reason |
| space-1 | `t.Space(1)` | 4 | context line of a result card; rows of its details |
| space-1.5 | `t.Space(1.5)` | 6 | platform filter buttons; the arrow to the new path in the stale-path prompt; the empty state's lines |
| space-2 | `t.Space(2)` | 8 | related controls, button rows, the context line's padding |
| space-2 (row) | `t.Space(2)` | 8 | vertical padding of a list row (rows are at least 48 high) |
| space-2.5 | `t.Space(2.5)` | 10 | badge to text in a list row; between blocks of the panel; lines of the read-only state |
| space-3 | `t.Space(3)` | 12 | panel content padding at the sides and below; card padding; list row left inset |
| space-4 | `t.Space(4)` | 16 | title row, left inset |
| symbol-button | `t.Space(6.5)` | 26 | borderless icon buttons: in the title row and the footer, in the context line of a card (open, copy, menu, clear), at the end of an overlap row (pin, minus); always 26 x 26 with a 16 glyph, 4 apart |
| chrome-title | `t.Space(11)` | 44 | title row of the panel, no hairline; the content starts 4 below it |
| chrome-footer | `t.Space(10)` | 40 | footer of the panel, a hairline above it |
| context-line | `t.Space(9)` | 36 | context line of a result card, its hairline included |
| chrome-inset | `t.Space(2)` / `t.Space(2.25)` | 8 / 9 | the footer's side insets / the title row's right inset |
| chrome-gap | `t.Space(1)` | 4 | between the footer's icon buttons |
| menu-entry | `t.Space(8)` | 32 | the stacked actions of the overlap card |
| bar-undo | `t.Space(9)` | 36 | undo bar of a result card; the stale-path prompt's Update and Ignore row |
| bar-actions | `t.Space(10)` | 40 | the stale-path prompt's header; the "New Identity" row of the bind list |
| path-row | `t.Space(7)` | 28 | the two path rows of the stale-path prompt; a finding row of the identity check; the HTTPS row (8 right inset, its icon button 24) |
| clone-input | `t.Space(7)` | 28 | the clone helper's input, filled with the `Background` role; its copy button is a 24 symbol-button; padding `Space(2)` x `Space(3)`, bottom `Space(2.5)`; `Space(1)` between input and output |
| clone-output | `t.Space(4)` | 16 | the clone helper's output or hint line (`small`, monospace for the command) |
| row-disclosure | `t.Space(6)` | 24 | the identity row's disclosure button (a 24 gap without an alias); the open helper is indented `Space(8)` (32) |
| finding-section | `t.Space(1)` | 4 | above and below the finding rows, under their hairline |
| row-bind | `t.Space(11)` | 44 | bind row, overlap row |
| button-confirm | `t.Space(7)` high, `t.Space(20)` wide | 28 x 80 | Cancel and Bind of the confirm block; both the same size |
| button-readonly | `t.Space(7)` high, `t.Space(30)` wide | 28 x 120 | Show in Finder and Read Again of the read-only state; both the same size |
| countdown | `t.Space(0.5)` | 2 | undo bar's countdown line on its bottom edge, `Accent` at 45%, in fifths of the card's width |
| badge-result | `t.Space(9)` | 36 | initial badge in a result card |
| badge-list | `t.Space(7)` | 28 | initial badge in a list row; the row hairline starts 50 from the card's edge (12 + 28 + 10) |
| badge-warning | `t.Space(11)` | 44 | warning badge of the read-only state |
| button-add | `t.Space(7)` high, 16 padding | 28 | the filled Add Identity button of the empty state |
| detail-label | `t.Space(14)` | 56 | label column of the result card's details |
| detail-inset | `t.Space(14.5)` | 58 | details lined up with the identity name (12 + 36 + 10) |
| radius | `Theme.Radius` | 6 | controls and symbol buttons |
| radius-card | 10 | 10 | result cards, the identity list, the drop zone, settings groups |
| radius-window | system | 12 | the windows; the system draws the corners |
| radius-hero | 14 | 14 | the empty state's key tile, 56 x 56 |
| radius-search | 8 | 8 | the panel's filter field, 28 high, `Surface` fill, no outline |
| radius-dialog | system | 12 | dialog windows, whose corners the system draws; their buttons share the width, 28 high, radius 8 |
| radius-badge | 8 / 10 / 11 | 8 / 10 / 11 | list badge, result badge, warning badge |
| radius-filter | 12 | 12 | platform filter buttons |
| outline-drop | 1.5, dashed | 1.5 | drop zone outline, `Accent` at 55% over an 8% tint; the drop zone is 48 high: a 20 folder glyph, the line (`secondary`) and a 24 high primary Choose button |
| title | `t.Rem(1.15)`, 600 | about 15 / 16 | panel title "Idset", a level 1 heading; the identity name in a result card and in the editor; pane, read-only and empty titles |
| emphasis | `t.Rem(1)`, 600 | 13 / 14 | identity names in a list, dialog titles, the first line of a notice |
| hairline | 0.5 | 0.5 (1 at 1x) | every divider and the outline of a field; cards have none |
| secondary | `t.Rem(0.92)` | about 12 / 13 | hints, notes, emails, status lines, the drop zone line |
| mono | `t.Rem(0.88)` / `t.Rem(0.85)`, monospace | about 11.5 / 11 | paths, aliases, commands and key names next to secondary text / next to small text |
| caption | `t.Rem(0.85)`, 600 | about 11 / 12 | group heading ("Identities · 3") |
| small | `t.Rem(0.85)` | about 11 / 12 | folders in a list row, the read-only reason, "Copied" |
| badge-letter | `t.Rem(1.25)`, 600 | 16 | initial in a result badge |
| icon | `t.Rem(1.2)` / `t.Rem(1.7)` | about 16 / 22 | glyphs of every icon button (cards, title bar, footer) / the read-only warning |
| body | theme `FontSize` | 13 / 14 | everything else, and all widget text |
| accounts-list | `t.Space(58)` | 232 | source list of the accounts window |
| accounts-row | `t.Space(8)` | 32 | height of a settings row |
| accounts-label | `t.Space(42)` | 168 | label column of a settings row |
| dot | `t.Space(2.5)` / `t.Space(3.5)` | 10 / 14 | color dot in a bind row / color choices |
| conn | `t.Space(1.75)` | 7 | connectivity dot of a list row |
| accounts-bar | `t.Space(11)` | 44 | height of the sidebar's add bar and of the detail's footer; their hairlines are one line |
| accounts-button | `t.Space(7)` high, `t.Space(25)` wide | 28 x 100 | footer buttons; Revert and Save share the width, Delete is a text button of the same height |
| accounts-dialog | `theme.DialogWidth` | 340 | width of a dialog window: the delete confirmation, the settings window's two confirmations and the key's identity choice; its height is its content's |
| settings-tab | `t.Space(22)` wide, `t.Space(13)` high | 88 x 52 | a tab of the settings window: an icon over a label, 4 apart; the bar pads 4 above, 6 below, 8 at the sides |
| settings-tab-icon | `t.Space(5)` | 20 | icon of a tab; the label is the small text size |
| settings-pane | `t.Space(140)` max width | 560 | content of a settings pane, centered, 16 above and 20 at the sides |
| settings-backup-row | `t.Space(11)` | 44 | height of a backup row; other settings rows use accounts-row (32) |
| panel | `Width`, `Height` | 340 x 480 | the panel window, fixed |
| accounts | `Width`, `Height` | 760 x 520 | the accounts window; resizable, with a minimum size |
| settings | `Width`, `Height` | 720 x 480 | the settings window; resizable, with a minimum size |
| tray-icon | 32 x 32 PNG | 16 x 16 points | drawn by `internal/trayicon/` |

## Material

Windows have no fill of their own where the system can draw a material behind them: MyGo's `WindowOptions.Vibrancy`
with a transparent page, and `c.Vibrancy()` reports whether it shows. Where it does not, the window draws the solid
fallback itself.

| Window | macOS | Windows 11 22H2+ | Linux, Windows 10, Reduce transparency |
|---|---|---|---|
| panel | `Popover` | `Acrylic` | solid |
| accounts | `Sidebar` | `Mica` | solid |
| settings, about | `Popover` | `Mica` | solid |
| dialogs | `Popover`, a modal child window | `Acrylic` | solid |
| card menu, select menus | the system's menu | the system's menu | the system's menu |

| Name | Light | Dark | Used for |
|---|---|---|---|
| material (mockups only) | `rgba(246, 246, 248, 0.62)` | `rgba(36, 36, 40, 0.58)` | how the material looks over a typical desktop; mockups blur 30 and saturate 180% under it |
| solid | `#f2f2f5` | `#2a2a2e` | the fallback window fill, and the color every contrast below is computed against |
| solid-detail | `#fbfbfd` | `#323236` | the accounts detail pane on the solid fallback |
| detail | `rgba(255, 255, 255, 0.45)` | `rgba(0, 0, 0, 0.18)` | the accounts detail pane over the material |
| card | `rgba(255, 255, 255, 0.2)` | `rgba(255, 255, 255, 0.08)` | result cards, the identity list, settings groups; no outline |

macOS turns the material opaque by itself when Reduce transparency is on; the theme does not change.

## Color roles

Idset's own `ui.Theme`, light and dark, given to every window. It does not follow the system accent color: Git
orange is the product's color, and every pair below was checked against the material, which a system accent was not.
It does follow Increase contrast (`Preferences().HighContrast`): `Border` and `TextMuted` move towards `Text` as
MyGo's default theme does, and the focus ring becomes opaque. Contrast is WCAG AA (4.5:1 for text, 3:1 for glyphs and
outlines) on the solid color and on a card over it.

| Role | Light | Dark | Used for |
|---|---|---|---|
| `Background` | transparent (material) / solid | transparent (material) / solid | the window |
| `Surface` | `rgba(0, 0, 0, 0.05)` | `rgba(255, 255, 255, 0.08)` | buttons, the filter field, segmented control track (`control` in mockups) |
| `SurfaceHover` | `rgba(0, 0, 0, 0.07)` | `rgba(255, 255, 255, 0.11)` | hovered control, symbol button, list row |
| `SurfacePressed` | `rgba(0, 0, 0, 0.12)` | `rgba(255, 255, 255, 0.16)` | pressed control; the row being confirmed in a result card |
| field | `rgba(255, 255, 255, 0.22)` | `rgba(0, 0, 0, 0.22)` | text inputs, with a hairline `Border`; light enough that the material shows through and a field does not read as a white box |
| `Border` | `rgba(60, 60, 67, 0.18)` | `rgba(255, 255, 255, 0.12)` | hairlines (`sep` in mockups) |
| `Text` | `#1d1d1f` | `#f5f5f7` | primary text |
| `TextMuted` | `#6b6b72` (4.7:1) | `#a3a3ab` (5.6:1) | hints, paths, group headings, symbols, the glyphs of the title row and footer |
| `Accent` | `#c63a1c` | `#ff6a4a` | primary buttons, the selected list row and menu entry, switch on, radio and checkbox, text buttons and links, the drop zone, the countdown line (at 45%), the overlap winner |
| `AccentHover` | `#ad3218` | `#ff8468` | hovered primary button |
| `AccentPressed` | `#962a14` | `#ff9a83` | pressed primary button |
| `AccentText` | `#ffffff` (5.2:1) | `#1d1d1f` (5.9:1) | text on the accent; dark mode uses dark text, as MyGo does for any light accent |
| accent-soft | `rgba(240, 80, 51, 0.16)` | `rgba(255, 106, 74, 0.22)` | Git orange `#f05033` as a tint: the selected settings tab, the drop zone while dragging over, the empty state's key tile |
| `Danger` | `#c8102e` | `#ff5c74` | `git` failed card; read-only warning (fill at 14%); save and field errors, Delete; error glyph of a finding, the 1 DIP inner outline of a card with an error |
| `Warning` | `#b86e00` (3.6:1) | `#ffb340` | never text, only glyphs and outlines: inner outline of cards without a match; warning glyph of a finding |
| `Success` | `#1e7d36` | `#3ccf62` | connectivity dot of a passing check; check mark of "Configuration OK" |
| `Selection` | `rgba(240, 80, 51, 0.25)` | `rgba(255, 106, 74, 0.35)` | selected text |
| `Focus` | `rgba(198, 58, 28, 0.55)` | `rgba(255, 132, 104, 0.6)` | keyboard focus ring, 3 DIPs; a focused input's outline turns `Accent` |
| `Inverse` / `InverseText` | `#1d1d1f` / `#f5f5f7` | `#f5f5f7` / `#1d1d1f` | tooltips (every icon button has one) and toasts |
| `Scrollbar` | `rgba(0, 0, 0, 0.32)` | `rgba(255, 255, 255, 0.35)` | scroll bar thumbs, 6 DIPs wide |

`#f05033` itself is never a fill under text: white on it is 3.6:1.

## Identity colors

Data, not theme roles; used on initial badges and as dots, always next to the identity's name; the color choices of
the accounts window are named.

| Preset | Value |
|---|---|
| blue | `#3A7BD5` |
| green | `#2E9E5B` |
| orange | `#E07A1F` |
| red | `#D64545` |
| purple | `#8E44AD` |
| yellow | `#D4A017` |

A badge is the value itself, solid, in both appearances. Its letter is `#1d1d1f` when the value is light
(`0.299 R + 0.587 G + 0.114 B > 165`) and white otherwise. On a selected (`Accent`) row the badge gets a 1.5 DIP
`AccentText` ring so an orange or red identity stays apart from the selection. Any other `#RRGGBB` follows the same
rule; an invalid value shows as blue (`internal/account/display.go`).

## Tray icon

| Variant | Ink |
|---|---|
| macOS (template) | black on transparent; the system tints it |
| Windows, Linux | `#8a8a8a` on transparent, visible on light and dark taskbars |
| Warning badge | triangle, `#d97706` with a white "!" (template: ink with the mark cut out) |
| Error badge | circle, `#dc2626` with a white "!" (template: ink with the mark cut out) |
| Badge gap | 1.5 px cleared around the badge |

## Motion

| Token | Value |
|---|---|
| default | none: state changes take effect immediately |
| copied (path, command, clone command), written, undone and updated acknowledgement | 1.2 s, then gone at once (`c.After`) |
| undo countdown | 5 s from an absolute deadline; the line shortens in fifths, once a second (`c.After`), no animation |
| element transition | 200 ms, ease-out (MyGo default); only as an agreed exception, see docs/design.md |
| reduced motion | jump to the target (`Preferences().ReduceMotion`) |

## Keeping this file true

Change a token here, in `docs/design.md` and in the code together. Mockups in `screens/` copy the color values above
as CSS variables (`--material`, `--card`, `--accent` and so on), simulate the material with a blur over a drawn
desktop, and show the solid fallback with `?material=solid`. When the MyGo version in `go.mod` changes, compare the
theme roles and the materials with its `ui/theme.go` and `WindowOptions`.
