// Package example demonstrates a native view with state owned by the UI thread.
package example

import (
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/egoist/mygo/ui"
)

//go:embed copy.json
var catalog []byte

type copy struct{ Title, Increment, Decrement, Reset, Hint string }

var translations = func() map[string]copy {
	var values map[string]copy
	if err := json.Unmarshal(catalog, &values); err != nil {
		panic(err)
	}
	return values
}()

// Screen keeps only the in-memory state of the example.
type Screen struct {
	count int
	text  copy
}

// New selects the best available translation, falling back to English.
func New(locale string) *Screen {
	locale = strings.ToLower(strings.ReplaceAll(locale, "_", "-"))
	language := "en"
	switch {
	case strings.HasPrefix(locale, "zh-hant"), strings.HasPrefix(locale, "zh-tw"), strings.HasPrefix(locale, "zh-hk"), strings.HasPrefix(locale, "zh-mo"):
		language = "zh-Hant"
	case strings.HasPrefix(locale, "zh"):
		language = "zh-Hans"
	case strings.HasPrefix(locale, "pt"):
		language = "pt-BR"
	default:
		if base := strings.Split(locale, "-")[0]; translations[base].Title != "" {
			language = base
		}
	}
	return &Screen{text: translations[language]}
}

// View handles input and draws the current count, without disk or network work.
func (s *Screen) View(c *ui.Context) {
	if c.Shortcut(0, ui.KeyUp) {
		s.count++
	}
	if c.Shortcut(0, ui.KeyDown) {
		s.count--
	}
	ui.Column(c).Fill().Center().Gap(16).Padding(24).Children(func() {
		ui.Text(c, s.text.Title).FontSize(24).Bold()
		ui.Textf(c, "%d", s.count).FontSize(48).Bold()
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "−").Label(s.text.Decrement).Width(44).Clicked() {
				s.count--
			}
			if ui.Button(c, s.text.Reset).Disabled(s.count == 0).Clicked() {
				s.count = 0
			}
			if ui.PrimaryButton(c, "+").Label(s.text.Increment).Width(44).Clicked() {
				s.count++
			}
		})
		ui.Text(c, s.text.Hint).FontSize(12).TextColor(c.Theme().TextMuted)
	})
}
