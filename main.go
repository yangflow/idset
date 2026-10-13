package main

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"
	"github.com/egoist/mygo/plugins/updater/native"
	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/apply"
	"github.com/yangflow/idset/internal/bind"
	"github.com/yangflow/idset/internal/match"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/store"
	"github.com/yangflow/idset/internal/trayicon"
	"github.com/yangflow/idset/internal/ui/accounts"
	"github.com/yangflow/idset/internal/ui/panel"
)

const gitTimeout = 10 * time.Second

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
	quitting  bool
	// accounts is the account window while it is open, nil otherwise.
	accounts *accountsWindow
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
		gitConfig: accounts.SystemGitConfig(gitTimeout)}
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
	a.text = panel.Text(mygo.App.Locale())
	a.screen = panel.New(a.text, a.env, panel.Actions{
		Resolve:      a.resolve,
		ChooseFolder: a.chooseFolder,
		CopyText:     func(s string) { mygo.Clipboard.WriteText(s) },
		Hide:         a.hide,
		Quit:         mygo.App.Quit,
		Reload:       func() { a.load() },
		Reveal:       mygo.Shell.ShowItemInFolder,
		Open:         a.openFolder,
		Run:          a.runBind,
		OpenAccounts: a.openIdentity,
		NewIdentity:  func() { a.openAccounts(accounts.Request{New: true}) },
		NewIdentityFor: func(folder string) {
			a.openAccounts(accounts.Request{New: true, Folder: folder})
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
		Icon: trayicon.PNG(runtime.GOOS == "darwin"), IconIsTemplate: runtime.GOOS == "darwin", ToolTip: a.text.TooltipIdle,
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
	a.menu = mygo.NewMenu([]*mygo.MenuItem{
		a.openItem, a.hideItem,
		mygo.Separator(),
		{Label: a.text.Quit, Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Quit() }},
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
		a.post(func() {
			if a.store == nil {
				a.store = s
			}
			a.showAccounts(list, readErr, path)
		})
	}()
}

// post runs fn on the UI thread and redraws the panel and the account window, if it is open.
func (a *app) post(fn func()) {
	a.win.Update(fn)
	if a.accounts != nil {
		a.accounts.win.Update(func() {})
	}
}

// showAccounts gives the list to both screens and updates the tray tooltip.
func (a *app) showAccounts(list []account.Account, readErr error, path string) {
	a.screen.SetAccounts(list, readErr, path)
	if a.accounts != nil {
		a.accounts.screen.SetAccounts(list, readErr, path)
	}
	a.updateTooltip()
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
			a.updateTooltip()
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
			a.updateTooltip()
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

func (a *app) updateTooltip() {
	if a.tray != nil {
		a.tray.SetToolTip(a.screen.Tooltip())
	}
}
