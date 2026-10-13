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
| space-1.5 | `t.Space(1.5)` | 6 | lines of the drop zone and the empty state; platform filter buttons |
| space-2 | `t.Space(2)` | 8 | related controls, button rows, the context line's padding |
| space-2.25 | `t.Space(2.25)` | 9 | vertical padding of a list row |
| space-2.5 | `t.Space(2.5)` | 10 | badge to text in a list row; lines of the read-only state |
| space-3 | `t.Space(3)` | 12 | between blocks of the panel; card padding; title bar and footer, vertical |
| space-4 | `t.Space(4)` | 16 | panel content padding; title bar and footer, horizontal |
| space-5 | `t.Space(5)` | 20 | vertical padding of the idle drop zone |
| symbol-button | `t.Space(6)` | 24 | borderless symbol buttons in cards |
| badge-row | `t.Space(7)` | 28 | initial badge in a list row |
| badge-result | `t.Space(9)` | 36 | initial badge in a result card |
| badge-warning | `t.Space(11)` | 44 | warning badge of the read-only state |
| detail-label | `t.Space(14)` | 56 | label column of the result card's details |
| detail-inset | `t.Space(15)` | 60 | details lined up with the identity name |
| radius | `Theme.Radius` | 6 | controls, symbol buttons, the read-only reason box |
| radius-card | 10 | 10 | result cards, the identity list, the drop zone |
| radius-badge | 7 / 9 / 11 | 7 / 9 / 11 | row badge, result badge, warning badge |
| radius-filter | 12 | 12 | platform filter buttons |
| outline-drop | 1.5, dashed | 1.5 | drop zone outline |
| title | `t.Rem(1.15)` bold | about 15 / 16 | panel title, a level 1 heading |
| emphasis | `t.Rem(1.15)`, 600 | about 15 / 16 | identity name in a result card; read-only title |
| secondary | `t.Rem(0.9)` | about 12 / 13 | hints, group heading, emails, paths in a result card |
| small | `t.Rem(0.85)` | about 11 / 12 | folders in a list row, the read-only reason, "Copied" |
| badge-letter | `t.Rem(1)` / `t.Rem(1.25)` bold | 13 / 16 | initial in a row badge / result badge |
| icon | `t.Rem(1.1)` / `t.Rem(1.7)` | about 14 / 22 | symbols in cards / the read-only warning |
| body | theme `FontSize` | 13 / 14 | everything else, and all widget text |
| accounts-list | `t.Space(58)` | 232 | source list of the accounts window |
| accounts-row | `t.Space(8)` | 32 | height of a settings row |
| accounts-label | `t.Space(42)` | 168 | label column of a settings row |
| dot | `t.Space(2.5)` / `t.Space(3)` / `t.Space(3.5)` | 10 / 12 / 14 | color dot in the list / header / color choices |
| accounts-dialog | `t.Space(90)` | 360 | width of the delete confirmation |
| panel | `Width`, `Height` | 340 x 480 | the panel window, fixed |
| accounts | `Width`, `Height` | 760 x 520 | the accounts window; resizable, with a minimum size |
| tray-icon | 32 x 32 PNG | 16 x 16 points | drawn by `internal/trayicon/` |

## Color roles

Values are the defaults of `ui.LightTheme()` and `ui.DarkTheme()` in MyGo v0.3.6. The app also follows the user's
accent color and contrast settings, so mockups approximate it.

| Role | Light | Dark | Used for |
|---|---|---|---|
| `Background` | `#ffffff` | `#18181b` | panel fill, read-only reason box |
| `Surface` | `#f4f4f5` | `#27272a` | cards, the identity list, inputs, buttons |
| `SurfaceHover` | `#e9e9ec` | `#323236` | hovered control, hovered symbol button |
| `SurfacePressed` | `#dddde1` | `#3c3c41` | pressed control; selected list row; segmented control track |
| `Border` | `#d9d9de` | `#3f3f46` | dividers, outlines, the drop zone's dashed outline |
| `Text` | `#18181b` | `#f4f4f5` | primary text |
| `TextMuted` | `#71717a` | `#a1a1aa` | hints, paths, group heading, symbols |
| `Accent` | `#2563eb` | `#3b82f6` | drop zone outline while dragging over; selected platform filter, switch on, radio and checkbox, text buttons, Save |
| `AccentHover` | `#1d4ed8` | `#60a5fa` | hovered primary button |
| `AccentPressed` | `#1e40af` | `#2563eb` | pressed primary button |
| `AccentText` | `#ffffff` | `#ffffff` | text on the accent |
| `Danger` | `#dc2626` | `#ef4444` | `git` failed card; read-only warning (fill at 14%); save error, field error, the Delete text button and the confirming Delete button |
| `Warning` | `#d97706` | `#f59e0b` | outline of cards without a match |
| `Success` | `#16a34a` | `#22c55e` | not used yet |
| `Selection` | `rgba(37, 99, 235, 0.25)` | `rgba(59, 130, 246, 0.4)` | drop zone fill while dragging over; selected text |
| `Focus` | `rgba(37, 99, 235, 0.55)` | `rgba(96, 165, 250, 0.6)` | keyboard focus ring |
| `Inverse` / `InverseText` | zero: `Text` / `Background` | zero: `Text` / `Background` | tooltips and toasts |
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

## Motion

| Token | Value |
|---|---|
| default | none: state changes take effect immediately |
| copied and written acknowledgement | 1.2 s, then gone at once (`c.After`) |
| element transition | 200 ms, ease-out (MyGo default); only as an agreed exception, see docs/design.md |
| reduced motion | jump to the target (`Preferences().ReduceMotion`) |

## Keeping this file true

Change a token here, in `docs/design.md` and in the code together. Mockups in `screens/` copy the color values above
as CSS variables. When the MyGo version in `go.mod` changes, compare the color values with its `ui/theme.go`.
