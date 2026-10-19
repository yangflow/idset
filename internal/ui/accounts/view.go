package accounts

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/keyfile"
	"github.com/yangflow/idset/internal/ui/icons"
	"github.com/yangflow/idset/internal/ui/theme"
)

// View draws the window. It reads state only; clicks go to the owner through Actions.
func (s *Screen) View(c *ui.Context) {
	theme.Apply(c)
	t := c.Theme()
	if c.Shortcut(ui.Cmd, ui.KeyS) {
		s.save()
	}
	band := bandHeight(c)
	ui.Column(c).Fill().Children(func() {
		switch {
		case s.loading:
			ui.Column(c).Grow(1).Padding(band, 0, 0, 0).Gap(t.Space(2)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				ui.Spinner(c)
				ui.Text(c, s.text.Loading).TextColor(t.TextMuted)
			})
		case s.readErr != nil:
			s.readOnly(c)
		case s.empty():
			s.emptyState(c)
		default:
			ui.Split(c, &s.splitW, func() { s.sidebar(c) }, func() { s.detail(c) }).Grow(1)
		}
		// The title band lies over the top of both panes, where the window's controls are.
		ui.Column(c).Absolute().Top(0).Left(0).Right(0).Children(func() { theme.TitleBar(c, s.text.Window) })
	})
	if s.announce != "" {
		c.Announce(s.announce)
		s.announce = ""
	}
}

// bandHeight is the height of the title band, which the content below it makes room for.
func bandHeight(c *ui.Context) float32 {
	return max(c.TitleBar().Height, c.Theme().Space(7))
}

// hairline is a 0.5 DIP line across its column.
func hairline(c *ui.Context) ui.Element {
	return ui.Column(c).Height(0.5).Shrink(0).Background(c.Theme().Border)
}

func (s *Screen) readOnly(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Grow(1).Padding(bandHeight(c), t.Space(6), 0, t.Space(6)).Gap(t.Space(2)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ui.Icon(c, icons.Warn).FontSize(t.Rem(1.7)).TextColor(t.Danger)
		ui.Text(c, s.text.ReadOnly).FontSize(t.Rem(1.15)).FontWeight(600).Role(ui.RoleHeading).Level(2)
		ui.Text(c, s.text.ReadOnlyHint).TextColor(t.TextMuted).TextAlign(ui.Center)
		ui.Text(c, s.env.Tilde(s.dataPath)).Font("monospace").FontSize(t.Rem(0.88)).TextColor(t.TextMuted).Selectable()
	})
}

// emptyState has no list and no footer; "Add Identity" stays in the toolbar.
func (s *Screen) emptyState(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Grow(1).Padding(bandHeight(c), 0, 0, 0).Gap(t.Space(2)).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ui.Text(c, s.text.NoneTitle).FontSize(t.Rem(1.15)).FontWeight(600).Role(ui.RoleHeading).Level(2)
		ui.Text(c, s.text.NoneHint).TextColor(t.TextMuted)
		ui.Row(c).Gap(t.Space(2)).Margin(t.Space(2), 0, 0, 0).Children(func() {
			if ui.Button(c, s.text.ImportExisting).Clicked() {
				s.act.OpenImport()
			}
			if ui.PrimaryButton(c, s.text.Create).Clicked() {
				s.newDraft("")
			}
		})
	})
}

// badge is the identity's initial on its color, solid, the letter colored by theme.Ink. ring draws the selection
// ring around it.
func badge(c *ui.Context, color, label string, size, radius, fontSize float32, ring bool) {
	t := c.Theme()
	r, g, b := account.Account{Color: color}.RGB()
	letter := "?"
	if l := []rune(strings.TrimSpace(label)); len(l) > 0 {
		letter = strings.ToUpper(string(l[0]))
	}
	e := ui.Column(c).Size(size, size).Shrink(0).Radius(radius).Background(ui.RGB(r, g, b)).AlignItems(ui.Center).Justify(ui.Center)
	if ring {
		e.Border(1.5, t.AccentText)
	}
	e.Children(func() {
		ui.Text(c, letter).FontSize(fontSize).FontWeight(600).TextColor(theme.Ink(r, g, b))
	})
}

