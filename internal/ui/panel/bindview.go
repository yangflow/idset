package panel

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/match"
	"github.com/yangflow/idset/internal/remote"
	"github.com/yangflow/idset/internal/ui/icons"
	"github.com/yangflow/idset/internal/ui/theme"
)

// textButton is a borderless button in the given color. name is its accessible name and tooltip when it says more
// than the visible text. Sizes come from the design: 5 x 4 DIPs of padding at the default spacing.
func (s *Screen) textButton(c *ui.Context, key, text, name string, icon *ui.SVG, color ui.Color) ui.Element {
	t := c.Theme()
	if name == "" {
		name = text
	}
	b := ui.Button(c.Key(key), "").Label(name).Disabled(s.busy).Background(ui.Color{}).Border(0, ui.Color{}).TextColor(color).
		Padding(t.Space(theme.Space0_5), t.Space(theme.Space1)).Children(func() {
		if icon != nil {
			ui.Icon(c, icon).FontSize(t.Space(theme.Icon))
		}
		ui.Text(c, text).SingleLine()
	})
	if name != text {
		b.Tooltip(name)
	}
	return b
}

// openLabel names the platform's file manager on the button that opens the folder.
func (s *Screen) openLabel() string {
	switch s.goos {
	case "darwin":
		return s.text.OpenFinder
	case "windows":
		return s.text.OpenExplorer
	}
	return s.text.OpenFiles
}

// bar is a row of a result card under a hairline: a note, a failure or the undo bar.
func (s *Screen) bar(c *ui.Context, text string, color ui.Color) {
	t := c.Theme()
	ui.Divider(c)
	theme.Secondary.On(c, ui.Text(c, text)).TextColor(color).Padding(t.Space(theme.Space2), t.Space(theme.Space3))
}

// matchedCard is the answer and what the last action left: a confirmation, a failure or the undo bar. The answer
// is the identity's name over the dropped folder beside its badge, then its email and alias; the actions are at
// the end of the answer: the clone toggle, the menu (open, copy, another folder, rebind and unbind) and clear.
func (s *Screen) matchedCard(c *ui.Context, r *match.Result, a account.Account) {
	t := c.Theme()
	others := s.others(r)
	var border ui.Color
	switch match.Worst(r.Findings) {
	case match.Error:
		border = t.Danger
	case match.Warning:
		border = t.Warning
	}
	fillCard(c, "result", border).Children(func() {
		theme.Cluster(c, ui.Row(c)).Padding(t.Space(theme.Space3), t.Space(theme.Space1_5), 0, t.Space(theme.Space3)).Children(func() {
			badge(c, a, t.Space(theme.BadgeM), theme.RadiusControl, theme.Title)
			theme.StackTight(c, ui.Column(c)).Grow(1).MinWidth(0).Children(func() {
				theme.Title.On(c, ui.Text(c, s.label(a))).SingleLine().Tooltip(s.text.UsesHere)
				pathLine(c, s.env.Tilde(r.Root), "", "", pathFace{style: theme.Mono, color: t.TextMuted})
			})
			theme.InlineTight(c, ui.Row(c)).AlignSelf(ui.Start).Shrink(0).Children(func() {
				s.status(c, true)
				if a.SSHAlias != "" {
					toggle := symbolButton(c, "card-clone", s.text.CloneToggle, icons.Terminal, false).Expanded(s.cardCloneOpen)
					tipAboveEnd(c, toggle, s.text.CloneToggle)
					if toggle.Clicked() {
						s.toggleCardClone()
					}
				}
				ghost(c, s.text.More, icons.More).Disabled(s.busy).Menu(func(m *ui.Menu) { s.cardMenu(c, m, r, others) })
				if ghost(c, s.text.Clear, icons.Close).Clicked() {
					s.clear()
				}
			})
		})
		// The meta line starts under the name: the card's padding, the badge and the gap.
		ui.Row(c).MinWidth(0).Padding(t.Space(theme.Space1_5), t.Space(theme.Space3), t.Space(theme.Space3), t.Space(theme.Space3+theme.BadgeM+theme.Space2)).Children(func() {
			theme.Secondary.On(c, ui.Text(c, orDefault(a.GitUserEmail, s.text.NoEmail))).TextColor(t.TextMuted).SingleLine().MinWidth(0)
			if a.SSHAlias != "" {
				theme.Mono.On(c, ui.Text(c, " · "+a.SSHAlias)).TextColor(t.TextMuted).SingleLine().Shrink(0)
			}
		})
		s.findings(c, r, a)
		s.confirmBlock(c)
		if s.cardCloneOpen {
			s.cardClone(c, a)
		}
		if s.failure != "" {
			s.bar(c, s.failure, t.Danger)
		}
		if s.undoLeft() > 0 {
			s.undoBar(c)
		}
	})
}

