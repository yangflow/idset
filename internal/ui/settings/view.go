package settings

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/ui/icons"
	"github.com/yangflow/idset/internal/ui/theme"
)

// Stored forms of the two files the Configuration pane names; they read the same on every platform.
const (
	sshConfig = "~/.ssh/config"
	gitConfig = "~/.gitconfig"
)

// View draws the window: its title in the hidden title bar, the tabs on the material, then the chosen pane. It reads
// state only; clicks go to the owner through Actions.
func (s *Screen) View(c *ui.Context) {
	theme.Apply(c)
	t := c.Theme()
	if s.shown != s.tab {
		s.shown = s.tab
		s.entered()
	}
	ui.Column(c).Fill().Children(func() {
		theme.TitleBar(c, s.text.Window)
		s.tabBar(c)
		ui.Divider(c)
		ui.Scroll(c).Grow(1).AlignItems(ui.Center).Padding(t.Space(4), t.Space(5), t.Space(5), t.Space(5)).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(t.Space(140)).Gap(t.Space(5)).Children(func() {
				switch Pane(s.tab + 1) {
				case General:
					s.general(c)
				case Keys:
					s.keysPane(c)
				case Import:
					s.importPane(c)
				case Backups:
					s.backupsPane(c)
				case Config:
					s.configPane(c)
				}
			})
		})
	})
	if s.announce != "" {
		c.Announce(s.announce)
		s.announce = ""
	}
}

// DialogOpen reports whether a dialog asks something: restoring a backup, removing the Include blocks or the
// identity of a new key. The owner shows it in a window of its own, which Dialog draws.
func (s *Screen) DialogOpen() bool { return s.confirmOpen || s.removeOpen || s.keys.pickOpen }

// Dialog draws the dialog that is open and returns its column, or a zero element when none is.
func (s *Screen) Dialog(c *ui.Context) ui.Element {
	switch {
	case s.confirmOpen:
		return s.restoreDialog(c)
	case s.removeOpen:
		return s.removeDialog(c)
	case s.keys.pickOpen:
		return s.pickDialog(c)
	}
	return ui.Element{}
}

// CloseDialog closes the dialog without doing what it asks, as Cancel does; its window was closed.
func (s *Screen) CloseDialog() { s.confirmOpen, s.removeOpen, s.keys.pickOpen = false, false, false }

// tabBar is the row of tabs over the panes: an icon over a label, the chosen one in the accent-soft fill.
func (s *Screen) tabBar(c *ui.Context) {
	t, f := c.Theme(), theme.Of(c)
	tabs := []struct {
		name string
		icon *ui.SVG
	}{{s.text.General, icons.Sliders}, {s.text.Keys, icons.Key}, {s.text.Imports, icons.Import}, {s.text.Backups, icons.Clock}, {s.text.Config, icons.Doc}}
	ui.Row(c).Justify(ui.Center).Padding(t.Space(1), t.Space(2), t.Space(1.5), t.Space(2)).Children(func() {
		parts := ui.TabsBase(c, &s.tab, len(tabs))
		parts.List.Gap(t.Space(1)).Label(s.text.Tabs).Children(func() {
			for i, tab := range tabs {
				el := parts.Tab(i).MinWidth(t.Space(22)).Height(t.Space(13)).PaddingX(t.Space(1)).Radius(8).AlignItems(ui.Center).Justify(ui.Center).Label(tab.name)
				color, weight := t.TextMuted, 400
				switch {
				case i == s.tab:
					el.Background(f.AccentSoft)
					color, weight = t.Accent, 500
				case el.Hovered():
					el.Background(t.SurfaceHover)
				}
				el.Children(func() {
					ui.Column(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, tab.icon).FontSize(t.Space(5)).TextColor(color)
						ui.Text(c, tab.name).FontSize(t.Rem(0.85)).FontWeight(weight).TextColor(color).SingleLine().Shrink(0)
					})
				})
			}
		})
	})
}

// rows lays out the rows of one group, with a hairline between them.
type rows struct {
	c *ui.Context
	n int
}

func (r *rows) next() {
	if r.n > 0 {
		ui.Column(r.c).Padding(0, 0, 0, r.c.Theme().Space(3)).Children(func() { ui.Divider(r.c) })
	}
	r.n++
}