// sidebar lists the draft, then the identities in the panel's order. It has no fill: the window's material shows.
func (s *Screen) sidebar(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(bandHeight(c), 0, 0, 0).Children(func() {
		// The list scrolls above the add bar, which stays at the bottom edge of the sidebar.
		ui.Scroll(c).Grow(1).Padding(t.Space(4), t.Space(2), t.Space(3), t.Space(2)).Gap(t.Space(0.5)).Children(func() {
			ui.Row(c).Padding(0, t.Space(1.5), t.Space(1), t.Space(1.5)).Children(func() {
				ui.Text(c, s.text.Identities).FontSize(t.Rem(0.85)).FontWeight(600).TextColor(t.TextMuted).Grow(1).Role(ui.RoleHeading).Level(2)
				n := len(s.list)
				if s.draftID != "" {
					n++
				}
				ui.Textf(c, "%d", n).FontSize(t.Rem(0.85)).FontWeight(600).TextColor(t.TextMuted)
			})
			if f := s.forms[s.draftID]; f != nil {
				s.item(c, f.id, f.cur.label, f.cur.email, f.cur.color, false, true)
			}
			for _, a := range account.Sorted(s.list) {
				label, email, color := a.Label, a.GitUserEmail, a.Color
				if f := s.forms[a.ID]; f != nil {
					// The row follows the edits, as the title of the detail does.
					label, email, color = f.cur.label, f.cur.email, f.cur.color
				}
				s.item(c, a.ID, label, email, color, !a.Scope.Global, false)
			}
		})
		hairline(c)
		ui.Row(c).Height(barHeight(t)).Shrink(0).PaddingX(t.Space(2)).AlignItems(ui.Center).Children(func() {
			if ghost(c, s.text.AddIdentity, icons.Plus).Disabled(s.readErr != nil).Clicked() {
				s.newDraft("")
			}
		})
	})
}

func (s *Screen) item(c *ui.Context, id, label, email, color string, scoped, draft bool) {
	t := c.Theme()
	row := ui.Row(c).Key("item-"+id).Focusable().Gap(t.Space(2.5)).Padding(t.Space(1.5), t.Space(2)).Radius(t.Radius).AlignItems(ui.Center)
	selected := id == s.sel
	secondary := t.TextMuted
	switch {
	case selected:
		row.Selected(true).Background(t.Accent).TextColor(t.AccentText)
		secondary = t.AccentText.Alpha(0.8)
	case row.Hovered():
		row.Background(t.SurfaceHover)
	}
	if row.Clicked() {
		s.selectID(id)
	}
	row.Children(func() {
		badge(c, color, label, t.Space(7), 8, t.Rem(1), selected)
		ui.Column(c).Grow(1).Gap(t.Space(0.25)).Children(func() {
			ui.Text(c, s.labelOf(label)).FontWeight(600)
			if strings.TrimSpace(email) != "" {
				ui.Text(c, email).FontSize(t.Rem(0.92)).TextColor(secondary)
			}
		})
		switch {
		case draft:
			ui.Text(c, s.text.NewTag).FontSize(t.Rem(0.92)).TextColor(secondary)
		case scoped:
			ui.Icon(c, icons.Folder).FontSize(t.Rem(1.1)).TextColor(secondary).Label(s.text.Scoped)
		}
	})
}

// detail is the editor of the selected identity and its footer. It is filled where the window's material shows
// through the list, and solid otherwise.
func (s *Screen) detail(c *ui.Context) {
	f := s.selectedForm()
	if f == nil {
		return
	}
	if f.platform == "" {
		f.platform = s.providerName(f.cur.provider)
	}
	// The widgets write their choice after the frame that shows them; take it over before drawing the next one.
	s.syncWidgets(f)
	if s.problems != nil && (s.problemsID != f.id || !reflect.DeepEqual(s.problemsState, f.cur)) {
		s.problems = nil
	}
	c = c.Key(f.id)
	t := c.Theme()
	fill := theme.Of(c).SolidDetail
	if c.Vibrancy() {
		fill = theme.Of(c).Detail
	}
	ui.Column(c).Fill().Padding(bandHeight(c), 0, 0, 0).Background(fill).Children(func() {
		body := ui.Scroll(c).Grow(1).Padding(t.Space(4), t.Space(5), t.Space(5), t.Space(5)).Gap(t.Space(5))
		if s.busy {
			body.Opacity(0.55)
		}
		body.Children(func() {
			s.header(c, f)
			s.identityGroup(c, f)
			s.sshGroup(c, f)
			s.scopeGroup(c, f)
			s.rewriteGroup(c, f)
			s.notesGroup(c, f)
			if !f.isNew {
				s.timesGroup(c, f)
			}
		})
		hairline(c)
		s.footer(c, f)
	})
}

