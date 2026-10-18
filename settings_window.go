package main

import (
	"log/slog"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/ui/accounts"
	"github.com/yangflow/idset/internal/ui/panel"
	"github.com/yangflow/idset/internal/ui/settings"
)

// The settings window opens at settingsWidth x settingsHeight and may shrink to the minimum.
const (
	settingsWidth, settingsHeight       = 720, 480
	settingsMinWidth, settingsMinHeight = 600, 400
)

// settingsWindow is the settings window and its screen, for as long as the window is open.
type settingsWindow struct {
	app    *app
	text   settings.Copy
	screen *settings.Screen
	win    *mygo.Window
}

// openSettings shows the settings window, creating it when it is not open, and hides the panel. Closing the
// window never quits.
func (a *app) openSettings(req settings.Request) {
	a.hide()
	if w := a.settings; w != nil {
		w.screen.Open(req)
		w.win.Show()
		w.win.Focus()
		return
	}
	w := &settingsWindow{app: a, text: settings.Text(a.locale())}
	w.screen = settings.New(w.text, a.env, w.actions())
	w.screen.SetSystem(mygo.App.Locale())
	w.screen.SetLanguage(a.prefs.Language())
	if s := a.store; s != nil {
		w.screen.SetReadOnly(s.ReadOnly() != nil)
	}
	w.screen.Open(req)
	w.win = mygo.NewWindow(mygo.WindowOptions{
		Title: w.text.Window, Width: settingsWidth, Height: settingsHeight, MinWidth: settingsMinWidth, MinHeight: settingsMinHeight,
		StateKey: "settings", Content: ui.View(w.screen.View),
	})
	a.settings = w
	w.win.OnClosed(func() {
		if a.settings == w {
			a.settings = nil
		}
		a.syncDock()
	})
	a.syncDock()
	mygo.App.Focus()
	w.win.Focus()
}

// locale is the language the interface uses: the stored choice, else the system's.
func (a *app) locale() string {
	if choice := a.prefs.Language(); choice != "" {
		return choice
	}
	return mygo.App.Locale()
}

// applyLanguage re-labels every open screen and the tray menu in the language now in effect.
func (a *app) applyLanguage() {
	loc := a.locale()
	a.text = panel.Text(loc)
	a.screen.SetText(a.text)
	if a.openItem != nil {
		a.openItem.SetLabel(a.text.Open)
		a.hideItem.SetLabel(a.text.HidePanel)
		a.settingsItem.SetLabel(a.text.SettingsMenu)
		a.quitItem.SetLabel(a.text.Quit)
	}
	if w := a.accounts; w != nil {
		w.text = accounts.Text(loc)
		w.screen.SetText(w.text)
		w.win.SetTitle(w.text.Window)
	}
	if w := a.settings; w != nil {
		w.text = settings.Text(loc)
		w.screen.SetText(w.text)
		w.win.SetTitle(w.text.Window)
	}
	a.updateTray()
}

// actions are the window's requests. Each starts background work and reports back on the UI thread.
func (w *settingsWindow) actions() settings.Actions {
	a := w.app
	backend := func() *settings.Backend {
		if a.store == nil {
			return nil
		}
		return &settings.Backend{Store: a.store, Env: a.env}
	}
	return settings.Actions{
		Language: func(choice string) {
			go func() {
				if err := a.prefs.SetLanguage(choice); err != nil {
					slog.Warn("Could not store the language", "error", err)
				}
				a.post(func() { a.applyLanguage() })
			}()
		},
		LoadLogin: func() {
			go func() {
				on := mygo.App.OpenAtLogin()
				a.post(func() { w.screen.SetLoginState(on) })
			}()
		},
		SetLogin: func(on bool) {
			go func() {
				err := mygo.App.SetOpenAtLogin(on)
				a.post(func() { w.screen.LoginDone(on, err) })
			}()
		},
		LoadBackups: func() {
			b := backend()
			if b == nil {
				return
			}
			go func() {
				list, err := b.Backups()
				a.post(func() { w.screen.SetBackups(list, err) })
			}()
		},
		Restore: func(path string) {
			b := backend()
			if b == nil {
				return
			}
			go func() {
				err := b.Restore(path)
				list, readErr, dataPath := b.Store.Accounts(), b.Store.ReadOnly(), b.Store.Path()
				a.post(func() {
					w.screen.RestoreDone(path, err)
					a.showAccounts(list, readErr, dataPath)
				})
			}()
		},
		DeleteBackup: func(path string) {
			b := backend()
			if b == nil {
				return
			}
			go func() {
				err := b.Delete(path)
				a.post(func() { w.screen.DeleteDone(path, err) })
			}()
		},
		LoadIncludes: func() {
			b := backend()
			if b == nil {
				return
			}
			go func() {
				state := b.Includes()
				a.post(func() { w.screen.SetIncludes(state) })
			}()
		},
		RemoveIncludes: func() {
			b := backend()
			if b == nil {
				return
			}
			go func() {
				before, after, err := b.RemoveIncludes()
				a.post(func() { w.screen.RemoveDone(before, after, err) })
			}()
		},
	}
}
