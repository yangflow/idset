package panel

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/diagnose"
	"github.com/yangflow/idset/internal/match"
	"github.com/yangflow/idset/internal/ui/icons"
	"github.com/yangflow/idset/internal/ui/theme"
)

// Diagnostics is what the owner knows of the connectivity checks: the current check of each identity, the
// identities whose check is running, and the Host names of the user's own ~/.ssh/config.
type Diagnostics struct {
	Checks    []diagnose.Check
	Pending   []string
	UserHosts []string
}

// diagState is the diagnostics part of the panel. known is set once the owner has reported; until then the rows
// have no status dots.
type diagState struct {
	Diagnostics
	known bool
	// open says whether the summary is expanded, and seen is the set of entries it last showed: a new set opens it.
	open bool
	seen string
	// The transient state of the problem bars, by identity.
	pubUntil  map[string]time.Time
	noPub     map[string]bool
	unlocking map[string]bool
	unlockErr map[string]bool
}

// SetDiagnostics shows the current checks. A check that was running and is done clears what its bar showed.
func (s *Screen) SetDiagnostics(d Diagnostics) {
	for _, id := range s.diag.Pending {
		if !slices.Contains(d.Pending, id) {
			delete(s.diag.noPub, id)
			delete(s.diag.pubUntil, id)
		}
	}
	s.diag.Diagnostics, s.diag.known = d, true
}

// UnlockDone ends an unlock; after a failure the bar shows the ssh-add command to run in a terminal. The owner
// checks the identity again after a success.
func (s *Screen) UnlockDone(id string, err error) {
	delete(s.diag.unlocking, id)
	if err != nil {
		setFlag(&s.diag.unlockErr, id)
	}
}

// PublicKeyDone ends a copy of the public key: "Copied" for a moment, or why nothing was copied.
func (s *Screen) PublicKeyDone(id string, err error) {
	if err != nil {
		setFlag(&s.diag.noPub, id)
		return
	}
	delete(s.diag.noPub, id)
	if s.diag.pubUntil == nil {
		s.diag.pubUntil = map[string]time.Time{}
	}
	s.diag.pubUntil[id] = s.now().Add(copiedFor)
	s.say = s.text.Copied
}

func setFlag(m *map[string]bool, id string) {
	if *m == nil {
		*m = map[string]bool{}
	}
	(*m)[id] = true
}

// Worst is the severity the tray icon shows: the worst entry of the summary, and an error when ssh is missing.
func (s *Screen) Worst() match.Severity {
	if s.noSSH() {
		return match.Error
	}
	return diagnose.Worst(s.entries())
}

func (s *Screen) check(id string) (diagnose.Check, bool) {
	i := slices.IndexFunc(s.diag.Checks, func(c diagnose.Check) bool { return c.ID == id })
	if i < 0 {
		return diagnose.Check{}, false
	}
	return s.diag.Checks[i], true
}

func (s *Screen) pending(id string) bool { return slices.Contains(s.diag.Pending, id) }

// noSSH reports whether the checks found no ssh: then one notice replaces the bars and the summary.
func (s *Screen) noSSH() bool {
	return slices.ContainsFunc(s.diag.Checks, func(c diagnose.Check) bool { return c.Failure == diagnose.NoCommand })
}

// ready reports whether a round of checks has finished: the summary waits for it.
func (s *Screen) ready() bool { return s.diag.known && len(s.diag.Pending) == 0 }

// entries are the lines of the summary, none before a round has finished or when ssh is missing.
func (s *Screen) entries() []diagnose.Entry {
	if !s.ready() || s.noSSH() {
		return nil
	}
	in := diagnose.Input{List: s.accounts, Checks: s.diag.Checks, UserHosts: s.diag.UserHosts}
	if r := s.result; r != nil && r.Kind == match.Matched {
		in.Matched, in.Findings = r.ID, r.Findings
	}
	return diagnose.Summarize(in)
}

