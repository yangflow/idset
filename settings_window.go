package main

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/diagnose"
	"github.com/yangflow/idset/internal/keygen"
	"github.com/yangflow/idset/internal/store"
	"github.com/yangflow/idset/internal/ui/about"
	"github.com/yangflow/idset/internal/ui/accounts"
	"github.com/yangflow/idset/internal/ui/panel"
	"github.com/yangflow/idset/internal/ui/settings"
)

// The settings window opens at settingsWidth x settingsHeight and may shrink to the minimum.
const (
	settingsWidth, settingsHeight       = 720, 480
	settingsMinWidth, settingsMinHeight = 600, 400
)

// keygenTimeout bounds ssh-keygen; an RSA key of 4096 bits can take a while on a slow machine.
const keygenTimeout = 60 * time.Second

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
		w.screen.SetAccounts(s.Accounts())
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
	if a.appAboutItem != nil {
		a.appAboutItem.SetLabel(a.text.About)
	}
	if a.openItem != nil {
		a.openItem.SetLabel(a.text.Open)
		a.hideItem.SetLabel(a.text.HidePanel)
		a.settingsItem.SetLabel(a.text.SettingsMenu)
		a.aboutItem.SetLabel(a.text.AboutMenu)
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
	if w := a.about; w != nil {
		text := about.Text(loc)
		w.screen.SetText(text)
		w.win.SetTitle(text.Window)
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
		Generate: func(req settings.KeyRequest) {
			go func() {
				key, err := keygen.Generate(context.Background(), diagnose.System(keygenTimeout), a.sshTools().Keygen, a.env,
					req.Type, req.Name, req.Comment)
				if err != nil {
					slog.Warn("Could not generate the key", "error", err)
				}
				a.post(func() { w.screen.GenerateDone(key, err) })
			}()
		},
		LoadKey: func(keyPath string) {
			go func() {
				err := keygen.Load(context.Background(), diagnose.System(sshTimeout), a.sshTools().Add, a.env, keyPath)
				if err != nil {
					slog.Warn("Could not load the key", "error", err)
				}
				a.post(func() { w.screen.LoadDone(err) })
			}()
		},
		UseKey: func(id, keyPath, fingerprint string) {
			s := a.store
			if s == nil {
				return
			}
			go func() {
				err := useKey(s.Accounts(), s.Update, id, keyPath, fingerprint)
				if err != nil {
					slog.Warn("Could not give the key to the identity", "error", err)
				}
				list, readErr, path := s.Accounts(), s.ReadOnly(), s.Path()
				a.post(func() {
					a.showAccounts(list, readErr, path)
					w.screen.UseDone(id, err)
					if err == nil {
						a.checkAgain([]string{id})
					}
					a.recheckFolder()
				})
			}()
		},
		NewIdentity: func(d settings.KeyDraft) {
			a.openAccounts(accounts.Request{New: true, Key: &accounts.Key{
				Provider: d.Provider, Email: d.Email, Path: d.KeyPath, Fingerprint: d.Fingerprint,
			}})
		},
		Scan: func() {
			b := settings.Backend{Store: a.store, Env: a.env}
			go func() {
				cands, err := b.Scan(context.Background(), diagnose.System(gitTimeout))
				if err != nil {
					slog.Warn("Could not read the configuration to import", "error", err)
				}
				a.post(func() { w.screen.ScanDone(cands, err) })
			}()
		},
		Import: func(list []account.Account) {
			b := backend()
			if b == nil {
				w.screen.ImportDone(nil, errNotLoaded)
				return
			}
			go func() {
				added, err := b.Import(list, time.Now())
				if err != nil {
					slog.Warn("Could not import the identities", "error", err)
				}
				s := b.Store
				all, readErr, path := s.Accounts(), s.ReadOnly(), s.Path()
				a.post(func() {
					a.showAccounts(all, readErr, path)
					w.screen.ImportDone(added, err)
					if len(added) > 0 {
						ids := make([]string, len(added))
						for i, x := range added {
							ids[i] = x.ID
						}
						a.checkAgain(ids)
					}
					a.recheckFolder()
				})
			}()
		},
		OpenAccounts: func(id string) { a.openAccounts(accounts.Request{ID: id}) },
		Reveal:       func(keyPath string) { mygo.Shell.ShowItemInFolder(a.env.Native(keyPath)) },
	}
}

// errNotLoaded is why nothing can be imported before accounts.json has been opened.
var errNotLoaded = errors.New("accounts.json is not loaded yet")

// useKey gives the identity id the key at keyPath and its fingerprint.
func useKey(list []account.Account, update func(account.Account) error, id, keyPath, fingerprint string) error {
	i := slices.IndexFunc(list, func(x account.Account) bool { return x.ID == id })
	if i < 0 {
		return store.ErrNotFound
	}
	x := list[i]
	x.KeyPath, x.KeyFingerprint = keyPath, fingerprint
	return update(x)
}