// cardMenu is the menu of the "⋯" button, one of the system's: open the folder, copy its path, choose another
// folder, then the rebind part.
func (s *Screen) cardMenu(c *ui.Context, m *ui.Menu, r *match.Result, others []account.Account) {
	if m.Item(s.openLabel()).Chosen() {
		s.act.Open(r.Root)
	}
	if m.Item(s.text.CopyPath).Chosen() {
		s.copyPath(c, r.Root)
	}
	if m.Item(s.text.ChooseOther).Chosen() {
		s.act.ChooseFolder()
	}
	if len(others) > 0 || r.Exact {
		m.Separator()
		s.rebindMenu(m, r, others)
	}
}

// rebindMenu is the rebind part of the card's menu: the identities to rebind to under a heading, and "Unbind" when
// the identity lists this very folder.
func (s *Screen) rebindMenu(m *ui.Menu, r *match.Result, others []account.Account) {
	if len(others) > 0 {
		m.Item(s.text.RebindTo).Disabled(true)
	}
	for _, a := range others {
		if m.Item(s.label(a)).Chosen() {
			s.rebindTo(a.ID)
		}
	}
	if r.Exact {
		if len(others) > 0 {
			m.Separator()
		}
		if m.Item(s.text.Unbind).Chosen() {
			s.unbind()
		}
	}
}

// confirmBlock asks before a global identity becomes one that applies to a single folder: one line, and two
// buttons of the same size.
func (s *Screen) confirmBlock(c *ui.Context) {
	t := c.Theme()
	a, ok := s.find(s.confirmTo)
	if s.confirmTo == "" || !ok {
		return
	}
	ui.Divider(c)
	theme.InsetCard(c, theme.Stack(c, ui.Column(c))).Children(func() {
		ui.Text(c, fmt.Sprintf(s.text.ConfirmShort, s.label(a)))
		theme.Cluster(c, ui.Row(c)).Justify(ui.End).Children(func() {
			if ui.Button(c, s.text.Cancel).Label(s.text.Cancel).Size(t.Space(theme.ButtonW), t.Space(theme.Control)).Padding(0).Background(theme.Of(c).Field).Clicked() {
				s.confirmTo = ""
			}
			if ui.PrimaryButton(c, s.text.Rebind).Label(s.text.Rebind).Size(t.Space(theme.ButtonW), t.Space(theme.Control)).Padding(0).Disabled(s.busy).Clicked() {
				s.confirmRebind()
			}
		})
	})
}

// undoBar is the last row of a card after a bind: what was done, Undo, and a line on its bottom edge that shortens
// in fifths of the card's width as the time runs out.
func (s *Screen) undoBar(c *ui.Context) {
	t := c.Theme()
	// Undo clears the bar inside this frame, so the rest of it works on what it started with.
	undo, left := s.undo, s.undoLeft()
	ui.Divider(c)
	ui.Row(c).Height(t.Space(theme.Bar)).Padding(0, t.Space(theme.Space2), 0, t.Space(theme.Space3)).AlignItems(ui.Center).Children(func() {
		ui.Text(c, fmt.Sprintf(s.text.BoundTo, undo.who)+" ·").SingleLine()
		if s.textButton(c, "undo", s.text.Undo, "", nil, t.Accent).Clicked() {
			s.Undo()
		}
		ui.Column(c).Absolute().Left(0).Bottom(0).Height(t.Space(theme.Countdown)).WidthPercent(float32(left) * 100 / undoSteps).Background(t.Accent)
	})
	// The next frame is due when the line steps, and when the time is up.
	step := undoFor / undoSteps
	next := undo.until.Sub(s.now()) % step
	if next <= 0 {
		next = step
	}
	c.After(next + 10*time.Millisecond)
}