// problem is the bar under an identity, if any. Without ssh the notice above the list says it once.
func (s *Screen) problem(a account.Account) (diagnose.Problem, diagnose.Check, bool) {
	c, ok := s.check(a.ID)
	if !ok {
		return 0, c, false
	}
	matched, origin := false, ""
	if r := s.result; r != nil && r.Kind == match.Matched && r.ID == a.ID {
		matched, origin = true, r.Origin
	}
	p, ok := diagnose.ProblemOf(a, c, matched, origin)
	return p, c, ok && p != diagnose.MissingCommand
}

// host is the server an identity connects to, for the sentences of the bars.
func host(a account.Account) string {
	if h := a.RealHost(); h != "" {
		return h
	}
	return a.SSHAlias
}

// platform names the platform of an identity in a sentence: its name, or the server for a custom one.
func platform(a account.Account) string {
	if a.Provider == account.Custom {
		return host(a)
	}
	return a.Provider.DisplayName()
}

// failure says in a few words why the check of a failed: the sentence of its bar, or the output of ssh.
func (s *Screen) why(a account.Account, p diagnose.Problem, c diagnose.Check) string {
	switch p {
	case diagnose.MissingKey:
		return strings.TrimSpace(s.text.NoKey)
	case diagnose.LockedKey:
		return s.text.Locked
	case diagnose.Unreachable:
		return fmt.Sprintf(s.unreachable(c.Failure), host(a))
	}
	if c.Failure == diagnose.NoCommand {
		return s.text.NoSSH
	}
	if c.Failure == diagnose.Other && c.Detail != "" {
		return c.Detail
	}
	return fmt.Sprintf(s.text.Denied, platform(a))
}

// unreachable is the sentence for a host that cannot be reached, with a %s for the host, and its reason.
func (s *Screen) unreachable(f diagnose.Failure) string {
	switch f {
	case diagnose.NoHost:
		return s.text.NoHost
	case diagnose.Refused:
		return s.text.Refused
	case diagnose.HostKey:
		return s.text.HostKey
	}
	return s.text.TimedOut
}

func (s *Screen) unreachableTip(f diagnose.Failure, h string) string {
	switch f {
	case diagnose.NoHost:
		return s.text.NoHostTip
	case diagnose.Refused:
		return s.text.RefusedTip
	case diagnose.HostKey:
		return fmt.Sprintf(s.text.HostKeyTip, h)
	}
	return s.text.TimedOutTip
}

// connDot is the status of the check at the end of a row: green when it passed, red when it failed, a ring while the
// first check runs. A row without an alias keeps the space, so that the folders line up.
func (s *Screen) connDot(c *ui.Context, a account.Account) {
	t := c.Theme()
	slot := theme.Center(c, ui.Column(c)).Size(t.Space(theme.IconS), t.Space(theme.IconS)).Shrink(0)
	if !diagnose.Checkable(a) {
		return
	}
	check, ok := s.check(a.ID)
	var tip string
	switch {
	case ok && check.OK && check.User != "":
		tip = fmt.Sprintf(s.text.ConnOK, platform(a), check.User)
	case ok && check.OK:
		tip = s.text.ConnOKPlain
	case ok:
		p, _ := diagnose.ProblemOf(a, check, false, "")
		tip = fmt.Sprintf(s.text.ConnFail, s.why(a, p, check))
	case s.pending(a.ID):
		tip = s.text.ConnPending
	default:
		return
	}
	slot.Role(ui.RoleImage).Label(tip)
	tipAboveEnd(c, slot, tip)
	slot.Children(func() {
		size := t.Space(theme.DotS)
		d := ui.Column(c).Size(size, size).Radius(size / 2)
		switch {
		case ok && check.OK:
			d.Background(t.Success)
		case ok:
			d.Background(t.Danger)
		default:
			d.Border(1.5, t.TextMuted)
		}
	})
}

