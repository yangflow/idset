# 0003 The drop zone lives in the window the tray icon opens

- Status: accepted
- Date: 2026-10-12
- Related: [docs/product.md](../product.md), "Dropping a folder" (拖入文件夹); [ADR 0001](0001-pure-go-native-ui.md)

## Context

The most direct gesture is to drag a project folder onto the tray icon to see which identity that repository uses.
In MyGo v0.3.5 a tray icon has only click and right-click events and does not accept drops; only a window receives
dropped files (`OnFileDrop`).

## Decision

A click on the tray icon opens a small frameless window (the "panel") below the icon, placed as in MyGo's menu bar
example. The panel has a drop zone; a folder dropped into the panel starts matching. The drop zone is most prominent
when there is no result; once there is one it shrinks to a one-line hint, with the result card above it.

## Cost

One more step: open the panel first, then drop. The panel cannot be opened while a folder is being dragged, so it has
to be opened before.

The panel cannot hide when it loses focus: the user has to click Finder or Explorer to start dragging, and that click
takes the focus away. So the panel floats above other windows and hides only on another click on the tray icon or on
Escape (on Linux, with "Hide Idset" in the tray menu). The cost is that the panel covers part of the screen until the
user puts it away.

## When to revisit

MyGo's tray icon supports drops. Dropping onto the icon can then be added as another way in, and the drop zone in the
panel stays.
