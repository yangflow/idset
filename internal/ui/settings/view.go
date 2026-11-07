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
		ui.Scroll(c).Grow(1).AlignItems(ui.Center).Padding(t.Space(theme.Space4), t.Space(theme.Space6), t.Space(theme.Space6), t.Space(theme.Space6)).Children(func() {
			theme.StackPage(c, ui.Column(c)).FillWidth().MaxWidth(t.Space(theme.PaneMax)).Children(func() {
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
	ui.Row(c).Justify(ui.Center).Padding(t.Space(theme.Space1), t.Space(theme.Space2), t.Space(theme.Space1_5), t.Space(theme.Space2)).Children(func() {
		parts := ui.TabsBase(c, &s.tab, len(tabs))
		theme.InlineTight(c, parts.List).Label(s.text.Tabs).Children(func() {
			for i, tab := range tabs {
				el := parts.Tab(i).MinWidth(t.Space(theme.TabW)).Height(t.Space(theme.TabH)).PaddingX(t.Space(theme.Space1)).Radius(theme.RadiusControl).AlignItems(ui.Center).Justify(ui.Center).Label(tab.name)
				color := t.TextMuted
				switch {
				case i == s.tab:
					el.Background(f.Selected)
					color = t.Text
				case el.Hovered():
					el.Background(t.SurfaceHover)
				}
				el.Children(func() {
					theme.StackSnug(c, ui.Column(c)).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, tab.icon).FontSize(t.Space(theme.IconM)).TextColor(color)
						theme.Caption.On(c, ui.Text(c, tab.name)).TextColor(color).SingleLine().Shrink(0)
					})
				})
			}
		})
	})
}

// rows lays out the rows of one group, with a hairline between them.
type rows struct {
	n int
}

func (r *rows) next(c *ui.Context) {
	if r.n > 0 {
		ui.Column(c).Padding(0, 0, 0, c.Theme().Space(theme.Space3)).Children(func() { ui.Divider(c) })
	}
	r.n++
}

// section is a titled group in the card fill; its notes follow it.
func (s *Screen) section(c *ui.Context, title string, body func(r *rows), after func()) {
	t := c.Theme()
	theme.StackSnug(c, ui.Column(c)).Children(func() {
		if title != "" {
			theme.Caption.On(c, ui.Text(c, title)).TextColor(t.TextMuted).Role(ui.RoleHeading).Level(2).PaddingX(t.Space(theme.Space3))
		}
		theme.Card(c, ui.Column(c)).Clip().Children(func() { body(&rows{}) })
		if after != nil {
			after()
		}
	})
}

// note is a line of 12 DIP text under a group with a 12 DIP icon in front: a warning, a failure or a success.
func (s *Screen) note(c *ui.Context, icon *ui.SVG, iconColor, textColor ui.Color, text string) {
	t := c.Theme()
	theme.InlineTight(c, ui.Row(c)).PaddingX(t.Space(theme.Space3)).Children(func() {
		ui.Icon(c, icon).FontSize(t.Space(theme.IconXs)).TextColor(iconColor)
		theme.Secondary.On(c, ui.Text(c, text)).TextColor(textColor).Grow(1)
	})
}

func (s *Screen) warn(c *ui.Context, text string) {
	t := c.Theme()
	s.note(c, icons.Warn, t.Danger, t.Danger, text)
}

func (s *Screen) footnote(c *ui.Context, text string) {
	t := c.Theme()
	theme.Secondary.On(c, ui.Text(c, text)).TextColor(t.TextMuted).PaddingX(t.Space(theme.Space3))
}

// line is a row of a group.
func (r *rows) line(c *ui.Context, content func()) {
	r.lineAt(c, c.Theme().Space(theme.Space3), content)
}

