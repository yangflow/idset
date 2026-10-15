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
	"github.com/yangflow/idset/internal/ui/icons"
)

const copiedFor = 1200 * time.Millisecond

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
		// No hide button: the tray icon and Escape hide the panel, as with other tray panels. The title bar and the
		// footer are the same height, hairline included, and give the rest to the content.
		ui.Row(c).Height(t.Space(9)-1).Padding(0, t.Space(2), 0, t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Idset").FontWeight(600).Role(ui.RoleHeading).Level(1)
		})
		ui.Divider(c)
		// States without a list never need scrolling, and centering only works in a container of fixed height.
		if s.loading || s.readErr != nil || len(s.accounts) == 0 {
			ui.Column(c).Grow(1).Padding(t.Space(4)).Gap(t.Space(3)).Children(func() { s.content(c) })
		} else {
			ui.Scroll(c).Grow(1).Padding(t.Space(4)).Gap(t.Space(3)).Children(func() { s.content(c) })
		}
		ui.Divider(c)
		ui.Row(c).Height(t.Space(9) - 1).PaddingX(t.Space(2)).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
			// Add Identity stays in the fixed footer, reachable however long the list is. Without a list the
			// empty state has its own button. Only actions that work are here.
			if !s.loading && s.readErr == nil && len(s.accounts) > 0 && ghost(c, s.text.AddIdentity, icons.Plus).Clicked() {
				s.act.NewIdentity()
			}
			ui.Spacer(c)
			if ghost(c, s.text.Quit, icons.Power).Clicked() {
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

// ghost is a borderless square symbol button, 24 DIPs at the default spacing, with a 16 DIP glyph. Its label is
// its accessible name and its tooltip.
func ghost(c *ui.Context, label string, icon *ui.SVG) ui.Element {
	t := c.Theme()
	b := ui.Button(c.Key("ghost-"+label), "").Label(label).Size(t.Space(6), t.Space(6)).Padding(0).Radius(t.Radius).Border(0, ui.Color{}).
		Background(ui.Color{}).TextColor(t.TextMuted).Tooltip(label).Children(func() { ui.Icon(c, icon).FontSize(t.Rem(1.2)) })
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
	if len(s.accounts) == 0 {
		// One centered block, with the drop zone under it.
		ui.Column(c).Grow(1).Gap(t.Space(1.5)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			ui.Text(c, s.text.Empty).FontSize(t.Rem(1.15)).FontWeight(600).Role(ui.RoleHeading).Level(2)
			ui.Text(c, s.text.EmptyHint).FontSize(t.Rem(0.9)).TextColor(t.TextMuted).TextAlign(ui.Center)
			ui.Row(c).PaddingY(t.Space(1)).Children(func() {
				if ui.PrimaryButton(c, s.text.AddIdentity).Height(t.Space(7)).PaddingX(t.Space(4)).Clicked() {
					s.act.NewIdentity()
				}
			})
		})
		s.dropZone(c)
		return
	}
	s.dropZone(c)
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
			ui.Icon(c, icons.Warn).FontSize(t.Rem(1.7)).TextColor(t.Danger)
		})
		ui.Text(c, s.text.ReadOnly).FontSize(t.Rem(1.15)).FontWeight(600).Role(ui.RoleHeading).Level(2)
		ui.Text(c, s.env.Tilde(s.dataPath)).Font("monospace").FontSize(t.Rem(0.9)).TextColor(t.TextMuted).TextAlign(ui.Center).Selectable()
		// The parser error is at most two lines; the tooltip has all of it.
		ui.Text(c, reason.Error()).Font("monospace").FontSize(t.Rem(0.85)).TextColor(t.TextMuted).TextAlign(ui.Center).MaxLines(2).
			Tooltip(reason.Error())
		ui.Text(c, s.text.ReadOnlyShort).FontSize(t.Rem(0.9)).TextColor(t.TextMuted).Tooltip(s.text.ReadOnlyHint)
		ui.Row(c).Gap(t.Space(2)).PaddingY(t.Space(1)).Children(func() {
			if ui.Button(c, s.revealLabel()).Label(s.revealLabel()).Size(t.Space(30), t.Space(7)).Padding(0).Clicked() {
				s.act.Reveal(s.dataPath)
			}
			if ui.Button(c, s.text.Reload).Label(s.text.Reload).Size(t.Space(30), t.Space(7)).Padding(0).Clicked() {
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

// folderTag is the short scope at the end of a list row: the first folder and, after it, how many more. The
// folder gives way in the middle before the row's name does, and " +N" always stays.
func (s *Screen) folderTag(c *ui.Context, a account.Account) {
	t := c.Theme()
	face := pathFace{mono: true, size: t.Rem(0.85), color: t.TextMuted}
	folders := usablePaths(a)
	switch {
	case a.Scope.Global:
		face.apply(ui.Text(c, s.text.Global)).SingleLine()
	case len(folders) == 0:
		face.apply(ui.Text(c, s.text.NoFolders)).SingleLine()
	default:
		suffix := ""
		if len(folders) > 1 {
			suffix = fmt.Sprintf(" +%d", len(folders)-1)
		}
		pathLine(c, folders[0], suffix, "", face).MaxWidthPercent(50).Shrink(0)
	}
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
		ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(0.25)).Children(func() {
			ui.Text(c, s.label(a)).FontWeight(600).SingleLine()
			ui.Text(c, orDefault(a.GitUserEmail, s.text.NoEmail)).FontSize(t.Rem(0.9)).TextColor(t.TextMuted).SingleLine()
		})
		s.folderTag(c, a)
	})
}

