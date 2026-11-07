# Design

Idset's design is kept in two claude.ai artifacts and two files in this folder. The design language is
[docs/design.md](../docs/design.md); screen behavior is in [docs/product.md](../docs/product.md) (in Chinese). A design
approximates the result; the app running on a real device is what counts.

## Design artifacts

| Artifact | What it holds |
|---|---|
| [Idset design system](https://claude.ai/artifact/NFxWM1yWYqsYye5kxtNy7r) | The tokens of `tokens.json`, the principles, and the components by layer (atoms, controls, rows, cards, windows) with a preview each, and the Logos group |
| [Idset screens](https://claude.ai/artifact/2cP14kKAaf6tajDXeJV555) | A canvas with every state of every screen, in English and the light appearance, one page per screen; it uses the design system |

Both are private to the maintainer; ask for access when you need it.

## Files

| Path | Purpose |
|---|---|
| `tokens.json` | Every value the UI uses, named by layer: color (light and dark), type, spacing, radius, shadow, size, motion, opacity, plus the contrast audit and the values that still differ from the code (`audit.pending`) |
| `layout.json` | The layout patterns (rows, stacks, insets, bars, splits) and each window's size and material |
| `icon.svg` | The product icon's source |
| `export-icon.sh` | Renders `icon.svg` to `resources/icon.png` with Chrome |

`tokens.json` is the source; the design system's tokens are copied from it. Spacing is in pixels at MyGo's default
unit of 4: `space-3` (12) is `t.Space(3)` in code. Colors without a MyGo theme role are in `internal/ui/theme`.

## Screens: canvas page → package

| Canvas page | Boards | Code |
|---|---|---|
| Panel | list, drop a folder, bind and undo, moved folders, what Git sees, HTTPS and clone, diagnostics, light and dark | `internal/ui/panel/` |
| Accounts | editor, saving, delete | `internal/ui/accounts/` |
| Settings | general, new key, key in use, backups, import, managed config | `internal/ui/settings/` |
| About | about | `internal/ui/about/` |

Each board's frames are named by state, and those are the states the package's `Screen` can be in. The words are the
strings of the package's `copy.json` (English and Simplified Chinese); the tests in the package cover each state.

## Changing the UI

A UI change starts from the design; the maintainer confirms the visual direction before UI code is written.

1. Change or add the frames on the screens canvas, every state, and the tokens in `tokens.json` (and the design
   system) and `docs/design.md` if a rule changes. Run the contrast audit again when a color changes.
2. The maintainer confirms the visual direction.
3. Write the native view in `internal/ui/<screen>/` with theme roles and `Space`/`Rem` sizes, localized copy and
   accessible names, plus tests. When a screen moves to a value in `audit.pending`, take it off the list.
4. Run `scripts/check.sh --cross`, then the design acceptance checklist in `docs/design.md` on real devices. A
   headless test is not device acceptance.

Tokens, design notes and code go in the same pull request; the pull request body links the canvas frames it changes.

## Icons

The tray icon is drawn in code by `internal/trayicon/`; change it there and run `go test ./internal/trayicon/`. The
UI's glyphs are the SVGs in `internal/ui/icons/`, 16 x 16 line drawings at a 1.5 stroke.

The product icon's source is `icon.svg`. After changing it, run `design/export-icon.sh` (it needs Google Chrome, or
a Chromium-based browser given as `CHROME`) and commit `resources/icon.png` with it; `mygo build` makes every
platform's icon from that PNG, and the about window shows it. MyGo does not draw SVG filters, which is why the app
uses the PNG.

The design system's Logos group holds the icon and its mark (one color, and in color for light and dark grounds), all
on the same 100 grid as `icon.svg` and `internal/trayicon/`. When the icon changes, upload the new files there and
keep the `logo-*` tokens the same in both `tokens.json` files.

Screenshots go in the pull request body, not into the repository; `design/screenshots/` is ignored by git. The one
exception is the README images in `docs/images/`, taken from demo identities on `example.com` in a throwaway home
directory, never from a real one.
