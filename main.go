package main

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"
	"github.com/egoist/mygo/plugins/updater/native"
	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/apply"
	"github.com/yangflow/idset/internal/bind"
	"github.com/yangflow/idset/internal/diagnose"
	"github.com/yangflow/idset/internal/match"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/settings"
	"github.com/yangflow/idset/internal/store"
	"github.com/yangflow/idset/internal/trayicon"
	"github.com/yangflow/idset/internal/ui/accounts"
	"github.com/yangflow/idset/internal/ui/panel"
	settingsui "github.com/yangflow/idset/internal/ui/settings"
)

const (
	gitTimeout = 10 * time.Second
	// sshTimeout limits each command of a connectivity check.
	sshTimeout = 20 * time.Second
)

// app owns the tray, the panel window, the account window and the store. Its fields are touched on the main
// thread only; background work reports back through post.
type app struct {
	env       paths.Env
	dataDir   string
	store     *store.Store
	git       match.Git
	gitConfig accounts.GitConfig
	text      panel.Copy
	screen    *panel.Screen
	win       *mygo.Window
	tray      *mygo.Tray
	menu      *mygo.Menu
	openItem  *mygo.MenuItem
	hideItem  *mygo.MenuItem
	// settingsItem and quitItem are the other tray menu entries, kept to re-label them on a language change.
	settingsItem *mygo.MenuItem
	quitItem     *mygo.MenuItem
	prefs        *settings.Settings
	quitting     bool
	// accounts is the account window while it is open, nil otherwise.
	accounts *accountsWindow
	// settings is the settings window while it is open, nil otherwise.
	settings *settingsWindow
	// diag keeps the connectivity checks, checking holds the identities whose check is running, userHosts the
	// Host names of the user's ~/.ssh/config as the last check read them, and badge the mark on the tray icon.
	diag      diagnose.Cache
	checking  map[string]bool
	userHosts []string
	badge     trayicon.Badge
	// tools are the ssh programs, found once, off the UI thread.
	toolsOnce sync.Once
	tools     diagnose.Tools
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.Time(attr.Key, attr.Value.Time().UTC())
			}
			return attr
		},
	})))
	mygo.App.SetName("Idset")
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	env, err := paths.Current()
	if err != nil {
		slog.Error("No home directory", "error", err)
		os.Exit(1)
	}
	a := &app{env: env, dataDir: apply.DefaultTarget(env).Dir, git: match.SystemGit(gitTimeout),
		gitConfig: accounts.SystemGitConfig(gitTimeout), checking: map[string]bool{}}
	a.prefs = settings.Open(a.dataDir)
	mygo.Use(native.Plugin)
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { a.quitting = true })
	mygo.App.OnSecondInstance(func([]string, string) { a.show() })
	mygo.App.WhenReady(a.ready)
	if err := mygo.App.Run(); err != nil {
		slog.Error("Application failed", "error", err)
		os.Exit(1)
	}
}

