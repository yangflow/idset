# Idset design language

This is the design language of Idset: the rules every screen follows, and the place to record where a screen
departs from them. Idset is a tray app that keeps a list of Git and SSH identities and generates the configuration
so every folder uses the right one (see [product.md](product.md), in Chinese). Its UI today is one window, the
panel that the tray icon opens: the identity list, and a drop zone that shows which identity Git uses in a folder.

The UI is built on the native toolkit of [MyGo](https://github.com/egoist/mygo). This document summarizes the parts
of its [UI guides](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/README.md) that matter for design; the guides
stay the authority on API details. Links point to MyGo v0.4.1, the version pinned in `go.mod`. When the pin moves,
check the pages again and update the links and this document together.

The panel (`internal/ui/panel/`) is the reference the values below call "the panel". The accounts, settings and
about windows follow it.

Related files: [design/README.md](../design/README.md) (mockups and how a UI change is made) and
[design/tokens.md](../design/tokens.md) (spacing, size and color values in one table).

## Contents

1. [Principles](#principles)
2. [Layout and spacing](#layout-and-spacing)
3. [Color and dark mode](#color-and-dark-mode)
4. [Typography](#typography)
5. [Icons and images](#icons-and-images)
6. [Component guide](#component-guide)
7. [The panel](#the-panel)
8. [The accounts window](#the-accounts-window)
9. [The settings window](#the-settings-window)
10. [The about window](#the-about-window)
11. [Windows, menus and the tray](#windows-menus-and-the-tray)
12. [Accessibility checklist](#accessibility-checklist)
13. [Motion](#motion)
14. [Localization](#localization)
15. [Design acceptance checklist](#design-acceptance-checklist)
16. [References](#references)

## Principles

- **Native first.** Use MyGo's native widgets as they are, on the system's material, and follow the system's light
  and dark appearance. A control that looks and behaves like the platform's is learned already. Style
  an element yourself only when no widget does the job (the drop zone, the initial badge, the result card), and
  build it from containers and widgets, or on the
  [custom widget](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/custom-widgets.md) bases, so focus, keyboard
  and accessibility stay intact. No WebView and no HTML frontend.
- **Consistent across platforms, respectful of each.** One view serves macOS, Windows and Linux. Follow each
  platform's conventions where MyGo exposes them: the menu bar or notification area for the tray, a template icon on
  macOS, menu roles, the system material, the system font, the name of the platform's file manager on buttons. Do not
  add chrome that imitates one platform on the others.
- **Answer first.** Idset is glanced at, not lived in. The panel answers one question, which identity Git uses in
  a folder, in one sentence and one badge; the identity list is reference below it. Controls stay quiet until they
  are needed.
- **Accessible by default.** Every element can be reached with the keyboard, named for a screen reader, read in both
  appearances, and still works at larger text sizes. See the [checklist](#accessibility-checklist).
- **No custom motion.** State changes take effect at once. See [Motion](#motion).
- **Meaning is never carried by color alone.** Say it in words; pair color with text, a symbol or a shape. Every
  identity's color badge sits next to its name, and the state of a result card is written as a sentence.
- **The UI thread draws.** A frame is built from in-memory state. No disk, network or subprocess work in a view,
  including reading `accounts.json` and running `git`; start that work in the background and change the state back
  on the UI thread (see [Views](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/views.md)).
- **Spec before code.** Every screen is specified in its mockup in `design/` and in this document before it is
  written. Only `design/` and this document count as the design reference for Idset.

## Layout and spacing

Layout uses flexbox in device-independent pixels (DIPs); see
[Layout](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/layout.md).

- **One spacing unit.** The theme's `Spacing` (4 DIPs by default; 3 is compact, 5 roomy) is the unit of every widget's
  padding and gap. Size your own elements with `t.Space(n)` so they follow it. Do not write raw pixel numbers for
  spacing. Fractions are allowed where an existing layout needs them (`t.Space(2.25)` is 9 DIPs).
- **Scale.** Prefer whole multiples of the unit: 1 (4) inside a control row, 2 (8) between related controls and
  between buttons, 3 (12) between blocks of the panel, 4 (16) for the panel's outer padding. Larger gaps need a
  reason, for example the idle drop zone.
- **Structure from containers.** `Column` and `Row` with `Gap` and `Padding`; `Grow(1)` and `Spacer` for free space;
  `Grid` for forms and aligned tables of controls. Prefer `Gap` to margins on individual children.
- **Sizes are bounds.** Let content reflow; do not fix a width a translation can overflow. Use `MinWidth`, `MaxWidth`
  and `Grow` instead of fixed widths, except for square symbol buttons, badges and the label column of the result
  card.
- **Alignment.** Left-align text and labels. Center only isolated focal content: the idle drop zone, the loading,
  empty and read-only states.
- **Do not truncate.** Paths, names and emails wrap rather than being cut; a path that is cut can point somewhere
  else.
- **Scrolling.** Content that can exceed the window goes in a
  [scroll view](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/scroll.md). The panel scrolls when it shows the
  identity list. The loading, read-only and no-identity states never need to scroll and are not in a scroll view, so
  their content can be centered vertically (centering only works in a container of fixed height).
- **The panel.** Fixed 340×480. The title row is `Space(11)` (44) high with no divider, the footer `Space(10)` (40)
  with a hairline above it; the content is the point, and the chrome takes 84 DIPs of the 480. Content padding
  `Space(1)` (4) on top and `Space(3)` (12) at the sides and below, gap between blocks `Space(2.5)` (10). Cards, the
  identity list and the drop zone have a 10 DIP corner radius. The full measurements are in [The panel](#the-panel) and
  [tokens.md](../design/tokens.md).

## Color and dark mode

Idset draws on the system's material and colors it with its own glass theme in Git orange. Every window sets the
theme as a copy of the default (`t := *c.Theme()`, then its colors, `c.SetTheme(&t)`), so spacing, font size and text
scale still follow the desktop; the values are in [tokens.md](../design/tokens.md). See
[Styling and themes](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/styling.md) and
[Vibrancy](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/windows.md#vibrancy).

- **Material.** Each window asks for a material with `WindowOptions.Vibrancy` and leaves its root transparent when
  `c.Vibrancy()` is true: the panel `Popover` on macOS and `Acrylic` on Windows 11, the accounts window `Sidebar` and the
  settings and about windows `Popover` on macOS (`Window` and `UnderWindow` are nearly opaque there, and the windows
  looked white), and `Mica` on Windows 11. Where no material shows (Linux, Windows 10) the
  window fills itself with the solid color, `#f2f2f5` light and `#2a2a2e` dark; macOS turns the material opaque by
  itself under Reduce transparency. Every contrast is computed against the solid color, which is what the material
  looks like at its most opaque, and checked on a card over it.
- **Fills, not outlines.** Content sits on the material as translucent fills: cards, the identity list and settings
  groups use the card fill with no outline; buttons and the filter field use `Surface`. Dividers are 0.5 DIP
  hairlines in `Border`. A card that needs attention adds a 1 DIP inner outline in `Warning` or `Danger`.
- **Always use theme roles, never literals.** Views read `c.Theme()`; the theme alone holds the values. The exceptions
  are identity colors, below, and the few fills the theme has no role for (card, field, accent-soft),
  which are named once beside the theme and switch with `c.Theme().Dark`.

  | Role | Use in Idset |
  |---|---|
  | `Background` | transparent over the material; the solid color where there is none |
  | `Surface` | buttons, the filter field, the segmented control track |
  | `SurfaceHover`, `SurfacePressed` | hovered and pressed controls and rows; the hover fill of borderless symbol buttons |
  | `Border` | hairlines; the outline of a text field |
  | `Text`, `TextMuted` | primary text; secondary text, hints, group headings, paths, symbols |
  | `Accent`, `AccentText` | primary buttons (the drop zone's Choose, Save, Bind, Rebind), the selected list row and menu entry, text buttons and links, switch on, choices, the drop zone's outline (55%) and tint (8%) |
  | accent-soft | Git orange as a tint: the selected settings tab, the drop zone while files are dragged over it, the empty state's key tile |
  | `Warning` | the inner outline of the "no match", "not a Git repository" and "not a folder" cards; the warning glyph of a finding, and the matched card's outline when its worst finding is a warning; never text |
  | `Danger` | the outline and text of the card when `git` cannot run; the read-only warning, on a 14% `Danger` fill; the error glyph of a finding and the outline of a card with one; Delete |
  | `Success` | the connectivity dot of a passing check and the check mark of the "Configuration OK" line, always with text (the dot's tooltip and accessible name) |
  | `Focus` | the keyboard focus ring, 3 DIPs |
  | `Inverse`, `InverseText` | tooltips and toasts |
  | `Scrollbar` | scroll bar thumbs |

- **Git orange.** The brand color is Git's `#f05033`. It is a tint, never a fill under text: white on it is 3.6:1.
  `Accent` is a darker `#c63a1c` in light (white text 5.2:1) and a lighter `#ff6a4a` in dark, where `AccentText` is
  near black (5.9:1), as MyGo itself does for a light accent. Idset does not follow the system accent color: the
  orange is the product's, and only its values were checked against the material.
- **Identity colors.** An identity's color is data the user chose, not a theme role. The six preset names map to
  fixed values (blue `#3A7BD5`, green `#2E9E5B`, orange `#E07A1F`, red `#D64545`, purple `#8E44AD`, yellow
  `#D4A017`); any other value is its own `#RRGGBB`. The initial badge is the color itself, solid, in a list row, a
  result card and the accounts window; the letter is near black on a light color and white otherwise. On a selected
  row the badge has a 1.5 DIP `AccentText` ring. Dots show the color in a bind row and the color choices (named,
  the chosen one ticked).
- **Appearance.** Do not implement your own dark mode switch; build the light or dark glass theme from
  `c.Theme().Dark`, which follows the system.
- **Selection.** A selected list row (the accounts window) is filled with `Accent`, its text in `AccentText`, inset
  8 DIPs from the list's sides with radius 6, like the system's source lists.
- **Accent.** One primary action per group uses `PrimaryButton`; the panel's is the drop zone's Choose.
- **Contrast.** Text meets WCAG 2.1 AA, 4.5:1, and glyphs, outlines and control boundaries 3:1, on the solid color
  and on a card over it. Under the desktop's high contrast setting (`c.Preferences().HighContrast`) the theme does
  what MyGo's default does: `Border` and `TextMuted` move towards `Text`, and the focus ring is opaque.
- **Status.** Color supplements a message; the message states the status in words. A warning or danger outline on a
  result card always comes with the sentence that explains it.
- **Mockups.** Mockups in `design/screens/` simulate the material with a blur over a drawn desktop and show the solid
  fallback with `?material=solid`; the app running on a real device is what counts.

## Typography

- **System font.** Leave `Font` empty so text uses the platform's UI font. The default text size is 13 on macOS and
  14 elsewhere, and scales with the desktop's text size (`Preferences().TextScale`) when you rely on the theme. See
  [Text](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/text.md). Paths, SSH aliases and folders use
  `Font("monospace")`.
- **Size relative to the theme.** Use `t.Rem(n)` for sizes (`n` times the theme font size) rather than fixed numbers,
  so custom text follows the user's text scale. Widgets use the theme size.
- **Five styles.** At the macOS default of 13, every screen uses these and nothing else:

  | Style | Size | Weight | Used for |
  |---|---|---|---|
  | title | `Rem(1.15)`, about 15 | 600 | the panel title "Idset", the identity name in a result card and in the accounts editor, a settings pane's heading, the read-only and empty titles |
  | emphasis | `Rem(1)`, 13 | 600 | identity names in a list, dialog titles, the first line of a notice |
  | body | `Rem(1)`, 13 | 400 | everything else, and all widget text |
  | secondary | `Rem(0.92)`, about 12 | 400, `TextMuted` | hints, notes, emails, status lines, the drop zone line |
  | caption | `Rem(0.85)`, about 11 | 600 for group headings, 400 otherwise, `TextMuted` | group headings ("Identities · 4"), window titles, tab labels, "Copied" |

  Buttons use body or secondary size; a primary or destructive button is 500, a text button 500, others 400. The about
  window's product name, 18 and 600, is the only other size. Nothing is 700.
- **Monospace.** Paths, aliases, folders, commands and key file names use `Font("monospace")` (SF Mono, Consolas,
  DejaVu Sans Mono), one step smaller than the text around them, since monospace reads larger: about 11.5 next to
  secondary text, 11 next to captions.
- **Chinese.** The platform font covers Chinese (PingFang SC, Microsoft YaHei UI, Noto Sans CJK); no size changes
  per language.
- **Hierarchy with few steps.** Weight and color before size; one larger size per screen. A card uses at most three text styles.
- **Headings are real headings.** Give the title of a screen or section `Role(ui.RoleHeading)` and a `Level` so
  assistive technology can navigate by them. In the panel the title is level 1; the "Identities · N" group heading, the
  read-only title and the empty title are level 2.
- **Do not truncate silently.** Prefer wrapping. Where a single line is needed (`SingleLine`, `MaxLines`), make the
  full text available in a tooltip or elsewhere on screen.
- **Numbers.** Counts that change in place can use `FontFeatures("tnum")`.

## Icons and images

- **Icons are SVG** shown with `ui.Icon`, which takes the text color and the font size: an icon follows theme colors
  and the text size automatically. MyGo has no built-in icon set and no SF Symbols: `ui.Icon` draws an SVG. The panel's symbols are 16×16 line
  drawings (1.5 stroke, round caps) defined in `internal/ui/panel/view.go` and parsed once at package level: clear,
  copy, check, warning, error (`exclamationmark.circle`: the info circle with its strokes turned upside down), folder,
  more (three dots), arrow, minus, pin, and for the title row and footer plus, gear (`gearshape`), info (`info.circle`), refresh
  (`arrow.clockwise`) and power (`power`), named after the SF Symbols they follow; there is no third-party icon set. If one
  is adopted, use one set for the whole app and record it and its license in `THIRD_PARTY.md`. See
  [Icon](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/icon.md).
- **An icon needs a name when it is alone.** Icons are decoration for assistive technology unless given a `Label`.
  A button that shows only an icon must get a readable name with `.Label()`; an icon next to a visible text label
  needs none.
- **Size icons with the text.** Card symbols are `Rem(1.1)` (about 14) in `TextMuted`; the glyphs of the title row and
  footer buttons are `Rem(1.2)` (about 16); the read-only warning is
  `Rem(1.7)` (about 22) in `Danger`.
- **The tray icon** is drawn in code by `internal/trayicon/`, so no image file is maintained by hand. It is the
  product icon's key, solid and pointing right: a rounded diamond bow with the Git branch cut out of it, the shaft
  and two bits. On macOS it is a 54×36 template icon, black on transparent, that the system tints for the menu bar
  and scales to 18 points tall; the bow's corners reach the top and the bottom, so it is as tall as the system's
  symbols. Elsewhere it is a 32×32 square with the key scaled to the width, in a mid gray that stays visible on
  light and dark taskbars. The worst entry of the diagnostics summary adds a badge at the lower right: a triangle
  for a warning, a circle for an error (and when ssh is missing), with a "!" in it, cut free of the key by a 1.5 px
  gap. Outside macOS the badge is `#d97706` or `#dc2626` with a white mark; in the template it is the ink with the
  mark cut out, since the system tints a template icon as a whole. The icon changes once a round of checks has
  finished, never while one runs, and does not flash.
- **The product icon** is an ivory key on a graphite tile: the bow is a rounded diamond carrying a Git branch in Git
  orange (`#f6643f` to `#c8361b`), whose tip is a glowing mint commit; the key is one piece, the bow joined to the
  shaft by fillets, and stands raised, with a dark edge below. It sits on the macOS grid (an 824 tile at 100 in
  1024, radius 185) with transparent corners, so every platform shows the same rounded tile. The source is
  [design/icon.svg](../design/icon.svg); see [design/README.md](../design/README.md) for exporting it.

## Component guide

Pick the widget by the job. Each name links to the upstream page, which also says what assistive technology sees of
the widget. The complete list is the
[UI index](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/README.md).

What Idset uses today:

| Component | Use |
|---|---|
| `ui.Button` | Secondary actions. A button that shows only a symbol must give a readable name with `.Label()`. |
| `ui.PrimaryButton` | The main action of a group of buttons; at most one. In the panel: "Add Identity" of the empty state, and "Rebind" of the confirm block (never on screen together). |
| `ui.Text` | Text. Secondary explanations use `TextMuted`. |
| `ui.TextInput` | The identity filter, shown from three identities on, with the placeholder "Filter identities": 28 DIPs high, `Surface` fill, radius 8, no outline. Not `ui.SearchField`: its placeholder ("Search") is fixed English. |
| `ui.Spinner` | Reading the configuration and checking a folder. |
| `ui.Divider` | A 0.5 DIP `Border` hairline: above the footer (the title row has none); between the two parts of a result card; between list rows. |
| Drop zone | A 48 DIP `ui.Row` with a 1.5 DIP dashed outline (`BorderStyle(ui.BorderDashed)`) in `Accent` at 55% over a fill of `Accent` at 8%, taking files with `DroppedFiles`; a 20 DIP folder glyph in `Accent`, the line, and a 24 DIP high `PrimaryButton` "Choose…" at the end. While `FileDragOver`, the outline is solid `Accent` and the fill accent-soft. See [Dropped files](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/input.md#dropped-files). |
| Initial badge | A rounded square with the first letter of the identity's name ("?" without a name), in the identity's color, solid (see [Color](#color-and-dark-mode)): 28 DIPs with radius 8 in a list row, 36 with radius 10 in a result card. |
| Borderless symbol button | `Space(6.5)` square (26), transparent, icon `Rem(1.1)` in `TextMuted`, `SurfaceHover` fill on hover, `SurfacePressed` while pressed. Must have a `.Label()`. After the path is copied, the icon turns into a check mark with "Copied" before it for 1.2 seconds. |
| Icon button (title row, footer) | The same 26×26 button with a `Rem(1.2)` (about 16) glyph, radius `Theme.Radius`. Always has `.Label()` and a `.Tooltip()` with the same text, in both languages: Settings, Add Identity, About Idset, Check Again, Quit (设置 / 添加身份 / 关于 Idset / 重新检查 / 退出 in Chinese). The tooltip is the `Inverse` fill, no shadow, above the button (below the title row's). Same size, 4 DIPs apart, so a missing button never moves another. |
| Result card | A context line (the path in monospace `TextMuted`, then the copy and clear buttons) and below it the answer. Card fill, no outline, radius 10; the cards without a match add a 1 DIP inner outline in `Warning`, the card for a failed `git` one in `Danger`; a matched card with findings takes the color of the worst one. |
| Bind row | In a result card: color dot (10 DIPs), name and one line of `TextMuted` (the identity's first folder in monospace `Rem(0.85)`, like the identity list's folders, middle-truncated; "Has <path> · adds another" is its tooltip), "Bind" in `Accent` at the end. 44 DIPs high, hairline above. The whole row is the target. |
| Card menu | The "⋯" icon button's menu in a matched card: the system's menu (`Element.Menu`), opened by the button on pointer down. A disabled "Rebind to" heading, the other identities by name, and under a separator "Unbind" (only when the identity lists this very folder). The system draws it on its own material, so it has the platform's look, keyboard and screen reader behavior; it cannot show color dots or folders. There is no actions row: open, copy, the menu and clear are icon buttons in the context line, in that order. |
| Confirm block | In a matched card, under the details, hairline above: one line ("<name> will apply only here") and two 80×28 buttons at the right, the plain one first ("Cancel"), then the filled one (`Accent`, "Rebind"). No extra identity row. |
| Overlap rows | In the overlap card: a row per identity in git's read order (dot, name, path in monospace `Rem(0.88)`), then the verdict: "Used" in `Accent` 600 with a check mark for the winner, "Overridden" in `TextMuted` for the others, whose names also lose their weight; and at the very end an icon button: a pin on the winner ("Use Only <name> Here"), a minus on each loser ("Remove <path> from <name>"). Title row: "<n> identities cover this folder" and an info icon button whose tooltip is "Git reads them in order; the last one wins." |
| Stale-path prompt | A card of its own above the no-match card: a 40 DIP header (color dot, "<name>'s folder is gone" / 「<name> 的路径已不在」, the name in 600, one line), two 28 DIP path rows under hairlines with no labels (the old path in `TextMuted`, then the new path after an arrow, monospace, one line, middle-truncated, full text in the tooltip), and a 36 DIP row with two text buttons at the right, "Update" in `Accent` and "Ignore" in `TextMuted`; their tooltips are "Change to This Path" and "Keep Old Path". The height is the same for any path. Card fill, no outline, no clear button. |
| Finding rows | In a matched card, under the details, in a section of their own: a hairline above, `Space(1)` (4) above and below the rows. One 28 DIP row per identity-check finding, email first, then key, like a path row: padding `Space(3)` (12), a `Rem(1.1)` (about 14) glyph, the warning triangle in `Warning` or the error circle in `Danger`, gap `Space(1.5)` (6), and one line of `Rem(0.92)` (about 12) `Text`: "Commits have no author", "Commits use <email> (<name>)", "Commits use <email>", "Pushes use <path>" (「提交没有作者」「提交会署名为 <邮箱>（<身份>）」「提交会署名为 <邮箱>」「推送会用 <路径>」). A long email gives way before its "@", so the domain and the identity name stay; a key path is middle-truncated in monospace like any path. The row's tooltip says why, with the full value: above the row, 6 DIPs away, its start under the row's start, at most 240 DIPs wide. No buttons: fixes belong to the diagnostics. The card's outline is `Danger` when any finding is an error, else `Warning`. |
| Connectivity dot | At the end of an identity row, after the folders: a 14 DIP slot holding a 6 DIP dot, `Success` when the last `ssh -T` passed, `Danger` when it failed, a 1.5 DIP `TextMuted` ring while the first check runs. Its tooltip and accessible name say it in words: "Connected · <platform> user <user>", "Check failed · <why>", "Checking the connection…". A row without an alias keeps the empty slot so the folders line up; before the first report there are no dots. |
| Problem bar | Under the row of an identity whose check failed, inside the same list, indented to the name (50 DIPs: 12 + the 28 DIP badge + 10), 28 DIP line: the error circle in `Danger` (the globe in `TextMuted` for an HTTPS remote), one `Rem(0.92)` sentence (a missing key names its path in monospace, middle-truncated), the reason as its tooltip, and "Check Again" (the refresh icon button, disabled while the check runs). Below it, indented 20 more, at most one line of next step: "Unlock with Keychain" on macOS (an `Accent` text button); elsewhere, or after the unlock failed, the `ssh-add <path>` command with a copy button; "Copy Public Key" (a copy icon button) and "Open <platform>" for a rejected key; "Add Rewrite" for an HTTPS remote; "Generate Key" for a missing key, which opens the Keys pane of the settings window filled in for the identity. No hairline between the row and its bar. |
| ssh notice | When the checks find no `ssh`, one card above the drop zone replaces every bar and the summary: card fill with a 1 DIP inner `Danger` outline, the error circle, "ssh not found" (600), "Check Again" over all identities, and one muted line on how to install it for the platform. |
| Diagnostics summary | After the identity list, once a round of checks has finished. Nothing wrong: one muted line with a `Success` check mark, "Configuration OK". Otherwise a card (card fill, radius 10): a 32 DIP header, a button with `aria-expanded` (the worst glyph, "1 error · 2 warnings" in 600, a chevron), and when open one entry per problem under hairlines: the glyph, what is wrong, how to look into it in `TextMuted`, an optional action ("Copy Command" as a terminal icon button for NET001, "Regenerate" as an `Accent` text button for GIT001), and the code (NET001, SSH001, GIT001, GIT002) in monospace `Rem(0.85)` at the end. It opens by itself whenever its set of entries changes, and stays as the user left it otherwise. |
| Undo bar | The last row of the result card, 36 DIPs high, hairline above: "Bound to <identity> ·" and an `Accent` "Undo" text button. A 2 DIP line on the bar's bottom edge, `Accent` at 45%, is as wide as the time left of the card's width, in fifths. |
| Identity list | A group heading "Identities · N" (`Rem(0.85)`, 600, `TextMuted`), then one card container (card fill, radius 10, no outline) with 0.5 DIP hairlines between rows, starting at the name (50 DIPs from the edge). |
| Source list row | A 28 DIP initial badge, name and email on two lines, an optional folder symbol; `Accent` fill with `AccentText` when selected. |
| Color choice | A 14 DIP dot in the preset color with its name as `Label`; the chosen one has a ring and a check mark. |
| Settings row | Label left, value right, 32 DIPs high, hairline between rows inside a card-fill group with radius 10. |
| Platform filter | A row of `ui.Button`s, one per platform in use when more than one is; the selected one is `Accent` with `AccentText` and is `Selected(true)` so screen readers read it as selected. |

The tables below say what to reach for as the accounts, settings and about windows are added.

### Actions

| Need | Use |
|---|---|
| The main action of a group, once (Save in the accounts window) | [Button](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/button.md) (`PrimaryButton`) |
| Other actions | [Button](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/button.md) |
| Open the GitHub page from the about window | A link-style [Button](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/button.md) (`Accent` text, no fill) with the external icon and `c.OpenURL` |
| "Rebind to" another identity | A menu button (`Element.Menu`): the system's menu, on the system's material. A menu drawn inside the panel could only blur the panel's own pixels and looked gray (issue 51). |
| Actions on a specific identity, on right click | [Context menu](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/context-menu.md) |
| Choose one of a few modes (global or limited scope) | [Segmented control](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/segmented.md) |
| Adding to a list ("Add Identity") | A symbol button in a bar at the bottom of the list; icon button (plus) in the panel footer |

### Choices

| Need | Use |
|---|---|
| A setting applied immediately (start at login, push on a URL rewrite) | [Switch](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/switch.md) |
| A yes/no value submitted with a form ("also delete the private key"), or several independent options (import candidates) | [Checkbox](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/checkbox.md) |
| One of two to five visible options (port 22 or 443) | [Radio](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/radio.md) |
| One of many options (platform) | `theme.Select`: a 26 high button as wide as the choice with a down chevron, and the options in the system's menu (`Element.Menu`), the chosen one checked; with typing to filter, [Combobox](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/combobox.md) |
| An identity's color | The six presets as named choices; any other value with a [color well](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/color-picker.md) (`ui.ColorWell`) |

### Text

| Need | Use |
|---|---|
| Free text (name, email, alias, key path, notes) | [Text input](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/text-input.md) |
| Filtering a list with localized copy | [Text input](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/text-input.md) with a translated `Label` and `Placeholder` |
| Labeled controls with help and errors (the identity form) | [Form](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/form.md) |

### Collections

| Need | Use |
|---|---|
| Identities in the accounts window, backups, long lists of one kind | [List](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/list.md) |
| Rows with sortable, resizable columns | [Table](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/table.md) |

### Containers and navigation

| Need | Use |
|---|---|
| Scrolling content | [Scroll view](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/scroll.md) |
| Columns and rows of controls | [Grid](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/grid.md) |
| List and detail of the accounts window | [Split view](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/split.md) |
| Optional detail (the clone helper of an identity row, a backup's contents) | [Collapsible](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/collapsible.md), [Accordion](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/accordion.md) for several in a stack; only one identity row is expanded at a time, which the screen's state enforces |
| The five panes of the settings window (General, Keys, Import, Backups, Configuration) | [Tabs](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/tabs.md); `ui.TabsBase` for the icon over the label |

### Dialogs, messages and status

| Need | Use |
|---|---|
| Confirm something that cannot be undone (deleting an identity, restoring a backup, removing the Include) | A dialog window: `theme.Dialog` (`RoleAlertDialog`) in a modal child window, see below |
| Choose a folder or a key file | `mygo.Dialog` ([desktop APIs](https://github.com/egoist/mygo/blob/v0.4.1/docs/native.md)) |
| A small panel tied to an element | [Popover](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/popover.md) |
| A short label or hint for a control | [Tooltip](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/tooltip.md) (never the only place for essential information) |
| Feedback that needs no decision | Text on the card or form that caused it ("Copied", "Saved"); a [Toast](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/toast.md) only where there is no such place |
| Unknown duration (reading, checking, `ssh -T`) | [Spinner](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/spinner.md) |
| A count or short status | [Badge](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/badge.md) |

Overlays share focus and Escape behavior; see
[Overlays](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/overlays.md). Rules of thumb:

- Prefer undo over confirmation. Binding a folder needs no confirmation and can be undone. Confirm only what cannot
  be undone, with a button named for the action ("Delete"), not "OK", and Cancel as the Escape action. The one
  exception is giving a global identity a folder, which changes what every other folder uses: it asks first, inside
  the result card, not in a dialog.
- Do not stack dialogs, and do not use a dialog for an error.
- A dialog is a window of its own (`dialog_window.go`, [ADR 0005](adr/0005-dialogs-and-menus-in-system-windows.md)): frameless, not resizable, modal and a child of the
  window it belongs to, 340 DIPs wide and as tall as its content, centered over its parent, on the system material
  (`Popover` on macOS, `Acrylic` on Windows 11, the solid color on Linux and Windows 10). There is no scrim: the
  parent shows its inactive look. Inside: `Space(5)` padding, title, text, then the buttons side by side sharing the
  width, 28 DIPs high, Cancel first and the action last. Escape and Cancel close it; closing it focuses the parent.
  Drawn inside the window, a dialog could only blur that window's own pixels and looked gray (issue 51).
- Errors say what happened and what to do, on the card or form where they happened (`Error`, or a
  [form](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/form.md) field).
- Confirmations of copying and saving are a 1.2 second "Copied" or "Saved" next to the control; a repeat restarts
  the timer.

## The panel

Mockup: [design/screens/panel.html](../design/screens/panel.html), and both appearances side by side in
[design/screens/panel-appearance.html](../design/screens/panel-appearance.html).

From top to bottom: the title row ("Idset", `Rem(1.15)` / 600, a level 1 heading, and the Settings icon
button at the right; no close button: the tray icon and Escape hide the panel), the content, and the footer. The title
row is `Space(11)` (44) high with no divider; the footer is `Space(10)` (40) with a hairline above it. The title row
has `Space(4)` left and `Space(2.25)` right padding; the footer has `Space(2)` at both sides. The icon buttons are
26×26 with a 16 glyph, 4 apart. The footer is four icon buttons: "Add Identity" (plus) at the left, then at the right About (`info.circle`), Check Again (`arrow.clockwise`)
and Quit (`power`). The footer after the content:

- With identities, all of them. Without identities, loading or read-only: no Add (the empty state has its own button;
  the other two cannot write) and no Check Again (nothing to check); About and Quit stay, so the footer still has a way
  out when the read-only error takes the whole panel.
- Only buttons that work are shown. About opens the about window. Check Again checks every identity again and waits while a check runs. Settings is always in the title
  row.

The content, in order:

1. Loading: only a spinner and "Reading configuration…", centered.
2. `accounts.json` cannot be read: nothing else can be done, so the error fills the content and is centered
   vertically: a `Space(11)` (44) warning badge (radius 11), the title (`Rem(1.15)`, 600), the file path in
   monospace `Rem(0.88)` `TextMuted`, the parser error in monospace `Rem(0.85)` `TextMuted`, at most two lines (the full
   text is its tooltip), one short line "Read-only until fixed" (the sentence "Idset does not write to it until it is
   fixed or restored." is its tooltip), and three buttons: "Restore from Backup" (primary, 28×248, opens the Backups pane of the settings window), then
   "Show in Finder" (Explorer on Windows, the file manager on Linux) and "Reload", 28×120 each on one row. There is no reason box.
3. Otherwise: the result card (when there is a result), the ssh notice (when `ssh` is missing), the one-line drop
   zone, the group heading "Identities" with the count, the filter (from three identities on), the identity list with
   its connectivity dots and problem bars, and the diagnostics summary. Without identities there is one centered block
   instead: a 56×56 tile (radius 14, accent-soft) with the key glyph in `Accent`, then "No identities yet" (`Rem(1.15)`, 600), one short line ("Each folder uses its own Git and SSH identity.")
   and the filled "Add Identity" button (28 high); the drop zone sits under it, at the bottom of the content.

The drop zone is always one row, with or without a result, 48 DIPs high: a 20 DIP folder glyph in `Accent`, the line
"Drop a folder to see its identity" (「拖入文件夹，看用哪个身份」, `Rem(0.92)`), and at the end a 24 DIP high primary
"Choose…" button. The box has a dashed 1.5 DIP `Accent` outline at 55% over a fill of `Accent` at 8%; while a folder
is dragged over it the outline is solid `Accent` and the fill accent-soft. "Shows which identity a folder uses" is the
tooltip of the line and "Choose Folder…" that of the button. Words that only explain are tooltips throughout the panel: the full sentence of
"Covered by …", the reason of "Read-only until fixed", the "adds another" of a bind row, the why of a finding.

The result card has two parts with a divider between them:

- The context line: the folder (the repository root, or the dropped path when it is not a repository) in monospace,
  middle-truncated, then a short transient status ("Copied", "Updated", "Undone", "Covered by …"), the icon buttons of
  a matched card (open, copy, "⋯") and the clear button (not while checking). 36 DIPs high with its hairline;
  padding about 6 right, `Space(3)` left (12).
- The answer, padding `Space(3)` (12). Checking: a spinner and "Checking…". Matched: a `Space(9)` (36) initial
  badge (radius 10, solid, its letter near black or white), the identity's name and "Git uses this identity here", gap `Space(3)`; below them three rows,
  "Alias", "Email" and "Folders", with a label column `Space(14)` (56) wide, lined up with the name (left padding
  `Space(15)`, 60). Each label is top-aligned with the first line of its value, so "Folders" stays by the first
  folder however many follow. Without a match: the sentence for the case, 600, with a 1 DIP inner `Warning` outline. `git` failed: "Could not
  run git" in `Danger`, then the reason in monospace `Rem(0.85)` `TextMuted`, at most two lines (the full text is its
  tooltip), with a 1 DIP inner `Danger` outline. In "Folders" each path has its own line.
- Identity check (docs/product.md): when the repository's own `user.email` or `core.sshCommand` disagrees with the
  matched identity, the finding rows follow the details inside the same card, and the outline takes the worst
  severity. Without findings the card is exactly the plain matched card. Mockup states: no email, another identity's
  email, an email no identity has, another key, and two findings with long values; `&tip=finding` pins the first
  row's tooltip.
- Remote and clone (docs/product.md, "Change-remote command" and "Clone helper"). The matched card is read from top
  to bottom as: who and details, what is wrong, what to do next. In that order: the details, the finding rows, the
  HTTPS row, the clone helper. The HTTPS row is the last row of the same block as the findings (one hairline, same
  28 DIP rows), because it is also a fact about the repository; but it has no severity: its glyph is a muted globe
  and it never changes the card's outline, since HTTPS is not an error. The clone helper is a tool, not a fact, so
  it has a hairline and a section of its own at the bottom, where the eye ends and where a pasted address is
  typed. Both appear only for an identity with an alias, and the helper is the one block that is always there.
  - HTTPS row: "Remote is HTTPS", with one icon button at the end (`terminal` glyph, 26 x 26, 6 from the edge like
    the context line). The button's name is "Copy Switch-to-SSH Command"; its tooltip is that name and the whole command
    (`git remote set-url origin git@github-work:org/repo.git`), so nothing on the card has to show a long mono
    string. Why it matters ("HTTPS pushes do not use this identity's alias or key.") is the text's tooltip.
    After the copy the glyph is a check and "Copied" (`Rem(0.85)`, `TextMuted`) stands before it for 1.2 seconds.
  - Clone helper: padding `Space(2)`x`Space(3)`, bottom `Space(2.5)`. A 28 DIP input (monospace `Rem(0.88)`, field fill,
    0.5 DIP `Border`, radius 6) with the copy button (`terminal` glyph, the same glyph as the HTTPS row, since both copy
    a command) at its end; under them a 16 DIP line, monospace `Rem(0.85)` `TextMuted`: the output, `git clone
    git@github-work:org/repo.git`, middle-truncated like a path ("git clone git@github-work:" and the last segment
    stay; the full command is its tooltip). Enter copies. When the input cannot be read the copy button is
    disabled (40% opacity) and the line is a quiet hint in the same style, "Enter owner/repo or a repository URL",
    not an error: an empty or half-typed input is not a mistake. The input's tooltip lists what it accepts. After
    a copy: check glyph and "Copied" left of the button for 1.2 seconds, the output stays. The input is prefilled
    with the repository's `owner/repo`, else the remembered prefix ("acme/"), else shows the placeholder "owner/repo".
  - Mockup states: `https`, `https-copied`, `https-check` (finding and HTTPS row stacked), `clone`, `clone-long`,
    `clone-bad`, `clone-copied`, `row-open`; `&tip=cmd` pins the copy-command tooltip, `&tip=copyclone` and
    `&tip=clone` those of the helper's button and an identity row's disclosure button.
- Binding. Mockup states: no match, rebinding to a global identity (the confirmation), matched (by a parent folder, or by the folder itself
  with "Unbind"), the rebind menu, already covered, overlap, the undo bar, and after undo.
  - No match: the sentence, a "Bind to" heading (`Rem(0.92)`, 600, `TextMuted`) and one bind row per identity, then a
    last 40 DIP row with "New Identity" (an `Accent` text button with a plus; "New Identity for This Folder" is its
    tooltip and accessible name). One tap binds; there is
    no confirmation. A folder has no match only when there is no global identity, so the list never has one.
  - Rebinding to a global identity: choosing it in the "Rebind to" menu closes the menu and adds, under the card's
    details, the confirm block: one line "<name> will apply only here" and two 80×28 buttons, "Cancel" (closes the
    block, as Escape does) and the filled "Rebind".
  - Matched: the actions are icon buttons in the context line (open, copy, "⋯", clear). "Unbind" is absent from the menu
    when only a parent folder covers the folder; nothing else explains it, since "Folders" lists the covering path. "Rebind
    to" lists every other identity.
  - Already covered: the bind did not change anything. The card keeps its answer and its context line shows "Covered by
    <path>" (`Rem(0.85)`, `TextMuted`, one line, ending in an ellipsis when short of room) before the buttons for 1.2 seconds,
    like "Copied"; the full sentence "<name> already covers this folder with <path>.
    Nothing changed." is its tooltip. The path keeps at least 80 DIPs; the status shortens first.
  - Overlap: a card of its own above the matched one, without a context line (the matched card under it shows the
    folder, and its one clear button clears both); its first row is "<n> identities cover this
    folder" with an info icon (the order sentence is its tooltip), and the overlap rows with their pin and minus buttons.
    The card has no outline, since nothing is wrong. It shows only while the winner has no path of the folder's own; once it has one (after "Use only" or a rebind) the choice is explicit and the card is gone.
  - Stale path: when the dropped folder looks like a renamed or moved `gitdir` path that no longer exists, a card of
    its own sits above the no-match card, the same way the overlap card sits above the matched one. The header is
    one line and the paths are middle-truncated, so the card's height never changes. The no-match card below is
    unchanged, so the bind list still works; in its bind rows the middle of the first folder gives way to an
    ellipsis and the rest stays: "~/…/payments-gateway/". Mockup states: rename, move (a long old path), after "Update" and after
    "Ignore".
    - "Update" (tooltip "Use New Path"): the card is gone and the folder matches, so the matched card shows with "Updated" next
      to the copy button for 1.2 seconds, like "Copied". There is no undo bar: only bind, rebind and "Use only" have
      one (docs/product.md), and the old path is gone from disk anyway.
    - "Ignore" (tooltip "Keep Old Path"): only the prompt goes; the plain no-match card stays and shows no acknowledgement.
  - Undo: after a bind, rebind or "Use only", the undo bar is the last row of the matched card. The countdown line
    shortens once a second (`c.After` from the absolute deadline), five steps; no animation. After "Undo" the card
    returns to what it was, with "Undone" next to the clear button for 1.2 seconds, like "Copied". Unbinding shows
    no bar.

The identity list starts with the group heading "Identities · N" (「身份 · N」, `Rem(0.85)`, 600, `TextMuted`), 6 in from
the card's edge, then one card. Identity list rows: at least 48 DIPs high, padding `Space(2)`×`Space(3)` (8×12), a
28×28 initial badge (radius 8, the identity's color, solid, the same as in the accounts window's source list; no color
dot), gap `Space(2.5)` (10), the name (`Rem(1)`, 600) and the email (`Rem(0.92)`, `TextMuted`) on two lines, and at
the end the folders: the first one, "+N" when there
are more, "Global" for a global identity, "no folders" without any. A long first folder is middle-truncated (at
most half the row) and " +N" always stays; the name and email are single lines too.

An identity with an alias ends in a 26 x 26 disclosure button ("Clone with This Identity": a chevron pointing right
when closed, down when open; `aria-expanded`); a row without an alias keeps the 26 DIPs empty so the folders stay
lined up. The row's body still opens the accounts window; only the button expands. Open, the row is followed, inside
the same list and without a hairline between them, by the clone helper described above, indented to the name
(50 DIPs: 12 + 28 + 10) and with no remembered prefix shown as the placeholder "owner/repo". Only one row is open: opening
another closes the first. Nothing else about the row changes while it is open.

A path on one line is shortened in the middle only, the same way everywhere (the stale-path prompt's rows, the first
folder of a bind row, the folder of an identity row, the context line of a result card): its start ("~/", "/" or a drive such
as "C:/") and its last folder always stay, and what lies between gives way to an ellipsis, "~/…/payments-gateway/".
Only when even those do not fit is the last folder cut at its end. The full path is in the tooltip.

The row hairlines are 0.5 DIP `Border` and start at the name (50 DIPs from the card's edge), not at the card's edge.

Identity rows open the accounts window on their identity. "Add Identity" (the plus icon button) sits at the left of the
fixed footer, so that it stays reachable however long the list is: the content between the title row and the footer is
the scroll view. In the empty state the centered "Add Identity" button
replaces it. Both open the accounts window on a new draft. Under that button, "Import Existing" is an
`Accent` text button that opens the settings window on Import.

Planned (product.md): the expand arrow and the HTTPS row and clone
helper are drawn in the mockup and shown once built (issue 26). Bind, unbind, rebind, "Use only" and undo on result
cards are built as the mockup shows.

## The accounts window

Mockup: [design/screens/accounts.html](../design/screens/accounts.html), every state in light and dark and in both
languages. Behavior is in product.md, "账户窗口" and "删除确认". The model is a native grouped-settings window
(System Settings, Mail accounts): few outlines, one hairline between rows, hierarchy from position and the muted
color rather than from boxes. Everything is a material, a translucent fill, a hairline or text; only dialogs have a
shadow.

The window is 760×520 with radius 12 corners the system draws. The title bar is transparent, with no hairline and no
toolbar: below it, a source list and a detail pane.

- **Source list** (left, `Space(58)`, 232, straight on the material: `Sidebar` on macOS, no fill of its own). Its
  content starts 16 under the title bar, level with the detail pane's. Group heading "Identities" with the count (11, 600, `TextMuted`), then rows in the panel's order, inset 8 DIPs from the
  list's sides: `Space(1.5)`×`Space(2)` padding, radius 6, a 28 DIP initial badge (radius 8), the name (13, 600) and
  the email (`Rem(0.92)`, `TextMuted`) below. Text wraps. An identity limited to folders has
  a folder symbol at the end of its row, named "Scoped" for assistive technology; a draft shows "New" there. The
  selected row is filled with `Accent` and its text is `AccentText`, like the system's source lists; its badge gets a
  1.5 DIP `AccentText` ring so an orange or red identity stays apart from the fill. The list scrolls above an
  add bar fixed to the bottom edge of the sidebar: a hairline, then a borderless "+" symbol button named "Add
  Identity" (also its tooltip), `Space(11)` (44) high with the button centered.
- **Detail** (right, scrolls on a slightly more opaque layer, the detail fill, separated from the list by a 0.5 DIP
  hairline). Padding `Space(4)`×`Space(5)` (16×20), `Space(5)` (20) between groups.
  Header: a 32 DIP initial badge (radius 10), the title (`Rem(1.15)`, 600, level 1), "Global" or "Scoped" (`Rem(0.92)`, `TextMuted`), and at the
  end the six preset colors as 14 DIP dots (the chosen one ringed and ticked, each named for assistive technology)
  and the hex value.
- **Groups.** A section title (11, 600, `TextMuted`, level 2) over an inset group in the card fill with radius 10 and no outline: Identity,
  SSH, Scope, URL rewrites, Notes, Times (not for a draft). A row is 32 DIPs high with `Space(3)` (12) side padding:
  the label on the left (a fixed label column, so rows line up), the value on the right, right-aligned. Rows are
  separated by a hairline inset 12 from the left. Values are plain text, not boxed text fields; a focused one shows
  the focus ring around its text. Paths, aliases, host names, ports and URLs are monospace 12 and are sized to fit
  one line. A footnote under a group (`Rem(0.92)`, `TextMuted`) carries help; an error there is `Danger` with the warning
  symbol.
- **Controls in rows.** Platform is a select (value and a down chevron). Port is two radios (22, 443); Custom shows a
  host name and a port value. Scope is a row "Applies to" with a segmented control, "Global" and "Limited to
  Folders"; its footnote says when a global identity applies. The private key row is the path in monospace and a folder symbol button that opens the file picker; a path too long for the
  field is shown shortened in the middle ("~/.ssh/keys…_personal_laptop") until it is clicked, and its tooltip
  holds all of it. A limited identity lists one row per folder (path,
  choose-folder button, a minus button, disabled for the last one) and a text row "+ Add Path". A URL rewrite is one
  row: the `from` URL, and under it `→` and the `to` URL, then "Push only" with a switch and a minus button; then the
  text rows "+ Add Rewrite" and "+ Add Default Rewrites" (disabled without an alias, the reason in the
  footnote). Text rows are `Accent` text with a leading plus, never boxed.
- **Hints.** A prefilled signature name and email carry "From git config --global" (`Rem(0.92)`, `TextMuted`) at the end of their
  row, right of the value; the row keeps the standard 32 height. A long value is cut before it reaches the hint. The
  hint goes as soon as the field is edited.
- **Scrolling.** The detail is a scroll view above the fixed footer; the header scrolls with the groups. The source
  list scrolls above its add bar.
- **Footer.** Hairline above, `Space(11)` (44) high like the sidebar's add bar so the two hairlines are one line, side padding
  `Space(5)`, everything centered vertically. Delete, Revert and Save are `Space(7)` (28) high; Revert and Save are `Space(25)`
  (100) wide, enough for "✓ Saved", "Saving", "已保存" and "保存中", so no label change resizes or moves anything. At the left "Delete" as plain `Danger` text (existing
  identities only), then the status (`Rem(0.92)`, `TextMuted`), then "Revert" (bordered) and "Save", the only filled button.
  Status: "Unsaved changes", or "Not saved" for a draft; a spinner with "Writing…" while saving,
  when the form is dimmed and disabled; "Saved" with a check mark on the Save button for 1.2 seconds after
  success; the reason in `Danger` with the warning symbol, wrapping, when the write failed (Save stays available).
  Save and Revert are disabled while there is nothing to save.
- **Empty.** No list and no footer: "No identities yet" (`Rem(1.15)`, 600), one sentence, "Import Existing"
  (bordered; opens the settings window on Import) and "New" (primary), centered, with no add bar.
- **Delete confirmation.** A dialog window (see Dialogs), 340 wide, `Space(5)` padding:
  "Delete identity <name>?" (`Rem(1)`, 600), a sentence, the private key path and the folders that are not
  deleted automatically between two hairlines (no box), a line that project folders are never touched, the "Also
  delete the private key" checkbox (unchecked by default) and the buttons Cancel (bordered; the Escape action and
  initial focus) and Delete (`Danger` fill, white text). When the key cannot be deleted (no key path, another
  identity uses it, a symbolic link, missing, not a regular file) the checkbox is disabled and the reason is
  written under it; when checked, a line says that only the private key is deleted and `.pub` stays.
- **Type.** The five styles of Typography: 13 body, 12 secondary for emails, notes and status, 11 captions for group headings, 15 600 for the editor title; list names are 600.

## The settings window

Mockup: [design/screens/settings.html](../design/screens/settings.html), every state in light and dark and in both
languages. Behavior is in product.md, "设置窗口". 720×480, a hidden title bar with the title drawn by `theme.TitleBar` ("Idset · Settings"), on the `Popover` material:
translucent fills, hairlines and text; only dialogs have a shadow.

- **Tabs.** A tab bar along the top, under the title bar, on the material, with a 0.5 DIP hairline under it. Five tabs centered as a group, each an icon over a label: General, Keys, Import, Backups, Configuration (通用 /
  密钥 / 导入 / 备份 / 配置). A tab is 88 wide and 52 high (`Space(22)`, `Space(13)`) with a 20 DIP icon (`Space(5)`), the 11 label 4 below it,
  and 4 DIP gaps; the bar has 4 above and 6 below. The selected tab has the accent-soft fill, its icon and label in `Accent`, and a
  8 DIP radius; the others are `TextMuted` on no fill. Arrows, Home and End move the choice, as `ui.Tabs` does. The
  pane has no heading of its own: the selected tab names it.
- **Pane.** Full width below the bar, 16 above and 20 at the sides, scrolling; its content is at most 560 wide and
  centered, so wider windows do not stretch the rows.
- **General.** One grouped list of 32 DIPs rows. Language is three tick rows: "Follow System" (shows the language it
  resolves to), "English" and "简体中文", the last two always in their own language. "Open at login" is a switch row; when
  the system asks for approval the switch stays off and a `Warning` note under it says where to approve (macOS: System
  Settings, General, Login Items). A failed change is an inline `Danger` note under the row.
- **Keys.** The form is one grouped list "New key" of 32 DIP rows, a 168 DIP label at the start and the control at
  the end: Type (a segmented control, Ed25519 / RSA 4096; its track is `Surface` and the selected segment white in light and 18% white in dark), Platform (a pop-up), File name (monospace) and Comment
  (borderless inputs, text at the end). Under it a muted note says where the two files go and that an existing file is
  never overwritten; a bad name or a failure replaces it with an inline `Danger` note. "Generate" is the primary
  button at the end; while it runs the group is dimmed to half and a muted "Generating…" and a spinner stand before
  it. Opened from a missing key's bar, a banner in the card fill with the key icon above the form names the identity and its
  key path. The result has a heading with an `Accent` check, "Key created" and a muted "type · platform", then three
  groups: "Public key" (the selectable monospace key, "Copy Public Key" as a copy icon button at the end that turns into a
  check, with "Copied" beside it, for 1.2 s), "Files" (private key with "Show in Finder" as a ghost button, public key,
  fingerprint, passphrase "None", 96 DIP labels, a footnote with the `ssh-keygen -p` command), and "Next steps" (add the
  public key on the platform with "Open <platform> Settings" and the external icon; "Load into ssh-agent" with Load,
  a spinner, then a check and "Loaded"; "Use for identity" with "Choose…", then the identity's dot and name and
  "Change…"). Notes under Next steps: the identity written, ssh-agent not running (`Warning`), or the platform hint.
  "Generate Another" is a text button at the end. "Choose…" opens a dialog window 340 wide: "Use this key for which
  identity?", one sentence, the identities as radio rows (dot, name, muted email, and "Now: <key>" or "No key yet"
  under it) and "New identity" with the email and platform it starts from, between two hairlines, a muted note,
  Cancel (initial focus) and Use.
- **Import.** Scanned each time the pane is entered; while it reads, a centered spinner, "Reading your SSH and Git
  configuration…" (15, 600) and a muted line naming the files. The candidates are one grouped list under "N
  identities found", rows 44 high: a checkbox, the Host name (muted when unchecked) over a muted "alias · email" (11;
  "No alias" when there is none), and at the end a muted "Already added", or a folder icon and "Imported as scoped"
  when that candidate is checked. Under the list, muted footnotes: the files are only read; the scoped ones apply to
  no folder yet. Below, "Scan Again" as a refresh icon button at the start and "Import N Identities" (primary) at the
  end, disabled with nothing checked. While importing the list is dimmed to half and a muted "Importing…" and a
  spinner stand before the button. A failure replaces the footnotes with a `Danger` note naming the identity, the
  reason and how many came in before it; those are marked "Already added" and unchecked. Done is a heading with an
  `Accent` check and "N identities imported", a group of the identities (dot, name, muted email), an `Accent` check note
  "Configuration regenerated.", then "Scan Again" (refresh icon button) and "Open Accounts" (bordered). No candidates is a centered title,
  a muted sentence and "Scan Again"; a failed scan is a `Danger` banner with the reason and "Scan Again". When
  `accounts.json` cannot be read, a `Danger` banner tops the pane and the import button is disabled.
- **Backups.** One grouped list, newest first. A row is 44 high: a chevron, the time (13), and a muted line "N identities
  · names · size" (11; the first three names and "+N" for the rest), then two ghost buttons in fixed 24 DIP slots:
  Restore (a back arrow, tooltip "Restore") and trash. The chevron expands the row to show its identities read-only, one at a time. Delete has no confirmation: the
  backup is a copy and the newest 10 are kept anyway. A backup that cannot be read has its sub-line in `Danger`, and
  Restore and the chevron disabled. Under the list, a note says when backups are made (before an identity is added,
  before a restore), that the newest 10 are kept and where. With no backups that note is the empty state.
- **Restore.** A dialog window, 340 wide, like the delete confirmation: "Restore backup <time>?", one
  sentence (the current `accounts.json` is backed up first, the configuration is regenerated), the identities that will
  exist afterwards between two hairlines, a line that identities not in the backup are removed, Cancel (Escape,
  initial focus) and Restore (`Accent` fill: it replaces data but is undoable through the backup it makes). While it
  runs the Restore slot is a spinner with a muted "Restoring…" before it, and the other rows are dimmed to half; on
  success the slot is an `Accent` check with a muted "Restored" before it for 1.2 s; on failure the reason is an inline `Danger` note under the row. When the window is opened from the
  read-only panel the Backups pane opens with a `Danger` banner explaining why.
- **Configuration.** The two files with a marker block (`~/.ssh/config`, `~/.gitconfig`) and whether it is present, and
  the row "Remove Include…" in `Accent` text. Its alert (same shape, primary button) says in
  one sentence what changes and that the blocks come back at the next identity change, then two boxes in the card fill with a
  small muted heading: "Will be removed" lists the marker block in each file, one per line behind a `Danger` minus;
  "Left as is" lists `accounts.json` and the identities, the managed files under `~/.config/idset/`, and private keys
  with the rest of the user's config, each behind a muted check. Paths are monospace. Afterwards the rows read "no
  marker block" with a "✓ Removed" note. A failure is inline and names the file that could not be changed, including the
  case where one file was changed and the other not.
- **Errors and feedback.** Errors are inline notes or banners, never dialogs; the only alerts are the two confirmations above and the identity choice of the Keys pane.
  Success feedback is the 1.2 s label of the panel.

## The about window

Mockup: [design/screens/about.html](../design/screens/about.html), every state in light and dark and in both
languages. Behavior is in product.md, "关于窗口". 320×320, the hidden title bar ("About Idset") included, not resizable,
minimizable or full-screen, on the `Popover` material; one column centered both ways in the space under the title
bar, `Space(6)` at the sides and `Space(5)` kept free below.

- **Identity.** The product icon tile at 64 DIP (`Space(16)`), from `resources/icon.png`. Under it, 12 apart, "Idset" (18, 600, a level 1
  heading), then the muted, selectable "Version 0.1.0", then 12 below the copyright and the license on two muted
  11 lines.
- **Actions.** 20 under the license, 8 apart and centered: "GitHub", a link in `Accent` (500, no fill) with the external
  icon, opens the repository in the browser; "Check for Updates…" is the primary button and opens the updater's own
  window.
- **Cannot update.** Whether this copy can update itself is found out in the background; until then the primary
  button is disabled without a note. When it cannot (a development build, no update feed, or a copy a package manager
  updates), the button stays disabled and a muted 12 line under the buttons, at most 260 wide, says to download new
  versions from GitHub.
- **One window.** The tray menu's "About", the panel footer's About and the macOS app menu's "About Idset" all open it;
  when it is open they bring it to the front.

## Windows, menus and the tray

- **The panel window.** 340×480, frameless, not resizable, minimizable, maximizable or full-screen, always on top and
  left out of the taskbar. It opens below the tray icon, centered on it and 6 DIPs away, or above it when there is no
  room, kept inside the work area of that display. It does not hide when it loses focus: the user clicks the file
  manager to start dragging a folder, which takes the focus away ([ADR 0003](adr/0003-drop-zone-in-tray-window.md)).
  The tray icon and Escape hide it. See
  [Windows with native UI](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/windows.md) and the
  [window options](https://github.com/egoist/mygo/blob/v0.4.1/docs/windows.md).
- **Without a tray** (Linux without AppIndicator), the panel is an ordinary window with the system title bar and the
  same content; Escape does not hide it, and closing it quits.
- **Other windows** (accounts 760×520, settings 720×480, about 320×320) hide the title bar
  (`TitleBarStyle: mygo.TitleBarHidden`) so the material runs to the top edge, as the mockups show: the system keeps
  the window controls, and the view draws the title in their band, centered, 11 and `TextMuted`, dragging the window
  (`theme.TitleBar`). They set `MinWidth` and `MinHeight` so content never clips (the about window does not resize),
  and hide the panel when they open.
- **The tray.** A click opens the panel and a second click hides it. The right-click menu has "Open Idset" (or "Hide
  Idset" while the panel shows) and "Quit", with the accounts window, settings and about between them. On Linux
  AppIndicator reports no clicks, so a click opens that menu and it is the only way to open and hide the panel. The
  tooltip is "Idset", "Idset · <identity>" after a match, or "Idset · no match". See
  [Menus and the tray](https://github.com/egoist/mygo/blob/v0.4.1/docs/menus.md#tray-icons).
- **Menus.** The macOS menu bar has an app menu (About, check for updates, Quit) and the Edit menu (`RoleEditMenu`)
  so copy, paste and the other editing shortcuts work in text fields. "About Idset" is a custom item that opens the
  about window rather than `RoleAbout`'s system panel. Use roles (`RoleQuit` and the others) for standard items: they carry the right label and shortcut. Give custom items an accelerator with
  `CmdOrCtrl`. See [Menus](https://github.com/egoist/mygo/blob/v0.4.1/docs/menus.md).
- **One instance.** A second launch opens the panel of the running instance and exits.
- **Notifications.** System notifications only for what the user would want while the panel is hidden; the first
  launch says once that Idset lives in the tray, without taking the focus.

## Accessibility checklist

MyGo exposes native UI to VoiceOver, Narrator, NVDA and Orca; see
[Accessibility](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/accessibility.md).

- [ ] Every interactive element has an accessible name: its text, or `Label` when it shows only an icon or a symbol
      (the copy and clear buttons, the filter input).
- [ ] `Description` carries help text; `Error` marks invalid values.
- [ ] Custom elements that act as widgets set `Role`, and the states they show (`Selected`, `Checked`, `Expanded`).
- [ ] Section and screen titles are headings with a `Level`.
- [ ] Changes that do not move the focus call `c.Announce(text)`: the panel announces the result for a dropped folder
      ("Git uses <identity> here", or the sentence of the card) and "Copied". Toasts announce their text.
- [ ] The whole UI works with the keyboard: the focus order is the standard one and follows reading order, there is
      a visible focus ring (do not remove it), Escape hides the panel and closes overlays.
- [ ] Text uses theme sizes (`Rem`) and scales; layouts reflow at larger text and in the longer translation.
- [ ] Colors are theme roles (identity colors only on badges next to the name); contrast holds in both appearances
      and with high contrast on.
- [ ] Nothing depends on color, sound or motion alone.
- [ ] Pointer targets are at least about 24×24 DIPs (symbol buttons are 26×26).
- [ ] `ui.NewTester` tests click controls by their accessible names, so a missing name fails the test, and check
      announcements with `Announcements()` (see
      [Testing](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/testing.md)).

Contrast, scaling, keyboard use and screen readers are verified on real devices; a passing headless test does not
count as accessibility acceptance.
The [Inspector](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/inspector.md) (F12 in development builds) shows
the element tree, roles and accessible names of a running window, which helps find a missing name before that check.

## Motion

Idset has no custom animation: state changes take effect immediately. See
[Transitions](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/transitions.md) and
[Drawing and animation](https://github.com/egoist/mygo/blob/v0.4.1/docs/ui/drawing.md) for what the toolkit offers.

- **Default is none.** Do not add `Transition`, `Animate` or page transitions without a mockup that shows the change
  they explain and the maintainer's agreement; record the exception here.
- **If an exception is agreed,** use the built-ins with their defaults (element transitions are 200 ms, ease-out)
  and nothing longer than 300 ms for interface feedback. Avoid frame-by-frame custom animation.
- **Reduced motion.** `Animate` and element and page transitions already jump to the target when the desktop asks
  for less motion (`c.Preferences().ReduceMotion`). Any motion you compute yourself must read the preference.
  Spinners keep turning: they show that work is going on.
- **Never animate to hide latency**, and do not loop decoration. The tray icon does not flash.
- **The panel.** None. The drop zone, the result card and the "Copied" acknowledgement change at once; the
  acknowledgement goes away after 1.2 seconds through `c.After`, not an animation.

## Localization

- The UI is in English and Simplified Chinese only. Every user-visible string is in the embedded catalog `copy.json`
  next to the screen, keyed `en` and `zh-Hans`. Change both together; do not add other languages. The tray menu and
  tooltip read the panel's catalog.
- `locale.Language` (in `internal/ui/locale/`) maps the system locale: every Chinese locale, Traditional included,
  reads Simplified Chinese; everything else reads English. Screens use it rather than their own mapping.
- Every key has both translations, and each screen's test fails on a missing or untranslated entry. Accessible
  names and announcements are copy too: translate `Label`, `Description` and what `Announce` reads.
- Data is shown as it is: identity names, emails, aliases and paths are not copy. Paths show the home directory as
  `~`.
- English labels: short imperatives. Button and menu item names use title case, as macOS does, whether they show text or only a symbol
  ("Choose Folder…", "Copy Path", "Hide Idset"); headings and sentences use sentence case ("No identities yet"). Explanatory hints that are full sentences end with a full stop;
  headings, labels and one-line hints do not. Chinese sentences use full-width punctuation. Do not concatenate sentence fragments, since word order differs; use full-sentence strings
  with placeholders ("Git uses %s here").
- An action a common symbol says clearly (copy, clear, refresh, reveal, remove, expand) is a symbol button with its
  name as tooltip and accessible name; text buttons are for actions no symbol says. Copy states facts: what is
  wrong, what to do, or what happened, in one short line. It does not explain why or repeat what the screen shows.
- The button that reveals a file names the platform's file manager: Finder, Explorer, or "File Manager".
- Layouts must tolerate the longer of the two translations, and English text 30–50% longer than the first draft.
  Do not fix widths of text controls.

## Design acceptance checklist

Run before a UI pull request is merged, and on each target platform before a release:

- [ ] Every state in the mockup is reached in the app (for the panel: loading, read-only, no identities, a list,
      drag over, filter and no filter result, checking, matched and copied, no match, not a repository, not a
      folder, `git` failed).
- [ ] Both languages, light and dark checked.
- [ ] The panel opens next to the tray icon on each platform, and as an ordinary window without a tray on Linux.
- [ ] Keyboard-only run-through, and a screen reader pass on each target platform.
- [ ] High contrast and larger text size checked.
- [ ] Screenshots of each state, light and dark, are in the pull request body.

## References

This design language draws on the UI documentation of MyGo (MIT License, Copyright (c) egoist):
<https://github.com/egoist/mygo/tree/v0.4.1/docs/ui>, together with `docs/native.md`, `docs/menus.md` and
`docs/windows.md` from the same version. The text here is a summary in our own words; the upstream pages are the
source for APIs and behavior. See also [THIRD_PARTY.md](../THIRD_PARTY.md).