// syncWidgets applies the choices the widgets wrote to their own state: the platform, the scope and the hex color.
func (s *Screen) syncWidgets(f *form) {
	for _, p := range account.Providers {
		if s.providerName(p) == f.platform {
			f.setProvider(p)
		}
	}
	f.setScope(f.scope)
	f.syncHex()
}

func (s *Screen) header(c *ui.Context, f *form) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(1.5)).Children(func() {
		ui.Row(c).Gap(t.Space(2.5)).AlignItems(ui.Center).Children(func() {
			badge(c, f.cur.color, f.cur.label, t.Space(8), 10, t.Rem(1.15), false)
			ui.Text(c, s.labelOf(f.cur.label)).FontSize(t.Rem(1.15)).FontWeight(600).Role(ui.RoleHeading).Level(1)
			mark := s.text.Scoped
			if f.cur.global {
				mark = s.text.Global
			}
			ui.Text(c, mark).FontSize(t.Rem(0.92)).TextColor(t.TextMuted)
			ui.Spacer(c)
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				for _, name := range account.ColorPresets {
					s.colorChoice(c, f, name)
				}
				ui.TextInput(c.Key("hex"), &f.hex).Label(s.text.Hex).Font("monospace").FontSize(t.Rem(0.85)).Width(t.Space(22)).Disabled(s.busy)
			})
		})
		s.notes(c, s.problemTexts(f, "color"))
	})
}

func (s *Screen) colorName(name string) string {
	return map[string]string{"blue": s.text.Blue, "green": s.text.Green, "orange": s.text.Orange, "red": s.text.Red,
		"purple": s.text.Purple, "yellow": s.text.Yellow}[name]
}

// colorChoice is a 14 DIP dot of the preset; the chosen one has its check and a ring of its own color around it.
func (s *Screen) colorChoice(c *ui.Context, f *form, name string) {
	t := c.Theme()
	size := t.Space(3.5)
	on := f.cur.color == name
	r, g, b := account.Account{Color: name}.RGB()
	color := ui.RGB(r, g, b)
	ring := ui.Color{}
	if on {
		ring = color
	}
	ui.Column(c).Size(size+7, size+7).Shrink(0).Radius((size+7)/2).AlignItems(ui.Center).Justify(ui.Center).Border(1.5, ring).Children(func() {
		btn := ui.Button(c.Key("color-"+name), "").Label(s.colorName(name)).Selected(on).Size(size, size).Padding(0).Radius(size/2).
			Background(color).Border(0, ui.Color{}).Disabled(s.busy)
		btn.Children(func() {
			if on {
				ui.Icon(c, icons.Check).FontSize(t.Rem(0.7)).TextColor(ui.RGB(255, 255, 255))
			}
		})
		if btn.Clicked() {
			f.chooseColor(name)
		}
	})
}

// rows lays out the rows of one group, with a hairline between them.
type rows struct {
	c *ui.Context
	n int
}

func (r *rows) next() {
	if r.n > 0 {
		hairline(r.c).Margin(0, 0, 0, r.c.Theme().Space(3))
	}
	r.n++
}

// field is a row with its label on the left and the value, drawn by value, filling the rest. hint is a small
// muted note at the end of the row, after the value; the value gives way to it and never overlaps it.
func (r *rows) field(label, hint string, value func()) {
	r.next()
	t := r.c.Theme()
	ui.Row(r.c).MinHeight(t.Space(8)).PaddingX(t.Space(3)).PaddingY(t.Space(1)).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
		ui.Text(r.c, label).Role(ui.RoleText).Width(t.Space(42)).Shrink(0)
		value()
		if hint != "" {
			ui.Text(r.c, hint).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).SingleLine().Shrink(0)
		}
	})
}