// dropZone is always one line; what it says beyond that is in the tooltip.
func (s *Screen) dropZone(c *ui.Context) {
	t := c.Theme()
	zone := ui.Column(c).Key("drop").Radius(10).Border(1.5, t.Border).BorderStyle(ui.BorderDashed)
	if zone.FileDragOver() {
		zone.Border(1.5, t.Accent).Background(t.Selection)
	}
	if files := zone.DroppedFiles(); len(files) > 0 {
		s.Drop(files)
	}
	zone.Row().PaddingY(t.Space(2)).PaddingX(t.Space(3)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
		ui.Text(c, s.text.DropLine).FontSize(t.Rem(0.9)).TextColor(t.TextMuted).SingleLine().MinWidth(0).Grow(1).Tooltip(s.text.DropHint)
		if ui.Button(c, s.text.ChooseShort).Label(s.text.Choose).Tooltip(s.text.Choose).Clicked() {
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
		s.frame(c, "result", t.Border, checking, head{}, func() {
			ui.Row(c).Gap(t.Space(2)).Padding(t.Space(3)).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c)
				ui.Text(c, s.text.Checking)
			})
		})
	case resultErr != nil:
		// The headline is short; the reason is at most two lines, and the tooltip has all of it.
		s.frame(c, "result", t.Danger, dir, head{}, func() {
			ui.Column(c).Gap(t.Space(1)).Padding(t.Space(3)).Children(func() {
				ui.Text(c, s.text.GitFailed).TextColor(t.Danger)
				ui.Text(c, resultErr.Error()).Font("monospace").FontSize(t.Rem(0.85)).TextColor(t.TextMuted).MaxLines(2).
					Tooltip(resultErr.Error())
			})
		})
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
		if a, ok := s.stale(); ok {
			s.staleCard(c, r.Stale, a, r.Root)
		}
		s.noMatchCard(c, r)
	default:
		path := r.Root
		if path == "" {
			path = r.Dir
		}
		s.frame(c, "result", t.Warning, path, head{}, say(s.unresolved(r.Kind), t.Text, 600))
	}
}

// head says what the context line of a result card holds besides the folder and the clear button.
type head struct {
	// actions are the icon buttons before the clear button; a card with them can be copied from.
	actions func()
	// menu is called after the context line, to attach a menu to one of its buttons.
	menu func()
}

// frame is a result card: the folder with its status and icon buttons, a hairline, then the body.
func (s *Screen) frame(c *ui.Context, key string, border ui.Color, path string, h head, body func()) {
	t := c.Theme()
	checking := s.checking
	ui.Column(c).Key(key).Radius(10).Background(t.Surface).Border(1, border).Clip().Children(func() {
		// The context line is as high as the title row and the footer, hairline included.
		ui.Row(c).Height(t.Space(9)-1).Gap(t.Space(1)).Padding(0, t.Space(2), 0, t.Space(3)).AlignItems(ui.Center).Children(func() {
			face := pathFace{mono: true, size: t.Rem(0.9), color: t.TextMuted}
			// The folder keeps at least 80 DIPs; the status gives way first.
			pathLine(c, s.env.Tilde(path), "", "", face).Grow(1).MinWidth(t.Space(20)).Shrink(0)
			s.status(c, h.actions != nil)
			if h.actions != nil {
				h.actions()
			}
			if checking == "" && ghost(c, s.text.Clear, icons.Close).Clicked() {
				s.clear()
			}
		})
		if h.menu != nil {
			h.menu()
		}
		ui.Divider(c)
		body()
	})
}

// status is the short, transient word before the buttons of the context line: Copied, Updated, Undone, or
// Covered by a folder, whose full sentence is the tooltip.
func (s *Screen) status(c *ui.Context, copyable bool) {
	t := c.Theme()
	for _, ack := range []struct {
		until      time.Time
		text, full string
		show       bool
	}{
		{s.undoneUntil, s.text.Undone, "", true},
		{s.updatedUntil, s.text.Updated, "", true},
		{s.coveredUntil, s.coveredBy, s.note, true},
		{s.copiedUntil, s.text.Copied, "", copyable},
	} {
		if left := ack.until.Sub(s.now()); ack.show && left > 0 {
			text := ui.Text(c, ack.text).FontSize(t.Rem(0.85)).TextColor(t.TextMuted).SingleLine().MinWidth(0)
			if ack.full != "" {
				text.Tooltip(ack.full)
			}
			c.After(left)
		}
	}
}

// copyButton copies the full path; for a moment afterwards it shows a check mark, and the status says "Copied".
func (s *Screen) copyButton(c *ui.Context, path string) {
	icon := icons.Copy
	if s.now().Before(s.copiedUntil) {
		icon = icons.Check
	}
	if ghost(c, s.text.CopyPath, icon).Clicked() {
		s.act.CopyText(path)
		s.copiedUntil = s.now().Add(copiedFor)
		c.After(copiedFor)
		// The focus stays on the button, whose name does not change, so say that the copy worked.
		c.Announce(s.text.Copied)
	}
}