// problemBar is the problem under a row: the sentence, whose reason is its tooltip, Check Again at the end, then the
// next step.
func (s *Screen) problemBar(c *ui.Context, a account.Account, p diagnose.Problem, check diagnose.Check) {
	t := c.Theme()
	icon, color := icons.Error, t.Danger
	if p == diagnose.HTTPSRemote {
		icon, color = icons.Globe, t.TextMuted
	}
	var tip, key string
	var lead string
	switch p {
	case diagnose.MissingKey:
		lead, key, tip = s.text.NoKey, a.KeyPath, fmt.Sprintf(s.text.NoKeyTip, a.KeyPath)
	case diagnose.LockedKey:
		lead, tip = s.text.Locked, s.text.LockedTip
	case diagnose.Unreachable:
		lead, tip = fmt.Sprintf(s.unreachable(check.Failure), host(a)), s.unreachableTip(check.Failure, host(a))
	case diagnose.Denied:
		lead, tip = fmt.Sprintf(s.text.Denied, platform(a)), fmt.Sprintf(s.text.DeniedTip, platform(a))
	case diagnose.HTTPSRemote:
		lead, tip = s.text.HTTPSRow, s.text.HTTPSTip
		if r := account.DefaultRewrites(a.Provider, a.SSHAlias); len(r) > 0 {
			tip = fmt.Sprintf(s.text.HTTPSBarTip, r[0].From, r[0].To)
		}
	}
	ui.Column(c.Key("problem-"+a.ID)).Padding(0, t.Space(theme.Space2), t.Space(theme.Space1_5), t.Space(theme.Space3+theme.BadgeS+theme.Space2)).Children(func() {
		theme.Inline(c, ui.Row(c)).Height(t.Space(theme.RowS)).MinWidth(0).Children(func() {
			ui.Icon(c, icon).FontSize(t.Space(theme.IconS)).TextColor(color).Shrink(0)
			says := ui.Row(c).Grow(1).MinWidth(0).AlignItems(ui.Center).Label(lead + s.text.Then + tip).Children(func() {
				theme.Secondary.On(c, ui.Text(c, strings.TrimSpace(lead))).SingleLine().Shrink(0).MinWidth(0)
				if key != "" {
					ui.Row(c).Width(t.Rem(0.28)).Shrink(0)
					pathLine(c, key, "", tip, pathFace{style: theme.MonoSmall, color: t.Text})
				}
			})
			tipAbove(c, says, tip)
			if p == diagnose.Denied {
				s.pubAck(c, a)
			}
			if p.Retryable() {
				s.retryButton(c, "retry-"+a.ID, []string{a.ID})
			}
		})
		s.nextStep(c, a, p)
	})
}

// pubAck says what the last copy of the public key did: "Copied" for a moment, or that there is no .pub file.
func (s *Screen) pubAck(c *ui.Context, a account.Account) {
	t := c.Theme()
	if s.diag.noPub[a.ID] {
		text := theme.Small.On(c, ui.Text(c, s.text.NoPub)).TextColor(t.TextMuted).SingleLine().Shrink(0)
		tipAboveEnd(c, text, fmt.Sprintf(s.text.NoPubTip, a.KeyPath))
		return
	}
	s.copied(c, "pub-"+a.ID, s.diag.pubUntil[a.ID])
}

// recheckButton in the footer checks every identity again.
func (s *Screen) recheckButton(c *ui.Context) {
	var ids []string
	for _, a := range s.accounts {
		if diagnose.Checkable(a) {
			ids = append(ids, a.ID)
		}
	}
	b := symbolButton(c, "ghost-"+s.text.Recheck, s.text.Recheck, icons.Refresh, false).Tooltip(s.text.Recheck).
		Disabled(len(ids) == 0 || slices.ContainsFunc(ids, s.pending))
	if b.Clicked() {
		s.diag.unlockErr = nil
		s.act.CheckAgain(nil)
	}
}

