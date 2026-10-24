package main

import (
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/ui/theme"
)

// dialogScreen is a screen whose dialogs show in a window of their own.
type dialogScreen interface {
	View(c *ui.Context)
	DialogOpen() bool
	Dialog(c *ui.Context) ui.Element
	CloseDialog()
}

// dialogWindow shows the dialog of a screen in a frameless, modal child window of the screen's window, so that the
// system's material shows through it as through the window: drawn inside the window, it could only blur what the
// window draws (issue 51). The screen keeps the dialog's state; the window follows it, opening as the dialog opens
// and closing as it closes.
type dialogWindow struct {
	screen dialogScreen
	parent *mygo.Window
	win    *mygo.Window
	// syncing is set while a sync waits for the main loop; shown once the window has its height.
	syncing, shown bool
}

// view is the content of the screen's window: the screen, then a sync when the dialog opened or closed.
func (d *dialogWindow) view(c *ui.Context) {
	d.screen.View(c)
	if d.parent != nil && !d.syncing && d.screen.DialogOpen() != (d.win != nil) {
		d.requestSync()
	}
}

// requestSync has the screen's window follow the dialog's state on the next pass of the main loop, and redraws it.
func (d *dialogWindow) requestSync() {
	if d.syncing {
		return
	}
	d.syncing = true
	d.parent.Update(d.sync)
}

func (d *dialogWindow) sync() {
	d.syncing = false
	open := d.screen.DialogOpen()
	switch {
	case open && d.win == nil:
		d.open()
	case !open && d.win != nil:
		d.win.Close()
	}
}

// open creates the dialog's window over the middle of the parent. It starts all but invisible, as tall as a guess,
// and shows once its first frames measured the dialog's height.
func (d *dialogWindow) open() {
	px, py := d.parent.Position()
	pw, ph := d.parent.Size()
	d.shown = false
	w := mygo.NewWindow(mygo.WindowOptions{
		X: px + (pw-theme.DialogWidth)/2, Y: py + ph/4, Width: theme.DialogWidth, Height: ph / 2, Opacity: 0.01,
		Frameless: true, DisableResize: true, Parent: d.parent, Modal: true, Vibrancy: dialogMaterial(),
		Content: ui.View(d.dialogView),
	})
	d.win = w
	w.OnClosed(func() {
		if d.win != w {
			return
		}
		d.win = nil
		d.screen.CloseDialog()
		d.parent.Update(func() {})
		d.parent.Focus()
	})
}

// dialogView draws the dialog; the window takes its height and closes once the dialog did.
func (d *dialogWindow) dialogView(c *ui.Context) {
	theme.Apply(c)
	pane := d.screen.Dialog(c)
	w := d.win
	if w == nil {
		c.AnimationFrame()
		return
	}
	// Every action of a dialog closes it, and the screen's window shows what it changed once the dialog is gone.
	if !d.screen.DialogOpen() {
		d.requestSync()
		return
	}
	cw, ch := w.ContentSize()
	h := int(pane.Bounds().H + 0.5)
	switch {
	case h == 0:
		c.AnimationFrame()
	case h != ch || !d.shown:
		d.parent.Update(func() {
			if d.win != w {
				return
			}
			w.SetContentSize(cw, h)
			if !d.shown {
				d.shown = true
				px, py := d.parent.Position()
				pw, ph := d.parent.Size()
				w.SetPosition(px+(pw-cw)/2, py+(ph-h)/2)
				w.SetOpacity(1)
				w.Focus()
			}
		})
	}
}

// update redraws the dialog after work in the background changed the screen.
func (d *dialogWindow) update() {
	if d.win != nil {
		d.win.Update(func() {})
	}
}

// dialogMaterial is the material of dialogs: the popover material on macOS, Acrylic on Windows 11. Linux has none, and
// the dialog draws the solid fill.
func dialogMaterial() mygo.Vibrancy {
	if runtime.GOOS == "windows" {
		return mygo.VibrancyAcrylic
	}
	return mygo.VibrancyPopover
}
