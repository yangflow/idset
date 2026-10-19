package main

import (
	"context"
	"errors"
	"log/slog"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/store"
	"github.com/yangflow/idset/internal/ui/accounts"
	settingsui "github.com/yangflow/idset/internal/ui/settings"
)

// The account window opens at accountsWidth x accountsHeight and may shrink to the minimum.
const (
	accountsWidth, accountsHeight       = 760, 520
	accountsMinWidth, accountsMinHeight = 640, 420
)

// accountsWindow is the account window and its screen, for as long as the window is open.
type accountsWindow struct {
	app    *app
	text   accounts.Copy
	screen *accounts.Screen
	win    *mygo.Window
	dialog *dialogWindow
}

// openIdentity opens the account window on an identity and records the use, which moves it up the list.
func (a *app) openIdentity(id string) {
	a.openAccounts(accounts.Request{ID: id})
	s := a.store
	if s == nil {
		return
	}
	go func() {
		if err := s.Touch(id); err != nil {
			slog.Warn("Could not record the use of an identity", "error", err)
			return
		}
		list, readErr, path := s.Accounts(), s.ReadOnly(), s.Path()
		a.post(func() { a.showAccounts(list, readErr, path) })
	}()
}

// openAccounts shows the account window, creating it when it is not open, and hides the panel. Closing the
// window never quits.
func (a *app) openAccounts(req accounts.Request) {
	a.hide()
	if w := a.accounts; w != nil {
		w.screen.Open(req)
		w.win.Show()
		w.win.Focus()
		return
	}
	w := &accountsWindow{app: a, text: accounts.Text(mygo.App.Locale())}
	w.screen = accounts.New(w.text, a.env, w.actions())
	if s := a.store; s != nil {
		w.screen.SetAccounts(s.Accounts(), s.ReadOnly(), s.Path())
	}
	w.screen.Open(req)
	w.dialog = &dialogWindow{screen: w.screen}
	w.win = mygo.NewWindow(mygo.WindowOptions{
		Title: w.text.Window, Width: accountsWidth, Height: accountsHeight, MinWidth: accountsMinWidth, MinHeight: accountsMinHeight,
		StateKey: "accounts", TitleBarStyle: mygo.TitleBarHidden, Vibrancy: mygo.VibrancySidebar, Content: ui.View(w.dialog.view),
	})
	w.dialog.parent = w.win
	a.accounts = w
	w.win.OnClosed(func() {
		if a.accounts == w {
			a.accounts = nil
		}
		a.syncDock()
	})
	a.syncDock()
	mygo.App.Focus()
	w.win.Focus()
}

// syncDock shows the Dock icon on macOS while a window with a title bar is open and hides it after the last one
// closes.
func (a *app) syncDock() {
	if runtime.GOOS != "darwin" {
		return
	}
	if a.accounts != nil || a.settings != nil || a.about != nil {
		mygo.App.SetActivationPolicy(mygo.ActivationPolicyRegular)
		return
	}
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
}

// actions are the window's requests. Each starts background work and reports back on the UI thread; w.screen
// is used even when the window closed meanwhile, which does no harm.
func (w *accountsWindow) actions() accounts.Actions {
	a := w.app
	// backend is nil until accounts.json has been loaded; the window shows no editor before that.
	backend := func() *accounts.Backend {
		if a.store == nil {
			return nil
		}
		return &accounts.Backend{Store: a.store, Env: a.env, Git: a.gitConfig}
	}
	return accounts.Actions{
		Save: func(acc account.Account, isNew bool) {
			b := backend()
			if b == nil {
				return
			}
			// A new identity opened for a folder of the panel's result card: look at that folder again once it is saved.
			folder := w.screen.DraftFolder(acc.ID)
			go func() {
				err := b.Save(acc, isNew)
				list, readErr, path := b.Store.Accounts(), b.Store.ReadOnly(), b.Store.Path()
				a.post(func() {
					w.screen.SaveDone(acc.ID, list, err)
					a.showAccounts(list, readErr, path)
					var regenerate *store.RegenerateError
					if folder != "" && (err == nil || errors.As(err, &regenerate)) {
						a.screen.Recheck(folder)
					}
				})
			}()
		},
		Delete: func(id string, deleteKey bool) {
			b := backend()
			if b == nil {
				return
			}
			go func() {
				res := b.Delete(id, deleteKey)
				list, readErr, path := b.Store.Accounts(), b.Store.ReadOnly(), b.Store.Path()
				a.post(func() {
					w.screen.DeleteDone(id, list, res)
					a.showAccounts(list, readErr, path)
				})
			}()
		},
		InspectKey: func(id string) {
			b := backend()
			if b == nil {
				return
			}
			go func() {
				reason, other := b.InspectKey(id)
				a.post(func() { w.screen.SetKeyReason(id, reason, other) })
			}()
		},
		Prefill: func(id string) {
			b := accounts.Backend{Git: a.gitConfig}
			go func() {
				name, email := b.Prefill(context.Background())
				a.post(func() { w.screen.SetPrefill(id, name, email) })
			}()
		},
		ChooseKey: func(id string) {
			go func() {
				chosen, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
					Parent: w.win, Title: w.text.ChooseKeyTitle, DefaultPath: a.env.Native("~/.ssh"), ShowHiddenFiles: true,
				})
				if err != nil || len(chosen) == 0 {
					return
				}
				a.post(func() { w.screen.SetKeyPath(id, chosen[0]) })
			}()
		},
		ChooseFolder: func(id string, index int) {
			go func() {
				chosen, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
					Parent: w.win, Title: w.text.ChooseFolderTitle, Directory: true, DefaultPath: a.env.Home, CreateDirectories: true,
				})
				if err != nil || len(chosen) == 0 {
					return
				}
				a.post(func() { w.screen.SetFolder(id, index, chosen[0]) })
			}()
		},
		OpenImport: func() { a.openSettings(settingsui.Request{Pane: settingsui.Import}) },
	}
}