// retryButton checks ids again; it waits while their check runs.
func (s *Screen) retryButton(c *ui.Context, key string, ids []string) {
	b := symbolButton(c, key, s.text.Retry, icons.Refresh, false).Disabled(slices.ContainsFunc(ids, s.pending))
	tipAboveEnd(c, b, s.text.Retry)
	if b.Clicked() {
		for _, id := range ids {
			delete(s.diag.unlockErr, id)
		}
		s.act.CheckAgain(ids)
	}
}

// underGlyph is the indent that lines a second line of a bar or notice up with its sentence: past the IconS
// glyph and the Space1_5 gap after it, 20 DIPs at the default spacing.
func underGlyph(t *ui.Theme) float32 { return t.Space(theme.IconS) + t.Space(theme.Space1_5) }

// nextStep is the line under the sentence of a bar: the fixed next step of the problem, when it has one.
func (s *Screen) nextStep(c *ui.Context, a account.Account, p diagnose.Problem) {
	t := c.Theme()
	actions := func(children func()) {
		theme.Cluster(c, ui.Row(c)).Height(t.Space(theme.ControlS)).Padding(0, 0, 0, underGlyph(t)).Children(children)
	}
	switch p {
	case diagnose.MissingKey:
		actions(func() {
			if s.link(c, "keygen-"+a.ID, s.text.GenerateKey, "", false).Clicked() {
				s.act.GenerateKey(a.ID)
			}
		})
	case diagnose.LockedKey:
		if s.goos == "darwin" && !s.diag.unlockErr[a.ID] {
			actions(func() {
				if s.link(c, "unlock-"+a.ID, s.text.Unlock, "", s.diag.unlocking[a.ID]).Clicked() {
					setFlag(&s.diag.unlocking, a.ID)
					s.act.Unlock(a.ID)
				}
			})
			return
		}
		if s.diag.unlockErr[a.ID] {
			theme.Secondary.On(c, ui.Text(c, s.text.UnlockFailed)).TextColor(t.TextMuted).Padding(0, t.Space(theme.Space1), t.Space(theme.Space1), underGlyph(t))
		}
		cmd := diagnose.UnlockCommand(s.env, a)
		theme.InlineTight(c, ui.Row(c)).Height(t.Space(theme.RowS)).Padding(0, 0, 0, underGlyph(t)).MinWidth(0).Children(func() {
			face := pathFace{style: theme.MonoSmall, color: t.TextMuted}
			ui.Row(c).Grow(1).MinWidth(0).AlignItems(ui.Center).Children(func() {
				face.apply(c, ui.Text(c, "ssh-add ")).Shrink(0)
				pathLine(c, strings.TrimPrefix(cmd, "ssh-add "), "", s.text.UnlockCmdTip, face)
			})
			b := symbolButton(c, "unlock-copy-"+a.ID, s.text.CopyUnlock, icons.Terminal, false)
			tipAboveEnd(c, b, s.text.CopyUnlock)
			if b.Clicked() {
				s.act.CopyText(cmd)
				c.Announce(s.text.Copied)
			}
		})
	case diagnose.Denied:
		actions(func() {
			if s.link(c, "pub-"+a.ID, s.text.CopyPub, "", false).Clicked() {
				delete(s.diag.noPub, a.ID)
				s.act.CopyPublicKey(a.ID)
			}
			if page := a.Provider.KeySettingsURL(); page != "" {
				name := a.Provider.DisplayName()
				if s.link(c, "keys-"+a.ID, fmt.Sprintf(s.text.OpenKeys, name), fmt.Sprintf(s.text.OpenKeysTip, name), false).Clicked() {
					c.OpenURL(page)
				}
			}
		})
	case diagnose.HTTPSRemote:
		if len(account.DefaultRewrites(a.Provider, a.SSHAlias)) == 0 {
			return
		}
		actions(func() {
			tip := fmt.Sprintf(s.text.AddRewriteTip, a.Provider.DisplayName())
			if s.link(c, "rewrite-"+a.ID, s.text.AddRewrite, tip, s.busy).Clicked() {
				s.act.AddRewrites(a.ID)
			}
		})
	}
}