// noMatchCard offers every identity to bind the folder to, and a new one.
func (s *Screen) noMatchCard(c *ui.Context, r *match.Result) {
	t := c.Theme()
	path := r.Root
	if path == "" {
		path = r.Dir
	}
	s.frame(c, "result", t.Warning, path, func() {
		theme.BodyStrong.On(c, ui.Text(c, s.text.NoMatch)).Padding(t.Space(theme.Space3), t.Space(theme.Space3), t.Space(theme.Space2), t.Space(theme.Space3))
		if len(s.accounts) > 0 {
			theme.Caption.On(c, ui.Text(c, s.text.BindTo)).TextColor(t.TextMuted).Padding(0, t.Space(theme.Space3), t.Space(theme.Space1_5), t.Space(theme.Space3)).
				Role(ui.RoleHeading).Level(3)
		}
		for _, a := range s.accounts {
			ui.Divider(c)
			s.bindRow(c.Key("bind-"+a.ID), a)
		}
		ui.Divider(c)
		ui.Row(c).Height(t.Space(theme.RowL)).PaddingX(t.Space(theme.Space2)).AlignItems(ui.Center).Children(func() {
			if s.textButton(c, "new", s.text.NewShort, s.text.NewBound, icons.Plus, t.Accent).Clicked() {
				s.act.NewIdentityFor(r.Root)
			}
		})
		if s.failure != "" && !s.staleShown() {
			s.bar(c, s.failure, t.Danger)
		}
	})
}

// bindRow is an identity in the list of a card without a match; the whole row binds.
func (s *Screen) bindRow(c *ui.Context, a account.Account) {
	t := c.Theme()
	row := theme.ItemRow(c, ui.Row(c)).Focusable().Role(ui.RoleButton).Label(fmt.Sprintf(s.text.BindLabel, s.label(a))).
		MinHeight(t.Space(theme.RowL))
	if row.Hovered() && !s.busy {
		row.Background(t.SurfaceHover)
	}
	if row.Clicked() && !s.busy {
		s.bindTo(a.ID)
	}
	row.Children(func() {
		dot(c, a, t.Space(theme.Dot))
		theme.StackTight(c, ui.Column(c)).Grow(1).Children(func() {
			// One line each, cut with an ellipsis, so that every row is as high as the others on any font.
			theme.BodyMedium.On(c, ui.Text(c, s.label(a))).SingleLine().Tooltip(s.label(a))
			face := pathFace{style: theme.Secondary, color: t.TextMuted}
			if p := usablePaths(a); !a.Scope.Global && len(p) > 0 {
				// A path is monospace, as everywhere in the panel, at the size of the identity list's folders.
				face.style = theme.MonoSmall
				// Only the first folder; "has it, and adds another" is the tooltip.
				pathLine(c, p[0], "", fmt.Sprintf(s.text.Has, p[0]), face)
			} else {
				note := orDefault(a.GitUserEmail, s.text.NoEmail)
				face.apply(c, ui.Text(c, note)).SingleLine().Tooltip(note)
			}
		})
		ui.Text(c, s.text.Bind).TextColor(t.Accent)
	})
}

// overlapCard lists the identities that cover the folder in the order git reads them. The winner's row ends in a pin
// ("use only it here"), each overridden row in a minus ("remove the path from it"). It has no context line: the
// matched card under it shows the folder, and its clear button clears both.
func (s *Screen) overlapCard(c *ui.Context, r *match.Result) {
	t := c.Theme()
	fillCard(c, "overlap", ui.Color{}).Children(func() {
		theme.InlineTight(c, ui.Row(c)).Padding(t.Space(theme.Space3), t.Space(theme.Space2), t.Space(theme.Space2), t.Space(theme.Space3)).Children(func() {
			theme.BodyStrong.On(c, ui.Text(c, fmt.Sprintf(s.text.OverlapTitle, len(r.Covering)))).Grow(1).MinWidth(0)
			// The sentence about Git's order is the tooltip of the info icon.
			ghost(c, s.text.OverlapHint, icons.Info)
		})
		last := len(r.Covering) - 1
		for i, cv := range r.Covering {
			a, ok := s.find(cv.ID)
			if !ok {
				continue
			}
			win := i == last
			ui.Divider(c)
			theme.Cluster(c, ui.Row(c.Key("cover-"+a.ID))).MinHeight(t.Space(theme.RowL)).Padding(0, t.Space(theme.Space2), 0, t.Space(theme.Space3)).Children(func() {
				dot(c, a, t.Space(theme.Dot))
				theme.StackTight(c, ui.Column(c)).Grow(1).MinWidth(0).Children(func() {
					name := ui.Text(c, s.label(a)).SingleLine()
					if win {
						theme.BodyStrong.On(c, name)
					} else {
						name.TextColor(t.TextMuted)
					}
					theme.Mono.On(c, ui.Text(c, cv.Path)).TextColor(t.TextMuted).SingleLine().Tooltip(cv.Path)
				})
				theme.InlineTight(c, ui.Row(c)).Children(func() {
					if win {
						ui.Icon(c, icons.Check).FontSize(t.Space(theme.IconS)).TextColor(t.Accent)
						theme.Caption.On(c, ui.Text(c, s.text.Used)).TextColor(t.Accent)
					} else {
						theme.Secondary.On(c, ui.Text(c, s.text.Overridden)).TextColor(t.TextMuted)
					}
				})
				if win {
					if ghost(c, fmt.Sprintf(s.text.OnlyUse, s.label(a)), icons.Pin).Disabled(s.busy).Clicked() {
						s.useOnly()
					}
				} else if ghost(c, fmt.Sprintf(s.text.RemoveFrom, s.label(a), cv.Path), icons.Minus).Disabled(s.busy).Clicked() {
					s.removePath(cv.ID, cv.Path)
				}
			})
		}
	})
}