func (a *app) ready() {
	a.text = panel.Text(a.locale())
	a.screen = panel.New(a.text, a.env, panel.Actions{
		Resolve:      a.resolve,
		ChooseFolder: a.chooseFolder,
		CopyText:     func(s string) { mygo.Clipboard.WriteText(s) },
		CloneCopied:  a.cloneCopied,
		Hide:         a.hide,
		Quit:         mygo.App.Quit,
		Reload:       func() { a.load() },
		OpenSettings: func() { a.openSettings(settingsui.Request{}) },
		OpenBackups:  func() { a.openSettings(settingsui.Request{Pane: settingsui.Backups}) },
		Reveal:       mygo.Shell.ShowItemInFolder,
		Open:         a.openFolder,
		Run:          a.runBind,
		OpenAccounts: a.openIdentity,
		NewIdentity:  func() { a.openAccounts(accounts.Request{New: true}) },
		NewIdentityFor: func(folder string) {
			a.openAccounts(accounts.Request{New: true, Folder: folder})
		},
		CheckAgain:    a.checkAgain,
		Unlock:        a.unlock,
		CopyPublicKey: a.copyPublicKey,
		AddRewrites:   a.addRewrites,
		Regenerate:    a.regenerate,
		GenerateKey: func(id string) {
			a.openSettings(settingsui.Request{Pane: settingsui.Keys, For: id})
		},
	})
	// The application menu keeps the standard editing shortcuts working in text fields.
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Label: "Idset", Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleAbout}, updater.MenuItem(), mygo.Separator(), {Role: mygo.RoleQuit},
		}},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleWindowMenu},
	}))
	tray, err := mygo.NewTray(mygo.TrayOptions{
		Icon: trayicon.PNG(runtime.GOOS == "darwin", trayicon.None), IconIsTemplate: runtime.GOOS == "darwin", ToolTip: a.text.TooltipIdle,
	})
	if err != nil {
		// Without a tray (Linux without AppIndicator) the panel is an ordinary window, the app's only one: closing
		// it quits.
		slog.Warn("No tray icon", "error", err)
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title: "Idset", Width: 340, Height: 480, DisableResize: true, Content: ui.View(a.screen.View),
		})
		mygo.App.OnWindowAllClosed(mygo.App.Quit)
		a.load()
		return
	}
	a.tray = tray
	// The panel does not hide when it loses focus: the user clicks a file manager to start dragging a folder
	// into it. The tray icon and Escape hide it.
	a.win = mygo.NewWindow(mygo.WindowOptions{
		Title: "Idset", Width: 340, Height: 480, Hidden: true, Frameless: true,
		DisableResize: true, DisableMinimize: true, DisableMaximize: true, DisableFullScreen: true,
		AlwaysOnTop: true, SkipTaskbar: true,
		Content: ui.View(a.screen.View),
	})
	a.win.OnClose(func(e *mygo.CloseEvent) {
		if !a.quitting {
			e.PreventDefault()
			a.hide()
		}
	})
	a.openItem = &mygo.MenuItem{Label: a.text.Open, Click: func(*mygo.MenuItem, *mygo.Window) { a.show() }}
	a.hideItem = &mygo.MenuItem{Label: a.text.HidePanel, Hidden: true, Click: func(*mygo.MenuItem, *mygo.Window) { a.hide() }}
	a.settingsItem = &mygo.MenuItem{Label: a.text.SettingsMenu, Click: func(*mygo.MenuItem, *mygo.Window) {
		a.openSettings(settingsui.Request{})
	}}
	a.quitItem = &mygo.MenuItem{Label: a.text.Quit, Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Quit() }}
	a.menu = mygo.NewMenu([]*mygo.MenuItem{
		a.openItem, a.hideItem, a.settingsItem,
		mygo.Separator(),
		a.quitItem,
	})
	if runtime.GOOS == "linux" {
		// AppIndicator reports no clicks; its menu is the only way to open and hide the panel.
		a.tray.SetMenu(a.menu)
	} else {
		a.tray.OnClick(a.toggle)
		a.tray.OnRightClick(func() {
			a.tray.SetMenu(a.menu)
			defer a.tray.SetMenu(nil)
			a.tray.PopUpMenu()
		})
	}
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) { a.tray.Destroy() })
	a.load()
}

// load reads accounts.json in the background, so that edits made outside the app show up each time. The store
// is opened once and read again afterwards, so that saves in the account window and loads never work on two
// copies of the list.
func (a *app) load() {
	s := a.store
	go func() {
		if s == nil {
			s = store.Open(a.dataDir, time.Now, store.WithRegenerate(apply.Regenerate(a.env, apply.DefaultTarget(a.env), time.Now)))
		} else {
			s.Reload()
		}
		list, readErr, path := s.Accounts(), s.ReadOnly(), s.Path()
		prefixes := settings.Open(a.dataDir).Prefixes()
		a.post(func() {
			if a.store == nil {
				a.store = s
			}
			a.showAccounts(list, readErr, path)
			a.screen.SetPrefixes(prefixes)
		})
	}()
}