// line is a row without a label.
func (r *rows) line(content func()) {
	r.next()
	t := r.c.Theme()
	ui.Row(r.c).MinHeight(t.Space(8)).PaddingX(t.Space(3)).PaddingY(t.Space(1)).Gap(t.Space(1)).AlignItems(ui.Center).Children(content)
}

// section is a titled group; its error messages, in Danger, and its help text follow it.
func (s *Screen) section(c *ui.Context, title, help string, errs []string, body func(r *rows)) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(1)).Children(func() {
		ui.Text(c, title).FontSize(t.Rem(0.85)).FontWeight(600).TextColor(t.TextMuted).Role(ui.RoleHeading).Level(2).PaddingX(t.Space(3))
		ui.Column(c).Radius(10).Background(theme.Of(c).Card).Clip().Children(func() { body(&rows{c: c}) })
		s.notes(c, errs)
		if help != "" {
			s.footnote(c, help)
		}
	})
}

func (s *Screen) notes(c *ui.Context, errs []string) {
	t := c.Theme()
	for _, e := range errs {
		ui.Row(c).Gap(t.Space(1)).Margin(t.Space(1), t.Space(3), 0, t.Space(3)).Children(func() {
			ui.Icon(c, icons.Warn).FontSize(t.Rem(0.92)).TextColor(t.Danger)
			ui.Text(c, e).FontSize(t.Rem(0.92)).TextColor(t.Danger).Grow(1)
		})
	}
}

func (s *Screen) footnote(c *ui.Context, text string) {
	t := c.Theme()
	ui.Text(c, text).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).Margin(t.Space(1), t.Space(3), 0, t.Space(3))
}

// input is a text input drawn as plain text in a row.
func (s *Screen) input(c *ui.Context, key string, value *string, label string) ui.Element {
	return ui.TextInput(c.Key(key), value).Label(label).Grow(1).TextAlign(ui.End).Background(ui.Color{}).Border(0, ui.Color{}).Disabled(s.busy)
}

func (s *Screen) identityGroup(c *ui.Context, f *form) {
	names := []string{}
	for _, p := range account.Providers {
		names = append(names, s.providerName(p))
	}
	s.section(c, s.text.GroupIdentity, "", s.problemTexts(f, "label", "username", "gitUserName", "gitUserEmail", "provider"), func(r *rows) {
		r.field(s.text.Name, "", func() { s.input(c, "label", &f.cur.label, s.text.Name).Placeholder(s.text.Unnamed) })
		r.field(s.text.Platform, "", func() {
			ui.Spacer(c)
			theme.Select(c.Key("platform"), &f.platform, names).Label(s.text.Platform).Disabled(s.busy)
		})
		r.field(s.text.Username, "", func() { s.input(c, "username", &f.cur.username, s.text.Username) })
		r.field(s.text.GitName, s.fromGit(f.cur.gitName, f.prefName), func() { s.input(c, "gitName", &f.cur.gitName, s.text.GitName) })
		r.field(s.text.Email, s.fromGit(f.cur.email, f.prefEmail), func() { s.input(c, "email", &f.cur.email, s.text.Email) })
	})
}

// fromGit is the hint at the end of a field that still holds what git config gave.
func (s *Screen) fromGit(value, prefilled string) string {
	if prefilled != "" && value == prefilled {
		return s.text.FromGit
	}
	return ""
}