// section is a titled group in the card fill; its notes follow it.
func (s *Screen) section(c *ui.Context, title string, body func(r *rows), after func()) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(1)).Children(func() {
		if title != "" {
			ui.Text(c, title).FontSize(t.Rem(0.85)).FontWeight(600).TextColor(t.TextMuted).Role(ui.RoleHeading).Level(2).PaddingX(t.Space(3))
		}
		ui.Column(c).Radius(10).Background(theme.Of(c).Card).Clip().Children(func() { body(&rows{c: c}) })
		if after != nil {
			after()
		}
	})
}

// note is a line of 12 DIP text under a group with a 12 DIP icon in front: a warning, a failure or a success.
func (s *Screen) note(c *ui.Context, icon *ui.SVG, iconColor, textColor ui.Color, text string) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(1)).PaddingX(t.Space(3)).Children(func() {
		ui.Icon(c, icon).FontSize(t.Space(3)).TextColor(iconColor)
		ui.Text(c, text).FontSize(t.Rem(0.92)).TextColor(textColor).Grow(1)
	})
}

func (s *Screen) warn(c *ui.Context, text string) {
	t := c.Theme()
	s.note(c, icons.Warn, t.Danger, t.Danger, text)
}

func (s *Screen) footnote(c *ui.Context, text string) {
	t := c.Theme()
	ui.Text(c, text).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).PaddingX(t.Space(3))
}

// line is a row of a group.
func (r *rows) line(content func()) { r.lineAt(r.c.Theme().Space(3), content) }

// lineAt is a row with its own left padding, for a row that holds a text button.
func (r *rows) lineAt(left float32, content func()) {
	r.next()
	t := r.c.Theme()
	ui.Row(r.c).MinHeight(t.Space(8)).Padding(t.Space(1), t.Space(3), t.Space(1), left).Gap(t.Space(2)).AlignItems(ui.Center).Children(content)
}

func (s *Screen) general(c *ui.Context) {
	t := c.Theme()
	choices := []struct{ value, name string }{{FollowSystem, s.text.FollowSystem}, {English, s.text.LangEnglish}, {Chinese, s.text.LangChinese}}
	chosen := s.language
	s.section(c, s.text.Language, func(r *rows) {
		for _, choice := range choices {
			r.next()
			on := s.language == choice.value
			// A radio, not Selected: Selected would paint the row in the accent color; the mockup marks the choice with a check.
			row := ui.RadioBase(c.Key("lang-"+choice.value), &chosen, choice.value).Label(choice.name).MinHeight(t.Space(8)).Padding(t.Space(1), t.Space(3), t.Space(1), t.Space(3)).
				Gap(t.Space(2)).AlignItems(ui.Center)
			if row.Hovered() {
				row.Background(t.SurfaceHover)
			}
			if row.Clicked() {
				s.chooseLanguage(choice.value)
			}
			row.Children(func() {
				ui.Text(c, choice.name).Grow(1)
				if on && choice.value == FollowSystem {
					ui.Text(c, fmt.Sprintf(s.text.Current, s.systemName())).FontSize(t.Rem(0.92)).TextColor(t.TextMuted)
				}
				ui.Row(c).Size(t.Space(3.5), t.Space(3.5)).Shrink(0).Children(func() {
					if on {
						ui.Icon(c, icons.Check).FontSize(t.Space(3.5)).TextColor(t.Accent)
					}
				})
			})
		}
	}, func() {
		if s.language == FollowSystem {
			s.footnote(c, fmt.Sprintf(s.text.SystemNote, s.systemName()))
			return
		}
		s.footnote(c, s.text.LangNote)
	})
	s.section(c, s.text.Startup, func(r *rows) {
		r.line(func() {
			ui.Text(c, s.text.Login).Grow(1)
			sw := ui.Switch(c.Key("login"), &s.login).Size(t.Space(7), t.Space(4)).Radius(t.Space(2)).Label(s.text.Login).Disabled(s.loginBusy)
			if sw.Changed() {
				s.toggleLogin()
			}
		})
	}, func() {
		switch s.loginNote {
		case loginApprove:
			s.note(c, icons.Warn, t.Warning, t.Warning, s.text.Approve)
		case loginFailed:
			s.warn(c, fmt.Sprintf(s.text.LoginFail, s.loginError))
		default:
			s.footnote(c, s.text.LoginNote)
		}
	})
}

