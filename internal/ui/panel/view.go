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
	"github.com/yangflow/idset/internal/ui/theme"
)

const copiedFor = 1200 * time.Millisecond

// View draws the panel. It reads state only; drops and clicks go to the owner through Actions. Spacing and type
// sizes derive from the theme so they follow its Spacing, FontSize and the desktop's text scale; see docs/design.md.
func (s *Screen) View(c *ui.Context) {
	theme.Apply(c)
	t := c.Theme()
	if c.Shortcut(0, ui.KeyEscape) {
		if s.confirmTo != "" {
			// Cancel is the Escape action of the confirmation.
			s.confirmTo = ""
		} else {
			s.act.Hide()
		}
	}
	root := ui.Column(c).Fill()
	// While a result shows, the drop zone is gone and the whole panel takes a dropped folder; dragging one over it
	// shows an overlay that says what releasing does.
	dropOver := false
	if s.showsResult() {
		dropOver = root.FileDragOver()
		if files := root.DroppedFiles(); len(files) > 0 {
			s.Drop(files)
		}
	}
	root.Children(func() {
		// No hide button: the tray icon and Escape hide the panel, as with other tray panels. The title bar and the
		// footer give the rest to the content: the title bar is 44 high and the footer 40, neither with a hairline.
		ui.Row(c).Height(t.Space(11)).Padding(0, t.Space(2.25), 0, t.Space(4)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Idset").FontSize(t.Rem(1.15)).FontWeight(600).Role(ui.RoleHeading).Level(1)
			ui.Spacer(c)
			if ghost(c, s.text.Settings, icons.Gear).Clicked() {
				s.act.OpenSettings()
			}
		})
		// States without a list never need scrolling, and centering only works in a container of fixed height. The
		// content has 12 at the sides and cards 10 apart.
		if s.loading || s.readErr != nil || len(s.accounts) == 0 {
			ui.Column(c).Grow(1).Padding(t.Space(1), t.Space(3), t.Space(3)).Gap(t.Space(2.5)).Children(func() { s.content(c) })
		} else {
			ui.Scroll(c).Grow(1).Padding(t.Space(1), t.Space(3), t.Space(3)).Gap(t.Space(2.5)).Children(func() { s.content(c) })
		}
		ui.Row(c).Height(t.Space(10)).PaddingX(t.Space(2)).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
			// Add Identity stays in the fixed footer, reachable however long the list is. Without a list the
			// empty state has its own button. Only actions that work are here.
			if !s.loading && s.readErr == nil && len(s.accounts) > 0 && ghost(c, s.text.AddIdentity, icons.Plus).Clicked() {
				s.act.NewIdentity()
			}
			ui.Spacer(c)
			if ghost(c, s.text.About, icons.Info).Clicked() {
				s.act.OpenAbout()
			}
			if !s.loading && s.readErr == nil && len(s.accounts) > 0 {
				s.recheckButton(c)
			}
			if ghost(c, s.text.Quit, icons.Power).Clicked() {
				s.act.Quit()
			}
		})
		if dropOver {
			s.dropOverlay(c)
		}
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

// dropOverlay covers the content while a folder is dragged over a panel with a result: a tinted card with the
// folder icon and what releasing does.
func (s *Screen) dropOverlay(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Key("drop-over").Absolute().Top(t.Space(11)).Left(t.Space(2)).Right(t.Space(2)).Bottom(t.Space(11.5)).
		Radius(t.Space(3.5)).Border(1.5, t.Accent.Alpha(0.7)).Background(theme.Of(c).AccentSoft).Gap(t.Space(2)).
		AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ui.Icon(c, icons.Folder).FontSize(t.Space(7)).TextColor(t.Accent)
		ui.Text(c, s.text.DropRelease).FontWeight(600).TextAlign(ui.Center)
	})
}

// showsResult reports whether a result card, or the check before it, is on the panel; the drop zone is then gone.
func (s *Screen) showsResult() bool {
	return !s.loading && s.readErr == nil && (s.result != nil || s.checking != "" || s.resultErr != nil)
}

// ghost is a borderless circular symbol button, 26 DIPs at the default spacing, with a 16 DIP glyph. Its label is
// its accessible name and its tooltip.
func ghost(c *ui.Context, label string, icon *ui.SVG) ui.Element {
	return tooltipGhost(c, "ghost-"+label, label, label, icon)
}

