package settings

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/ui/icons"
	"github.com/yangflow/idset/internal/ui/theme"
)

func (s *Screen) importPane(c *ui.Context) {
	t := c.Theme()
	if s.readOnly {
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2.5), t.Space(3), t.Space(2.5), t.Space(3)).Radius(theme.CardRadius).Background(t.Danger.Alpha(0.12)).Children(func() {
			ui.Icon(c, icons.Warn).FontSize(t.Space(3.5)).TextColor(t.Danger)
			ui.Text(c, s.text.ImportReadOnly).Grow(1)
		})
	}
	im := &s.imports
	switch {
	case im.scanning || !im.scanned:
		s.centered(c, func() {
			ui.Spinner(c)
			ui.Text(c, s.text.Scanning).FontSize(t.Rem(1.15)).FontWeight(600).Role(ui.RoleHeading).Level(2)
		}, s.text.ScanningSub, false)
	case im.scanErr != "":
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2.5), t.Space(3), t.Space(2.5), t.Space(3)).Radius(theme.CardRadius).Background(t.Danger.Alpha(0.12)).Children(func() {
			ui.Icon(c, icons.Warn).FontSize(t.Space(3.5)).TextColor(t.Danger)
			ui.Text(c, im.scanErr).Grow(1)
		})
		ui.Row(c).Justify(ui.End).Children(func() {
			if ui.Button(c.Key("rescan"), s.text.Rescan).Clicked() {
				s.scan()
			}
		})
	case im.added != nil:
		s.imported(c)
	case len(im.cands) == 0:
		s.centered(c, func() {
			ui.Text(c, s.text.ImportNone).FontSize(t.Rem(1.15)).FontWeight(600).Role(ui.RoleHeading).Level(2)
		}, s.text.ImportNoneBody, true)
	default:
		s.candidates(c)
	}
}

// centered is an empty state: a heading over a muted line, with Scan Again under them when rescan is set.
func (s *Screen) centered(c *ui.Context, head func(), body string, rescan bool) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(1)).AlignItems(ui.Center).Padding(t.Space(8), t.Space(4), t.Space(8), t.Space(4)).Children(func() {
		ui.Column(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(head)
		ui.Text(c, body).TextColor(t.TextMuted).TextAlign(ui.Center).MaxWidth(t.Space(105)).Wrap()
		if rescan {
			ui.Row(c).Padding(t.Space(4), 0, 0, 0).Children(func() {
				if ui.Button(c.Key("rescan"), s.text.Rescan).Clicked() {
					s.scan()
				}
			})
		}
	})
}

// candidates is the list to import from: a check box per candidate, then Scan Again and the import button.
func (s *Screen) candidates(c *ui.Context) {
	t := c.Theme()
	im := &s.imports
	n := s.checkedCount()
	list := ui.Column(c)
	if im.busy {
		list.Opacity(0.5)
	}
	list.Children(func() {
		s.section(c, s.foundTitle(len(im.cands)), func(r *rows) {
			for i, cand := range im.cands {
				r.next()
				a := cand.Account
				ui.Row(c.Key("cand-"+strconv.Itoa(i))).MinHeight(t.Space(11)).Padding(t.Space(1.5), t.Space(3), t.Space(1.5), t.Space(3)).Gap(t.Space(2.5)).
					AlignItems(ui.Center).Children(func() {
					ui.Checkbox(c, &im.checked[i], "").Label(a.Label).Disabled(im.busy)
					ui.Column(c).Grow(1).Children(func() {
						name := ui.Text(c, a.Label).SingleLine()
						if !im.checked[i] {
							name.TextColor(t.TextMuted)
						}
						alias := a.SSHAlias
						if alias == "" {
							alias = s.text.NoAlias
						}
						sub := []string{alias}
						if a.GitUserEmail != "" {
							sub = append(sub, a.GitUserEmail)
						}
						ui.Text(c, strings.Join(sub, " · ")).FontSize(t.Rem(0.85)).TextColor(t.TextMuted).Wrap()
					})
					switch {
					case cand.Exists:
						ui.Text(c, s.text.Exists).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).SingleLine().Shrink(0)
					case cand.Limited && im.checked[i]:
						ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Shrink(0).Children(func() {
							ui.Icon(c, icons.Folder).FontSize(t.Space(3)).TextColor(t.TextMuted)
							ui.Text(c, s.text.AsLimited).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).SingleLine()
						})
					}
				})
			}
		}, func() {
			if im.err != "" {
				s.warn(c, im.err)
				return
			}
			s.footnote(c, s.text.ReadNote)
			if s.limited() {
				s.footnote(c, s.text.LimitedNote)
			}
		})
	})
	ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
		if ghost(c, "rescan", s.text.Rescan, icons.Refresh).Disabled(im.busy).Clicked() {
			s.scan()
		}
		ui.Row(c).Grow(1)
		if im.busy {
			ui.Text(c, s.text.Importing).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).SingleLine()
			ui.Spinner(c)
		}
		if ui.PrimaryButton(c.Key("import"), s.importLabel(n)).Disabled(im.busy || s.readOnly || n == 0).Clicked() {
			s.importChecked()
		}
	})
}

// imported is the result: the identities added, then Scan Again and Open Identities.
func (s *Screen) imported(c *ui.Context) {
	t := c.Theme()
	im := &s.imports
	ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icons.Check).FontSize(t.Space(4)).TextColor(t.Accent)
		ui.Text(c, s.doneTitle(len(im.added))).FontSize(t.Rem(1.15)).FontWeight(600).Role(ui.RoleHeading).Level(2).SingleLine()
	})
	s.section(c, "", func(r *rows) {
		for _, a := range im.added {
			r.line(func() {
				dot(c, a.Color)
				ui.Text(c, s.label(a)).SingleLine().Shrink(0)
				ui.Text(c, a.GitUserEmail).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).Grow(1).Wrap()
			})
		}
	}, func() {
		if im.err != "" {
			s.warn(c, im.err)
			return
		}
		s.note(c, icons.Check, t.Accent, t.TextMuted, s.text.DoneNote)
	})
	ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
		if ghost(c, "rescan", s.text.Rescan, icons.Refresh).Clicked() {
			s.scan()
		}
		ui.Row(c).Grow(1)
		if ui.Button(c.Key("open-accounts"), s.text.OpenAccounts).Clicked() {
			s.openImported()
		}
	})
}