func (s *Screen) sshGroup(c *ui.Context, f *form) {
	t := c.Theme()
	s.section(c, s.text.GroupSSH, "", s.problemTexts(f, "sshAlias", "keyPath", "hostName", "sshPort"), func(r *rows) {
		if f.custom() {
			r.field(s.text.Host, "", func() { s.input(c, "host", &f.cur.hostName, s.text.Host).Font("monospace").FontSize(t.Rem(0.88)) })
		}
		r.field(s.text.Alias, "", func() { s.input(c, "alias", &f.cur.alias, s.text.Alias).Font("monospace").FontSize(t.Rem(0.88)) })
		r.field(s.text.Key, "", func() {
			s.keyValue(c, f)
			if ghost(c, s.text.ChooseFile, icons.Folder).Disabled(s.busy).Clicked() {
				s.act.ChooseKey(f.id)
			}
		})
		r.field(s.text.Port, "", func() {
			if f.custom() {
				s.input(c, "port", &f.cur.portText, s.text.Port).Font("monospace").FontSize(t.Rem(0.88))
				return
			}
			ui.Spacer(c)
			ui.RadioGroup(c, func() {
				ui.Radio(c, &f.cur.port, 22, "22").Disabled(s.busy)
				ui.Radio(c, &f.cur.port, 443, "443").Disabled(s.busy || !f.cur.provider.Supports443())
			}).Row().Gap(t.Space(4)).Label(s.text.Port)
		})
	})
}

// keyShown is how many characters of a private key path fit the value field; a longer path is shown shortened
// in the middle, so that the file name stays visible, until the field is clicked.
const keyShown = 34

// middle shortens text to n runes by replacing the middle with an ellipsis.
func middle(text string, n int) string {
	runes := []rune(text)
	if len(runes) <= n || n < 3 {
		return text
	}
	head := (n - 1) / 3
	tail := n - 1 - head
	return string(runes[:head]) + "…" + string(runes[len(runes)-tail:])
}

// keyValue is the private key path: a text input, or for a long path that is not being edited the path
// shortened in the middle. Either way the tooltip and the focused input show all of it.
func (s *Screen) keyValue(c *ui.Context, f *form) {
	t := c.Theme()
	long := len([]rune(f.cur.keyPath)) > keyShown
	if long && !f.keyEditing {
		short := ui.Text(c.Key("key-short"), middle(f.cur.keyPath, keyShown)).Font("monospace").FontSize(t.Rem(0.88)).Grow(1).
			Tooltip(f.cur.keyPath).Focusable()
		if short.Clicked() && !s.busy {
			f.keyEditing, f.keyFrames = true, 0
		}
		return
	}
	in := s.input(c, "key", &f.cur.keyPath, s.text.Key).Font("monospace").FontSize(t.Rem(0.88)).Tooltip(f.cur.keyPath)
	if f.keyEditing {
		in.AutoFocus()
	}
	switch {
	case in.Focused():
		f.keyEditing, f.keyFrames = true, 1
	case f.keyEditing && f.keyFrames > 0:
		// It had the focus and lost it.
		f.keyEditing, f.keyFrames = false, 0
	case f.keyEditing:
		f.keyFrames++
	}
}

func (s *Screen) scopeGroup(c *ui.Context, f *form) {
	t := c.Theme()
	help := ""
	if f.cur.global {
		help = s.text.ScopeHint
	}
	s.section(c, s.text.GroupScope, help, s.problemTexts(f, "scope"), func(r *rows) {
		r.field(s.text.Applies, "", func() {
			ui.Spacer(c)
			ui.Segmented(c, &f.scope, s.text.ScopeGlobal, s.text.ScopeGitdir).Label(s.text.Applies).Disabled(s.busy)
		})
		if f.cur.global {
			return
		}
		remove := -1
		for i := range f.cur.paths {
			r.line(func() {
				s.input(c, fmt.Sprint("path-", i), &f.cur.paths[i], s.text.Path).TextAlign(ui.Start).Font("monospace").FontSize(t.Rem(0.88))
				if ghost(c, s.text.ChooseFolder, icons.Folder).Disabled(s.busy).Clicked() {
					s.act.ChooseFolder(f.id, i)
				}
				if ghost(c, s.text.RemovePath, icons.Minus).Disabled(s.busy || len(f.cur.paths) == 1).Clicked() {
					remove = i
				}
			})
		}
		if remove >= 0 {
			f.cur.paths = slices.Delete(f.cur.paths, remove, remove+1)
		}
		r.line(func() {
			if textButton(c, s.text.AddPath, t.Accent, icons.Plus).Disabled(s.busy).Clicked() {
				f.cur.paths = append(f.cur.paths, "")
			}
		})
	})
}