// post runs fn on the UI thread and redraws the panel and the account window, if it is open.
func (a *app) post(fn func()) {
	a.win.Update(fn)
	if a.accounts != nil {
		a.accounts.win.Update(func() {})
	}
	if a.settings != nil {
		a.settings.win.Update(func() {})
	}
}

// showAccounts gives the list to both screens, checks the identities that are due and updates the tray.
func (a *app) showAccounts(list []account.Account, readErr error, path string) {
	a.screen.SetAccounts(list, readErr, path)
	if a.accounts != nil {
		a.accounts.screen.SetAccounts(list, readErr, path)
	}
	if a.settings != nil {
		a.settings.screen.SetReadOnly(readErr != nil)
		a.settings.screen.SetAccounts(list)
	}
	a.diagnose(list, false, nil)
	a.updateTray()
}

func (a *app) toggle() {
	if a.win.IsVisible() {
		a.hide()
		return
	}
	a.show()
}

// hide hides the panel; without a tray the panel is the only window and stays.
func (a *app) hide() {
	if a.tray == nil {
		return
	}
	a.win.Hide()
	a.openItem.SetVisible(true)
	a.hideItem.SetVisible(false)
}

// show places the panel by the tray icon, inside the work area of its display, and reloads the list.
func (a *app) show() {
	if a.tray != nil {
		anchor := a.tray.Bounds()
		if anchor.Width == 0 || anchor.Height == 0 {
			a.win.Center()
		} else {
			area := mygo.Screen.DisplayNearestPoint(mygo.Point{X: anchor.X + anchor.Width/2, Y: anchor.Y + anchor.Height/2}).WorkArea
			b := a.win.Bounds()
			x := anchor.X + (anchor.Width-b.Width)/2
			y := anchor.Y + anchor.Height + 6
			if y+b.Height > area.Y+area.Height {
				y = anchor.Y - b.Height - 6
			}
			x = max(area.X, min(x, area.X+max(0, area.Width-b.Width)))
			y = max(area.Y, min(y, area.Y+max(0, area.Height-b.Height)))
			a.win.SetPosition(x, y)
		}
	}
	a.win.Show()
	if a.tray != nil {
		a.openItem.SetVisible(false)
		a.hideItem.SetVisible(true)
	}
	a.load()
}

func (a *app) resolve(dir string) {
	if a.store == nil {
		return
	}
	list := a.store.Accounts()
	s := a.store
	go func() {
		r, err := match.Resolve(context.Background(), a.git, dir, list, a.env)
		touched := false
		if err == nil && r.Kind == match.Matched {
			if err := s.Touch(r.ID); err != nil {
				slog.Warn("Could not record the use of an identity", "error", err)
			} else {
				touched = true
			}
		}
		a.post(func() {
			a.screen.SetResult(dir, r, err)
			if touched {
				// The use moves the identity up the list.
				a.showAccounts(s.Accounts(), s.ReadOnly(), s.Path())
			}
			a.updateTray()
		})
	}()
}

// cloneCopied remembers the owner prefix of a copied clone command and records a use of the identity, in the
// background; the list is shown again because the use moves the identity up.
func (a *app) cloneCopied(id, prefix string) {
	s := a.store
	if s == nil {
		return
	}
	go func() {
		prefixes := settings.Open(a.dataDir)
		if prefix != "" && prefixes.Prefixes()[id] != prefix {
			if err := prefixes.SetPrefix(id, prefix); err != nil {
				slog.Warn("Could not remember the owner prefix", "error", err)
			}
		}
		if err := s.Touch(id); err != nil {
			slog.Warn("Could not record the use of an identity", "error", err)
		}
		list, readErr, path, remembered := s.Accounts(), s.ReadOnly(), s.Path(), prefixes.Prefixes()
		a.post(func() {
			a.showAccounts(list, readErr, path)
			a.screen.SetPrefixes(remembered)
		})
	}()
}

