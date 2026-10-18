# Design tokens

The values Idset's UI uses, in one place. Spacing, type and color are not new tokens: each is a role or a unit of the
MyGo theme (`c.Theme()`), so the code reads the theme and never copies the numbers. Identity colors and corner radii
of cards are Idset's own and are listed as such. The rules behind them are in [docs/design.md](../docs/design.md).

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
| space-2.25 | `t.Space(2.25)` | 9 | vertical padding of a list row |
| space-2.5 | `t.Space(2.5)` | 10 | badge to text in a list row; lines of the read-only state |
| space-3 | `t.Space(3)` | 12 | between blocks of the panel; card padding |
| space-4 | `t.Space(4)` | 16 | panel content padding; title bar, left inset |
| symbol-button | `t.Space(6)` | 24 | borderless icon buttons: in the context line of a card (open, copy, menu, clear), at the end of an overlap row (pin, minus), the title bar and the footer; always 24 x 24 with a 16 glyph, 4 apart |
| chrome-bar | `t.Space(9)` | 36 | title bar and footer of the panel and the context line of a result card, hairline included (they were 42, 56 and 41 before) |
| chrome-inset | `t.Space(2)` | 8 | the title bar's right inset and the footer's side insets, so the 24 DIP buttons sit 8 from the edge |
| chrome-gap | `t.Space(1)` | 4 | between the footer's icon buttons |
| menu-entry | `t.Space(8)` | 32 | entry of the rebind menu; the stacked actions of the overlap card |
| bar-undo | `t.Space(9)` | 36 | undo bar of a result card; the stale-path prompt's Update and Ignore row |
| bar-actions | `t.Space(10)` | 40 | the stale-path prompt's header; the "New Identity" row of the bind list |
| path-row | `t.Space(7)` | 28 | the two path rows of the stale-path prompt; a finding row of the identity check; the HTTPS row (8 right inset, its icon button 24) |
| clone-input | `t.Space(7)` | 28 | the clone helper's input, filled with the `Background` role; its copy button is a 24 symbol-button; padding `Space(2)` x `Space(3)`, bottom `Space(2.5)`; `Space(1)` between input and output |
| clone-output | `t.Space(4)` | 16 | the clone helper's output or hint line (`small`, monospace for the command) |
| row-disclosure | `t.Space(6)` | 24 | the identity row's disclosure button (a 24 gap without an alias); the open helper is indented `Space(8)` (32) |
| finding-section | `t.Space(1)` | 4 | above and below the finding rows, under their hairline |
| row-bind | `t.Space(11)` | 44 | bind row, overlap row |
| button-confirm | `t.Space(7)` high, `t.Space(20)` wide | 28 x 80 | Cancel and Bind of the confirm block; both the same size |
| menu-width | `t.Space(56)` | 224 | the card menu ("Rebind to", identities, "Unbind"); its heading is 28 high |
| button-readonly | `t.Space(7)` high, `t.Space(30)` wide | 28 x 120 | Show in Finder and Read Again of the read-only state; both the same size |
| countdown | `t.Space(0.5)` | 2 | undo bar's countdown line on its bottom edge, `Accent` at 45%, in fifths of the card's width |
| badge-result | `t.Space(9)` | 36 | initial badge in a result card |
| badge-warning | `t.Space(11)` | 44 | warning badge of the read-only state |
| button-add | `t.Space(7)` high, 16 padding | 28 | the filled Add Identity button of the empty state |
| detail-label | `t.Space(14)` | 56 | label column of the result card's details |
| detail-inset | `t.Space(15)` | 60 | details lined up with the identity name |
| radius | `Theme.Radius` | 6 | controls, symbol buttons, the read-only reason box |
| radius-card | 10 | 10 | result cards, the identity list, the drop zone |
| radius-menu | 8 | 8 | the card menu |
| radius-badge | 9 / 11 | 9 / 11 | result badge, warning badge |
| radius-filter | 12 | 12 | platform filter buttons |
| outline-drop | 1.5, dashed | 1.5 | drop zone outline; the drop zone is one line, `Space(2)` x `Space(3)` padding (about 46 high) |
| title | `t.Rem(1)`, 600 | 13 / 14 | panel title "Idset", a level 1 heading |
| emphasis | `t.Rem(1.15)`, 600 | about 15 / 16 | identity name in a result card; read-only title |
| secondary | `t.Rem(0.9)` | about 12 / 13 | hints, group heading, emails, paths in a result card |
| small | `t.Rem(0.85)` | about 11 / 12 | folders in a list row, the read-only reason, "Copied" |
| badge-letter | `t.Rem(1.25)` bold | 16 | initial in a result badge |
| icon | `t.Rem(1.2)` / `t.Rem(1.7)` | about 16 / 22 | glyphs of every icon button (cards, title bar, footer) / the read-only warning |
| body | theme `FontSize` | 13 / 14 | everything else, and all widget text |
| accounts-list | `t.Space(58)` | 232 | source list of the accounts window |
| accounts-row | `t.Space(8)` | 32 | height of a settings row |
| accounts-label | `t.Space(42)` | 168 | label column of a settings row |
| dot | `t.Space(2.5)` / `t.Space(3)` / `t.Space(3.5)` | 10 / 12 / 14 | color dot in the panel and account lists / header / color choices |
| accounts-bar | `t.Space(11)` | 44 | height of the sidebar's add bar and of the detail's footer; their hairlines are one line |
| accounts-button | `t.Space(7)` high, `t.Space(25)` wide | 28 x 100 | footer buttons; Revert and Save share the width, Delete is a text button of the same height |
| accounts-dialog | `t.Space(90)` | 360 | width of the delete confirmation; the settings window's two confirmations reuse it |
| settings-tab | `t.Space(22)` wide, `t.Space(13)` high | 88 x 52 | a tab of the settings window: an icon over a label, 4 apart; the bar pads 4 above, 6 below, 8 at the sides |
| settings-tab-icon | `t.Space(5)` | 20 | icon of a tab; the label is the small text size |
| settings-pane | `t.Space(140)` max width | 560 | content of a settings pane, centered, 16 above and 20 at the sides |
| settings-backup-row | `t.Space(11)` | 44 | height of a backup row; other settings rows use accounts-row (32) |
| panel | `Width`, `Height` | 340 x 480 | the panel window, fixed |
| accounts | `Width`, `Height` | 760 x 520 | the accounts window; resizable, with a minimum size |
| settings | `Width`, `Height` | 720 x 480 | the settings window; resizable, with a minimum size |
| tray-icon | 32 x 32 PNG | 16 x 16 points | drawn by `internal/trayicon/` |