func (s *Screen) rewriteGroup(c *ui.Context, f *form) {
	t := c.Theme()
	help := ""
	if !f.custom() && strings.TrimSpace(f.cur.alias) == "" {
		help = s.text.NeedAlias
	}
	s.section(c, s.text.GroupRewrites, help, s.problemTexts(f, "urlRewrites"), func(r *rows) {
		remove := -1
		for i := range f.cur.rewrites {
			r.line(func() {
				ui.Column(c).Grow(1).Children(func() {
					s.input(c, fmt.Sprint("from-", i), &f.cur.rewrites[i].From, s.text.From).Grow(0).FillWidth().TextAlign(ui.Start).Font("monospace").FontSize(t.Rem(0.88))
					ui.Row(c).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "→").TextColor(t.TextMuted).PaddingX(t.Space(1))
						s.input(c, fmt.Sprint("to-", i), &f.cur.rewrites[i].To, s.text.To).TextAlign(ui.Start).TextColor(t.TextMuted).Font("monospace").FontSize(t.Rem(0.88))
					})
				})
				ui.Text(c, s.text.PushOnly).FontSize(t.Rem(0.92)).TextColor(t.TextMuted)
				ui.Switch(c.Key(fmt.Sprint("push-", i)), &f.cur.rewrites[i].Push).Label(s.text.PushOnly).Disabled(s.busy)
				if ghost(c, s.text.RemoveRewrite, icons.Minus).Disabled(s.busy).Clicked() {
					remove = i
				}
			})
		}
		if remove >= 0 {
			f.cur.rewrites = slices.Delete(f.cur.rewrites, remove, remove+1)
		}
		r.line(func() {
			if textButton(c, s.text.AddRewrite, t.Accent, icons.Plus).Disabled(s.busy).Clicked() {
				f.cur.rewrites = append(f.cur.rewrites, account.Rewrite{})
			}
		})
		if f.custom() {
			return
		}
		next, ok := f.nextDefaultRewrite()
		r.line(func() {
			if textButton(c, s.text.AddDefault, t.Accent, icons.Plus).Disabled(s.busy || !ok).Clicked() {
				f.cur.rewrites = append(f.cur.rewrites, next)
			}
		})
	})
}

func (s *Screen) notesGroup(c *ui.Context, f *form) {
	s.section(c, s.text.GroupNotes, "", nil, func(r *rows) {
		r.line(func() {
			ui.TextArea(c.Key("notes"), &f.cur.notes).Label(s.text.GroupNotes).Grow(1).TextAlign(ui.Start).Background(ui.Color{}).Border(0, ui.Color{}).Disabled(s.busy)
		})
	})
}

func (s *Screen) timesGroup(c *ui.Context, f *form) {
	t := c.Theme()
	stamp := func(v time.Time) string { return v.Local().Format("2006-01-02 15:04") }
	s.section(c, s.text.GroupTimes, "", nil, func(r *rows) {
		r.field(s.text.Created, "", func() { ui.Text(c, stamp(f.base.CreatedAt)).Grow(1).TextAlign(ui.End).TextColor(t.TextMuted) })
		r.field(s.text.Updated, "", func() { ui.Text(c, stamp(f.base.UpdatedAt)).Grow(1).TextAlign(ui.End).TextColor(t.TextMuted) })
		r.field(s.text.Used, "", func() {
			used := s.text.Never
			if f.base.LastUsedAt != nil {
				used = stamp(*f.base.LastUsedAt)
			}
			ui.Text(c, used).Grow(1).TextAlign(ui.End).TextColor(t.TextMuted)
		})
	})
}

// ghost is a borderless square symbol button.
func ghost(c *ui.Context, label string, icon *ui.SVG) ui.Element {
	t := c.Theme()
	b := ui.Button(c.Key("ghost-"+label), "").Label(label).Size(t.Space(6), t.Space(6)).Padding(0).Radius(t.Radius).Border(0, ui.Color{}).
		Background(ui.Color{}).TextColor(t.TextMuted).Tooltip(label).Children(func() { ui.Icon(c, icon).FontSize(t.Rem(1.1)) })
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	return b
}

