package panel

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/match"
)

const copiedFor = 1200 * time.Millisecond

func svg(paths string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="#000" stroke-linecap="round" stroke-linejoin="round">` + paths + `</svg>`))
}

var (
	closeIcon = svg(`<path d="M4 4l8 8M12 4l-8 8" stroke-width="1.6"/>`)
	copyIcon  = svg(`<rect x="5.5" y="5.5" width="8" height="8" rx="1.5" stroke-width="1.4"/><path d="M10.5 3.5v-.5a1 1 0 0 0-1-1h-6a1 1 0 0 0-1 1v6a1 1 0 0 0 1 1h.5" stroke-width="1.4"/>`)
	checkIcon = svg(`<path d="M3.5 8.5l3 3 6-7" stroke-width="1.6"/>`)
	plusIcon  = svg(`<path d="M8 3v10M3 8h10" stroke-width="1.6"/>`)
	chevIcon  = svg(`<path d="M4 6l4 4 4-4" stroke-width="1.6"/>`)
	warnIcon  = svg(`<path d="M8 2.5l6 11H2z" stroke-width="1.6"/><path d="M8 6.5v3M8 11.6v.1" stroke-width="1.6"/>`)
)

// View draws the panel. It reads state only; drops and clicks go to the owner through Actions. Spacing and type
// sizes derive from the theme so they follow its Spacing, FontSize and the desktop's text scale; see docs/design.md.
func (s *Screen) View(c *ui.Context) {
	t := c.Theme()
	if c.Shortcut(0, ui.KeyEscape) {
		if s.confirmTo != "" {
			// Cancel is the Escape action of the confirmation.
			s.confirmTo = ""
		} else {
			s.act.Hide()
		}
	}
	ui.Column(c).Fill().Children(func() {
		// No hide button: the tray icon and Escape hide the panel, as with other tray panels.
		ui.Row(c).PaddingY(t.Space(3)).PaddingX(t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Idset").FontSize(t.Rem(1.15)).Bold().Role(ui.RoleHeading).Level(1)
		})
		ui.Divider(c)
		// States without a list never need scrolling, and centering only works in a container of fixed height.
		if s.loading || s.readErr != nil || len(s.accounts) == 0 {
			ui.Column(c).Grow(1).Padding(t.Space(4)).Gap(t.Space(3)).Children(func() { s.content(c) })
		} else {
			ui.Scroll(c).Grow(1).Padding(t.Space(4)).Gap(t.Space(3)).Children(func() { s.content(c) })
		}
		ui.Divider(c)
		ui.Row(c).PaddingY(t.Space(3)).PaddingX(t.Space(4)).AlignItems(ui.Center).Children(func() {
			// Add Identity stays in the fixed footer, reachable however long the list is. Without a list the
			// empty state has its own button.
			if !s.loading && s.readErr == nil && len(s.accounts) > 0 {
				if ui.Button(c, "").Label(s.text.AddIdentity).Background(ui.Color{}).Border(0, ui.Color{}).TextColor(t.Accent).
					Children(func() {
						ui.Icon(c, plusIcon).FontSize(t.Rem(1.1))
						ui.Text(c, s.text.AddIdentity).SingleLine()
					}).Clicked() {
					s.act.NewIdentity()
				}
			}
			ui.Spacer(c)
			if ui.Button(c, s.text.Quit).Clicked() {
				s.act.Quit()
			}
		})
	})
	if s.say != "" {
		c.Announce(s.say)
		s.say = ""
	}
	if s.announcedSeq != s.resultSeq {
		// The result card is not focusable, so tell screen readers what the dropped folder resolved to.
		s.announcedSeq = s.resultSeq
		if msg := s.resultMessage(); msg != "" {
			c.Announce(msg)
		}
	}
}

// ghost is a borderless square symbol button, 24 DIPs at the default spacing, for actions inside a card.
func ghost(c *ui.Context, label string, icon *ui.SVG) ui.Element {
	t := c.Theme()
	b := ui.Button(c, "").Label(label).Size(t.Space(6), t.Space(6)).Padding(0).Radius(t.Radius).Border(0, ui.Color{}).
		Background(ui.Color{}).TextColor(t.TextMuted).Children(func() { ui.Icon(c, icon).FontSize(t.Rem(1.1)) })
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	return b
}

func (s *Screen) content(c *ui.Context) {
	t := c.Theme()
	switch {
	case s.loading:
		ui.Column(c).Grow(1).Gap(t.Space(2)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			ui.Spinner(c)
			ui.Text(c, s.text.Loading).TextColor(t.TextMuted)
		})
		return
	case s.readErr != nil:
		s.readOnly(c)
		return
	}
	s.resultCard(c)
	s.dropZone(c)
	if len(s.accounts) == 0 {
		ui.Column(c).Grow(1).Gap(t.Space(1.5)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			ui.Text(c, s.text.Empty).Bold().Role(ui.RoleHeading).Level(2)
			ui.Text(c, s.text.EmptyHint).FontSize(t.Rem(0.9)).TextColor(t.TextMuted).TextAlign(ui.Center)
			ui.Row(c).PaddingY(t.Space(1)).Children(func() {
				if ui.Button(c, s.text.AddIdentity).Clicked() {
					s.act.NewIdentity()
				}
			})
		})
		return
	}
	ui.Row(c).PaddingX(t.Space(0.5)).Children(func() {
		ui.Text(c, s.text.Identities).FontSize(t.Rem(0.9)).FontWeight(600).TextColor(t.TextMuted).Grow(1).
			Role(ui.RoleHeading).Level(2)
		ui.Textf(c, "%d", len(s.accounts)).FontSize(t.Rem(0.9)).FontWeight(600).TextColor(t.TextMuted)
	})
	if len(s.accounts) >= FilterThreshold {
		ui.TextInput(c.Key("filter"), &s.query).Label(s.text.Filter).Placeholder(s.text.Filter)
		if used := account.ProvidersInUse(s.accounts); len(used) > 1 {
			ui.Row(c).Gap(t.Space(1.5)).Children(func() {
				for _, p := range used {
					s.providerChip(c, p)
				}
			})
		}
	}
	visible := s.visible()
	if len(visible) == 0 {
		ui.Text(c, s.text.NoneMatch).TextColor(t.TextMuted).AlignSelf(ui.Center)
		return
	}
	ui.Column(c).Radius(10).Background(t.Surface).Border(1, t.Border).Clip().Children(func() {
		for i, a := range visible {
			if i > 0 {
				ui.Divider(c)
			}
			s.row(c.Key(a.ID), a)
		}
	})
}

// readOnly takes the whole panel: while accounts.json cannot be read there is nothing else to do.
func (s *Screen) readOnly(c *ui.Context) {
	t := c.Theme()
	reason := s.readErr
	if inner := errors.Unwrap(reason); inner != nil {
		reason = inner
	}
	ui.Column(c).Grow(1).Gap(t.Space(2.5)).PaddingX(t.Space(2)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ui.Column(c).Size(t.Space(11), t.Space(11)).Radius(11).Background(t.Danger.Alpha(0.14)).AlignItems(ui.Center).
			Justify(ui.Center).Children(func() {
			ui.Icon(c, warnIcon).FontSize(t.Rem(1.7)).TextColor(t.Danger)
		})
		ui.Text(c, s.text.ReadOnly).FontSize(t.Rem(1.15)).FontWeight(600).Role(ui.RoleHeading).Level(2)
		ui.Text(c, s.text.ReadOnlyHint).FontSize(t.Rem(0.9)).TextColor(t.TextMuted).TextAlign(ui.Center)
		ui.Column(c).FillWidth().Gap(t.Space(0.5)).PaddingY(t.Space(1.5)).PaddingX(t.Space(2)).Radius(t.Radius).
			Background(t.Background).Border(1, t.Border).Children(func() {
			ui.Text(c, s.env.Tilde(s.dataPath)).Font("monospace").FontSize(t.Rem(0.85)).TextColor(t.TextMuted).Selectable()
			ui.Text(c, reason.Error()).Font("monospace").FontSize(t.Rem(0.85)).TextColor(t.TextMuted).Selectable()
		})
		ui.Row(c).Gap(t.Space(2)).PaddingY(t.Space(1)).Children(func() {
			if ui.Button(c, s.revealLabel()).Clicked() {
				s.act.Reveal(s.dataPath)
			}
			if ui.Button(c, s.text.Reload).Clicked() {
				s.act.Reload()
			}
		})
	})
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
	b := ui.Button(c.Key("chip-"+string(p)), s.providerName(p)).Selected(on).FontSize(t.Rem(0.9)).Radius(12)
	if on {
		b.Background(t.Accent).TextColor(t.AccentText)
	}
	if b.Clicked() {
		s.toggleProvider(p)
	}
}

// dot is the identity's color as a small round mark, lighter on a dark theme for contrast; the account window's
// source list uses the same one.
func dot(c *ui.Context, a account.Account, size float32) {
	r, g, b := a.RGB()
	fill := ui.RGB(r, g, b)
	if c.Theme().Dark {
		light := func(v uint8) uint8 { return v + uint8(float32(255-v)*0.4) }
		fill = ui.RGB(light(r), light(g), light(b))
	}
	ui.Column(c).Size(size, size).Radius(size / 2).Background(fill)
}

// initial is the first letter of the label in upper case, or "?" without a label.
func initial(a account.Account) string {
	for _, r := range strings.TrimSpace(a.Label) {
		return string(unicode.ToUpper(r))
	}
	return "?"
}

// badge draws the identity's initial on a square tinted with its color, lighter on a dark theme for contrast.
func badge(c *ui.Context, a account.Account, size, radius, font float32) {
	r, g, b := a.RGB()
	fill, ink := ui.RGBA(r, g, b, 0.16), ui.RGB(r, g, b)
	if c.Theme().Dark {
		light := func(v uint8) uint8 { return v + uint8(float32(255-v)*0.4) }
		fill, ink = ui.RGBA(r, g, b, 0.24), ui.RGB(light(r), light(g), light(b))
	}
	ui.Column(c).Size(size, size).Radius(radius).Background(fill).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ui.Text(c, initial(a)).FontSize(font).Bold().TextColor(ink)
	})
}

func usablePaths(a account.Account) []string {
	var out []string
	for _, p := range a.Scope.Paths {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}

// folders is the short scope at the end of a list row: the first folder and how many more.
func (s *Screen) folders(a account.Account) string {
	paths := usablePaths(a)
	switch {
	case a.Scope.Global:
		return s.text.Global
	case len(paths) == 0:
		return s.text.NoFolders
	case len(paths) == 1:
		return paths[0]
	}
	return fmt.Sprintf("%s +%d", paths[0], len(paths)-1)
}

// folderLines are the folders an identity applies to, one per line: "Global" or "no folders" in a line of their own.
func (s *Screen) folderLines(a account.Account) []string {
	paths := usablePaths(a)
	switch {
	case a.Scope.Global:
		return []string{s.text.Global}
	case len(paths) == 0:
		return []string{s.text.NoFolders}
	}
	return paths
}

func orDefault(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// row is one identity; a click opens it in the accounts window.
func (s *Screen) row(c *ui.Context, a account.Account) {
	t := c.Theme()
	row := ui.Row(c).Focusable().Gap(t.Space(2.5)).PaddingY(t.Space(2.25)).PaddingX(t.Space(3)).AlignItems(ui.Center)
	if row.Hovered() {
		row.Background(t.SurfaceHover)
	}
	if row.Clicked() {
		s.act.OpenAccounts(a.ID)
	}
	row.Children(func() {
		dot(c, a, t.Space(2.5))
		ui.Column(c).Grow(1).Gap(t.Space(0.25)).Children(func() {
			ui.Text(c, s.label(a)).FontWeight(600)
			ui.Text(c, orDefault(a.GitUserEmail, s.text.NoEmail)).FontSize(t.Rem(0.9)).TextColor(t.TextMuted)
		})
		ui.Text(c, s.folders(a)).Font("monospace").FontSize(t.Rem(0.85)).TextColor(t.TextMuted)
	})
}

func (s *Screen) dropZone(c *ui.Context) {
	t := c.Theme()
	compact := s.result != nil || s.resultErr != nil || s.checking != ""
	zone := ui.Column(c).Key("drop").Radius(10).Border(1.5, t.Border).BorderStyle(ui.BorderDashed)
	if zone.FileDragOver() {
		zone.Border(1.5, t.Accent).Background(t.Selection)
	}
	if files := zone.DroppedFiles(); len(files) > 0 {
		s.Drop(files)
	}
	if compact {
		zone.Row().PaddingY(t.Space(2)).PaddingX(t.Space(3)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, s.text.DropAgain).FontSize(t.Rem(0.9)).TextColor(t.TextMuted).Grow(1)
			if ui.Button(c, s.text.Choose).Clicked() {
				s.act.ChooseFolder()
			}
		})
		return
	}
	zone.PaddingY(t.Space(5)).PaddingX(t.Space(4)).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
		ui.Text(c, s.text.DropTitle).Bold()
		ui.Text(c, s.text.DropHint).FontSize(t.Rem(0.9)).TextColor(t.TextMuted)
		if ui.Button(c, s.text.Choose).Clicked() {
			s.act.ChooseFolder()
		}
	})
}

// resultCard is a context line with the folder and its actions, then the answer.
func (s *Screen) resultCard(c *ui.Context) {
	t := c.Theme()
	// Clearing happens inside the card, so the rest of this frame works on the state it started with.
	r, checking, resultErr, dir := s.result, s.checking, s.resultErr, s.lastDir
	say := func(text string, color ui.Color, weight int) func() {
		return func() { ui.Text(c, text).TextColor(color).FontWeight(weight).Padding(t.Space(3)) }
	}
	switch {
	case checking != "":
		s.frame(c, "result", t.Border, checking, false, func() {
			ui.Row(c).Gap(t.Space(2)).Padding(t.Space(3)).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c)
				ui.Text(c, s.text.Checking)
			})
		})
	case resultErr != nil:
		s.frame(c, "result", t.Danger, dir, false, say(fmt.Sprintf(s.text.GitFailed, resultErr), t.Danger, 400))
	case r == nil:
	case r.Kind == match.Matched:
		a, ok := s.find(r.ID)
		if !ok {
			return
		}
		// Once the winner has the folder's own path the choice is explicit, and nothing is offered.
		if len(r.Covering) >= 2 && !r.Exact {
			s.overlapCard(c, r)
		}
		s.matchedCard(c, r, a)
	case r.Kind == match.NoMatch:
		s.noMatchCard(c, r)
	default:
		path := r.Root
		if path == "" {
			path = r.Dir
		}
		s.frame(c, "result", t.Warning, path, false, say(s.unresolved(r.Kind), t.Text, 600))
	}
}

// frame is a result card: the folder with its acknowledgements and buttons, a hairline, then the body.
func (s *Screen) frame(c *ui.Context, key string, border ui.Color, path string, copyable bool, body func()) {
	t := c.Theme()
	checking := s.checking
	ui.Column(c).Key(key).Radius(10).Background(t.Surface).Border(1, border).Clip().Children(func() {
		ui.Row(c).Gap(t.Space(1)).Padding(t.Space(2), t.Space(2), t.Space(2), t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, s.env.Tilde(path)).Font("monospace").FontSize(t.Rem(0.9)).TextColor(t.TextMuted).Grow(1).Selectable()
			if left := s.undoneUntil.Sub(s.now()); left > 0 {
				ui.Text(c, s.text.Undone).FontSize(t.Rem(0.85)).TextColor(t.TextMuted)
				c.After(left)
			}
			if copyable {
				s.copyButton(c, path)
			}
			if checking == "" && ghost(c, s.text.Clear, closeIcon).Clicked() {
				s.clear()
			}
		})
		ui.Divider(c)
		body()
	})
}

// copyButton copies the full path; for a moment afterwards it shows a check mark and "Copied".
func (s *Screen) copyButton(c *ui.Context, path string) {
	t := c.Theme()
	icon := copyIcon
	if s.now().Before(s.copiedUntil) {
		ui.Text(c, s.text.Copied).FontSize(t.Rem(0.85)).TextColor(t.TextMuted)
		icon = checkIcon
	}
	if ghost(c, s.text.CopyPath, icon).Clicked() {
		s.act.CopyText(path)
		s.copiedUntil = s.now().Add(copiedFor)
		c.After(copiedFor)
		// The focus stays on the button, whose name does not change, so say that the copy worked.
		c.Announce(s.text.Copied)
	}
}