// ghost is a borderless 24 DIP symbol button.
func ghost(c *ui.Context, key, label string, icon *ui.SVG) ui.Element {
	t := c.Theme()
	b := ui.Button(c.Key(key), "").Label(label).Size(t.Space(6), t.Space(6)).Padding(0).Radius(t.Radius).Border(0, ui.Color{}).
		Background(ui.Color{}).TextColor(t.TextMuted).Tooltip(label).Children(func() { ui.Icon(c, icon).FontSize(t.Rem(1.1)) })
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	return b
}

// dialogButton makes the two buttons of a dialog share its width, 28 DIPs high.
func dialogButton(b ui.Element, t *ui.Theme) ui.Element {
	return b.Grow(1).Height(t.Space(7)).Radius(8)
}

// dot is an identity's color, lighter on a dark theme for contrast.
func dot(c *ui.Context, color string) {
	size := c.Theme().Space(2.5)
	r, g, b := account.Account{Color: color}.RGB()
	if c.Theme().Dark {
		light := func(v uint8) uint8 { return v + uint8(float32(255-v)*0.4) }
		r, g, b = light(r), light(g), light(b)
	}
	ui.Column(c).Size(size, size).Radius(size / 2).Background(ui.RGB(r, g, b))
}

func (s *Screen) backupsPane(c *ui.Context) {
	t := c.Theme()
	if s.readOnly {
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2.5), t.Space(3), t.Space(2.5), t.Space(3)).Radius(10).Background(t.Danger.Alpha(0.12)).Children(func() {
			ui.Icon(c, icons.Warn).FontSize(t.Space(3.5)).TextColor(t.Danger)
			ui.Text(c, s.text.ReadOnlyBanner).Grow(1)
		})
	}
	switch {
	case !s.loaded:
		ui.Row(c).Gap(t.Space(2)).Justify(ui.Center).Padding(t.Space(6)).Children(func() {
			ui.Spinner(c)
			ui.Text(c, s.text.Loading).TextColor(t.TextMuted)
		})
	case s.listErr != "":
		s.warn(c, s.listErr)
	case len(s.backups) == 0:
		ui.Column(c).Gap(t.Space(2)).AlignItems(ui.Center).Padding(t.Space(8), t.Space(4), t.Space(8), t.Space(4)).Children(func() {
			ui.Text(c, s.text.EmptyTitle).FontSize(t.Rem(1.15)).FontWeight(600).Role(ui.RoleHeading).Level(2)
			ui.Text(c, s.text.EmptyBody).TextColor(t.TextMuted).TextAlign(ui.Center).Wrap()
		})
	default:
		s.section(c, "", func(r *rows) {
			for _, b := range s.backups {
				r.next()
				s.backupRow(c.Key(b.Path), b)
			}
		}, func() { s.footnote(c, s.text.BackupNote) })
	}
}

// backupRow is one backup: the row, then its identities when expanded and its failure when there is one.
func (s *Screen) backupRow(c *ui.Context, b Backup) {
	t := c.Theme()
	unreadable := b.Err != nil
	busy := s.restoring == b.Path
	flash := s.restored == b.Path && s.now().Before(s.until)
	if flash {
		c.After(s.until.Sub(s.now()))
	}
	open := s.expanded[b.Path] && !unreadable
	ui.Column(c).Children(func() {
		other := s.restoring != "" && !busy
		row := ui.Row(c).MinHeight(t.Space(11)).Padding(t.Space(1.5), t.Space(3), t.Space(1.5), t.Space(1)).Gap(t.Space(1)).AlignItems(ui.Center)
		if other {
			row.Opacity(0.5)
		}
		row.Children(func() {
			chevron, label := icons.ChevronRight, s.text.Expand
			if open {
				chevron, label = icons.ChevronDown, s.text.Collapse
			}
			if ghost(c, "expand", label, chevron).Disabled(unreadable || other).Clicked() {
				s.toggle(b.Path)
			}
			ui.Column(c).Grow(1).Children(func() {
				ui.Text(c, s.stamp(b.Time)).SingleLine()
				if unreadable {
					ui.Text(c, s.text.Unreadable+" · "+size(b.Size)).FontSize(t.Rem(0.85)).TextColor(t.Danger).SingleLine()
					return
				}
				sub := s.identityCount(len(b.Accounts))
				if len(b.Accounts) > 0 {
					sub += " · " + s.names(b.Accounts)
				}
				ui.Text(c, sub+" · "+size(b.Size)).FontSize(t.Rem(0.85)).TextColor(t.TextMuted).SingleLine()
			})
			switch {
			case busy:
				ui.Text(c, s.text.Restoring).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).SingleLine()
				ui.Spinner(c)
			case flash:
				ui.Text(c, s.text.Restored).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).SingleLine()
				ui.Icon(c, icons.Check).FontSize(t.Rem(1.1)).TextColor(t.Accent)
			default:
				if ghost(c, "restore", s.text.Restore, icons.Restore).Disabled(unreadable || s.restoring != "").Clicked() {
					s.askRestore(b.Path)
				}
			}
			if ghost(c, "delete", s.text.Delete, icons.Trash).Disabled(s.restoring != "" || s.deleting).Clicked() {
				s.deleteBackup(b.Path)
			}
		})
		if open {
			for i, a := range b.Accounts {
				if i > 0 {
					ui.Column(c).Padding(0, 0, 0, t.Space(11)).Children(func() { ui.Divider(c) })
				}
				s.identityRow(c, a)
			}
		}
		if msg := s.failed[b.Path]; msg != "" {
			ui.Column(c).Padding(0, t.Space(3), t.Space(2), t.Space(11)).Children(func() { s.warn(c, msg) })
		}
	})
}

