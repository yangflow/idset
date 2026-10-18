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
	updates Updates
	act     Actions
}

// New returns the screen for a version, such as "0.1.0".
func New(text Copy, version string, act Actions) *Screen {
	return &Screen{text: text, version: version, act: act}
}

// SetText changes the language.
func (s *Screen) SetText(text Copy) { s.text = text }

// SetUpdates reports whether this copy can update itself.
func (s *Screen) SetUpdates(u Updates) { s.updates = u }

// View draws the window: one centered column, the buttons at the bottom.
func (s *Screen) View(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().AlignItems(ui.Center).Padding(t.Space(7), t.Space(6), t.Space(6), t.Space(6)).Children(func() {
		// The tray icon's identity card on an Accent tile, until the product icon replaces it.
		ui.Column(c).Width(t.Space(16)).Height(t.Space(16)).Radius(14).Background(t.Accent).AlignItems(ui.Center).
			Justify(ui.Center).Children(func() {
			ui.Icon(c, icons.Card).FontSize(t.Space(11)).TextColor(t.AccentText)
		})
		ui.Text(c, "Idset").FontSize(t.Space(4.5)).FontWeight(600).Role(ui.RoleHeading).Level(1).Margin(t.Space(3), 0, 0, 0)
		ui.Text(c, fmt.Sprintf(s.text.Version, s.version)).TextColor(t.TextMuted).Selectable()
		ui.Column(c).AlignItems(ui.Center).Margin(t.Space(3), 0, 0, 0).Children(func() {
			ui.Text(c, s.text.Copyright).FontSize(t.Space(2.75)).TextColor(t.TextMuted)
			ui.Text(c, s.text.License).FontSize(t.Space(2.75)).TextColor(t.TextMuted)
		})
		ui.Spacer(c)
		ui.Row(c).Gap(t.Space(2)).Margin(t.Space(5), 0, 0, 0).Children(func() {
			github := ui.Button(c.Key("github"), "").Label(s.text.GitHub)
			github.Children(func() {
				ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, icons.External).FontSize(t.Space(3))
					ui.Text(c, s.text.GitHub).SingleLine()
				})
			})
			if github.Clicked() {
				c.OpenURL(Repository)
			}
			if ui.PrimaryButton(c.Key("check"), s.text.Check).Disabled(s.updates != Available).Clicked() {
				s.act.CheckForUpdates()
			}
		})
		if s.updates == Unavailable {
			ui.Text(c, s.text.Cannot).FontSize(t.Space(2.75)).TextColor(t.TextMuted).TextAlign(ui.Center).
				MaxWidth(t.Space(65)).Wrap().Margin(t.Space(2), 0, 0, 0)
		}
	})
}