// staleShown reports whether the stale path prompt is above the card.
func (s *Screen) staleShown() bool {
	_, ok := s.stale()
	return ok
}

// staleCard offers to change an identity's gone path to the folder that replaced it. It is a card of its own above
// the no-match card, and every line is one line, so its height is the same for any path: the header, the old path,
// an arrow and the new path, and the two buttons.
func (s *Screen) staleCard(c *ui.Context, st *match.Stale, a account.Account, root string) {
	t := c.Theme()
	fillCard(c, "stale", ui.Color{}).Children(func() {
		theme.ItemRow(c, ui.Row(c)).Height(t.Space(theme.RowL)).Children(func() {
			dot(c, a, t.Space(theme.Dot))
			ui.Row(c).Grow(1).MinWidth(0).Children(func() {
				theme.BodyStrong.On(c, ui.Text(c, s.label(a))).SingleLine().Shrink(0).MaxWidthPercent(50).Tooltip(s.label(a))
				ui.Text(c, s.text.StaleHead).SingleLine().MinWidth(0).Grow(1)
			})
		})
		face := pathFace{style: theme.Mono, color: t.Text}
		oldFace := face
		oldFace.color = t.TextMuted
		old := strings.TrimSuffix(s.env.Tilde(s.env.Expand(st.Path)), "/")
		ui.Divider(c)
		ui.Row(c.Key("stale-old")).Height(t.Space(theme.RowS)).PaddingX(t.Space(theme.Space3)).AlignItems(ui.Center).Children(func() {
			pathLine(c, old, "", "", oldFace).Grow(1)
		})
		theme.Inline(c, ui.Row(c.Key("stale-new"))).Height(t.Space(theme.RowS)).PaddingX(t.Space(theme.Space3)).Children(func() {
			ui.Icon(c, icons.Arrow).FontSize(t.Space(theme.IconS)).TextColor(t.TextMuted)
			pathLine(c, strings.TrimSuffix(s.env.Tilde(root), "/"), "", "", face).Grow(1)
		})
		ui.Divider(c)
		theme.InlineTight(c, ui.Row(c)).PaddingX(t.Space(theme.Space2)).Height(t.Space(theme.Bar)).Justify(ui.End).Children(func() {
			if s.textButton(c, "stale-fix", s.text.StaleUpdate, s.text.StaleFix, nil, t.Accent).Clicked() {
				s.fixStale()
			}
			if s.textButton(c, "stale-keep", s.text.StaleIgnore, s.text.StaleKeep, nil, t.TextMuted).Clicked() {
				s.keepStale()
			}
		})
		if s.failure != "" {
			s.bar(c, s.failure, t.Danger)
		}
	})
}

// said is a finding as the card says it: a short sentence of a lead, a value that gives way when the line is
// short, and a tail that does not, and the full reason for the tooltip and the accessible name.
type said struct {
	severity          match.Severity
	lead, value, tail string
	path              bool
	why               string
}

func (d said) sentence() string { return d.lead + d.value + d.tail }

func (s *Screen) describe(f match.Finding, a account.Account) said {
	d := said{severity: f.Severity(), value: f.Value}
	switch f.Kind {
	case match.NoEmail:
		d.lead, d.why = s.text.FindNoEmail, s.text.FindNoEmailTip
	case match.OtherEmail:
		if owner, ok := s.find(f.Owner); ok {
			d.lead, d.tail = s.text.FindCommits, fmt.Sprintf(s.text.FindOwner, s.label(owner))
			d.why = fmt.Sprintf(s.text.FindOtherTip, f.Value, s.label(owner), s.label(a))
			break
		}
		// The owner is gone since the check; what is left is an email no identity uses.
		fallthrough
	case match.UnknownEmail:
		d.lead, d.why = s.text.FindCommits, fmt.Sprintf(s.text.FindUnknownTip, f.Value)
		d.severity = match.Warning
	case match.OtherKey:
		d.lead, d.path, d.value = s.text.FindPushes, true, s.env.Tilde(f.Value)
		d.why = fmt.Sprintf(s.text.FindKeyTip, d.value, s.label(a))
	}
	return d
}