// runBind does a request of the result card in the background: one store write for everything it changes, then
// the folder looked at again. The list and the card are updated together, so that the card finds the identities
// it names.
func (a *app) runBind(op bind.Op) {
	s := a.store
	if s == nil {
		return
	}
	runner := bind.Runner{Store: s, Env: a.env, Git: a.git, Now: time.Now}
	go func() {
		out := runner.Run(context.Background(), op)
		list, readErr, path := s.Accounts(), s.ReadOnly(), s.Path()
		a.post(func() {
			a.showAccounts(list, readErr, path)
			a.screen.OpDone(out)
			a.updateTray()
		})
	}()
}

// openFolder shows a folder of the result card in the system's file manager.
func (a *app) openFolder(path string) {
	if err := mygo.Shell.OpenPath(a.env.Native(path)); err != nil {
		slog.Warn("Could not open the folder", "error", err)
	}
}

func (a *app) chooseFolder() {
	go func() {
		chosen, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.win, Title: a.text.ChooseTitle, Directory: true, DefaultPath: a.env.Home})
		if err != nil || len(chosen) == 0 {
			return
		}
		a.win.Update(func() { a.screen.Drop(chosen) })
	}()
}

// updateTray shows the panel's tooltip and, once no check is running, the worst problem as the icon's badge.
func (a *app) updateTray() {
	if a.tray == nil {
		return
	}
	a.tray.SetToolTip(a.screen.Tooltip())
	if len(a.checking) > 0 {
		return
	}
	badge := map[match.Severity]trayicon.Badge{match.Warning: trayicon.Warning, match.Error: trayicon.Error}[a.screen.Worst()]
	if badge == a.badge {
		return
	}
	darwin := runtime.GOOS == "darwin"
	if err := a.tray.SetIcon(trayicon.PNG(darwin, badge), darwin); err != nil {
		slog.Warn("Could not change the tray icon", "error", err)
		return
	}
	a.badge = badge
}

// diagnose checks in the background the identities of list that are due, or with force those of ids, all of them
// when ids is nil, and meanwhile shows the checks kept so far. An identity whose check is running is not started
// again.
func (a *app) diagnose(list []account.Account, force bool, ids []string) {
	due := slices.DeleteFunc(a.diag.Due(list, time.Now(), force), func(x account.Account) bool {
		return a.checking[x.ID] || ids != nil && !slices.Contains(ids, x.ID)
	})
	for _, x := range due {
		a.checking[x.ID] = true
	}
	a.showDiagnostics(list)
	if len(due) == 0 {
		return
	}
	go func() {
		runner := diagnose.Runner{Run: diagnose.System(sshTimeout), Tools: a.sshTools(), Env: a.env, Now: time.Now}
		checks := runner.CheckAll(context.Background(), due)
		var hosts []string
		if config, err := os.ReadFile(a.env.Native("~/.ssh/config")); err == nil {
			hosts = diagnose.UserHosts(string(config))
		}
		// A check stands only for the identity as it was checked: one edited meanwhile is checked again.
		var done diagnose.Cache
		done.Put(due, checks)
		a.post(func() {
			for _, x := range due {
				delete(a.checking, x.ID)
			}
			a.userHosts = hosts
			list := a.store.Accounts()
			var current []diagnose.Check
			for _, x := range list {
				if c, ok := done.Get(x); ok {
					current = append(current, c)
				}
			}
			a.diag.Put(list, current)
			a.diagnose(list, false, nil)
			a.updateTray()
		})
	}()
}

// showDiagnostics gives the panel the kept checks of list and the identities being checked.
func (a *app) showDiagnostics(list []account.Account) {
	var pending []string
	for _, x := range list {
		if a.checking[x.ID] {
			pending = append(pending, x.ID)
		}
	}
	a.screen.SetDiagnostics(panel.Diagnostics{Checks: a.diag.Checks(list), Pending: pending, UserHosts: a.userHosts})
}