// tooltipGhost is a ghost button whose tooltip says more than its label.
func tooltipGhost(c *ui.Context, key, label, tip string, icon *ui.SVG) ui.Element {
	return symbolButton(c, key, label, icon, false).Tooltip(tip)
}

// symbolButton is the ghost button without a tooltip. A done button, one that shows the check mark of a copy, keeps
// its plain look even while the pointer is still over it.
func symbolButton(c *ui.Context, key, label string, icon *ui.SVG, done bool) ui.Element {
	t := c.Theme()
	b := ui.Button(c.Key(key), "").Label(label).Size(t.Space(6.5), t.Space(6.5)).Padding(0).Radius(t.Space(6.5)/2).Border(0, ui.Color{}).
		Background(ui.Color{}).TextColor(t.TextMuted).Children(func() { ui.Icon(c, icon).FontSize(t.Space(4)) })
	if b.Hovered() && !done {
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
		// One centered block, with the drop zone under it: the product's key in a tinted tile, the title and the hint.
		ui.Column(c).Grow(1).Gap(t.Space(2)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			ui.Column(c).Size(t.Space(14), t.Space(14)).Radius(14).Background(theme.Of(c).AccentSoft).AlignItems(ui.Center).
				Justify(ui.Center).Children(func() {
				ui.Icon(c, icons.Key).FontSize(t.Space(7)).TextColor(t.Accent)
			})
			ui.Text(c, s.text.Empty).FontSize(t.Rem(1.15)).FontWeight(600).Role(ui.RoleHeading).Level(2)
			ui.Text(c, s.text.EmptyHint).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).TextAlign(ui.Center)
			ui.Row(c).PaddingY(t.Space(1)).Children(func() {
				if ui.PrimaryButton(c, s.text.AddIdentity).Height(t.Space(7)).PaddingX(t.Space(4)).Clicked() {
					s.act.NewIdentity()
				}
			})
			if s.textButton(c, "import", s.text.ImportExisting, "", nil, t.Accent).Clicked() {
				s.act.OpenImport()
			}
		})
		if !s.showsResult() {
			s.dropZone(c)
		}
		return
	}
	if s.noSSH() {
		s.noSSHNotice(c)
	}
	if !s.showsResult() {
		s.dropZone(c)
	}
	ui.Row(c).PaddingX(t.Space(1.5)).Children(func() {
		ui.Text(c, s.text.Identities).FontSize(t.Rem(0.85)).FontWeight(600).TextColor(t.TextMuted).Grow(1).
			Role(ui.RoleHeading).Level(2)
		ui.Textf(c, "%d", len(s.accounts)).FontSize(t.Rem(0.85)).FontWeight(600).TextColor(t.TextMuted)
	})
	if len(s.accounts) >= FilterThreshold {
		// A capsule search field with a magnifier inside its start, as macOS 26 draws one.
		ui.Row(c).Height(t.Space(7)).AlignItems(ui.Center).Children(func() {
			ui.TextInput(c.Key("filter"), &s.query).Label(s.text.Filter).Placeholder(s.text.Filter).Fill().Radius(theme.Capsule).
				Border(0, ui.Color{}).Background(t.Surface).Padding(0, t.Space(2.5), 0, t.Space(7.25))
			ui.Row(c).Absolute().Top(0).Bottom(0).Left(t.Space(2.5)).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icons.Search).FontSize(t.Rem(1)).TextColor(t.TextMuted)
			})
		})
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
		s.summary(c)
		return
	}
	fillCard(c, "list", ui.Color{}).Children(func() {
		for i, a := range visible {
			if i > 0 {
				insetDivider(c, t.Space(12.5))
			}
			s.row(c.Key(a.ID), a)
		}
	})
	s.summary(c)
}

