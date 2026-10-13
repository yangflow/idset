package panel

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/match"
)

// textButton is a borderless button in the accent color, with an optional icon before the label. Sizes come
// from the mockup: 5 x 4 DIPs of padding at the default spacing.
func (s *Screen) textButton(c *ui.Context, key, label string, icon *ui.SVG, after *ui.SVG) ui.Element {
	t := c.Theme()
	return ui.Button(c.Key(key), "").Label(label).Disabled(s.busy).Background(ui.Color{}).Border(0, ui.Color{}).TextColor(t.Accent).
		Padding(t.Space(1.25), t.Space(1)).Children(func() {
		if icon != nil {
			ui.Icon(c, icon).FontSize(t.Rem(1.1))
		}
		ui.Text(c, label).SingleLine()
		if after != nil {
			ui.Icon(c, after).FontSize(t.Rem(0.9))
		}
	})
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

// firstFolder is the first folder of a scoped identity, or "Global", for the end of a menu entry.
func (s *Screen) firstFolder(a account.Account) string {
	if a.Scope.Global {
		return s.text.Global
	}
	if p := usablePaths(a); len(p) > 0 {
		return p[0]
	}
	return s.text.NoFolders
}

// bar is a row of a result card under a hairline: a note, a failure or the undo bar.
func (s *Screen) bar(c *ui.Context, text string, color ui.Color) {
	t := c.Theme()
	ui.Divider(c)
	ui.Text(c, text).FontSize(t.Rem(0.9)).TextColor(color).Padding(t.Space(2), t.Space(3))
}

// matchedCard is the answer, the actions on the folder, and what the last action left: a confirmation, a note or
// the undo bar.
func (s *Screen) matchedCard(c *ui.Context, r *match.Result, a account.Account) {
	t := c.Theme()
	s.frame(c, "result", t.Border, r.Root, true, func() {
		ui.Row(c).Gap(t.Space(3)).Padding(t.Space(3.5), t.Space(3), t.Space(2.5), t.Space(3)).AlignItems(ui.Center).Children(func() {
			badge(c, a, t.Space(9), 9, t.Rem(1.25))
			ui.Column(c).Gap(t.Space(0.25)).Children(func() {
				ui.Text(c, s.label(a)).FontSize(t.Rem(1.15)).FontWeight(600)
				ui.Text(c, s.text.UsesHere).FontSize(t.Rem(0.9)).TextColor(t.TextMuted)
			})
		})
		// The left padding lines the details up with the identity name: badge plus gap.
		ui.Column(c).Gap(t.Space(1)).Padding(0, t.Space(3), t.Space(3), t.Space(15)).Children(func() {
			detail := func(label string, values []string, mono bool) {
				ui.Row(c).Gap(t.Space(2)).Children(func() {
					ui.Text(c, label).FontSize(t.Rem(0.9)).TextColor(t.TextMuted).Width(t.Space(14))
					ui.Column(c).Grow(1).Gap(t.Space(0.5)).Children(func() {
						for _, value := range values {
							v := ui.Text(c, value).FontSize(t.Rem(0.9))
							if mono {
								v.Font("monospace")
							}
						}
					})
				})
			}
			detail(s.text.Alias, []string{orDefault(a.SSHAlias, s.text.NoAlias)}, a.SSHAlias != "")
			detail(s.text.Email, []string{orDefault(a.GitUserEmail, s.text.NoEmail)}, false)
			// Each folder has its own line.
			detail(s.text.Folders, s.folderLines(a), !a.Scope.Global && len(usablePaths(a)) > 0)
		})
		ui.Divider(c)
		others := s.others(r)
		actions := ui.Row(c).Gap(t.Space(1)).PaddingX(t.Space(2)).Height(t.Space(10)).AlignItems(ui.Center).Children(func() {
			if s.textButton(c, "open", s.openLabel(), nil, nil).Clicked() {
				s.act.Open(r.Root)
			}
			ui.Spacer(c)
			if len(others) > 0 && s.textButton(c, "rebind", s.text.RebindTo, nil, chevIcon).Clicked() {
				s.menuOpen = !s.menuOpen
			}
			if r.Exact && s.textButton(c, "unbind", s.text.Unbind, nil, nil).Clicked() {
				s.unbind()
			}
		})
		s.rebindMenu(c, actions, others)
		s.confirmBlock(c)
		if s.failure != "" {
			s.bar(c, s.failure, t.Danger)
		}
		switch {
		case s.note != "":
			s.bar(c, s.note, t.TextMuted)
		case s.undoLeft() > 0:
			s.undoBar(c)
		}
	})
}

// rebindMenu lists the identities to rebind to under the card's actions, right-aligned to them.
func (s *Screen) rebindMenu(c *ui.Context, actions ui.Element, others []account.Account) {
	t := c.Theme()
	ui.PopoverBase(c, actions, &s.menuOpen, func(panel ui.Element) {
		panel.AttachTo(actions, ui.AnchorBottomRight, ui.AnchorTopRight).Margin(t.Space(1), 0, 0, 0).Right(t.Space(2)).
			Width(t.Space(56)).Padding(t.Space(1)).Radius(8).Background(t.Background).Border(1, t.Border).
			Role(ui.RoleMenu).Label(s.text.RebindTo)
		for _, a := range others {
			entry := ui.Row(c.Key("entry-" + a.ID)).Focusable().Role(ui.RoleMenuItem).Label(s.text.RebindTo + " " + s.label(a)).Gap(t.Space(2)).
				Height(t.Space(8)).PaddingX(t.Space(2)).Radius(t.Radius).AlignItems(ui.Center)
			if entry.Hovered() {
				entry.Background(t.SurfaceHover)
			}
			if entry.Clicked() {
				s.rebindTo(a.ID)
			}
			entry.Children(func() {
				dot(c, a, t.Space(2.5))
				ui.Text(c, s.label(a)).Grow(1).SingleLine()
				ui.Text(c, s.firstFolder(a)).Font("monospace").FontSize(t.Rem(0.85)).TextColor(t.TextMuted).SingleLine()
			})
		}
	})
}

// confirmBlock asks before a global identity becomes one that applies to a single folder.
func (s *Screen) confirmBlock(c *ui.Context) {
	t := c.Theme()
	a, ok := s.find(s.confirmTo)
	if s.confirmTo == "" || !ok {
		return
	}
	ui.Divider(c)
	ui.Row(c).Gap(t.Space(2.5)).MinHeight(t.Space(11)).PaddingX(t.Space(3)).AlignItems(ui.Center).Background(t.SurfacePressed).Children(func() {
		dot(c, a, t.Space(2.5))
		ui.Column(c).Grow(1).Gap(t.Space(0.25)).Children(func() {
			ui.Text(c, s.label(a)).FontWeight(600)
			ui.Text(c, s.text.GlobalIdentity).FontSize(t.Rem(0.9)).TextColor(t.TextMuted)
		})
	})
	ui.Column(c).Gap(t.Space(3)).Padding(t.Space(3)).Children(func() {
		ui.Text(c, fmt.Sprintf(s.text.ConfirmGlobal, s.label(a)))
		ui.Row(c).Gap(t.Space(2)).Justify(ui.End).Children(func() {
			if ui.Button(c, s.text.Cancel).Label(s.text.Cancel).Size(t.Space(20), t.Space(7)).Padding(0).Background(t.Background).Clicked() {
				s.confirmTo = ""
			}
			if ui.PrimaryButton(c, s.text.Rebind).Label(s.text.Rebind).Size(t.Space(20), t.Space(7)).Padding(0).Disabled(s.busy).Clicked() {
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
	ui.Row(c).Height(t.Space(9)).Padding(0, t.Space(2), 0, t.Space(3)).AlignItems(ui.Center).Children(func() {
		ui.Text(c, fmt.Sprintf(s.text.BoundTo, undo.who)+" ·").SingleLine()
		if s.textButton(c, "undo", s.text.Undo, nil, nil).Clicked() {
			s.Undo()
		}
		ui.Column(c).Absolute().Left(0).Bottom(0).Height(t.Space(0.5)).WidthPercent(float32(left) * 100 / undoSteps).
			Background(t.Accent.Alpha(0.45))
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
	s.frame(c, "result", t.Warning, path, false, func() {
		ui.Text(c, s.text.NoMatch).FontWeight(600).Padding(t.Space(3), t.Space(3), t.Space(2), t.Space(3))
		if len(s.accounts) > 0 {
			ui.Text(c, s.text.BindTo).FontSize(t.Rem(0.9)).FontWeight(600).TextColor(t.TextMuted).Padding(0, t.Space(3), t.Space(1.5), t.Space(3)).
				Role(ui.RoleHeading).Level(3)
		}
		for _, a := range s.accounts {
			ui.Divider(c)
			s.bindRow(c.Key("bind-"+a.ID), a)
		}
		ui.Divider(c)
		ui.Row(c).Height(t.Space(10)).PaddingX(t.Space(2)).AlignItems(ui.Center).Children(func() {
			if s.textButton(c, "new", s.text.NewBound, plusIcon, nil).Clicked() {
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
	row := ui.Row(c).Focusable().Role(ui.RoleButton).Label(fmt.Sprintf(s.text.BindLabel, s.label(a))).Gap(t.Space(2.5)).
		MinHeight(t.Space(11)).PaddingX(t.Space(3)).AlignItems(ui.Center)
	if row.Hovered() && !s.busy {
		row.Background(t.SurfaceHover)
	}
	if row.Clicked() && !s.busy {
		s.bindTo(a.ID)
	}
	row.Children(func() {
		dot(c, a, t.Space(2.5))
		ui.Column(c).Grow(1).Gap(t.Space(0.25)).Children(func() {
			// One line each, cut with an ellipsis, so that every row is as high as the others on any font.
			ui.Text(c, s.label(a)).FontWeight(600).SingleLine().Tooltip(s.label(a))
			face := pathFace{size: t.Rem(0.9), color: t.TextMuted}
			if p := usablePaths(a); !a.Scope.Global && len(p) > 0 {
				// The path gives way in the middle; "Has" and "adds another" stay.
				before, after, _ := strings.Cut(s.text.Has, "%s")
				pathLine(c, before, p[0], after, face)
			} else {
				note := orDefault(a.GitUserEmail, s.text.NoEmail)
				face.apply(ui.Text(c, note)).SingleLine().Tooltip(note)
			}
		})
		ui.Text(c, s.text.Bind).TextColor(t.Accent).Width(t.Space(11)).TextAlign(ui.End)
	})
}

// overlapCard lists the identities that cover the folder in the order git reads them, and offers to settle it.
func (s *Screen) overlapCard(c *ui.Context, r *match.Result) {
	t := c.Theme()
	s.frame(c, "overlap", t.Border, r.Root, false, func() {
		ui.Text(c, fmt.Sprintf(s.text.OverlapTitle, len(r.Covering))).FontWeight(600).Padding(t.Space(3), t.Space(3), t.Space(2), t.Space(3))
		ui.Text(c, s.text.OverlapHint).FontSize(t.Rem(0.9)).TextColor(t.TextMuted).Padding(0, t.Space(3), t.Space(2.5), t.Space(3))
		last := len(r.Covering) - 1
		for i, cv := range r.Covering {
			a, ok := s.find(cv.ID)
			if !ok {
				continue
			}
			win := i == last
			ui.Divider(c)
			ui.Row(c.Key("cover-" + a.ID)).Gap(t.Space(2.5)).MinHeight(t.Space(11)).PaddingX(t.Space(3)).AlignItems(ui.Center).Children(func() {
				dot(c, a, t.Space(2.5))
				ui.Column(c).Grow(1).Gap(t.Space(0.25)).Children(func() {
					name := ui.Text(c, s.label(a))
					if win {
						name.FontWeight(600)
					} else {
						name.TextColor(t.TextMuted)
					}
					ui.Text(c, cv.Path).Font("monospace").FontSize(t.Rem(0.9)).TextColor(t.TextMuted)
				})
				ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
					if win {
						ui.Icon(c, checkIcon).FontSize(t.Rem(1.1)).TextColor(t.Accent)
						ui.Text(c, s.text.Used).FontSize(t.Rem(0.9)).FontWeight(600).TextColor(t.Accent)
					} else {
						ui.Text(c, s.text.Overridden).FontSize(t.Rem(0.9)).TextColor(t.TextMuted)
					}
				})
			})
		}
		ui.Divider(c)
		ui.Column(c).Padding(t.Space(1), t.Space(2)).AlignItems(ui.Start).Children(func() {
			if winner, ok := s.find(r.ID); ok && s.textButton(c, "only", fmt.Sprintf(s.text.OnlyUse, s.label(winner)), nil, nil).
				Height(t.Space(8)).Clicked() {
				s.useOnly()
			}
			for _, cv := range r.Covering[:last] {
				loser, ok := s.find(cv.ID)
				if ok && s.textButton(c, "remove-"+cv.ID, fmt.Sprintf(s.text.RemoveFrom, s.label(loser), cv.Path), nil, nil).
					Height(t.Space(8)).Clicked() {
					s.removePath(cv.ID, cv.Path)
				}
			}
		})
	})
}

// staleShown reports whether the stale path prompt is above the card.
func (s *Screen) staleShown() bool {
	_, ok := s.stale()
	return ok
}

// staleCard offers to change an identity's gone path to the folder that replaced it. It is a card of its own above
// the no-match card, and every line is one line, so its height is the same for any path: the header, the old and
// new path, and the two buttons.
func (s *Screen) staleCard(c *ui.Context, st *match.Stale, a account.Account, root string) {
	t := c.Theme()
	ui.Column(c).Key("stale").Radius(10).Background(t.Surface).Border(1, t.Border).Clip().Children(func() {
		ui.Row(c).Height(t.Space(11)).PaddingX(t.Space(3)).Gap(t.Space(2.5)).AlignItems(ui.Center).Children(func() {
			dot(c, a, t.Space(2.5))
			ui.Text(c, s.label(a)).FontWeight(600).SingleLine().Shrink(0).MaxWidthPercent(50).Tooltip(s.label(a))
			ui.Text(c, s.text.StaleHead).SingleLine().MinWidth(0).Grow(1)
		})
		face := pathFace{mono: true, size: t.Rem(0.85), color: t.Text}
		for _, row := range []struct{ key, label, path string }{
			{"old", s.text.StaleOld, s.env.Tilde(s.env.Expand(st.Path))},
			{"new", s.text.StaleNew, s.env.Tilde(root)},
		} {
			ui.Divider(c)
			ui.Row(c.Key("stale-" + row.key)).Height(t.Space(9)).PaddingX(t.Space(3)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, row.label).FontSize(t.Rem(0.85)).TextColor(t.TextMuted).Width(t.Space(10)).Shrink(0)
				pathLine(c, "", strings.TrimSuffix(row.path, "/"), "", face).Grow(1)
			})
		}
		ui.Divider(c)
		ui.Row(c).Gap(t.Space(1)).PaddingX(t.Space(2)).Height(t.Space(10)).AlignItems(ui.Center).Children(func() {
			if s.textButton(c, "stale-fix", s.text.StaleFix, nil, nil).Clicked() {
				s.fixStale()
			}
			ui.Spacer(c)
			if s.textButton(c, "stale-keep", s.text.StaleKeep, nil, nil).Clicked() {
				s.keepStale()
			}
		})
		if s.failure != "" {
			s.bar(c, s.failure, t.Danger)
		}
	})
}