// sshTools finds the ssh programs the first time it is called; it runs git, so it is called off the UI thread.
func (a *app) sshTools() diagnose.Tools {
	a.toolsOnce.Do(func() {
		command := ""
		if runtime.GOOS == "windows" {
			command, _ = a.gitConfig(context.Background(), "core.sshCommand")
		}
		a.tools = diagnose.FindTools(runtime.GOOS, command, exec.LookPath, func(p string) bool {
			info, err := os.Stat(p)
			return err == nil && info.Mode().IsRegular()
		})
	})
	return a.tools
}

// checkAgain checks identities again at once, all of them when ids is empty.
func (a *app) checkAgain(ids []string) {
	if a.store == nil {
		return
	}
	if len(ids) == 0 {
		ids = nil
	}
	a.diagnose(a.store.Accounts(), true, ids)
	a.updateTray()
}

// identity is the stored identity with id.
func (a *app) identity(id string) (account.Account, bool) {
	if a.store == nil {
		return account.Account{}, false
	}
	list := a.store.Accounts()
	i := slices.IndexFunc(list, func(x account.Account) bool { return x.ID == id })
	if i < 0 {
		return account.Account{}, false
	}
	return list[i], true
}

// unlock loads the key of an identity into the agent with the keychain, then checks the identity again.
func (a *app) unlock(id string) {
	x, ok := a.identity(id)
	if !ok {
		a.screen.UnlockDone(id, os.ErrNotExist)
		return
	}
	go func() {
		err := diagnose.Unlock(context.Background(), diagnose.System(sshTimeout), a.sshTools(), a.env, x)
		if err != nil {
			slog.Warn("Could not unlock the key", "error", err)
		}
		a.post(func() {
			a.screen.UnlockDone(id, err)
			if err == nil {
				a.checkAgain([]string{id})
			}
		})
	}()
}

// copyPublicKey copies the public key next to an identity's key.
func (a *app) copyPublicKey(id string) {
	x, ok := a.identity(id)
	if !ok {
		a.screen.PublicKeyDone(id, diagnose.ErrNoPublicKey)
		return
	}
	go func() {
		text, err := diagnose.PublicKey(a.env, x)
		a.post(func() {
			if err == nil {
				mygo.Clipboard.WriteText(text)
			}
			a.screen.PublicKeyDone(id, err)
		})
	}()
}

// addRewrites gives an identity the default rewrites of its platform it does not have, then looks at the folder
// again.
func (a *app) addRewrites(id string) {
	s := a.store
	if s == nil {
		return
	}
	go func() {
		list := s.Accounts()
		if i := slices.IndexFunc(list, func(x account.Account) bool { return x.ID == id }); i >= 0 {
			x := list[i]
			add := slices.DeleteFunc(account.DefaultRewrites(x.Provider, x.SSHAlias), func(r account.Rewrite) bool {
				return slices.Contains(x.URLRewrites, r)
			})
			if len(add) > 0 {
				x.URLRewrites = append(slices.Clone(x.URLRewrites), add...)
				if err := s.Update(x); err != nil {
					slog.Warn("Could not add the rewrites", "error", err)
				}
			}
		}
		list, readErr, path := s.Accounts(), s.ReadOnly(), s.Path()
		a.post(func() {
			a.showAccounts(list, readErr, path)
			a.recheckFolder()
		})
	}()
}

// regenerate writes the managed files again from the stored list, then looks at the folder again.
func (a *app) regenerate() {
	s := a.store
	if s == nil {
		return
	}
	go func() {
		if err := apply.Regenerate(a.env, apply.DefaultTarget(a.env), time.Now)(s.Accounts()); err != nil {
			slog.Warn("Could not write the managed files", "error", err)
		}
		a.post(a.recheckFolder)
	}()
}

// recheckFolder looks again at the folder on the result card, if there is one.
func (a *app) recheckFolder() {
	if r := a.screen.Result(); r != nil {
		a.screen.Recheck(r.Root)
	}
}