// readOnly takes the whole panel: while accounts.json cannot be read there is nothing else to do.
func (s *Screen) readOnly(c *ui.Context) {
	t := c.Theme()
	reason := s.readErr
	if inner := errors.Unwrap(reason); inner != nil {
		reason = inner
	}
	ui.Column(c).Grow(1).Gap(t.Space(2.5)).PaddingX(t.Space(2)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ui.Column(c).Size(t.Space(12), t.Space(12)).Radius(12).Background(t.Danger.Alpha(0.14)).AlignItems(ui.Center).
			Justify(ui.Center).Children(func() {
			ui.Icon(c, icons.Warn).FontSize(t.Space(6)).TextColor(t.Danger)
		})
		ui.Text(c, s.text.ReadOnly).FontSize(t.Rem(1.15)).FontWeight(600).Role(ui.RoleHeading).Level(2)
		ui.Text(c, s.env.Tilde(s.dataPath)).Font("monospace").FontSize(t.Rem(0.88)).TextColor(t.TextMuted).TextAlign(ui.Center).Selectable()
		// The parser error is at most two lines; the tooltip has all of it.
		ui.Text(c, reason.Error()).Font("monospace").FontSize(t.Rem(0.85)).TextColor(t.TextMuted).TextAlign(ui.Center).MaxLines(2).
			Tooltip(reason.Error())
		ui.Text(c, s.text.ReadOnlyShort).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).Tooltip(s.text.ReadOnlyHint)
		ui.Column(c).Gap(t.Space(2)).PaddingY(t.Space(1)).AlignItems(ui.Center).Children(func() {
			if ui.PrimaryButton(c, s.text.RestoreBackup).Label(s.text.RestoreBackup).Size(t.Space(62), t.Space(7)).Padding(0).Clicked() {
				s.act.OpenBackups()
			}
			ui.Row(c).Gap(t.Space(2)).Children(func() {
				if ui.Button(c, s.revealLabel()).Label(s.revealLabel()).Size(t.Space(30), t.Space(7)).Padding(0).Clicked() {
					s.act.Reveal(s.dataPath)
				}
				if ui.Button(c, s.text.Reload).Label(s.text.Reload).Size(t.Space(30), t.Space(7)).Padding(0).Clicked() {
					s.act.Reload()
				}
			})
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
	b := ui.Button(c.Key("chip-"+string(p)), s.providerName(p)).Selected(on).FontSize(t.Rem(0.92)).Height(t.Space(6)).Radius(12)
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

// badge draws the identity's initial on a solid square of its color, in white or near black by theme.Ink.
func badge(c *ui.Context, a account.Account, size, radius, font float32) {
	r, g, b := a.RGB()
	ui.Column(c).Size(size, size).Radius(radius).Background(ui.RGB(r, g, b)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ui.Text(c, initial(a)).FontSize(font).FontWeight(600).TextColor(theme.Ink(r, g, b))
	})
}

// fillCard is a result card or the identity list: a fill with the card rim, not an outline. One that needs attention adds a 1 DIP inner
// outline in Warning or Danger.
func fillCard(c *ui.Context, key string, attention ui.Color) ui.Element {
	e := theme.Card(c, ui.Column(c).Key(key)).Clip()
	if attention.A != 0 {
		e.InsetShadow(0, 0, 0, 1, attention)
	}
	return e
}

// insetDivider is a hairline that starts left DIPs in, where a row's name does.
func insetDivider(c *ui.Context, left float32) {
	ui.Column(c).Padding(0, 0, 0, left).Children(func() { ui.Divider(c) })
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
		pathLine(c, folders[0], suffix, "", face).MaxWidthPercent(42).Shrink(0)
	}
}

func orDefault(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// row is one identity; a click opens it in the accounts window. One with an alias ends in a disclosure button
// that opens the clone helper under it.
func (s *Screen) row(c *ui.Context, a account.Account) {
	t := c.Theme()
	open := s.openRow == a.ID && a.SSHAlias != ""
	problem, check, hasProblem := s.problem(a)
	ui.Column(c).Children(func() {
		// A problem bar sits close under its row, without a hairline, as part of it.
		bottom := t.Space(2)
		if hasProblem {
			bottom = t.Space(0.5)
		}
		row := ui.Row(c).Focusable().MinHeight(t.Space(12)).Gap(t.Space(2.5)).Padding(t.Space(2), t.Space(1.5), bottom, t.Space(3)).
			AlignItems(ui.Center)
		if row.Hovered() {
			row.Background(t.SurfaceHover)
		}
		if row.Clicked() {
			s.act.OpenAccounts(a.ID)
		}
		row.Children(func() {
			badge(c, a, t.Space(7), 8, t.Rem(1))
			ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(0.25)).Children(func() {
				ui.Text(c, s.label(a)).FontWeight(600).SingleLine()
				ui.Text(c, orDefault(a.GitUserEmail, s.text.NoEmail)).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).SingleLine()
			})
			s.folderTag(c, a)
			if s.diag.known {
				s.connDot(c, a)
			}
			if a.SSHAlias == "" {
				// The space stays, so that the folders line up.
				ui.Column(c).Size(t.Space(6.5), t.Space(6.5)).Shrink(0)
				return
			}
			icon := icons.ChevronRight
			if open {
				icon = icons.ChevronDown
			}
			toggle := symbolButton(c, "clone-"+a.ID, s.text.CloneToggle, icon, false).Expanded(open)
			tipAboveEnd(c, toggle, s.text.CloneToggle)
			if toggle.Clicked() {
				s.toggleRow(a.ID)
			}
		})
		if hasProblem {
			s.problemBar(c, a, problem, check)
		}
		if open {
			s.openRowClone(c, a)
		}
	})
}

