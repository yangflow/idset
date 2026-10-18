# Idset design language

This is the design language of Idset: the rules every screen follows, and the place to record where a screen
departs from them. Idset is a tray app that keeps a list of Git and SSH identities and generates the configuration
so every folder uses the right one (see [product.md](product.md), in Chinese). Its UI today is one window, the
panel that the tray icon opens: the identity list, and a drop zone that shows which identity Git uses in a folder.

The UI is built on the native toolkit of [MyGo](https://github.com/egoist/mygo). This document summarizes the parts
of its [UI guides](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/README.md) that matter for design; the guides
stay the authority on API details. Links point to MyGo v0.3.6, the version pinned in `go.mod`. When the pin moves,
check the pages again and update the links and this document together.

The panel (`internal/ui/panel/`) is the reference the values below call "the panel". The settings and about
windows are described in product.md and are plans until their mockup and code are merged.

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
10. [Windows, menus and the tray](#windows-menus-and-the-tray)
11. [Accessibility checklist](#accessibility-checklist)
12. [Motion](#motion)
13. [Localization](#localization)
14. [Design acceptance checklist](#design-acceptance-checklist)
15. [References](#references)

## Principles

- **Native first.** Use MyGo's native widgets and the theme defaults as they are, and follow the system's light and
  dark appearance and accent color. A control that looks and behaves like the platform's is learned already. Style
  an element yourself only when no widget does the job (the drop zone, the initial badge, the result card), and
  build it from containers and widgets, or on the
  [custom widget](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/custom-widgets.md) bases, so focus, keyboard
  and accessibility stay intact. No WebView and no HTML frontend.
- **Consistent across platforms, respectful of each.** One view serves macOS, Windows and Linux. Follow each
  platform's conventions where MyGo exposes them: the menu bar or notification area for the tray, a template icon on
  macOS, menu roles, the accent color, the system font, the name of the platform's file manager on buttons. Do not
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
  on the UI thread (see [Views](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/views.md)).
- **Spec before code.** Every screen is specified in its mockup in `design/` and in this document before it is
  written. Only `design/` and this document count as the design reference for Idset.

## Layout and spacing

Layout uses flexbox in device-independent pixels (DIPs); see
[Layout](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/layout.md).

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
  [scroll view](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/scroll.md). The panel scrolls when it shows the
  identity list. The loading, read-only and no-identity states never need to scroll and are not in a scroll view, so
  their content can be centered vertically (centering only works in a container of fixed height).
- **The panel.** Fixed 340×480. Title bar and footer are `Space(9)` (36) high, hairline included, each separated from
  the content by a divider; they are compact on purpose: the content is the point, and the chrome takes 72 DIPs of the
  480 and leaves 406 inside the outline. Content padding `Space(4)` (16), gap between blocks `Space(3)` (12). Cards, the identity list
  and the drop zone have a 10 DIP corner radius. The full measurements are in [The panel](#the-panel) and
  [tokens.md](../design/tokens.md).

## Color and dark mode

Colors come from the theme, `c.Theme()`: it is the light or dark theme, switches when the system appearance changes,
and carries the user's accent color and contrast settings. See
[Styling and themes](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/styling.md).

- **Always use theme roles, never literals.** Use only MyGo theme colors; for example, secondary text uses
  `TextMuted`. Hard-coded colors do not follow dark mode, accent or high contrast. The one exception is identity
  colors, below.

  | Role | Use in Idset |
  |---|---|
  | `Background` | panel fill; the reason box of the read-only state |
  | `Surface` | result cards, the identity list, the filter input and buttons |
  | `SurfaceHover`, `SurfacePressed` | hovered and pressed controls; the hover fill of borderless symbol buttons |
  | `Border` | dividers, outlines of cards and the list, the dashed outline of the drop zone |
  | `Text`, `TextMuted` | primary text; secondary text, hints, group headings, paths |
  | `Accent`, `AccentText` | the drop zone's outline while files are dragged over it; a selected platform filter, with its text in `AccentText` |
  | `Selection` | the drop zone's fill while files are dragged over it |
  | `Warning` | the outline of the "no match", "not a Git repository" and "not a folder" cards; the warning glyph of an identity-check finding, and the matched card's outline when its worst finding is a warning |
  | `Danger` | outline and text of the card when `git` cannot run; the warning symbol of the read-only state, on a 14% `Danger` fill; the error glyph of an identity-check finding, and the matched card's outline when a finding is an error |
  | `Success` | the connectivity dot of a passing check and the check mark of the "Configuration OK" line, always with text (the dot's tooltip and accessible name) |
  | `Focus` | the keyboard focus ring |
  | `Inverse`, `InverseText` | tooltips and toasts (left zero, they are `Text` and `Background`) |
  | `Scrollbar` | scroll bar thumbs |

- **Identity colors.** An identity's color is data the user chose, not a theme role. The six preset names map to
  fixed values (blue `#3A7BD5`, green `#2E9E5B`, orange `#E07A1F`, red `#D64545`, purple `#8E44AD`, yellow
  `#D4A017`); any other value is its own `#RRGGBB`. Identity colors appear on the initial badge of a result card and, as dots, in the panel list and the accounts
  window (list, header, color choices; the choices are named, the chosen one ticked). On the badge the fill is the
  color at 16% opacity (24% in dark), and the letter is the color itself (in dark, mixed 40% towards white for
  contrast).
- **Appearance.** Do not implement your own dark mode switch; the theme follows the system. If a screen draws custom
  elements, use only theme colors, or read `c.Theme().Dark` where a choice cannot be expressed with them, as the
  initial badge does. For derived colors use `Mix` and `Alpha` on theme colors rather than new constants.
- **Selection.** A selected list row (the accounts window) is filled with `SurfacePressed`, its text stays `Text`.
- **Accent.** One primary action per group uses `PrimaryButton`; the panel has none. Idset has no brand color and
  does not recolor the accent; it is the user's system accent.
- **Contrast.** Text and meaningful graphics should meet WCAG 2.1 AA: 4.5:1 for body text, 3:1 for large text and
  control boundaries. Test with the desktop's high contrast setting on (`c.Preferences().HighContrast`).
- **Status.** Color supplements a message; the message states the status in words. A warning or danger outline on a
  result card always comes with the sentence that explains it.
- **Mockups.** Colors in `design/screens/` approximate the default theme; the app running on a real device is what
  counts.

## Typography

- **System font.** Leave `Font` empty so text uses the platform's UI font. The default text size is 13 on macOS and
  14 elsewhere, and scales with the desktop's text size (`Preferences().TextScale`) when you rely on the theme. See
  [Text](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/text.md). Paths, SSH aliases and folders use
  `Font("monospace")`.
- **Size relative to the theme.** Use `t.Rem(n)` for sizes (`n` times the theme font size) rather than fixed numbers,
  so custom text follows the user's text scale. Widgets use the theme size.
- **The panel.** At the macOS default of 13:

  | Text | Size | Weight |
  |---|---|---|
  | Panel title "Idset" | body, 13 | 600 |
  | Identity name in the result card; read-only title | `Rem(1.15)`, about 15 | 600 |
  | Identity name in the list; drop zone title; empty title | body | 600 / bold |
  | Secondary text, hints, group heading | `Rem(0.9)`, about 12 | 400 (group heading 600) |
  | Paths, aliases and folders in the result card | `Rem(0.9)` monospace, about 12 | 400 |
  | Folders at the end of a list row; the read-only reason; "Copied" | `Rem(0.85)` (monospace for paths), about 11 | 400 |

  These replace the fixed 15, 12 and 11 used before sizes followed the theme.
- **Hierarchy with few steps.** Body, secondary (`TextMuted`, about 0.9 rem), section heading (600 or bold) and one
  larger size per screen. Weight and color before size. A card uses at most three text styles.
- **Headings are real headings.** Give the title of a screen or section `Role(ui.RoleHeading)` and a `Level` so
  assistive technology can navigate by them. In the panel the title is level 1; the "Identities" group heading, the
  read-only title and the empty title are level 2.
- **Do not truncate silently.** Prefer wrapping. Where a single line is needed (`SingleLine`, `MaxLines`), make the
  full text available in a tooltip or elsewhere on screen.
- **Numbers.** Counts that change in place can use `FontFeatures("tnum")`.

## Icons and images

- **Icons are SVG** shown with `ui.Icon`, which takes the text color and the font size: an icon follows theme colors
  and the text size automatically. MyGo has no built-in icon set and no SF Symbols: `ui.Icon` draws an SVG. The panel's symbols are 16×16 line
  drawings (1.5 stroke, round caps) defined in `internal/ui/panel/view.go` and parsed once at package level: clear,
  copy, check, warning, error (`exclamationmark.circle`: the info circle with its strokes turned upside down), folder,
  more (three dots), arrow, minus, pin, and for the title bar and footer plus, gear (`gearshape`), info (`info.circle`), refresh
  (`arrow.clockwise`) and power (`power`), named after the SF Symbols they follow; there is no third-party icon set. If one
  is adopted, use one set for the whole app and record it and its license in `THIRD_PARTY.md`. See
  [Icon](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/icon.md).
- **An icon needs a name when it is alone.** Icons are decoration for assistive technology unless given a `Label`.
  A button that shows only an icon must get a readable name with `.Label()`; an icon next to a visible text label
  needs none.
- **Size icons with the text.** Card symbols are `Rem(1.1)` (about 14) in `TextMuted`; the glyphs of the title bar and
  footer buttons are `Rem(1.2)` (about 16); the read-only warning is
  `Rem(1.7)` (about 22) in `Danger`.
- **The tray icon** is drawn in code by `internal/trayicon/`, so no image file is maintained by hand: a 32×32 PNG of
  an identity card. On macOS it is a template icon, black on transparent, that the system tints for the menu bar;
  elsewhere it is a mid gray that stays visible on light and dark taskbars. The worst entry of the diagnostics summary
  adds a badge at the lower right: a triangle for a warning, a circle for an error (and when ssh is missing), with a
  "!" in it, cut free of the card by a 1.5 px gap. Outside macOS the badge is `#d97706` or `#dc2626` with a white
  mark; in the template it is the ink with the mark cut out, since the system tints a template icon as a whole. The
  icon changes once a round of checks has finished, never while one runs, and does not flash.
- **The product icon** is still to be designed (see [design/README.md](../design/README.md)).

## Component guide

Pick the widget by the job. Each name links to the upstream page, which also says what assistive technology sees of
the widget. The complete list is the
[UI index](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/README.md).

What Idset uses today:

| Component | Use |
|---|---|
| `ui.Button` | Secondary actions. A button that shows only a symbol must give a readable name with `.Label()`. |
| `ui.PrimaryButton` | The main action of a group of buttons; at most one. In the panel: "Add Identity" of the empty state, and "Rebind" of the confirm block (never on screen together). |
| `ui.Text` | Text. Secondary explanations use `TextMuted`. |
| `ui.TextInput` | The identity filter, shown from three identities on, with the placeholder "Filter identities". Not `ui.SearchField`: its placeholder ("Search") is fixed English. |
| `ui.Spinner` | Reading the configuration and checking a folder. |
| `ui.Divider` | Between the title bar, content and footer; between the two parts of a result card; between list rows. |
| Drop zone | A `ui.Column` with a 1.5 DIP dashed `Border` outline (`BorderStyle(ui.BorderDashed)`), taking files with `DroppedFiles`; while `FileDragOver`, the outline turns `Accent` and the fill `Selection`. See [Dropped files](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/input.md#dropped-files). |
| Initial badge | A rounded square with the first letter of the identity's name ("?" without a name), in the identity's color (see [Color](#color-and-dark-mode)). |
| Borderless symbol button | `Space(6)` square (24), transparent, icon `Rem(1.1)` in `TextMuted`, `SurfaceHover` fill on hover, `SurfacePressed` while pressed. Must have a `.Label()`. After the path is copied, the icon turns into a check mark with "Copied" before it for 1.2 seconds. |
| Icon button (title bar, footer) | The same 24×24 button with a `Rem(1.2)` (about 16) glyph, radius `Theme.Radius`. Always has `.Label()` and a `.Tooltip()` with the same text, in both languages: Settings, Add Identity, About Idset, Re-check, Quit (设置 / 添加身份 / 关于 Idset / 重新检查 / 退出 in Chinese). The tooltip is the `Inverse` fill, no shadow, above the button (below the title bar's). Same size, 4 DIPs apart, so a missing button never moves another. |
| Result card | A context line (the path in monospace `TextMuted`, then the copy and clear buttons) and below it the answer. `Surface` fill, `Border` outline; `Warning` for the cards without a match, `Danger` when `git` fails; a matched card with findings takes the color of the worst one. |
| Bind row | In a result card: color dot, name and one line of `TextMuted` (the identity's first folder in monospace `Rem(0.85)`, like the identity list's folders, middle-truncated; "Has <path> · adds another" is its tooltip), "Bind" in `Accent` at the end. 44 DIPs high, hairline above. The whole row is the target. |
| Card menu | The "⋯" icon button's menu in a matched card, 224 DIPs wide, anchored to the button: its right edge is the button's, and it opens 4 DIPs below the card's context line: `Background` fill, `Border` outline, radius 8, 4 DIPs of padding, a 28 DIP heading "Rebind to" (`Rem(0.9)`, `TextMuted`, centered in its row and inset 8 DIPs like the entries), 32 DIP entries (color dot, name, first folder or "Global" in monospace `Rem(0.85)`), and under a hairline "Unbind" (only when the identity lists this very folder). No shadow. There is no actions row: open, copy, the menu and clear are icon buttons in the context line, in that order. |
| Confirm block | In a matched card, under the details, hairline above: one line ("<name> will apply only here") and two 80×28 buttons at the right, the plain one first ("Cancel"), then the filled one (`Accent`, "Rebind"). No extra identity row. |
| Overlap rows | In the overlap card: a row per identity in git's read order (dot, name, path in monospace `Rem(0.9)`), then the verdict: "Used" in `Accent` 600 with a check mark for the winner, "Overridden" in `TextMuted` for the others, whose names also lose their weight; and at the very end an icon button: a pin on the winner ("Use only <name> here"), a minus on each loser ("Remove <path> from <name>"). Title row: "<n> identities cover this folder" and an info icon button whose tooltip is "Git reads them in this order; the last one wins." |
| Stale-path prompt | A card of its own above the no-match card: a 40 DIP header (color dot, "<name>'s folder is gone" / 「<name> 的路径已不在」, the name in 600, one line), two 28 DIP path rows under hairlines with no labels (the old path in `TextMuted`, then the new path after an arrow, monospace, one line, middle-truncated, full text in the tooltip), and a 36 DIP row with two text buttons at the right, "Update" in `Accent` and "Ignore" in `TextMuted`; their tooltips are "Change to This Path" and "Keep Old Path". The height is the same for any path. Plain `Border` outline, no clear button. |
| Finding rows | In a matched card, under the details, in a section of their own: a hairline above, `Space(1)` (4) above and below the rows. One 28 DIP row per identity-check finding, email first, then key, like a path row: padding `Space(3)` (12), a `Rem(1.1)` (about 14) glyph, the warning triangle in `Warning` or the error circle in `Danger`, gap `Space(1.5)` (6), and one line of `Rem(0.9)` (about 12) `Text`: "Commits have no author", "Commits use <email> (<name>)", "Commits use <email>", "Pushes use <path>" (「提交没有作者」「提交会署名为 <邮箱>（<身份>）」「提交会署名为 <邮箱>」「推送会用 <路径>」). A long email gives way before its "@", so the domain and the identity name stay; a key path is middle-truncated in monospace like any path. The row's tooltip says why, with the full value: above the row, 6 DIPs away, its start under the row's start, at most 240 DIPs wide. No buttons: fixes belong to the diagnostics. The card's outline is `Danger` when any finding is an error, else `Warning`. |
| Connectivity dot | At the end of an identity row, after the folders: a 14 DIP slot holding a 6 DIP dot, `Success` when the last `ssh -T` passed, `Danger` when it failed, a 1.5 DIP `TextMuted` ring while the first check runs. Its tooltip and accessible name say it in words: "Connected · <platform> user <user>", "Check failed · <why>", "Checking the connection…". A row without an alias keeps the empty slot so the folders line up; before the first report there are no dots. |
| Problem bar | Under the row of an identity whose check failed, inside the same list, indented to the name (`Space(8)`, 32), 28 DIP line: the error circle in `Danger` (the globe in `TextMuted` for an HTTPS remote), one `Rem(0.9)` sentence (a missing key names its path in monospace, middle-truncated), the reason as its tooltip, and "Check Again" (the refresh icon button, disabled while the check runs). Below it, indented 20 more, at most one line of next step: "Unlock with Keychain" on macOS (an `Accent` text button); elsewhere, or after the unlock failed, the `ssh-add <path>` command with a copy button; "Copy Public Key" and "Open <platform> Settings" for a rejected key; "Add Rewrite" for an HTTPS remote; "Generate Key" for a missing key, which opens the Keys pane of the settings window filled in for the identity. No hairline between the row and its bar. |
| ssh notice | When the checks find no `ssh`, one card above the drop zone replaces every bar and the summary: `Danger` outline, the error circle, "ssh not found" (600), "Check Again" over all identities, and one muted line on how to install it for the platform. |
| Diagnostics summary | After the identity list, once a round of checks has finished. Nothing wrong: one muted line with a `Success` check mark, "Configuration OK". Otherwise a `Surface` card with radius 10: a 32 DIP header, a button with `aria-expanded` (the worst glyph, "1 error · 2 warnings" in 600, a chevron), and when open one entry per problem under hairlines: the glyph, what is wrong, how to look into it in `TextMuted`, an optional `Accent` text button ("Copy the Command" for NET001, "Regenerate" for GIT001), and the code (NET001, SSH001, GIT001, GIT002) in monospace `Rem(0.85)` at the end. It opens by itself whenever its set of entries changes, and stays as the user left it otherwise. |
| Undo bar | The last row of the result card, 36 DIPs high, hairline above: "Bound to <identity> ·" and an `Accent` "Undo" text button. A 2 DIP line on the bar's bottom edge, `Accent` at 45%, is as wide as the time left of the card's width, in fifths. |
| Identity list | A group heading, then one `Surface` container with `Border` dividers between rows. |
| Source list row | A color dot, name and email on two lines, an optional folder symbol; `SurfacePressed` fill when selected. |
| Color choice | A 14 DIP dot in the preset color with its name as `Label`; the chosen one has a ring and a check mark. |
| Settings row | Label left, value right, 32 DIPs high, hairline between rows inside a `Surface` group with radius 10. |
| Platform filter | A row of `ui.Button`s, one per platform in use when more than one is; the selected ones are `Accent` with `AccentText` and are `Selected(true)` so screen readers read them as selected. |

The tables below say what to reach for as the accounts, settings and about windows are added.

### Actions

| Need | Use |
|---|---|
| The main action of a group, once (Save in the accounts window) | [Button](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/button.md) (`PrimaryButton`) |
| Other actions | [Button](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/button.md) |
| Open the GitHub page from the about window | [Link](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/link.md) |
| "Rebind to" another identity | [Popover](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/popover.md) (`PopoverBase`) with entries of our own: a menu button's menu is the system's and cannot show a color dot and a monospace folder at 224 wide |
| Actions on a specific identity, on right click | [Context menu](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/context-menu.md) |
| Choose one of a few modes (global or limited scope) | [Segmented control](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/segmented.md) |
| Adding to a list ("Add Identity") | A symbol button in a bar at the bottom of the list; icon button (plus) in the panel footer |

### Choices

| Need | Use |
|---|---|
| A setting applied immediately (start at login, push on a URL rewrite) | [Switch](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/switch.md) |
| A yes/no value submitted with a form ("also delete the private key"), or several independent options (import candidates) | [Checkbox](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/checkbox.md) |
| One of two to five visible options (port 22 or 443) | [Radio](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/radio.md) |
| One of many options (platform, interface language) | [Select](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/select.md); with typing to filter, [Combobox](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/combobox.md) |
| An identity's color | The six presets as named choices; any other value with a [color well](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/color-picker.md) (`ui.ColorWell`) |

### Text

| Need | Use |
|---|---|
| Free text (name, email, alias, key path, notes) | [Text input](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/text-input.md) |
| Filtering a list with localized copy | [Text input](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/text-input.md) with a translated `Label` and `Placeholder` |
| Labeled controls with help and errors (the identity form) | [Form](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/form.md) |

### Collections

| Need | Use |
|---|---|
| Identities in the accounts window, backups, long lists of one kind | [List](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/list.md) |
| Rows with sortable, resizable columns | [Table](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/table.md) |

### Containers and navigation

| Need | Use |
|---|---|
| Scrolling content | [Scroll view](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/scroll.md) |
| Columns and rows of controls | [Grid](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/grid.md) |
| List and detail of the accounts window | [Split view](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/split.md) |
| Optional detail (the clone helper of an identity row, a backup's contents) | [Collapsible](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/collapsible.md), [Accordion](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/accordion.md) for several in a stack; only one identity row is expanded at a time, which the screen's state enforces |
| The four panes of the settings window (General, Keys, Backups, Configuration) | [Tabs](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/tabs.md); `ui.TabsBase` for the icon over the label |

### Dialogs, messages and status

| Need | Use |
|---|---|
| Confirm something that cannot be undone (deleting an identity, restoring a backup, removing the Include) | [Alert dialog](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/alert-dialog.md) |
| Choose a folder or a key file | `mygo.Dialog` ([desktop APIs](https://github.com/egoist/mygo/blob/v0.3.6/docs/native.md)) |
| A small panel tied to an element | [Popover](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/popover.md) |
| A short label or hint for a control | [Tooltip](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/tooltip.md) (never the only place for essential information) |
| Feedback that needs no decision | Text on the card or form that caused it ("Copied", "Saved"); a [Toast](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/toast.md) only where there is no such place |
| Unknown duration (reading, checking, `ssh -T`) | [Spinner](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/spinner.md) |
| A count or short status | [Badge](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/badge.md) |

Overlays share focus and Escape behavior; see
[Overlays](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/overlays.md). Rules of thumb:

- Prefer undo over confirmation. Binding a folder needs no confirmation and can be undone. Confirm only what cannot
  be undone, with a button named for the action ("Delete"), not "OK", and Cancel as the Escape action. The one
  exception is giving a global identity a folder, which changes what every other folder uses: it asks first, inside
  the result card, not in a dialog.
- Do not stack dialogs, and do not use a dialog for an error.
- Errors say what happened and what to do, on the card or form where they happened (`Error`, or a
  [form](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/form.md) field).
- Confirmations of copying and saving are a 1.2 second "Copied" or "Saved" next to the control; a repeat restarts
  the timer.

## The panel

Mockup: [design/screens/panel.html](../design/screens/panel.html), and both appearances side by side in
[design/screens/panel-appearance.html](../design/screens/panel-appearance.html).

From top to bottom: the title bar ("Idset", 13 / 600, a level 1 heading, and the Settings icon
button at the right; no close button: the tray icon and Escape hide the panel), the content, and the footer. Both bars are 36 DIPs high. The
title bar has `Space(4)` left and `Space(2)` right padding; the footer has `Space(2)` at both sides. The footer is four
icon buttons: "Add Identity" (plus) at the left, then at the right About (`info.circle`), Re-check (`arrow.clockwise`)
and Quit (`power`). The footer after the content:

- With identities, all of them. Without identities, loading or read-only: no Add (the empty state has its own button;
  the other two cannot write) and no Re-check (nothing to check); About and Quit stay, so the footer still has a way
  out when the read-only error takes the whole panel.
- Only buttons that work are shown: About joins when it is built (the mockup's "Built today" switch shows the footer
  without it). Re-check checks every identity again and waits while a check runs. Settings is always in the title
  bar.

The content, in order:

1. Loading: only a spinner and "Reading configuration…", centered.
2. `accounts.json` cannot be read: nothing else can be done, so the error fills the content and is centered
   vertically: a `Space(11)` (44) warning badge (radius 11), the title (`Rem(1.15)`, 600), the file path in
   monospace `Rem(0.9)` `TextMuted`, the parser error in monospace `Rem(0.85)` `TextMuted`, at most two lines (the full
   text is its tooltip), one short line "Read-only until fixed" (the sentence "Idset will not write to it until it is
   fixed or restored from a backup." is its tooltip), and three buttons: "Restore from Backup" (primary, 28×248, opens the Backups pane of the settings window), then
   "Show in Finder" (Explorer on Windows, the file manager on Linux) and "Read Again", 28×120 each on one row. There is no reason box.
3. Otherwise: the result card (when there is a result), the ssh notice (when `ssh` is missing), the one-line drop
   zone, the group heading "Identities" with the count, the filter (from three identities on), the identity list with
   its connectivity dots and problem bars, and the diagnostics summary. Without identities there is one centered block
   instead: "No identities yet" (`Rem(1.15)`, 600), one short line ("Give each folder its own Git and SSH identity.")
   and the filled "Add Identity" button (28 high); the drop zone sits under it, at the bottom of the content.

The drop zone is always one line, with or without a result: a dashed box with "Drop a folder, or" and "Choose…",
padding `Space(2)`×`Space(3)` (about 46 DIPs high). "See which identity pushes from it" is the tooltip of the text
and "Choose Folder…" that of the button. Words that only explain are tooltips throughout the panel: the full sentence of
"Covered by …", the reason of "Read-only until fixed", the "adds another" of a bind row, the why of a finding.

The result card has two parts with a divider between them:

- The context line: the folder (the repository root, or the dropped path when it is not a repository) in monospace,
  middle-truncated, then a short transient status ("Copied", "Updated", "Undone", "Covered by …"), the icon buttons of
  a matched card (open, copy, "⋯") and the clear button (not while checking). 36 DIPs high with its hairline, like
  the title row and the footer; padding `Space(2)` right, `Space(3)` left (8, 12).
- The answer, padding `Space(3)` (12). Checking: a spinner and "Checking…". Matched: a `Space(9)` (36) initial
  badge (radius 9), the identity's name and "Git uses this identity here", gap `Space(3)`; below them three rows,
  "Alias", "Email" and "Folders", with a label column `Space(14)` (56) wide, lined up with the name (left padding
  `Space(15)`, 60). Each label is top-aligned with the first line of its value, so "Folders" stays by the first
  folder however many follow. Without a match: the sentence for the case, 600, on a `Warning` outline. `git` failed: "Could not
  run git" in `Danger`, then the reason in monospace `Rem(0.85)` `TextMuted`, at most two lines (the full text is its
  tooltip), on a `Danger` outline. In "Folders" each path has its own line.
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
  - HTTPS row: "Remote is HTTPS", with one icon button at the end (`terminal` glyph, 24 x 24, 8 from the edge like
    the context line). The button's name is "Copy command to use SSH"; its tooltip is that name and the whole command
    (`git remote set-url origin git@github-work:org/repo.git`), so nothing on the card has to show a long mono
    string. Why it matters ("Pushes over HTTPS do not use this identity's SSH alias or key.") is the text's tooltip.
    After the copy the glyph is a check and "Copied" (`Rem(0.85)`, `TextMuted`) stands before it for 1.2 seconds.
  - Clone helper: padding `Space(2)`x`Space(3)`, bottom `Space(2.5)`. A 28 DIP input (monospace `Rem(0.9)`, `Background`
    fill, `Border`, radius 6) with the copy button (`terminal` glyph, the same glyph as the HTTPS row, since both copy
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
  - No match: the sentence, a "Bind to" heading (`Rem(0.9)`, 600, `TextMuted`) and one bind row per identity, then a
    last 40 DIP row with "New Identity" (an `Accent` text button with a plus; "New Identity and Bind Here" is its
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
    The card has the plain `Border`, since nothing is wrong. It shows only while the winner has no path of the folder's own; once it has one (after "Use only" or a rebind) the choice is explicit and the card is gone.
  - Stale path: when the dropped folder looks like a renamed or moved `gitdir` path that no longer exists, a card of
    its own sits above the no-match card, the same way the overlap card sits above the matched one. The header is
    one line and the paths are middle-truncated, so the card's height never changes. The no-match card below is
    unchanged, so the bind list still works; in its bind rows the middle of the first folder gives way to an
    ellipsis and the rest stays: "~/…/payments-gateway/". Mockup states: rename, move (a long old path), after "Change to This Path" and after
    "Keep Old Path".
    - "Update" (tooltip "Change to This Path"): the card is gone and the folder matches, so the matched card shows with "Updated" next
      to the copy button for 1.2 seconds, like "Copied". There is no undo bar: only bind, rebind and "Use only" have
      one (docs/product.md), and the old path is gone from disk anyway.
    - "Ignore" (tooltip "Keep Old Path"): only the prompt goes; the plain no-match card stays and shows no acknowledgement.
  - Undo: after a bind, rebind or "Use only", the undo bar is the last row of the matched card. The countdown line
    shortens once a second (`c.After` from the absolute deadline), five steps; no animation. After "Undo" the card
    returns to what it was, with "Undone" next to the clear button for 1.2 seconds, like "Copied". Unbinding shows
    no bar.

Identity list rows: padding `Space(2.25)`×`Space(3)` (9×12), a 10 DIP color dot (the same as in the accounts window's source list), gap
`Space(2.5)` (10), the name and the email on two lines, and at the end the folders: the first one, "+N" when there
are more, "Global" for a global identity, "no folders" without any. A long first folder is middle-truncated (at
most half the row) and " +N" always stays; the name and email are single lines too.

An identity with an alias ends in a 24 x 24 disclosure button ("Clone with this identity": a chevron pointing right
when closed, down when open; `aria-expanded`); a row without an alias keeps the 24 DIPs empty so the folders stay
lined up. The row's body still opens the accounts window; only the button expands. Open, the row is followed, inside
the same list and without a hairline between them, by the clone helper described above, indented to the name
(`Space(8)`, 32) and with no remembered prefix shown as the placeholder "owner/repo". Only one row is open: opening
another closes the first. Nothing else about the row changes while it is open.

A path on one line is shortened in the middle only, the same way everywhere (the stale-path prompt's rows, the first
folder of a bind row, the folder of an identity row, the context line of a result card): its start ("~/", "/" or a drive such
as "C:/") and its last folder always stay, and what lies between gives way to an ellipsis, "~/…/payments-gateway/".
Only when even those do not fit is the last folder cut at its end. The full path is in the tooltip.

Identity rows open the accounts window on their identity. "Add Identity" (the plus icon button) sits at the left of the
fixed footer, so that it stays reachable however long the list is: the content between the title row and the footer is
the scroll view. In the empty state the centered "Add Identity" button
replaces it. Both open the accounts window on a new draft.

Planned (product.md): the About button (drawn in the mockup, shown once it works); "Import" when there are no
identities. The expand arrow and the HTTPS row and clone
helper are drawn in the mockup and shown once built (issue 26). Bind, unbind, rebind, "Use only" and undo on result
cards are built as the mockup shows.

## The accounts window

Mockup: [design/screens/accounts.html](../design/screens/accounts.html), every state in light and dark and in both
languages. Behavior is in product.md, "账户窗口" and "删除确认". The model is a native grouped-settings window
(System Settings, Mail accounts): few borders, one hairline between rows, hierarchy from position and the muted
color rather than from boxes. Everything is a fill, a hairline or text: no shadow, gradient or blur.

The window is 760×520. It has the system title bar and no toolbar: below the title bar, a source list and a detail
pane.

- **Source list** (left, `Space(58)`, 232, `Surface` fill, hairline to the detail). Group heading "Identities" with the
  count (11, `TextMuted`), then rows in the panel's order: `Space(1.5)`×`Space(2)` padding, radius 6, a 10 DIP color
  dot, the name (13, regular) and the email (11, `TextMuted`) below. Text wraps. An identity limited to folders has
  a folder symbol at the end of its row, named "Scoped" for assistive technology; a draft shows "New" there. The
  selected row is filled with `SurfacePressed` and keeps `Text`: calm, not an accent slab. The list scrolls above an
  add bar fixed to the bottom edge of the sidebar: a hairline, then a borderless "+" symbol button named "Add
  Identity" (also its tooltip), `Space(11)` (44) high with the button centered.
- **Detail** (right, scrolls on `Background`). Padding `Space(4)`×`Space(5)` (16×20), `Space(5)` (20) between groups.
  Header: a 12 DIP color dot, the title (`Rem(1.15)`, 600, level 1), "Global" or "Scoped" (11, `TextMuted`), and at the
  end the six preset colors as 14 DIP dots (the chosen one ringed and ticked, each named for assistive technology)
  and the hex value.
- **Groups.** A section title (11, `TextMuted`, regular, level 2) over an inset `Surface` group with radius 10: Identity,
  SSH, Scope, URL rewrites, Notes, Times (not for a draft). A row is 32 DIPs high with `Space(3)` (12) side padding:
  the label on the left (a fixed label column, so rows line up), the value on the right, right-aligned. Rows are
  separated by a hairline inset 12 from the left. Values are plain text, not boxed text fields; a focused one shows
  the focus ring around its text. Paths, aliases, host names, ports and URLs are monospace 12 and are sized to fit
  one line. A footnote under a group (11, `TextMuted`) carries help; an error there is `Danger` with the warning
  symbol.
- **Controls in rows.** Platform is a select (value and an up-down symbol). Port is two radios (22, 443); Custom shows a
  host name and a port value. Scope is a row "Applies to" with a segmented control, "Global" and "Limited to
  folders"; its footnote says when a global identity applies. The private key row is the path in monospace and a folder symbol button that opens the file picker; a path too long for the
  field is shown shortened in the middle ("~/.ssh/keys…_personal_laptop") until it is clicked, and its tooltip
  holds all of it. A limited identity lists one row per folder (path,
  choose-folder button, a minus button, disabled for the last one) and a text row "+ Add Path". A URL rewrite is one
  row: the `from` URL, and under it `→` and the `to` URL, then "Push only" with a switch and a minus button; then the
  text rows "+ Add Rewrite" and "+ Add Platform Default Rewrite" (disabled without an alias, the reason in the
  footnote). Text rows are `Accent` text with a leading plus, never boxed.
- **Hints.** A prefilled signature name and email carry "From git config --global" (11, `TextMuted`) at the end of their
  row, right of the value; the row keeps the standard 32 height. A long value is cut before it reaches the hint. The
  hint goes as soon as the field is edited.
- **Scrolling.** The detail is a scroll view above the fixed footer; the header scrolls with the groups. The source
  list scrolls above its add bar.
- **Footer.** Hairline above, `Space(11)` (44) high like the sidebar's add bar so the two hairlines are one line, side padding
  `Space(5)`, everything centered vertically. Delete, Revert and Save are `Space(7)` (28) high; Revert and Save are `Space(25)`
  (100) wide, enough for "✓ Written", "Saving", "已写入" and "保存中", so no label change resizes or moves anything. At the left "Delete" as plain `Danger` text (existing
  identities only), then the status (11, `TextMuted`), then "Revert" (bordered) and "Save", the only filled button.
  Status: "Unsaved changes", or "New identity, not saved yet" for a draft; a spinner with "Writing…" while saving,
  when the form is dimmed and disabled; "Written" with a check mark on the Save button for 1.2 seconds after
  success; the reason in `Danger` with the warning symbol, wrapping, when the write failed (Save stays available).
  Save and Revert are disabled while there is nothing to save.
- **Empty.** No list and no footer: "No identities yet" (`Rem(1.15)`, 600), one sentence, "Import Existing
  Configuration" and "New" (primary), centered, with no add bar: "New" is the only way to add.
- **Delete confirmation.** An alert over the dimmed window, 360 wide, `Space(5)` padding, an outline and no shadow:
  "Delete identity <name>?" (`Rem(1.15)`, 600), a sentence, the private key path and the folders that are not
  deleted automatically between two hairlines (no box), a line that project folders are never touched, the "Also
  delete the private key" checkbox (unchecked by default) and the buttons Cancel (bordered; the Escape action and
  initial focus) and Delete (`Danger` fill, white text). When the key cannot be deleted (no key path, another
  identity uses it, a symbolic link, missing, not a regular file) the checkbox is disabled and the reason is
  written under it; when checked, a line says that only the private key is deleted and `.pub` stays.
- **Type.** 13 body, 11 secondary and footnotes, one larger size (`Rem(1.15)`, 600) for titles; nothing else is bold.

## The settings window

Mockup: [design/screens/settings.html](../design/screens/settings.html), every state in light and dark and in both
languages. Behavior is in product.md, "设置窗口". 720×480, the system title bar ("Idset · Settings"), same
material as the accounts window: fills, hairlines and text, no shadow, gradient or blur.

- **Tabs.** A tab bar along the top, under the title bar and part of the same `Surface` band, one hairline at its
  bottom. Four tabs centered as a group, each an icon over a label: General, Keys, Backups, Configuration (通用 / 密钥 /
  备份 / 配置). A tab is 88 wide and 52 high (`Space(22)`, `Space(13)`) with a 20 DIP icon (`Space(5)`), the 11 label 4 below it,
  and 4 DIP gaps; the bar has 4 above and 6 below. The selected tab has the `Pressed` fill, `Text` icon and label and a
  6 DIP radius; the others are `TextMuted` on no fill. Arrows, Home and End move the choice, as `ui.Tabs` does. The
  pane has no heading of its own: the selected tab names it.
- **Pane.** Full width below the bar, 16 above and 20 at the sides, scrolling; its content is at most 560 wide and
  centered, so wider windows do not stretch the rows.
- **General.** One grouped list of 32 DIPs rows. Language is three tick rows: "Follow System" (shows the language it
  resolves to), "English" and "简体中文", the last two always in their own language. "Open at login" is a switch row; when
  the system asks for approval the switch stays off and a `Warning` note under it says where to approve (macOS: System
  Settings, General, Login Items). A failed change is an inline `Danger` note under the row.
- **Keys.** The form is one grouped list "New key" of 32 DIP rows, a 168 DIP label at the start and the control at
  the end: Type (a segmented control, Ed25519 / RSA 4096), Platform (a pop-up), File name (monospace) and Comment
  (borderless inputs, text at the end). Under it a muted note says where the two files go and that an existing file is
  never overwritten; a bad name or a failure replaces it with an inline `Danger` note. "Generate" is the primary
  button at the end; while it runs the group is dimmed to half and a muted "Generating…" and a spinner stand before
  it. Opened from a missing key's bar, a `Surface` banner with the key icon above the form names the identity and its
  key path. The result has a heading with an `Accent` check, "Key created" and a muted "type · platform", then three
  groups: "Public key" (the selectable monospace key, "Copy Public Key" as an `Accent` text button that turns into a
  check and "Copied" for 1.2 s), "Files" (private key with "Show in Finder" as a ghost button, public key,
  fingerprint, passphrase "None", 96 DIP labels, a footnote with the `ssh-keygen -p` command), and "Next steps" (add the
  public key on the platform with "Open <platform> Settings" and the external icon; "Load into ssh-agent" with Load,
  a spinner, then a check and "Loaded"; "Use for identity" with "Choose…", then the identity's dot and name and
  "Change…"). Notes under Next steps: the identity written, ssh-agent not running (`Warning`), or the platform hint.
  "Generate Another" is a text button at the end. "Choose…" opens an alert 360 wide: "Use this key for which
  identity?", one sentence, the identities as radio rows (dot, name, muted email, and "Now: <key>" or "No key yet"
  under it) and "New identity" with the email and platform it starts from, between two hairlines, a muted note,
  Cancel (initial focus) and Use.
- **Backups.** One grouped list, newest first. A row is 44 high: a chevron, the time (13), and a muted line "N identities
  · names · size" (11; the first three names and "+N" for the rest), then two ghost buttons in fixed 24 DIP slots:
  Restore (a back arrow, tooltip "Restore") and trash. The chevron expands the row to show its identities read-only, one at a time. Delete has no confirmation: the
  backup is a copy and the newest 10 are kept anyway. A backup that cannot be read has its sub-line in `Danger`, and
  Restore and the chevron disabled. Under the list, a note says when backups are made (before an identity is added,
  before a restore), that the newest 10 are kept and where. With no backups that note is the empty state.
- **Restore.** An alert over the dimmed window, 360 wide, like the delete confirmation: "Restore backup <time>?", one
  sentence (the current `accounts.json` is backed up first, the configuration is regenerated), the identities that will
  exist afterwards between two hairlines, a line that identities not in the backup are removed, Cancel (Escape,
  initial focus) and Restore (`Accent` fill: it replaces data but is undoable through the backup it makes). While it
  runs the Restore slot is a spinner with a muted "Restoring…" before it, and the other rows are dimmed to half; on
  success the slot is an `Accent` check with a muted "Restored" before it for 1.2 s; on failure the reason is an inline `Danger` note under the row. When the window is opened from the
  read-only panel the Backups pane opens with a `Danger` banner explaining why.
- **Configuration.** The two files with a marker block (`~/.ssh/config`, `~/.gitconfig`) and whether it is present, and
  the row "Remove Include (keep accounts.json)" in `Accent` text. Its alert (same shape, primary button) says in
  one sentence what changes and that the blocks come back at the next identity change, then two `Surface` boxes with a
  small muted heading: "Will be removed" lists the marker block in each file, one per line behind a `Danger` minus;
  "Left as is" lists `accounts.json` and the identities, the managed files under `~/.config/idset/`, and private keys
  with the rest of the user's config, each behind a muted check. Paths are monospace. Afterwards the rows read "no
  marker block" with a "✓ Removed" note. A failure is inline and names the file that could not be changed, including the
  case where one file was changed and the other not.
- **Errors and feedback.** Errors are inline notes, never dialogs; the only alerts are the two confirmations above and the identity choice of the Keys pane.
  Success feedback is the 1.2 s label of the panel.

## Windows, menus and the tray

- **The panel window.** 340×480, frameless, not resizable, minimizable, maximizable or full-screen, always on top and
  left out of the taskbar. It opens below the tray icon, centered on it and 6 DIPs away, or above it when there is no
  room, kept inside the work area of that display. It does not hide when it loses focus: the user clicks the file
  manager to start dragging a folder, which takes the focus away ([ADR 0003](adr/0003-drop-zone-in-tray-window.md)).
  The tray icon and Escape hide it. See
  [Windows with native UI](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/windows.md) and the
  [window options](https://github.com/egoist/mygo/blob/v0.3.6/docs/windows.md).
- **Without a tray** (Linux without AppIndicator), the panel is an ordinary window with the system title bar and the
  same content; Escape does not hide it, and closing it quits.
- **Planned windows** (accounts 760×520, settings 720×480, about) keep the system title bar, set `MinWidth` and
  `MinHeight` so content never clips, and hide the panel when they open. A hidden title bar
  (`TitleBarStyle: mygo.TitleBarHidden`) needs a mockup that shows why.
- **The tray.** A click opens the panel and a second click hides it. The right-click menu has "Open Idset" (or "Hide
  Idset" while the panel shows) and "Quit"; the accounts window, settings and about join it when they exist. On Linux
  AppIndicator reports no clicks, so a click opens that menu and it is the only way to open and hide the panel. The
  tooltip is "Idset", "Idset · <identity>" after a match, or "Idset · no match". See
  [Menus and the tray](https://github.com/egoist/mygo/blob/v0.3.6/docs/menus.md#tray-icons).
- **Menus.** The macOS menu bar has an app menu (About, check for updates, Quit) and the Edit menu (`RoleEditMenu`)
  so copy, paste and the other editing shortcuts work in text fields. Use roles (`RoleAbout`, `RoleQuit` and the
  others) for standard items: they carry the right label and shortcut. Give custom items an accelerator with
  `CmdOrCtrl`. See [Menus](https://github.com/egoist/mygo/blob/v0.3.6/docs/menus.md).
- **One instance.** A second launch opens the panel of the running instance and exits.
- **Notifications.** System notifications only for what the user would want while the panel is hidden; the first
  launch says once that Idset lives in the tray, without taking the focus.

## Accessibility checklist

MyGo exposes native UI to VoiceOver, Narrator, NVDA and Orca; see
[Accessibility](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/accessibility.md).

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
- [ ] Pointer targets are at least about 24×24 DIPs.
- [ ] `ui.NewTester` tests click controls by their accessible names, so a missing name fails the test, and check
      announcements with `Announcements()` (see
      [Testing](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/testing.md)).

Contrast, scaling, keyboard use and screen readers are verified on real devices; a passing headless test does not
count as accessibility acceptance.
The [Inspector](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/inspector.md) (F12 in development builds) shows
the element tree, roles and accessible names of a running window, which helps find a missing name before that check.

## Motion

Idset has no custom animation: state changes take effect immediately. See
[Transitions](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/transitions.md) and
[Drawing and animation](https://github.com/egoist/mygo/blob/v0.3.6/docs/ui/drawing.md) for what the toolkit offers.

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
- English labels: short imperatives. Buttons and menu items that show text use title case, as macOS does ("Choose
  Folder…", "Read Again", "Hide Idset"); accessible names of symbol buttons, headings and sentences use sentence case
  ("Copy the full path", "No identities yet"). Explanatory hints that are full sentences end with a full stop;
  headings, labels and one-line hints do not. Chinese sentences use full-width punctuation. Do not concatenate sentence fragments, since word order differs; use full-sentence strings
  with placeholders ("Git uses %s here").
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
<https://github.com/egoist/mygo/tree/v0.3.6/docs/ui>, together with `docs/native.md`, `docs/menus.md` and
`docs/windows.md` from the same version. The text here is a summary in our own words; the upstream pages are the
source for APIs and behavior. See also [THIRD_PARTY.md](../THIRD_PARTY.md).
