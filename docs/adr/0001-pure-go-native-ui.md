# 0001 Pure Go native UI

- Status: accepted
- Date: 2026-10-12
- Related: [docs/product.md](../product.md), "Technical baseline" (技术基线)

## Context

Idset manages several Git and SSH identities on one machine. The problem is the same on macOS, Windows and Linux:
`~/.ssh/config`, `~/.gitconfig`, `includeIf "gitdir:"` and the behavior of `ssh` are essentially the same on all
three. Idset wants a single language, reproducible builds and a truly native window, with one code base for the
three platforms. References: [MyGo native UI docs](https://mygo.egoist.dev/docs/ui) and
[the source at the pinned version](https://github.com/egoist/mygo/tree/v0.3.5).

## Decision

Use MyGo native UI and one Go module, with the command-line tool pinned in `go.mod` with `tool`. The app lives in a
tray icon, in the menu bar (macOS) or the notification area (Windows, Linux). Updates use the native updater plugin.

## Cost

Controls are drawn on the GPU, so accessibility and behavior on each platform have to be tested on real devices. The
tray icon does not accept drops, so dropping can only happen in a window (see
[ADR 0003](0003-drop-zone-in-tray-window.md)). The tray on Linux depends on AppIndicator; without that library the
app can only open an ordinary window. MyGo is at an early version; every upgrade needs a look at API and packaging
changes.

## When to revisit

A required platform capability or accessibility requirement cannot be met by native UI.
