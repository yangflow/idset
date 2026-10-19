# 0005 Dialogs open in child windows, menus are the system's

- Status: accepted
- Date: 2026-10-19
- Related: [docs/design.md](../design.md), "Dialogs, messages and status"; [design/tokens.md](../../design/tokens.md),
  "Material"; issue 51

## Context

The windows sit on the system's material, which shows the desktop behind them. Dialogs and menus were drawn inside
the window with MyGo's glass material. A material drawn by the window can only blur what that window draws: over a
window that is mostly transparent it blurred nearly nothing and looked gray and dirty, nothing like the system's
sheets and menus.

## Decision

A dialog is a window of its own: frameless, not resizable, modal and a child of the window it belongs to, on the
system material (`Popover` on macOS, `Acrylic` on Windows 11, the solid fill elsewhere). The screen keeps the dialog's
state and draws its content; `dialog_window.go` opens the window when the dialog opens, sizes it to the content,
centers it over the parent and closes it when the dialog closes. It starts all but transparent, because a hidden
window draws no frames to measure, and shows once it has its height.

The card's "⋯" menu and the select pop-ups use the system's menu (`Element.Menu`).

## Cost

- The rebind menu loses its color dots and folders: a system menu shows text only.
- A dialog window has a moment before it shows while its height is measured.
- There is no scrim. On macOS the parent shows its inactive look while the dialog is open.
- Headless tests draw the dialog in the screen's tester, not in a window; the window itself is checked by hand.

## When to revisit

MyGo can blur what is behind a window from inside it, or a menu or sheet API gains the system material with custom
content.