// lineAt is a row with its own left padding, for a row that holds a text button.
func (r *rows) lineAt(c *ui.Context, left float32, content func()) {
	r.next(c)
	t := c.Theme()
	theme.Cluster(c, ui.Row(c)).MinHeight(t.Space(theme.Row)).Padding(t.Space(theme.Space1), t.Space(theme.Space3), t.Space(theme.Space1), left).Children(content)
}

func (s *Screen) general(c *ui.Context) {
	t := c.Theme()
	choices := []struct{ value, name string }{{FollowSystem, s.text.FollowSystem}, {English, s.text.LangEnglish}, {Chinese, s.text.LangChinese}}
	chosen := s.language
	s.section(c, s.text.Language, func(r *rows) {
		for _, choice := range choices {
			r.next(c)
			on := s.language == choice.value
			// A radio, not Selected: Selected would paint the row in the accent color; the design marks the choice with a check.
			row := theme.ItemRow(c, ui.RadioBase(c.Key("lang-"+choice.value), &chosen, choice.value)).Label(choice.name).MinHeight(t.Space(theme.Row)).Padding(t.Space(theme.Space1), t.Space(theme.Space3), t.Space(theme.Space1), t.Space(theme.Space3))
			if row.Hovered() {
				row.Background(t.SurfaceHover)
			}
			if row.Clicked() {
				s.chooseLanguage(choice.value)
			}
			row.Children(func() {
				ui.Text(c, choice.name).Grow(1)
				if on && choice.value == FollowSystem {
					theme.Secondary.On(c, ui.Text(c, fmt.Sprintf(s.text.Current, s.systemName()))).TextColor(t.TextMuted)
				}
				ui.Row(c).Size(t.Space(theme.IconS), t.Space(theme.IconS)).Shrink(0).Children(func() {
					if on {
						ui.Icon(c, icons.Check).FontSize(t.Space(theme.IconS)).TextColor(t.Accent)
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
		r.line(c, func() {
			ui.Text(c, s.text.Login).Grow(1)
			sw := ui.Switch(c.Key("login"), &s.login).Size(t.Space(theme.SwitchW), t.Space(theme.SwitchH)).Radius(theme.RadiusPill).Label(s.text.Login).Disabled(s.loginBusy)
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
	b := ui.Button(c.Key(key), "").Label(label).Size(t.Space(theme.ControlS), t.Space(theme.ControlS)).Padding(0).Radius(theme.RadiusRow).Border(0, ui.Color{}).
		Background(ui.Color{}).TextColor(t.TextMuted).Tooltip(label).Children(func() { ui.Icon(c, icon).FontSize(t.Space(theme.Icon)) })
	theme.IconButtonState(t, b)
	return b
}

// dialogButton makes the two buttons of a dialog share its width, 28 DIPs high.
func dialogButton(b ui.Element, t *ui.Theme) ui.Element {
	return b.Grow(1).Height(t.Space(theme.Control))
}

// dot is an identity's color, lighter on a dark theme for contrast.
func dot(c *ui.Context, color string) {
	size := c.Theme().Space(theme.Dot)
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
		theme.InsetCard(c, theme.Cluster(c, ui.Row(c))).AlignItems(ui.Start).Radius(theme.RadiusCard).Background(theme.Of(c).DangerSoft).Children(func() {
			ui.Icon(c, icons.Warn).FontSize(t.Space(theme.IconS)).TextColor(t.Danger)
			ui.Text(c, s.text.ReadOnlyBanner).Grow(1)
		})
	}
	switch {
	case !s.loaded:
		theme.InsetPane(c, theme.Cluster(c, ui.Row(c))).Justify(ui.Center).Children(func() {
			ui.Spinner(c)
			ui.Text(c, s.text.Loading).TextColor(t.TextMuted)
		})
	case s.listErr != "":
		s.warn(c, s.listErr)
	case len(s.backups) == 0:
		theme.Stack(c, ui.Column(c)).AlignItems(ui.Center).Padding(t.Space(theme.Space8), t.Space(theme.Space4), t.Space(theme.Space8), t.Space(theme.Space4)).Children(func() {
			theme.Title.On(c, ui.Text(c, s.text.EmptyTitle)).Role(ui.RoleHeading).Level(2)
			ui.Text(c, s.text.EmptyBody).TextColor(t.TextMuted).TextAlign(ui.Center).Wrap()
		})
	default:
		s.section(c, "", func(r *rows) {
			for _, b := range s.backups {
				r.next(c)
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
		row := theme.InlineTight(c, ui.Row(c)).MinHeight(t.Space(theme.RowL)).Padding(t.Space(theme.Space1_5), t.Space(theme.Space3), t.Space(theme.Space1_5), t.Space(theme.Space1))
		if other {
			row.Opacity(theme.OpacityFaded)
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
					theme.Small.On(c, ui.Text(c, s.text.Unreadable+" · "+size(b.Size))).TextColor(t.Danger).SingleLine()
					return
				}
				sub := s.identityCount(len(b.Accounts))
				if len(b.Accounts) > 0 {
					sub += " · " + s.names(b.Accounts)
				}
				theme.Small.On(c, ui.Text(c, sub+" · "+size(b.Size))).TextColor(t.TextMuted).SingleLine()
			})
			switch {
			case busy:
				theme.Secondary.On(c, ui.Text(c, s.text.Restoring)).TextColor(t.TextMuted).SingleLine()
				ui.Spinner(c)
			case flash:
				theme.FadeIn(theme.InlineTight(c, ui.Row(c.Key("restored")))).Children(func() {
					theme.Secondary.On(c, ui.Text(c, s.text.Restored)).TextColor(t.TextMuted).SingleLine()
					ui.Icon(c, icons.Check).FontSize(t.Space(theme.Icon)).TextColor(t.Accent)
				})
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
					ui.Column(c).Padding(0, 0, 0, t.Space(theme.Space1+theme.ControlS+theme.Space1)).Children(func() { ui.Divider(c) })
				}
				s.identityRow(c, a)
			}
		}
		if msg := s.failed[b.Path]; msg != "" {
			ui.Column(c).Padding(0, t.Space(theme.Space3), t.Space(theme.Space2), t.Space(theme.Space1+theme.ControlS+theme.Space1)).Children(func() { s.warn(c, msg) })
		}
	})
}

// identityRow is an identity inside a backup: read-only, with a 24 DIP slot at the end under the delete column.
func (s *Screen) identityRow(c *ui.Context, a account.Account) {
	t := c.Theme()
	theme.Cluster(c, ui.Row(c.Key("id-"+a.ID))).MinHeight(t.Space(theme.Row)).Padding(0, t.Space(theme.Space3), 0, t.Space(theme.Space1+theme.ControlS+theme.Space1)).Children(func() {
		dot(c, a.Color)
		ui.Text(c, s.label(a)).SingleLine().Shrink(0)
		theme.Secondary.On(c, ui.Text(c, a.GitUserEmail)).TextColor(t.TextMuted).Grow(1).Wrap()
		if a.Scope.Global {
			ui.Row(c).MinWidth(t.Space(theme.ControlS)).Justify(ui.End).Padding(0, t.Space(theme.Space1_5), 0, 0).Shrink(0).Children(func() {
				theme.Secondary.On(c, ui.Text(c, s.text.Global)).TextColor(t.TextMuted).SingleLine()
			})
			return
		}
		theme.Center(c, ui.Column(c)).Size(t.Space(theme.ControlS), t.Space(theme.ControlS)).Shrink(0).Children(func() {
			ui.Icon(c, icons.Folder).FontSize(t.Space(theme.IconS)).TextColor(t.TextMuted).Label(s.text.Scoped)
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
		theme.BodyStrong.On(c, ui.Text(c, title)).Role(ui.RoleHeading).Level(2).Wrap()
		ui.Text(c, s.text.ConfirmBody).Wrap()
		ui.Divider(c)
		theme.StackSnug(c, ui.Column(c)).Children(func() {
			for _, a := range b.Accounts {
				theme.Cluster(c, ui.Row(c)).Children(func() {
					dot(c, a.Color)
					ui.Text(c, s.label(a)).SingleLine().Shrink(0)
					theme.Secondary.On(c, ui.Text(c, a.GitUserEmail)).TextColor(t.TextMuted).Grow(1).Wrap()
				})
			}
		})
		ui.Divider(c)
		theme.Secondary.On(c, ui.Text(c, s.text.ConfirmNote)).TextColor(t.TextMuted).Wrap()
		theme.Cluster(c, ui.Row(c)).Margin(t.Space(theme.Space1_5), 0, 0, 0).Children(func() {
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
			theme.Secondary.On(c, ui.Text(c, cause(f.err))).TextColor(t.Danger).SingleLine()
		case f.present:
			theme.Secondary.On(c, ui.Text(c, s.text.Present)).TextColor(t.TextMuted).SingleLine()
		default:
			theme.Secondary.On(c, ui.Text(c, s.text.Absent)).TextColor(t.TextMuted).SingleLine()
		}
	}
	done := s.removal == removalDone
	s.section(c, s.text.Include, func(r *rows) {
		for _, f := range []struct {
			name  string
			state fileState
		}{{sshConfig, fileState{s.includes.SSH.Present, s.includes.SSH.Err}}, {gitConfig, fileState{s.includes.Git.Present, s.includes.Git.Err}}} {
			r.line(c, func() {
				theme.Mono.On(c, ui.Text(c, f.name)).Grow(1).SingleLine()
				state(f.state)
			})
		}
		r.lineAt(c, t.Space(theme.Space1), func() {
			b := ui.Button(c.Key("remove"), "").Label(s.text.Remove).Background(ui.Color{}).Border(0, ui.Color{}).TextColor(t.Accent).
				Disabled(s.removing || !s.includesRead || done)
			b.Children(func() {
				if done {
					ui.Icon(c, icons.Check).FontSize(t.Space(theme.IconXs))
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
		theme.BodyStrong.On(c, ui.Text(c, s.text.RemoveTitle)).Role(ui.RoleHeading).Level(2).Wrap()
		ui.Text(c, s.text.RemoveBody).Wrap()
		// item is a line of a box: a 12 DIP icon, then the text, whose path (mono) sits between the two plain parts.
		item := func(icon *ui.SVG, color ui.Color, pre, mono, post string) {
			theme.Cluster(c, ui.Row(c)).Children(func() {
				ui.Icon(c, icon).FontSize(t.Space(theme.IconXs)).TextColor(color)
				theme.InlineTight(c, ui.Row(c)).Children(func() {
					for _, part := range []struct {
						text string
						mono bool
					}{{pre, false}, {mono, true}, {post, false}} {
						switch {
						case part.text == "":
						case part.mono:
							theme.Mono.On(c, ui.Text(c, part.text)).SingleLine()
						default:
							ui.Text(c, part.text).SingleLine()
						}
					}
				})
			})
		}
		box := func(title string, items func()) {
			theme.InsetCard(c, theme.StackSnug(c, theme.Card(c, ui.Column(c)))).Children(func() {
				theme.Caption.On(c, ui.Text(c, title)).TextColor(t.TextMuted)
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
		theme.Cluster(c, ui.Row(c)).Margin(t.Space(theme.Space1_5), 0, 0, 0).Children(func() {
			if dialogButton(ui.Button(c.Key("remove-cancel"), s.text.Cancel), t).AutoFocus().Clicked() {
				s.removeOpen = false
			}
			if dialogButton(ui.PrimaryButton(c.Key("remove-confirm"), s.text.RemoveButton), t).Clicked() {
				s.confirmRemove()
			}
		})
	})
}
