package main

import (
	_ "embed"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"
	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/ui/about"
)

// The about window has a fixed size.
const aboutWidth, aboutHeight = 320, 292

//go:embed resources/icon.png
var iconPNG []byte

// productIcon is the product icon, decoded the first time the about window opens.
var productIcon = sync.OnceValue(func() *ui.Bitmap {
	b, err := ui.DecodeBitmap(iconPNG)
	if err != nil {
		panic(err)
	}
	return b
})

// aboutWindow is the about window and its screen, for as long as the window is open.
type aboutWindow struct {
	screen *about.Screen
	win    *mygo.Window
}

// openAbout shows the about window, creating it when it is not open, and hides the panel. Closing the window never
// quits.
func (a *app) openAbout() {
	a.hide()
	if w := a.about; w != nil {
		w.win.Show()
		w.win.Focus()
		return
	}
	text := about.Text(a.locale())
	w := &aboutWindow{screen: about.New(text, mygo.App.Version(), productIcon(), about.Actions{CheckForUpdates: updater.CheckForUpdates})}
	w.win = mygo.NewWindow(mygo.WindowOptions{
		Title: text.Window, Width: aboutWidth, Height: aboutHeight,
		DisableResize: true, DisableMinimize: true, DisableMaximize: true, DisableFullScreen: true,
		Content: ui.View(w.screen.View),
	})
	a.about = w
	w.win.OnClosed(func() {
		if a.about == w {
			a.about = nil
		}
		a.syncDock()
	})
	// Whether the app can update itself takes a probe of the folder it is installed in.
	go func() {
		u := about.Unavailable
		if mygo.Updater.Enabled() {
			u = about.Available
		}
		a.post(func() { w.screen.SetUpdates(u) })
	}()
	a.syncDock()
	mygo.App.Focus()
	w.win.Focus()
}
