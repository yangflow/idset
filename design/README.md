# Design

Mockups for Idset's screens. The design language (colors, type, spacing, components) is in
[docs/design-system.md](../docs/design-system.md); screen behavior is in [docs/product.md](../docs/product.md).

## What is here

- `screens/<screen>.html`: one self-contained HTML mockup per screen, matching the package
  `internal/ui/<screen>/`. Today there is `screens/panel.html` for the tray panel. Mockups approximate the
  result; the app running on a real machine is what counts.

## Preview

Open a mockup in a browser. The buttons on the left switch state, light or dark scheme and language; the left and
right arrow keys step through the states. The address bar follows as `?state=…&scheme=light|dark&lang=en|zh`, so
a link opens a specific state. New mockups work the same way and show every state.

## Icons

There are no icon files to regenerate. The tray icon is drawn in code by `internal/trayicon/`; change it there and
run `go test ./internal/trayicon/`.

## Changing a screen

A UI change starts from the mockup here; the maintainer confirms the visual direction before UI code is written.
The mockup, `docs/design-system.md` and the code change go in the same pull request. Screenshots go in the pull
request body, not in the repository (`design/screenshots/` is ignored).