## Color roles

Values are the defaults of `ui.LightTheme()` and `ui.DarkTheme()` in MyGo v0.3.6. The app also follows the user's
accent color and contrast settings, so mockups approximate it.

| Role | Light | Dark | Used for |
|---|---|---|---|
| `Background` | `#ffffff` | `#18181b` | panel fill, read-only reason box |
| `Surface` | `#f4f4f5` | `#27272a` | cards, the identity list, inputs, buttons |
| `SurfaceHover` | `#e9e9ec` | `#323236` | hovered control, hovered symbol button and icon button |
| `SurfacePressed` | `#dddde1` | `#3c3c41` | pressed control and icon button; selected list row; segmented control track; the row being confirmed in a result card |
| `Border` | `#d9d9de` | `#3f3f46` | dividers, outlines, the drop zone's dashed outline |
| `Text` | `#18181b` | `#f4f4f5` | primary text |
| `TextMuted` | `#71717a` | `#a1a1aa` | hints, paths, group heading, symbols, the glyphs of the title bar and footer |
| `Accent` | `#2563eb` | `#3b82f6` | drop zone outline while dragging over; selected platform filter, switch on, radio and checkbox, text buttons, Save, the confirm block's Rebind, the countdown line (at 45%), the overlap winner |
| `AccentHover` | `#1d4ed8` | `#60a5fa` | hovered primary button |
| `AccentPressed` | `#1e40af` | `#2563eb` | pressed primary button |
| `AccentText` | `#ffffff` | `#ffffff` | text on the accent |
| `Danger` | `#dc2626` | `#ef4444` | `git` failed card; read-only warning (fill at 14%); save error, field error, the Delete text button and the confirming Delete button; error glyph of a finding, outline of a matched card with an error finding |
| `Warning` | `#d97706` | `#f59e0b` | outline of cards without a match; warning glyph of a finding, outline of a matched card whose worst finding is a warning |
| `Success` | `#16a34a` | `#22c55e` | connectivity dot of a passing check; check mark of "Configuration OK" |
| `Selection` | `rgba(37, 99, 235, 0.25)` | `rgba(59, 130, 246, 0.4)` | drop zone fill while dragging over; selected text |
| `Focus` | `rgba(37, 99, 235, 0.55)` | `rgba(96, 165, 250, 0.6)` | keyboard focus ring (also the clone helper's focused input, 2 DIPs) |
| `Inverse` / `InverseText` | zero: `Text` / `Background` | zero: `Text` / `Background` | tooltips (every icon button has one) and toasts |
| `Scrollbar` | `rgba(0, 0, 0, 0.32)` | `rgba(255, 255, 255, 0.35)` | scroll bar thumbs, 6 DIPs wide |

## Identity colors

Data, not theme roles; used on initial badges and as dots, always next to the identity's name; the color choices of the accounts window are named.

| Preset | Value | Badge fill (light / dark) | Letter (light / dark) |
|---|---|---|---|
| blue | `#3A7BD5` | value at 16% / 24% | value / value mixed 40% towards white |
| green | `#2E9E5B` | same rule | same rule |
| orange | `#E07A1F` | same rule | same rule |
| red | `#D64545` | same rule | same rule |
| purple | `#8E44AD` | same rule | same rule |
| yellow | `#D4A017` | same rule | same rule |

Any other `#RRGGBB` follows the same rule; an invalid value shows as blue (`internal/account/display.go`).

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
as CSS variables. When the MyGo version in `go.mod` changes, compare the color values with its `ui/theme.go`.