// textButton is a button drawn as colored text with a leading symbol.
func textButton(c *ui.Context, label string, color ui.Color, icon *ui.SVG) ui.Element {
	t := c.Theme()
	return ui.Button(c.Key("text-"+label), "").Label(label).Background(ui.Color{}).Border(0, ui.Color{}).TextColor(color).Children(func() {
		if icon != nil {
			ui.Icon(c, icon).FontSize(t.Rem(1.1))
		}
		ui.Text(c, label).SingleLine()
	})
}

// barHeight is the height of the sidebar's add bar and of the detail's footer, so their hairlines are one line.
func barHeight(t *ui.Theme) float32 { return t.Space(11) }

// buttonHeight is the height of every footer button; buttonWidth is the width of Revert and Save, wide enough
// for the longest label of either language, so that a label change moves nothing.
func buttonHeight(t *ui.Theme) float32 { return t.Space(7) }
func buttonWidth(t *ui.Theme) float32  { return t.Space(25) }

// footer holds Delete, the status, Revert and Save.
func (s *Screen) footer(c *ui.Context, f *form) {
	t := c.Theme()
	dirty := f.isNew || f.dirty()
	flash := !s.busy && s.now().Before(s.savedUntil)
	if flash {
		c.After(s.savedUntil.Sub(s.now()))
	}
	bw, bh := buttonWidth(t), buttonHeight(t)
	ui.Row(c).MinHeight(barHeight(t)).Shrink(0).PaddingX(t.Space(5)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
		if !f.isNew && textButton(c, s.text.Delete, t.Danger, nil).Height(bh).Disabled(s.busy).Clicked() {
			s.askDelete()
		}
		ui.Row(c).Grow(1).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() { s.status(c, f, dirty) })
		if ui.Button(c, s.text.Revert).Size(bw, bh).Disabled(!dirty || s.busy).Clicked() {
			s.revert()
		}
		var save ui.Element
		switch {
		case flash:
			save = ui.PrimaryButton(c, "").Label(s.text.Saved).Size(bw, bh).Children(func() {
				ui.Icon(c, icons.Check).FontSize(t.Rem(1.1))
				ui.Text(c, s.text.Saved).SingleLine()
			}).Disabled(true)
		case s.busy:
			save = ui.PrimaryButton(c, s.text.Saving).Size(bw, bh).Disabled(true)
		default:
			save = ui.PrimaryButton(c, s.text.Save).Size(bw, bh).Disabled(!dirty)
		}
		if save.Clicked() {
			s.save()
		}
	})
}

// status is the line in the footer: an error first, then what is pending.
func (s *Screen) status(c *ui.Context, f *form, dirty bool) {
	t := c.Theme()
	small := func(text string, color ui.Color) {
		ui.Text(c, text).FontSize(t.Rem(0.92)).TextColor(color).Grow(1)
	}
	switch {
	case s.busy:
		ui.Spinner(c)
		small(s.text.Writing, t.TextMuted)
	case s.notice != "":
		ui.Icon(c, icons.Warn).FontSize(t.Rem(1)).TextColor(t.Danger)
		small(s.notice, t.Danger)
	case len(s.problems) > 0 && s.problemsID == f.id:
		format := s.text.ProblemsMany
		if len(s.problems) == 1 {
			format = s.text.ProblemsOne
		}
		small(fmt.Sprintf(format, len(s.problems)), t.Danger)
	case f.isNew:
		small(s.text.DraftStatus, t.TextMuted)
	case dirty:
		small(s.text.Dirty, t.TextMuted)
	}
}

// DialogOpen reports whether the delete confirmation shows. The owner shows it in a window of its own, which Dialog
// draws.
func (s *Screen) DialogOpen() bool { return s.confirmOpen }

// Dialog draws the delete confirmation and returns its column, or a zero element when it is closed.
func (s *Screen) Dialog(c *ui.Context) ui.Element {
	if !s.confirmOpen {
		return ui.Element{}
	}
	return s.deleteDialog(c)
}

// CloseDialog closes the delete confirmation without deleting, as Cancel does; its window was closed.
func (s *Screen) CloseDialog() { s.confirmOpen, s.confirm = false, nil }