// link is a text button in the accent color, the size of the text around it. tip is its tooltip when it says more
// than the text.
func (s *Screen) link(c *ui.Context, key, text, tip string, disabled bool) ui.Element {
	t := c.Theme()
	b := ui.Button(c.Key(key), "").Label(text).Disabled(disabled).Background(ui.Color{}).Border(0, ui.Color{}).
		TextColor(t.Accent).Padding(t.Space(theme.Space0_5), 0).Children(func() { theme.Secondary.On(c, ui.Text(c, text)).SingleLine() })
	if tip != "" {
		b.Description(tip)
		tipAbove(c, b, tip)
	}
	return b
}

// noSSHNotice replaces the bars when ssh is missing: every check fails the same way, so it is said once.
func (s *Screen) noSSHNotice(c *ui.Context) {
	t := c.Theme()
	hint := s.text.NoSSHLinux
	switch s.goos {
	case "windows":
		hint = s.text.NoSSHWindows
	case "darwin":
		hint = s.text.NoSSHMac
	}
	var ids []string
	for _, a := range s.accounts {
		if diagnose.Checkable(a) {
			ids = append(ids, a.ID)
		}
	}
	// A notice sits on the danger tint, not in an outline: the glyph and the words say how bad it is.
	ui.Column(c.Key("no-ssh")).Radius(theme.RadiusCard).Background(theme.Of(c).DangerSoft).Padding(0, t.Space(theme.Space2), t.Space(theme.Space1_5), t.Space(theme.Space3)).
		Children(func() {
			theme.Inline(c, ui.Row(c)).Height(t.Space(theme.RowS)).Children(func() {
				ui.Icon(c, icons.Error).FontSize(t.Space(theme.IconS)).TextColor(t.Danger).Shrink(0)
				theme.BodyStrong.On(c, ui.Text(c, s.text.NoSSH)).Grow(1).MinWidth(0)
				s.retryButton(c, "retry-ssh", ids)
			})
			theme.Secondary.On(c, ui.Text(c, hint)).TextColor(theme.Of(c).Fg2).Padding(0, t.Space(theme.Space1), t.Space(theme.Space1), underGlyph(t))
		})
}

// summary is the end of the list: one quiet line when nothing is wrong, else the counts, which expand to the
// entries. Nothing shows while a round of checks runs.
func (s *Screen) summary(c *ui.Context) {
	if !s.ready() || s.noSSH() {
		return
	}
	t := c.Theme()
	entries := s.entries()
	var seen []string
	for _, e := range entries {
		seen = append(seen, string(e.Code)+" "+e.ID)
	}
	if sig := strings.Join(seen, "\n"); sig != s.diag.seen {
		s.diag.seen, s.diag.open = sig, sig != ""
	}
	if len(entries) == 0 {
		theme.Inline(c, ui.Row(c.Key("all-ok"))).PaddingX(t.Space(theme.Space0_5)).Children(func() {
			ui.Icon(c, icons.Check).FontSize(t.Space(theme.IconS)).TextColor(t.Success)
			theme.Secondary.On(c, ui.Text(c, s.text.AllOK)).TextColor(t.TextMuted)
		})
		return
	}
	errs := 0
	for _, e := range entries {
		if e.Severity == match.Error {
			errs++
		}
	}
	count := s.count(errs, len(entries)-errs)
	fillCard(c, "summary", ui.Color{}).Children(func() {
		head := theme.BarRow(c, ui.Row(c)).Focusable().Role(ui.RoleButton).Label(count).Expanded(s.diag.open).Height(t.Space(theme.Row))
		if head.Hovered() {
			head.Background(t.SurfaceHover)
		}
		if head.Clicked() {
			s.diag.open = !s.diag.open
		}
		head.Children(func() {
			icon, color := icons.Warn, t.Warning
			if errs > 0 {
				icon, color = icons.Error, t.Danger
			}
			ui.Icon(c, icon).FontSize(t.Space(theme.IconS)).TextColor(color).Shrink(0)
			theme.BodyStrong.On(c, ui.Text(c, count)).Grow(1).MinWidth(0).SingleLine()
			// The chevron turns down as the details open.
			turn := float32(0)
			if s.diag.open {
				turn = 90
			}
			theme.Turn(ui.Icon(c, icons.ChevronRight).FontSize(t.Space(theme.IconXs)).TextColor(t.TextMuted), turn)
		})
		if !s.diag.open {
			return
		}
		theme.Arrive(c, ui.Column(c.Key("details")), theme.Collapse(), theme.Collapse()).Children(func() {
			for _, e := range entries {
				ui.Divider(c)
				s.entry(c, e)
			}
		})
	})
}