// identityRow is an identity inside a backup: read-only, with a 24 DIP slot at the end under the delete column.
func (s *Screen) identityRow(c *ui.Context, a account.Account) {
	t := c.Theme()
	ui.Row(c.Key("id-"+a.ID)).MinHeight(t.Space(8)).Padding(0, t.Space(3), 0, t.Space(11)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
		dot(c, a.Color)
		ui.Text(c, s.label(a)).SingleLine().Shrink(0)
		ui.Text(c, a.GitUserEmail).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).Grow(1).Wrap()
		if a.Scope.Global {
			ui.Row(c).MinWidth(t.Space(6)).Justify(ui.End).Padding(0, t.Space(1.25), 0, 0).Shrink(0).Children(func() {
				ui.Text(c, s.text.Global).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).SingleLine()
			})
			return
		}
		ui.Row(c).Size(t.Space(6), t.Space(6)).Justify(ui.Center).AlignItems(ui.Center).Shrink(0).Children(func() {
			ui.Icon(c, icons.Folder).FontSize(t.Rem(1.1)).TextColor(t.TextMuted).Label(s.text.Scoped)
		})
	})
}

// restoreDialog asks before restoring and lists the identities there will be afterwards.
func (s *Screen) restoreDialog(c *ui.Context) ui.Element {
	b := s.backup(s.confirmPath)
	if b == nil {
		s.confirmOpen = false
		return ui.Element{}
	}
	t := c.Theme()
	title := fmt.Sprintf(s.text.ConfirmTitle, s.stamp(b.Time))
	return theme.Dialog(c, &s.confirmOpen, ui.RoleAlertDialog, title, func() {
		ui.Text(c, title).FontWeight(600).Role(ui.RoleHeading).Level(2).Wrap()
		ui.Text(c, s.text.ConfirmBody).Wrap()
		ui.Divider(c)
		ui.Column(c).Gap(t.Space(1.5)).Children(func() {
			for _, a := range b.Accounts {
				ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
					dot(c, a.Color)
					ui.Text(c, s.label(a)).SingleLine().Shrink(0)
					ui.Text(c, a.GitUserEmail).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).Grow(1).Wrap()
				})
			}
		})
		ui.Divider(c)
		ui.Text(c, s.text.ConfirmNote).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).Wrap()
		ui.Row(c).Gap(t.Space(2)).Margin(t.Space(1.5), 0, 0, 0).Children(func() {
			if dialogButton(ui.Button(c.Key("confirm-cancel"), s.text.Cancel), t).AutoFocus().Clicked() {
				s.confirmOpen = false
			}
			if dialogButton(ui.PrimaryButton(c.Key("confirm-restore"), s.text.Restore), t).Clicked() {
				s.confirmRestore()
			}
		})
	})
}

