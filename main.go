package main

import (
	"log/slog"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"
	"github.com/egoist/mygo/plugins/updater/native"
	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/ui/example"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.Time(attr.Key, attr.Value.Time().UTC())
			}
			return attr
		},
	})))
	mygo.Use(native.Plugin)
	mygo.App.WhenReady(func() {
		screen := example.New(mygo.App.Locale())
		mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
			{Label: "Idset", Submenu: []*mygo.MenuItem{
				{Role: mygo.RoleAbout}, updater.MenuItem(), mygo.Separator(), {Role: mygo.RoleQuit},
			}},
			{Role: mygo.RoleEditMenu}, {Role: mygo.RoleWindowMenu},
		}))
		mygo.NewWindow(mygo.WindowOptions{
			Title: "Idset", Width: 480, Height: 320, MinWidth: 360, MinHeight: 280,
			Content: ui.View(screen.View),
		})
	})
	mygo.App.OnWindowAllClosed(func() { mygo.App.Quit() })
	if err := mygo.App.Run(); err != nil {
		slog.Error("Application failed", "error", err)
		os.Exit(1)
	}
}
