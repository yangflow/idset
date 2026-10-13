# Design

Mockups and design notes for Idset's native UI. The design language is [docs/design.md](../docs/design.md); the
values it uses are in [tokens.md](tokens.md); screen behavior is in [docs/product.md](../docs/product.md) (in
Chinese). Mockups approximate the result; the app running on a real device is what counts.

## Layout

| Path | Purpose |
|---|---|
| `screens/<screen>.html` | One self-contained mockup per screen with every state: static HTML and CSS with a small script for switching states, no external resources |
| `screens/accounts.html` | The accounts window: every state, light and dark, English and Chinese, switched like the panel |
| `screens/panel-appearance.html` | The panel side by side in light and dark, with the drag-over state and a focus ring |
| `tokens.md` | Spacing, size, color and motion values, mapped to MyGo theme roles, plus identity colors |

The tray panel (`screens/panel.html`) is implemented; the accounts window has its mockup (`screens/accounts.html`)
and no code yet. The settings and about windows get their mockups before their code.

## Preview

Open a file in `screens/` in a browser. In `screens/panel.html` the buttons on the left switch state, light or dark
scheme and language; the left and right arrow keys step through the states. The address bar follows as
`?state=…&scheme=light|dark&lang=en|zh`, so a link opens a specific state. New mockups work the same way and show
every state. `screens/panel-appearance.html` needs no parameters.

## Screens

| Screen | Design file | Code |
|---|---|---|
| Panel | `screens/panel.html`, `screens/panel-appearance.html` | `internal/ui/panel/` |
| Accounts | `screens/accounts.html` | `internal/ui/accounts/` (planned) |

A mockup maps to a package: `screens/<screen>.html` is implemented by `internal/ui/<screen>/`. The text and states in
the mockup are the strings in that package's `copy.json` (English and Simplified Chinese) and the states its `Screen`
can be in; the test in the package covers each state.

## Changing the UI

A UI change starts from the mockup in design/; the maintainer confirms the visual direction before UI code is
written.

1. Edit or add the mockup in `screens/` (light and dark, every state, both languages), and update `tokens.md` and
   `docs/design.md` if a rule changes.
2. The maintainer confirms the visual direction.
3. Write the native view in `internal/ui/<screen>/` with theme roles and `Space`/`Rem` sizes, localized copy and
   accessible names, plus tests.
4. Run `scripts/check.sh --cross`, then the design acceptance checklist in `docs/design.md` on real devices. A
   headless test is not device acceptance.

Mockup, design notes and code go in the same pull request.

## Icons

There are no icon files to regenerate. The tray icon is drawn in code by `internal/trayicon/`; change it there and
run `go test ./internal/trayicon/`. The panel's symbols are inline SVG in `internal/ui/panel/view.go`. The product
icon is still to be designed; when it lands, its source and the script that exports it will live here.

Screenshots go in the pull request body, not into the repository; `design/screenshots/` is ignored by git.