// count says how many errors and warnings there are: "1 error · 2 warnings".
func (s *Screen) count(errs, warns int) string {
	var parts []string
	for _, n := range []struct {
		n         int
		one, many string
	}{{errs, s.text.OneError, s.text.Errors}, {warns, s.text.OneWarning, s.text.Warnings}} {
		switch {
		case n.n == 1:
			parts = append(parts, n.one)
		case n.n > 1:
			parts = append(parts, fmt.Sprintf(n.many, n.n))
		}
	}
	return strings.Join(parts, " · ")
}

// entry is one line of the summary: what is wrong, how to look into it or fix it, and the code.
func (s *Screen) entry(c *ui.Context, e diagnose.Entry) {
	t := c.Theme()
	a, _ := s.find(e.ID)
	var what, how string
	switch e.Code {
	case diagnose.NET001:
		what, how = fmt.Sprintf(s.text.Net001, e.Alias), fmt.Sprintf(s.text.Net001How, e.Alias)
	case diagnose.SSH001:
		what, how = fmt.Sprintf(s.text.Ssh001, e.Alias), s.text.Ssh001How
	case diagnose.GIT001:
		what, how = fmt.Sprintf(s.text.Git001, s.label(a)), fmt.Sprintf(s.text.Git001Global, s.label(a))
		if e.Regenerate {
			how = s.text.Git001How
		}
	case diagnose.GIT002:
		globals := 0
		for _, g := range s.accounts {
			if g.Scope.Global {
				globals++
			}
		}
		what, how = fmt.Sprintf(s.text.Git002, globals), fmt.Sprintf(s.text.Git002How, s.label(a))
	}
	icon, color := icons.Warn, t.Warning
	if e.Severity == match.Error {
		icon, color = icons.Error, t.Danger
	}
	theme.Inline(c, ui.Row(c.Key("entry-"+string(e.Code)+"-"+e.ID))).Padding(t.Space(theme.Space2), t.Space(theme.Space3)).AlignItems(ui.Start).
		Children(func() {
			ui.Icon(c, icon).FontSize(t.Space(theme.IconS)).TextColor(color).Shrink(0)
			theme.StackTight(c, ui.Column(c)).Grow(1).MinWidth(0).AlignItems(ui.Start).Children(func() {
				theme.Secondary.On(c, ui.Text(c, what))
				theme.Secondary.On(c, ui.Text(c, how)).TextColor(t.TextMuted)
				switch {
				case e.Code == diagnose.NET001:
					if s.link(c, "debug-"+e.ID, s.text.CopyDebug, "", false).Clicked() {
						s.act.CopyText("ssh -vT git@" + e.Alias)
						c.Announce(s.text.Copied)
					}
				case e.Code == diagnose.GIT001 && e.Regenerate:
					if s.link(c, "regen-"+e.ID, s.text.Regenerate, "", s.busy).Clicked() {
						s.act.Regenerate()
					}
				}
			})
			theme.MonoSmall.On(c, ui.Text(c, string(e.Code))).TextColor(t.TextMuted).Shrink(0)
		})
}