// dropZone is always one line, and only there while no result shows; what it says beyond that is in the tooltip.
func (s *Screen) dropZone(c *ui.Context) {
	t := c.Theme()
	zone := ui.Column(c).Key("drop").Height(t.Space(12)).Radius(theme.CardRadius).Border(1, t.Accent.Alpha(0.5)).BorderStyle(ui.BorderDashed).
		Background(t.Accent.Alpha(0.06))
	if zone.FileDragOver() {
		zone.Border(1.5, t.Accent).BorderStyle(ui.BorderSolid).Background(theme.Of(c).AccentSoft)
	}
	if files := zone.DroppedFiles(); len(files) > 0 {
		s.Drop(files)
	}
	zone.Row().Padding(0, t.Space(2.5), 0, t.Space(3)).Gap(t.Space(2.5)).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icons.Folder).FontSize(t.Space(5)).TextColor(t.Accent).Shrink(0)
		ui.Text(c, s.text.DropLine).FontSize(t.Rem(0.92)).SingleLine().MinWidth(0).Grow(1).Tooltip(s.text.DropHint)
		if ui.PrimaryButton(c, s.text.ChooseShort).Label(s.text.Choose).Tooltip(s.text.Choose).Height(t.Space(6)).
			PaddingX(t.Space(2.5)).FontSize(t.Rem(0.92)).FontWeight(500).Shrink(0).Clicked() {
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
		s.frame(c, "result", ui.Color{}, checking, func() {
			ui.Row(c).Gap(t.Space(2)).Padding(t.Space(3)).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c)
				ui.Text(c, s.text.Checking)
			})
		})
	case resultErr != nil:
		// The headline is short; the reason is at most two lines, and the tooltip has all of it.
		s.frame(c, "result", t.Danger, dir, func() {
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
		s.frame(c, "result", t.Warning, path, say(s.unresolved(r.Kind), t.Text, 600))
	}
}

// frame is a result card without an answer: the folder with its status and the clear button, a hairline, then the
// body. border is the outline of a card that needs attention, Warning or Danger, and zero for any other.
func (s *Screen) frame(c *ui.Context, key string, border ui.Color, path string, body func()) {
	t := c.Theme()
	checking := s.checking
	fillCard(c, key, border).Children(func() {
		ui.Row(c).Height(t.Space(9)).Gap(t.Space(1)).Padding(0, t.Space(1.5), 0, t.Space(3)).AlignItems(ui.Center).Children(func() {
			face := pathFace{mono: true, size: t.Rem(0.88), color: t.TextMuted}
			// The folder keeps at least 80 DIPs; the status gives way first.
			pathLine(c, s.env.Tilde(path), "", "", face).Grow(1).MinWidth(t.Space(20)).Shrink(0)
			s.status(c, false)
			if checking == "" && ghost(c, s.text.Clear, icons.Close).Clicked() {
				s.clear()
			}
		})
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

// copyPath copies the full path; for a moment afterwards the status says "Copied".
func (s *Screen) copyPath(c *ui.Context, path string) {
	s.act.CopyText(path)
	s.copiedUntil = s.now().Add(copiedFor)
	c.After(copiedFor)
	c.Announce(s.text.Copied)
}
