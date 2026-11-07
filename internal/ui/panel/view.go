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

const copiedFor = theme.MotionAck

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
		theme.Cluster(c, ui.Row(c)).Height(t.Space(theme.Bar)).Padding(0, t.Space(theme.Space2), 0, t.Space(theme.Space4)).Children(func() {
			theme.Title.On(c, ui.Text(c, "Idset")).Role(ui.RoleHeading).Level(1)
			ui.Spacer(c)
			if ghost(c, s.text.Settings, icons.Gear).Clicked() {
				s.act.OpenSettings()
			}
		})
		// States without a list never need scrolling, and centering only works in a container of fixed height. The
		// content has 12 at the sides and cards 12 apart.
		if s.loading || s.readErr != nil || len(s.accounts) == 0 {
			theme.StackBlock(c, ui.Column(c)).Grow(1).Padding(t.Space(theme.Space1), t.Space(theme.Space3), t.Space(theme.Space3)).Children(func() { s.content(c) })
		} else {
			theme.StackBlock(c, ui.Scroll(c)).Grow(1).Padding(t.Space(theme.Space1), t.Space(theme.Space3), t.Space(theme.Space3)).Children(func() { s.content(c) })
		}
		theme.InlineTight(c, ui.Row(c)).Height(t.Space(theme.Bar)).PaddingX(t.Space(theme.Space2)).Children(func() {
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
	theme.Center(c, theme.Stack(c, ui.Column(c.Key("drop-over")))).Absolute().Top(t.Space(theme.Bar)).Left(t.Space(theme.Space2)).Right(t.Space(theme.Space2)).Bottom(t.Space(theme.Bar+theme.Space1_5)).
		Radius(theme.RadiusCard).Border(1.5, t.Accent.Alpha(0.7)).Background(theme.Of(c).AccentSoft).Children(func() {
		ui.Icon(c, icons.Folder).FontSize(t.Space(theme.IconL)).TextColor(t.Accent)
		theme.BodyStrong.On(c, ui.Text(c, s.text.DropRelease)).TextAlign(ui.Center)
	})
}

// showsResult reports whether a result card, or the check before it, is on the panel; the drop zone is then gone.
func (s *Screen) showsResult() bool {
	return !s.loading && s.readErr == nil && (s.result != nil || s.checking != "" || s.resultErr != nil)
}

// ghost is a borderless circular symbol button, 28 DIPs at the default spacing, with a 16 DIP glyph. Its label is
// its accessible name and its tooltip.
func ghost(c *ui.Context, label string, icon *ui.SVG) ui.Element {
	return tooltipGhost(c, "ghost-"+label, label, label, icon)
}

// tooltipGhost is a ghost button whose tooltip says more than its label.
func tooltipGhost(c *ui.Context, key, label, tip string, icon *ui.SVG) ui.Element {
	return symbolButton(c, key, label, icon, false).Tooltip(tip)
}

// symbolButton is the ghost button without a tooltip: the design system's icon button, a control-s square with
// radius-row. A done button, one that shows the check mark of a copy, keeps its plain look even while the pointer is
// still over it.
func symbolButton(c *ui.Context, key, label string, icon *ui.SVG, done bool) ui.Element {
	t := c.Theme()
	b := ui.Button(c.Key(key), "").Label(label).Size(t.Space(theme.ControlS), t.Space(theme.ControlS)).Padding(0).Radius(theme.RadiusRow).Border(0, ui.Color{}).
		Background(ui.Color{}).TextColor(t.TextMuted).Children(func() { ui.Icon(c, icon).FontSize(t.Space(theme.Icon)) })
	if !done {
		theme.IconButtonState(t, b)
	}
	// The symbol cross-fades to the check mark when the copy is done, and back.
	return theme.Swap(b)
}

func (s *Screen) content(c *ui.Context) {
	t := c.Theme()
	switch {
	case s.loading:
		theme.Center(c, theme.Stack(c, ui.Column(c))).Grow(1).Children(func() {
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
		theme.Center(c, theme.Stack(c, ui.Column(c))).Grow(1).Children(func() {
			ui.Column(c).Size(t.Space(theme.BadgeL), t.Space(theme.BadgeL)).Radius(theme.RadiusCard).Background(theme.Of(c).AccentSoft).AlignItems(ui.Center).
				Justify(ui.Center).Children(func() {
				ui.Icon(c, icons.Key).FontSize(t.Space(theme.IconL)).TextColor(t.Accent)
			})
			theme.Title.On(c, ui.Text(c, s.text.Empty)).Role(ui.RoleHeading).Level(2)
			theme.Secondary.On(c, ui.Text(c, s.text.EmptyHint)).TextColor(t.TextMuted).TextAlign(ui.Center)
			ui.Row(c).PaddingY(t.Space(theme.Space1)).Children(func() {
				if ui.PrimaryButton(c, s.text.AddIdentity).Height(t.Space(theme.Control)).PaddingX(t.Space(theme.Space4)).Clicked() {
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
	ui.Row(c).PaddingX(t.Space(theme.Space1_5)).Children(func() {
		theme.Caption.On(c, ui.Text(c, s.text.Identities)).TextColor(t.TextMuted).Grow(1).
			Role(ui.RoleHeading).Level(2)
		theme.Caption.On(c, ui.Textf(c, "%d", len(s.accounts))).TextColor(t.TextMuted)
	})
	if len(s.accounts) >= FilterThreshold {
		// A capsule search field with a magnifier inside its start, as macOS 26 draws one.
		ui.Row(c).Height(t.Space(theme.Control)).AlignItems(ui.Center).Children(func() {
			ui.TextInput(c.Key("filter"), &s.query).Label(s.text.Filter).Placeholder(s.text.Filter).Fill().Radius(theme.RadiusPill).
				Border(0, ui.Color{}).Background(t.Surface).Padding(0, t.Space(theme.Space2), 0, t.Space(theme.Space2+theme.Icon+theme.Space1))
			ui.Row(c).Absolute().Top(0).Bottom(0).Left(t.Space(theme.Space2)).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icons.Search).FontSize(t.Space(theme.IconS)).TextColor(t.TextMuted)
			})
		})
		if used := account.ProvidersInUse(s.accounts); len(used) > 1 {
			theme.Inline(c, ui.Row(c)).Children(func() {
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
				insetDivider(c, t.Space(theme.Space3+theme.BadgeS+theme.Space2))
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
	theme.Center(c, theme.Stack(c, ui.Column(c))).Grow(1).PaddingX(t.Space(theme.Space2)).Children(func() {
		ui.Column(c).Size(t.Space(theme.BadgeL), t.Space(theme.BadgeL)).Radius(theme.RadiusCard).Background(theme.Of(c).DangerSoft).AlignItems(ui.Center).
			Justify(ui.Center).Children(func() {
			ui.Icon(c, icons.Warn).FontSize(t.Space(theme.IconL)).TextColor(t.Danger)
		})
		theme.Title.On(c, ui.Text(c, s.text.ReadOnly)).Role(ui.RoleHeading).Level(2)
		theme.Mono.On(c, ui.Text(c, s.env.Tilde(s.dataPath))).TextColor(t.TextMuted).TextAlign(ui.Center).Selectable()
		// The parser error is at most two lines; the tooltip has all of it.
		theme.MonoSmall.On(c, ui.Text(c, reason.Error())).TextColor(t.TextMuted).TextAlign(ui.Center).MaxLines(2).
			Tooltip(reason.Error())
		theme.Secondary.On(c, ui.Text(c, s.text.ReadOnlyShort)).TextColor(t.TextMuted).Tooltip(s.text.ReadOnlyHint)
		theme.Stack(c, ui.Column(c)).PaddingY(t.Space(theme.Space1)).AlignItems(ui.Center).Children(func() {
			if ui.PrimaryButton(c, s.text.RestoreBackup).Label(s.text.RestoreBackup).Size(t.Space(2*theme.ButtonW+theme.Space2), t.Space(theme.Control)).Padding(0).Clicked() {
				s.act.OpenBackups()
			}
			theme.Cluster(c, ui.Row(c)).Children(func() {
				if ui.Button(c, s.revealLabel()).Label(s.revealLabel()).Size(t.Space(theme.ButtonW), t.Space(theme.Control)).Padding(0).Clicked() {
					s.act.Reveal(s.dataPath)
				}
				if ui.Button(c, s.text.Reload).Label(s.text.Reload).Size(t.Space(theme.ButtonW), t.Space(theme.Control)).Padding(0).Clicked() {
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
	b := theme.Secondary.On(c, ui.Button(c.Key("chip-"+string(p)), s.providerName(p))).Selected(on).Height(t.Space(theme.ControlS)).PaddingX(t.Space(theme.Space3)).Radius(theme.RadiusPill)
	if on {
		b.Background(theme.Of(c).Selected)
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
func badge(c *ui.Context, a account.Account, size, radius float32, style theme.Style) {
	r, g, b := a.RGB()
	theme.Center(c, ui.Column(c)).Size(size, size).Radius(radius).Background(ui.RGB(r, g, b)).Children(func() {
		style.On(c, ui.Text(c, initial(a))).TextColor(theme.Ink(r, g, b))
	})
}

// fillCard is a result card or the identity list: a fill with the card rim, not an outline. One that needs attention adds a 1 DIP inner
// outline in Warning or Danger.
func fillCard(c *ui.Context, key string, attention ui.Color) ui.Element {
	e := theme.Card(c, ui.Column(c.Key(key))).Clip()
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
	face := pathFace{style: theme.MonoSmall, color: t.TextMuted}
	folders := usablePaths(a)
	switch {
	case a.Scope.Global:
		face.apply(c, ui.Text(c, s.text.Global)).SingleLine()
	case len(folders) == 0:
		face.apply(c, ui.Text(c, s.text.NoFolders)).SingleLine()
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
		bottom := t.Space(theme.Space2)
		if hasProblem {
			bottom = t.Space(theme.Space0_5)
		}
		row := theme.Cluster(c, ui.Row(c)).Focusable().MinHeight(t.Space(theme.RowList)).Padding(t.Space(theme.Space2), t.Space(theme.Space1_5), bottom, t.Space(theme.Space3))
		if row.Hovered() {
			row.Background(t.SurfaceHover)
		}
		if row.Clicked() {
			s.act.OpenAccounts(a.ID)
		}
		row.Children(func() {
			badge(c, a, t.Space(theme.BadgeS), theme.RadiusControl, theme.BodyStrong)
			theme.StackTight(c, ui.Column(c)).Grow(1).MinWidth(0).Children(func() {
				theme.BodyMedium.On(c, ui.Text(c, s.label(a))).SingleLine()
				theme.Secondary.On(c, ui.Text(c, orDefault(a.GitUserEmail, s.text.NoEmail))).TextColor(t.TextMuted).SingleLine()
			})
			s.folderTag(c, a)
			if s.diag.known {
				s.connDot(c, a)
			}
			if a.SSHAlias == "" {
				// The toggle's space stays, so that the folders line up.
				ui.Column(c).Size(t.Space(theme.ControlS), t.Space(theme.ControlS)).Shrink(0)
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
	zone := ui.Column(c.Key("drop")).Height(t.Space(theme.RowL)).Radius(theme.RadiusCard).Border(1, t.Accent.Alpha(0.5)).BorderStyle(ui.BorderDashed).
		Background(t.Accent.Alpha(0.06))
	if zone.FileDragOver() {
		zone.Border(1.5, t.Accent).BorderStyle(ui.BorderSolid).Background(theme.Of(c).AccentSoft)
	}
	// It takes a folder in color, and fades out as the result card comes in.
	theme.Recolor(zone)
	if files := zone.DroppedFiles(); len(files) > 0 {
		s.Drop(files)
	}
	theme.Cluster(c, zone.Row()).Padding(0, t.Space(theme.Space2), 0, t.Space(theme.Space3)).Children(func() {
		ui.Icon(c, icons.Folder).FontSize(t.Space(theme.IconM)).TextColor(t.Accent).Shrink(0)
		theme.Secondary.On(c, ui.Text(c, s.text.DropLine)).SingleLine().MinWidth(0).Grow(1).Tooltip(s.text.DropHint)
		if theme.BodyMedium.On(c, ui.PrimaryButton(c, s.text.ChooseShort)).Label(s.text.Choose).Tooltip(s.text.Choose).Height(t.Space(theme.ControlS)).
			PaddingX(t.Space(theme.Space3)).Shrink(0).Clicked() {
			s.act.ChooseFolder()
		}
	})
}

// resultCard is a context line with the folder and its actions, then the answer.
func (s *Screen) resultCard(c *ui.Context) {
	t := c.Theme()
	// Clearing happens inside the card, so the rest of this frame works on the state it started with.
	r, checking, resultErr, dir := s.result, s.checking, s.resultErr, s.lastDir
	say := func(text string, color ui.Color) func() {
		return func() { theme.BodyStrong.On(c, ui.Text(c, text)).TextColor(color).Padding(t.Space(theme.Space3)) }
	}
	switch {
	case checking != "":
		s.frame(c, "result", ui.Color{}, checking, func() {
			theme.InsetCard(c, theme.Cluster(c, ui.Row(c))).Children(func() {
				ui.Spinner(c)
				ui.Text(c, s.text.Checking)
			})
		})
	case resultErr != nil:
		// The headline is short; the reason is at most two lines, and the tooltip has all of it.
		s.frame(c, "result", t.Danger, dir, func() {
			theme.InsetCard(c, theme.StackSnug(c, ui.Column(c))).Children(func() {
				ui.Text(c, s.text.GitFailed).TextColor(t.Danger)
				theme.MonoSmall.On(c, ui.Text(c, resultErr.Error())).TextColor(t.TextMuted).MaxLines(2).
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
		s.frame(c, "result", t.Warning, path, say(s.unresolved(r.Kind), t.Text))
	}
}

// frame is a result card without an answer: the folder with its status and the clear button, a hairline, then the
// body. border is the outline of a card that needs attention, Warning or Danger, and zero for any other.
func (s *Screen) frame(c *ui.Context, key string, border ui.Color, path string, body func()) {
	t := c.Theme()
	checking := s.checking
	// The card settles from above as the answer arrives, where the drop zone was, and fades as it is cleared.
	theme.Arrive(c, fillCard(c, key, border), theme.Slide(-1), theme.Fade()).Children(func() {
		theme.InlineTight(c, ui.Row(c)).Height(t.Space(theme.Bar)).Padding(0, t.Space(theme.Space1_5), 0, t.Space(theme.Space3)).Children(func() {
			face := pathFace{style: theme.Mono, color: t.TextMuted}
			// The folder keeps at least a button's width; the status gives way first.
			pathLine(c, s.env.Tilde(path), "", "", face).Grow(1).MinWidth(t.Space(theme.ButtonW)).Shrink(0)
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
	for i, ack := range []struct {
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
			// The word says what happened and fades in and out; the path beside it stays put.
			text := theme.FadeIn(theme.Small.On(c, ui.Text(c.Key(fmt.Sprintf("ack-%d", i)), ack.text)).TextColor(t.TextMuted).SingleLine().MinWidth(0))
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