// deleteDialog asks before deleting the selected identity and says what is not deleted with it.
func (s *Screen) deleteDialog(c *ui.Context) ui.Element {
	cf := s.confirm
	if cf == nil {
		return ui.Element{}
	}
	a := s.stored(cf.id)
	if a == nil {
		s.confirmOpen, s.confirm = false, nil
		return ui.Element{}
	}
	t := c.Theme()
	pane := theme.Dialog(c, &s.confirmOpen, ui.RoleAlertDialog, fmt.Sprintf(s.text.DelTitle, s.label(*a)), func() {
		ui.Text(c, fmt.Sprintf(s.text.DelTitle, s.label(*a))).FontWeight(600).Role(ui.RoleHeading).Level(2)
		ui.Text(c, s.text.DelBody).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).Wrap()
		ui.Column(c).Children(func() {
			hairline(c)
			ui.Row(c).Gap(t.Space(3)).PaddingY(t.Space(2)).Children(func() {
				ui.Text(c, s.text.PrivateKey).TextColor(t.TextMuted).Width(t.Space(16)).Shrink(0)
				key := s.text.None
				if strings.TrimSpace(a.KeyPath) != "" {
					key = a.KeyPath
				}
				ui.Text(c, key).Font("monospace").FontSize(t.Rem(0.88)).Grow(1).Wrap()
			})
			hairline(c)
			ui.Row(c).Gap(t.Space(3)).PaddingY(t.Space(2)).Children(func() {
				ui.Text(c, s.text.Folders).TextColor(t.TextMuted).Width(t.Space(16)).Shrink(0)
				ui.Column(c).Grow(1).Children(func() {
					folders := usablePaths(*a)
					if a.Scope.Global || len(folders) == 0 {
						ui.Text(c, s.text.None).Font("monospace").FontSize(t.Rem(0.88))
					}
					for _, p := range folders {
						ui.Text(c, p).Font("monospace").FontSize(t.Rem(0.88)).Wrap()
					}
				})
			})
			hairline(c)
		})
		ui.Text(c, s.text.DelNote).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).Wrap()
		ui.Column(c).Gap(t.Space(1)).Children(func() {
			ui.Checkbox(c, &cf.alsoKey, s.text.AlsoKey).Disabled(cf.checking || cf.reason != keyfile.None || s.busy)
			if text := s.keyReason(cf); text != "" {
				ui.Text(c, text).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).Wrap().PaddingX(t.Space(6))
			} else if cf.alsoKey {
				ui.Text(c, s.text.KeyNote).FontSize(t.Rem(0.92)).TextColor(t.TextMuted).Wrap().PaddingX(t.Space(6))
			}
		})
		if cf.err != "" {
			ui.Text(c, cf.err).FontSize(t.Rem(0.92)).TextColor(t.Danger).Wrap()
		}
		ui.Row(c).Gap(t.Space(2)).Justify(ui.End).Children(func() {
			if ui.Button(c, s.text.Cancel).AutoFocus().Disabled(s.busy).Clicked() {
				s.confirmOpen = false
			}
			del := ui.Button(c, s.text.Delete).Background(t.Danger).TextColor(ui.RGB(255, 255, 255)).Disabled(cf.checking || s.busy)
			if del.Clicked() {
				s.confirmDelete()
			}
		})
	})
	if s.busy {
		// Escape and a click outside do not dismiss the dialog while the identity is being deleted.
		s.confirmOpen = true
	}
	if !s.confirmOpen {
		s.confirm = nil
	}
	return pane
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

// keyReason says why the private key cannot be deleted, "" when it can or is still being looked at.
func (s *Screen) keyReason(cf *confirm) string {
	switch cf.reason {
	case keyfile.NoPath:
		return s.text.ReasonNoKey
	case keyfile.Shared:
		return fmt.Sprintf(s.text.ReasonShared, s.labelOf(cf.other))
	case keyfile.Symlink:
		return s.text.ReasonSymlink
	case keyfile.Missing:
		return s.text.ReasonMissing
	case keyfile.NotFile:
		return s.text.ReasonNotFile
	}
	return ""
}