// findings is the identity check under the details of the matched card: a hairline, then a row for each
// disagreement between the repository and the identity. There are no buttons; the reason is the tooltip.
func (s *Screen) findings(c *ui.Context, r *match.Result, a account.Account) {
	if _, https := remote.RemoteCommand(a, r.Origin); len(r.Findings) == 0 && !https {
		return
	}
	t := c.Theme()
	ui.Divider(c)
	ui.Column(c).PaddingY(t.Space(theme.Space1)).Children(func() {
		for i, f := range r.Findings {
			d := s.describe(f, a)
			icon, color := icons.Warn, t.Warning
			if d.severity == match.Error {
				icon, color = icons.Error, t.Danger
			}
			row := theme.Inline(c, ui.Row(c.Key(fmt.Sprintf("finding-%d", i)))).Height(t.Space(theme.RowS)).PaddingX(t.Space(theme.Space3)).
				MinWidth(0).Label(d.sentence() + s.text.Then + d.why).Description(d.why).Children(func() {
				ui.Icon(c, icon).FontSize(t.Space(theme.IconS)).TextColor(color).Shrink(0)
				s.findingLine(c, d)
			})
			tipAbove(c, row, d.why)
		}
		s.remoteRow(c, r, a)
	})
}

// findingLine is the sentence of a finding on one line. A long email gives way before its "@", so that the domain
// and the identity name stay; a key path gives way in its middle like any path. The reason is the tooltip.
func (s *Screen) findingLine(c *ui.Context, d said) {
	t := c.Theme()
	space := t.Rem(0.28)
	piece := func(text string) ui.Element {
		left, right := float32(0), float32(0)
		if strings.HasPrefix(text, " ") {
			left = space
		}
		if strings.HasSuffix(text, " ") {
			right = space
		}
		return theme.Secondary.On(c, ui.Text(c, strings.TrimSpace(text))).SingleLine().Shrink(0).Padding(0, right, 0, left)
	}
	ui.Row(c).Grow(1).MinWidth(0).AlignItems(ui.Center).Children(func() {
		piece(d.lead)
		switch {
		case d.path:
			pathLine(c, d.value, "", d.why, pathFace{style: theme.Mono, color: t.Text})
		case d.value != "":
			// The text gives way before the "@" and the ellipsis carries the domain and the name, so that what
			// the width leaves over after the cut stays at the end of the line, not before the "@".
			domain := ""
			if at := strings.LastIndex(d.value, "@"); at >= 0 {
				domain = d.value[at:]
			}
			theme.Secondary.On(c, ui.Text(c, d.value+d.tail)).SingleLine().Ellipsis("…" + domain + d.tail).MinWidth(0)
		}
		if d.tail != "" && (d.path || d.value == "") {
			piece(d.tail)
		}
	})
}

// tipAbove is the tooltip of anchor placed as the design has it: above it, 6 DIPs away, its start under the anchor's
// start, at most 240 DIPs wide, which Element.Tooltip does not offer (it goes by the pointer, below it, 320 wide).
func tipAbove(c *ui.Context, anchor ui.Element, text string) {
	tipPlaced(c, anchor, text, false)
}

// tipAboveEnd is tipAbove for a button at the end of a line: the tooltip ends where the button ends, so that it
// stays inside the panel.
func tipAboveEnd(c *ui.Context, anchor ui.Element, text string) {
	tipPlaced(c, anchor, text, true)
}

func tipPlaced(c *ui.Context, anchor ui.Element, text string, end bool) {
	t := c.Theme()
	ui.TooltipBase(c, anchor, func(tip ui.Element) {
		if end {
			tip.AttachTo(anchor, ui.AnchorTopRight, ui.AnchorBottomRight).Margin(t.Space(theme.Space1_5), 0, 0, 0)
		} else {
			tip.AttachTo(anchor, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(t.Space(theme.Space1_5), 0, 0, 0)
		}
		theme.Small.On(c, tip.MaxWidth(t.Space(theme.Measure))).Padding(t.Space(theme.Space1), t.Space(theme.Space2)).Radius(theme.RadiusRow).
			Background(t.Background).Border(0.5, t.Border).TextColor(theme.Of(c).Fg2)
		tip.Children(func() { ui.Text(c, text) })
	})
}
