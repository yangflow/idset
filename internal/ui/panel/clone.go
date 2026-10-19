package panel

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/match"
	"github.com/yangflow/idset/internal/remote"
	"github.com/yangflow/idset/internal/ui/icons"
	"github.com/yangflow/idset/internal/ui/theme"
)

// repoPlaceholder is the same in every language.
const repoPlaceholder = "owner/repo"

// cloneField is the text of one clone helper and, after a copy, until when it says "Copied".
type cloneField struct {
	value string
	until time.Time
}

// cloneSeed is what the helper of the matched card starts with: the owner/repo of the origin, else the remembered
// owner prefix, else nothing.
func cloneSeed(r match.Result, prefix string) string {
	if repo := remote.Repo(r.Origin); repo != "" {
		return repo
	}
	return prefix
}

// copied says "Copied" before a button, for the time that is left after a copy; the button shows the check mark.
func (s *Screen) copied(c *ui.Context, until time.Time) bool {
	left := until.Sub(s.now())
	if left <= 0 {
		return false
	}
	t := c.Theme()
	ui.Text(c, s.text.Copied).FontSize(t.Rem(0.85)).TextColor(t.TextMuted).SingleLine().Shrink(0)
	c.After(left)
	return true
}

// remoteRow is the last row of the findings block when the origin is an HTTPS address that has an SSH form: a
// quiet note, never a finding, and the button that copies the command that changes the origin.
func (s *Screen) remoteRow(c *ui.Context, r *match.Result, a account.Account) {
	cmd, ok := remote.RemoteCommand(a, r.Origin)
	if !ok {
		return
	}
	t := c.Theme()
	ui.Row(c.Key("https")).Height(t.Space(7)).Padding(0, t.Space(2), 0, t.Space(3)).Gap(t.Space(1.5)).AlignItems(ui.Center).
		MinWidth(0).Children(func() {
		ui.Icon(c, icons.Globe).FontSize(t.Rem(1.1)).TextColor(t.TextMuted).Shrink(0)
		text := ui.Text(c, s.text.HTTPSRow).FontSize(t.Rem(0.9)).SingleLine().Grow(1).MinWidth(0)
		tipAbove(c, text, s.text.HTTPSTip)
		s.copied(c, s.httpsUntil)
		icon, done := icons.Terminal, s.now().Before(s.httpsUntil)
		if done {
			icon = icons.Check
		}
		button := symbolButton(c, "https-copy", s.text.CopyCmd, icon, done)
		tipAboveEnd(c, button, fmt.Sprintf(s.text.CopyCmdTip, cmd))
		if button.Clicked() {
			s.act.CopyText(cmd)
			s.httpsUntil = s.now().Add(copiedFor)
			c.After(copiedFor)
			c.Announce(s.text.Copied)
		}
	})
}

// cloneBlock is the clone helper: an input that takes owner/repo, an address or a whole git clone command, the
// button that copies the command, and under them the command itself.
func (s *Screen) cloneBlock(c *ui.Context, key string, a account.Account, f *cloneField, left float32, focus bool) {
	if a.SSHAlias == "" {
		return
	}
	t := c.Theme()
	cmd, ok := remote.CloneCommand(a, f.value)
	ui.Column(c.Key(key)).Gap(t.Space(1)).Padding(t.Space(2), t.Space(3), t.Space(2.5), left).Children(func() {
		ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
			in := ui.TextInput(c.Key(key+"-input"), &f.value).Label(s.text.CloneLabel).Placeholder(repoPlaceholder).
				Tooltip(s.text.CloneTip).Font("monospace").FontSize(t.Rem(0.88)).Height(t.Space(7)).Radius(theme.FieldRadius).Border(0.5, t.Border).Background(theme.Of(c).Field).Grow(1).MinWidth(0)
			if focus {
				in.AutoFocus()
			}
			enter := in.Submitted()
			s.copied(c, f.until)
			icon, done := icons.Terminal, s.now().Before(f.until)
			if done {
				icon = icons.Check
			}
			button := symbolButton(c, key+"-copy", s.text.CloneCopy, icon, done).Disabled(!ok)
			tipAboveEnd(c, button, s.text.CloneCopy)
			if ok && (button.Clicked() || enter) {
				s.act.CopyText(cmd)
				f.until = s.now().Add(copiedFor)
				c.After(copiedFor)
				c.Announce(s.text.Copied)
				s.act.CloneCopied(a.ID, remote.Prefix(f.value))
			}
		})
		ui.Row(c).Height(t.Space(4)).PaddingX(t.Space(0.5)).AlignItems(ui.Center).MinWidth(0).Children(func() {
			if !ok {
				ui.Text(c, s.text.CloneHint).FontSize(t.Rem(0.85)).TextColor(t.TextMuted).SingleLine().MinWidth(0)
				return
			}
			// Only the middle gives way: "git clone git@alias:" and the last segment stay.
			pathLine(c, cmd, "", cmd, pathFace{mono: true, size: t.Rem(0.85), color: t.TextMuted})
		})
	})
}

// openRowClone is the helper under the open identity row.
func (s *Screen) openRowClone(c *ui.Context, a account.Account) {
	t := c.Theme()
	s.cloneBlock(c, "row-clone", a, &s.rowClone, t.Space(8), true)
}

// toggleRow opens the clone helper of an identity row, or closes it; only one helper is open at a time.
func (s *Screen) toggleRow(id string) {
	if s.openRow == id {
		s.openRow = ""
		return
	}
	s.openRow = id
	s.cardCloneOpen = false
	s.rowClone = cloneField{value: s.prefixes[id]}
}

// toggleCardClone opens the clone helper of the matched card, or closes it; opening it closes a row's helper.
func (s *Screen) toggleCardClone() {
	s.cardCloneOpen = !s.cardCloneOpen
	if s.cardCloneOpen {
		s.openRow = ""
	}
}

// cardClone is the helper at the bottom of the matched card, under a hairline, opened by the card's toggle.
func (s *Screen) cardClone(c *ui.Context, a account.Account) {
	if a.SSHAlias == "" {
		return
	}
	ui.Divider(c)
	s.cloneBlock(c, "clone", a, &s.matchClone, c.Theme().Space(3), true)
}
