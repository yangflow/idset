// Package about is the about window: the app's icon, name, version, copyright and license, a button that opens the
// repository and Check for Updates. Whether this copy can update itself is found out in the owner's background work
// and reported through SetUpdates; until then Check for Updates is disabled without a note.
package about

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/ui/icons"
	"github.com/yangflow/idset/internal/ui/locale"
	"github.com/yangflow/idset/internal/ui/theme"
)

//go:embed copy.json
var catalog []byte

// Copy is the window's text in one language. Main reads the window title from it.
type Copy struct {
	Window, Version, Copyright, License, GitHub, Check, Cannot string
}

var translations = func() map[string]Copy {
	var values map[string]Copy
	if err := json.Unmarshal(catalog, &values); err != nil {
		panic(err)
	}
	return values
}()

// Text returns the copy for a locale: a system locale such as "zh_TW", or a UI language such as "zh-Hans".
func Text(loc string) Copy { return translations[locale.Language(loc)] }

// Repository is the page the GitHub button opens.
const Repository = "https://github.com/yangflow/idset"

// Updates is whether this copy can update itself.
type Updates int

const (
	// Unknown: not found out yet.
	Unknown Updates = iota
	// Available: built with an update feed and installed where it can write.
	Available
	// Unavailable: a development build, a build without an update feed, or a copy a package manager updates.
	Unavailable
)

// Actions are the window's requests. CheckForUpdates opens the updater's own window.
type Actions struct {
	CheckForUpdates func()
}

// Screen holds the window's state.
type Screen struct {
	text    Copy
	version string
	icon    *ui.Bitmap
	updates Updates
	act     Actions
}

// New returns the screen for a version, such as "0.1.0", with the product icon.
func New(text Copy, version string, icon *ui.Bitmap, act Actions) *Screen {
	return &Screen{text: text, version: version, icon: icon, act: act}
}

// SetText changes the language.
func (s *Screen) SetText(text Copy) { s.text = text }

// SetUpdates reports whether this copy can update itself.
func (s *Screen) SetUpdates(u Updates) { s.updates = u }

// View draws the window: its title in the hidden title bar, then one column centered under it, over the material.
func (s *Screen) View(c *ui.Context) {
	theme.Apply(c)
	t := c.Theme()
	theme.TitleBar(c, s.text.Window)
	theme.Center(c, ui.Column(c)).Grow(1).Padding(0, t.Space(theme.Space6), t.Space(theme.Space6), t.Space(theme.Space6)).Children(func() {
		ui.Image(c, s.icon).Width(t.Space(theme.AppIcon)).Height(t.Space(theme.AppIcon))
		theme.Display.On(c, ui.Text(c, "Idset")).Role(ui.RoleHeading).Level(1).Margin(t.Space(theme.Space3), 0, 0, 0)
		ui.Text(c, fmt.Sprintf(s.text.Version, s.version)).TextColor(t.TextMuted).Selectable().Margin(t.Space(theme.Space0_5), 0, 0, 0)
		ui.Column(c).AlignItems(ui.Center).Margin(t.Space(theme.Space3), 0, 0, 0).Children(func() {
			theme.Small.On(c, ui.Text(c, s.text.Copyright)).TextColor(t.TextMuted)
			theme.Small.On(c, ui.Text(c, s.text.License)).TextColor(t.TextMuted)
		})
		theme.Cluster(c, ui.Row(c)).Margin(t.Space(theme.Space6), 0, 0, 0).Children(func() {
			// GitHub is a link: Accent text with the external symbol, no fill.
			github := ui.Button(c.Key("github"), "").Label(s.text.GitHub).Padding(0, t.Space(theme.Space1)).Border(0, ui.Color{}).
				Background(ui.Color{}).TextColor(t.Accent)
			github.Children(func() {
				theme.InlineTight(c, ui.Row(c)).Children(func() {
					ui.Icon(c, icons.External).FontSize(t.Space(theme.IconXs))
					theme.BodyMedium.On(c, ui.Text(c, s.text.GitHub)).SingleLine()
				})
			})
			if github.Hovered() {
				github.TextColor(t.AccentHover)
			}
			if github.Clicked() {
				c.OpenURL(Repository)
			}
			if ui.PrimaryButton(c.Key("check"), s.text.Check).Disabled(s.updates != Available).Clicked() {
				s.act.CheckForUpdates()
			}
		})
		if s.updates == Unavailable {
			theme.Secondary.On(c, ui.Text(c, s.text.Cannot)).TextColor(t.TextMuted).TextAlign(ui.Center).
				MaxWidth(t.Space(theme.Measure)).Wrap().Margin(t.Space(theme.Space2), 0, 0, 0)
		}
	})
}
