package panel

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/match"
)

const copiedFor = 1200 * time.Millisecond

var closeIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><path d="M4 4l8 8M12 4l-8 8" stroke="#000" stroke-width="1.6" stroke-linecap="round"/></svg>`))

// View draws the panel. It reads state only; drops and clicks go to the owner through Actions.
func (s *Screen) View(c *ui.Context) {
	if c.Shortcut(0, ui.KeyEscape) {
		s.act.Hide()
	}
	ui.Column(c).Fill().Children(func() {
		ui.Row(c).PaddingY(12).PaddingX(16).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Idset").FontSize(15).Bold().Grow(1)
			if iconButton(c, s.text.Hide).Clicked() {
				s.act.Hide()
			}
		})
		ui.Divider(c)
		ui.Scroll(c).Grow(1).Padding(16).Gap(12).Children(func() { s.content(c) })
		ui.Divider(c)
		ui.Row(c).PaddingY(12).PaddingX(16).Children(func() {
			ui.Spacer(c)
			if ui.Button(c, s.text.Quit).Clicked() {
				s.act.Quit()
			}
		})
	})
}

func iconButton(c *ui.Context, label string) ui.Element {
	return ui.Button(c, "").Label(label).Width(28).Children(func() { ui.Icon(c, closeIcon) })
}

func (s *Screen) content(c *ui.Context) {
	t := c.Theme()
	switch {
	case s.loading:
		ui.Column(c).Grow(1).Gap(8).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			ui.Spinner(c)
			ui.Text(c, s.text.Loading).TextColor(t.TextMuted)
		})
		return
	case s.readErr != nil:
		card(c, t.Danger).Children(func() {
			ui.Text(c, s.text.ReadOnly).Bold().TextColor(t.Danger)
			ui.Text(c, s.env.Tilde(s.dataPath)).Font("monospace").FontSize(12).Selectable()
			ui.Text(c, s.text.ReadOnlyHint).FontSize(12)
		})
		return
	}
	s.resultCard(c)
	s.dropZone(c)
	if len(s.accounts) == 0 {
		ui.Column(c).Grow(1).Gap(6).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			ui.Text(c, s.text.Empty).Bold()
			ui.Text(c, s.text.EmptyHint).FontSize(12).TextColor(t.TextMuted)
		})
		return
	}
	if len(s.accounts) >= FilterThreshold {
		ui.TextInput(c.Key("filter"), &s.query).Label(s.text.Filter).Placeholder(s.text.Filter)
		if used := account.ProvidersInUse(s.accounts); len(used) > 1 {
			ui.Row(c).Gap(6).Children(func() {
				for _, p := range used {
					s.providerChip(c, p)
				}
			})
		}
	}
	visible := s.visible()
	if len(visible) == 0 {
		ui.Text(c, s.text.NoneMatch).TextColor(t.TextMuted).AlignSelf(ui.Center)
	}
	ui.Column(c).Gap(2).Children(func() {
		for _, a := range visible {
			s.row(c.Key(a.ID), a)
		}
	})
}

func card(c *ui.Context, border ui.Color) ui.Element {
	t := c.Theme()
	return ui.Column(c).Gap(4).Padding(12).Radius(8).Background(t.Surface).Border(1, border)
}

func (s *Screen) providerName(p account.Provider) string {
	if p == account.Custom {
		return s.text.Custom
	}
	return p.DisplayName()
}

func (s *Screen) providerChip(c *ui.Context, p account.Provider) {
	t := c.Theme()
	on := false
	for _, q := range s.providers {
		on = on || q == p
	}
	b := ui.Button(c.Key("chip-"+string(p)), s.providerName(p)).Selected(on).FontSize(12).Radius(12)
	if on {
		b.Background(t.Accent).TextColor(t.AccentText)
	}
	if b.Clicked() {
		s.toggleProvider(p)
	}
}

func dot(c *ui.Context, a account.Account) {
	r, g, b := a.RGB()
	ui.Box(c).Size(8, 8).Radius(4).Background(ui.RGB(r, g, b))
}

func (s *Screen) subtitle(a account.Account) string {
	alias, email := a.SSHAlias, a.GitUserEmail
	if alias == "" {
		alias = s.text.NoAlias
	}
	if email == "" {
		email = s.text.NoEmail
	}
	return alias + " · " + email
}

func (s *Screen) row(c *ui.Context, a account.Account) {
	t := c.Theme()
	ui.Row(c).Gap(10).PaddingY(8).PaddingX(4).AlignItems(ui.Center).Children(func() {
		dot(c, a)
		ui.Column(c).Grow(1).Children(func() {
			ui.Text(c, s.label(a)).SingleLine()
			ui.Text(c, s.subtitle(a)).FontSize(12).TextColor(t.TextMuted).SingleLine()
		})
	})
}

func (s *Screen) scopeText(a account.Account) string {
	if a.Scope.Global {
		return s.text.ScopeGlobal
	}
	var parts []string
	for _, p := range a.Scope.Paths {
		if strings.TrimSpace(p) != "" {
			parts = append(parts, p)
		}
	}
	return fmt.Sprintf(s.text.ScopePaths, strings.Join(parts, " · "))
}

func (s *Screen) dropZone(c *ui.Context) {
	t := c.Theme()
	compact := s.result != nil || s.resultErr != nil || s.checking != ""
	zone := ui.Column(c).Key("drop").Radius(8).Border(1.5, t.Border).BorderStyle(ui.BorderDashed)
	if zone.FileDragOver() {
		zone.Border(1.5, t.Accent).Background(t.Selection)
	}
	if files := zone.DroppedFiles(); len(files) > 0 {
		s.Drop(files)
	}
	if compact {
		zone.Row().PaddingY(8).PaddingX(12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, s.text.DropAgain).FontSize(12).TextColor(t.TextMuted).Grow(1)
			if ui.Button(c, s.text.Choose).Clicked() {
				s.act.ChooseFolder()
			}
		})
		return
	}
	zone.PaddingY(20).PaddingX(16).Gap(6).AlignItems(ui.Center).Children(func() {
		ui.Text(c, s.text.DropTitle).Bold()
		ui.Text(c, s.text.DropHint).FontSize(12).TextColor(t.TextMuted)
		if ui.Button(c, s.text.Choose).Clicked() {
			s.act.ChooseFolder()
		}
	})
}

func (s *Screen) resultCard(c *ui.Context) {
	t := c.Theme()
	// Clearing happens inside the card, so the rest of this frame works on the result it started with.
	r := s.result
	header := func(title string, a *account.Account) {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			if a != nil {
				dot(c, *a)
			}
			ui.Text(c, title).Bold().Grow(1)
			if iconButton(c, s.text.Clear).Clicked() {
				s.clear()
			}
		})
	}
	switch {
	case s.checking != "":
		card(c, t.Border).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c)
				ui.Text(c, fmt.Sprintf(s.text.Checking, s.env.Tilde(s.checking))).FontSize(12).Grow(1)
			})
		})
	case s.resultErr != nil:
		card(c, t.Danger).Children(func() {
			header(fmt.Sprintf(s.text.GitFailed, s.resultErr), nil)
		})
	case r == nil:
	case r.Kind == match.Matched:
		a, ok := s.find(r.ID)
		if !ok {
			return
		}
		card(c, t.Border).Children(func() {
			header(s.label(a), &a)
			ui.Text(c, s.subtitle(a)).FontSize(12).TextColor(t.TextMuted)
			ui.Text(c, s.scopeText(a)).FontSize(12)
			s.pathButton(c, r.Root)
		})
	default:
		msg := map[match.Kind]string{match.NoMatch: s.text.NoMatch, match.NotRepo: s.text.NotRepo, match.NotFolder: s.text.NotFolder}[r.Kind]
		card(c, t.Warning).Children(func() {
			header(msg, nil)
			path := r.Root
			if path == "" {
				path = r.Dir
			}
			ui.Text(c, s.env.Tilde(path)).Font("monospace").FontSize(12).SingleLine()
		})
	}
}

func (s *Screen) pathButton(c *ui.Context, path string) {
	t := c.Theme()
	copied := s.now().Before(s.copiedUntil)
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		b := ui.Text(c, s.env.Tilde(path)).Font("monospace").FontSize(12).SingleLine().Underline().
			Cursor(ui.CursorPointer).Tooltip(s.text.CopyPath).Description(s.text.CopyPath).Focusable()
		if b.Clicked() {
			s.act.CopyText(path)
			s.copiedUntil = s.now().Add(copiedFor)
			c.After(copiedFor)
		}
		if copied {
			ui.Text(c, s.text.Copied).FontSize(12).TextColor(t.TextMuted)
		}
	})
}