func (s *Screen) configPane(c *ui.Context) {
	t := c.Theme()
	state := func(f fileState) {
		switch {
		case !s.includesRead:
		case f.err != nil:
			ui.Text(c, cause(f.err)).FontSize(t.Rem(0.92)).TextColor(t.Danger).SingleLine()
		case f.present:
			ui.Text(c, s.text.Present).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).SingleLine()
		default:
			ui.Text(c, s.text.Absent).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).SingleLine()
		}
	}
	done := s.removal == removalDone
	s.section(c, s.text.Include, func(r *rows) {
		for _, f := range []struct {
			name  string
			state fileState
		}{{sshConfig, fileState{s.includes.SSH.Present, s.includes.SSH.Err}}, {gitConfig, fileState{s.includes.Git.Present, s.includes.Git.Err}}} {
			r.line(func() {
				ui.Text(c, f.name).Font("monospace").FontSize(t.Rem(0.88)).Grow(1).SingleLine()
				state(f.state)
			})
		}
		r.lineAt(t.Space(1), func() {
			b := ui.Button(c.Key("remove"), "").Label(s.text.Remove).Background(ui.Color{}).Border(0, ui.Color{}).TextColor(t.Accent).
				Disabled(s.removing || !s.includesRead || done)
			b.Children(func() {
				if done {
					ui.Icon(c, icons.Check).FontSize(t.Space(3))
					ui.Text(c, s.text.Removed).SingleLine()
					return
				}
				ui.Text(c, s.text.Remove).SingleLine()
			})
			if b.Clicked() {
				s.askRemove()
			}
			if s.removing {
				ui.Spacer(c)
				ui.Spinner(c)
			}
		})
	}, func() {
		switch {
		case s.removal == removalFailed:
			s.warn(c, s.removeErr)
		case done:
			s.note(c, icons.Check, t.Accent, t.TextMuted, s.text.RemovedNote)
		default:
			s.footnote(c, s.text.IncludeNote)
		}
	})
}

type fileState struct {
	present bool
	err     error
}

// removeDialog asks before the Include blocks are removed and says what goes and what stays.
func (s *Screen) removeDialog(c *ui.Context) ui.Element {
	t := c.Theme()
	return theme.Dialog(c, &s.removeOpen, ui.RoleAlertDialog, s.text.RemoveTitle, func() {
		ui.Text(c, s.text.RemoveTitle).FontWeight(600).Role(ui.RoleHeading).Level(2).Wrap()
		ui.Text(c, s.text.RemoveBody).Wrap()
		// item is a line of a box: a 12 DIP icon, then the text, whose path (mono) sits between the two plain parts.
		item := func(icon *ui.SVG, color ui.Color, pre, mono, post string) {
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, icon).FontSize(t.Space(3)).TextColor(color)
				ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Children(func() {
					for _, part := range []struct {
						text string
						mono bool
					}{{pre, false}, {mono, true}, {post, false}} {
						switch {
						case part.text == "":
						case part.mono:
							ui.Text(c, part.text).Font("monospace").FontSize(t.Rem(0.88)).SingleLine()
						default:
							ui.Text(c, part.text).SingleLine()
						}
					}
				})
			})
		}
		box := func(title string, items func()) {
			ui.Column(c).Gap(t.Space(1.5)).Padding(t.Space(2.5), t.Space(3), t.Space(2.5), t.Space(3)).Radius(8).Background(theme.Of(c).Card).Children(func() {
				ui.Text(c, title).FontSize(t.Rem(0.85)).FontWeight(600).TextColor(t.TextMuted)
				items()
			})
		}
		box(s.text.Removes, func() {
			for _, file := range []string{sshConfig, gitConfig} {
				item(icons.Minus, t.Danger, s.text.MarkerPre, file, s.text.MarkerPost)
			}
		})
		box(s.text.Keeps, func() {
			item(icons.Check, t.TextMuted, "", "accounts.json", s.text.Kept1Post)
			item(icons.Check, t.TextMuted, s.text.Kept2Pre, "~/.config/idset/", s.text.Kept2Post)
			item(icons.Check, t.TextMuted, s.text.Kept3, "", "")
		})
		ui.Row(c).Gap(t.Space(2)).Margin(t.Space(1.5), 0, 0, 0).Children(func() {
			if dialogButton(ui.Button(c.Key("remove-cancel"), s.text.Cancel), t).AutoFocus().Clicked() {
				s.removeOpen = false
			}
			if dialogButton(ui.PrimaryButton(c.Key("remove-confirm"), s.text.RemoveButton), t).Clicked() {
				s.confirmRemove()
			}
		})
	})
}
