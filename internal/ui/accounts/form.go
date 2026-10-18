package accounts

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/gitdir"
	"github.com/yangflow/idset/internal/paths"
)

// values are the fields the form edits, in the shape the widgets bind to. Slices are never nil, so that two
// equal forms compare equal.
type values struct {
	label, username, gitName, email, alias, keyPath, hostName, notes string
	provider                                                         account.Provider
	port                                                             int
	portText                                                         string
	global                                                           bool
	paths                                                            []string
	rewrites                                                         []account.Rewrite
	color                                                            string
}

func valuesOf(a account.Account) values {
	return values{
		label: a.Label, username: a.Username, gitName: a.GitUserName, email: a.GitUserEmail, alias: a.SSHAlias,
		keyPath: a.KeyPath, hostName: a.HostName, notes: a.Notes, provider: a.Provider, port: a.SSHPort,
		portText: strconv.Itoa(a.SSHPort), global: a.Scope.Global, paths: append([]string{}, a.Scope.Paths...),
		rewrites: append([]account.Rewrite{}, a.URLRewrites...), color: a.Color,
	}
}

func (v values) clone() values {
	v.paths = append([]string{}, v.paths...)
	v.rewrites = append([]account.Rewrite{}, v.rewrites...)
	return v
}

// form is the edit state of one identity, saved or new.
type form struct {
	id    string
	isNew bool
	// base is the identity as stored; the fields the form does not edit come from it.
	base      account.Account
	orig, cur values
	// Widget state that mirrors cur: a segment index, the platform's name, the hex field.
	scope    int
	platform string
	hex      string
	hexShown string
	// The signature read from git config, while the fields still hold it.
	prefName, prefEmail string
	// keyEditing is set while a long key path is shown as the input; keyFrames counts the frames since the click.
	keyEditing bool
	keyFrames  int
}

func newForm(a account.Account, isNew bool) *form {
	v := valuesOf(a)
	f := &form{id: a.ID, isNew: isNew, base: a, orig: v, cur: v}
	if !v.global {
		f.scope = 1
	}
	f.hex = hexOf(v.color)
	f.hexShown = f.hex
	return f
}

// hexOf is the #RRGGBB of a preset name or of a hex color; anything else is shown as typed.
func hexOf(color string) string {
	if strings.HasPrefix(color, "#") {
		return color
	}
	r, g, b := account.Account{Color: color}.RGB()
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

func (f *form) dirty() bool { return !reflect.DeepEqual(f.cur, f.orig) }

func (f *form) custom() bool { return f.cur.provider == account.Custom }

// setScope switches between global and folders; folders start with one path, and global drops the paths.
func (f *form) setScope(index int) {
	f.scope = index
	if global := index == 0; global != f.cur.global {
		f.cur.global = global
		f.cur.paths = []string{}
		if !global {
			f.cur.paths = []string{"~/"}
		}
	}
}

// addFolder adds a stored path to the folders, switching a global identity to folders with that one path.
func (f *form) addFolder(stored string) {
	if f.cur.global {
		f.setScope(1)
		f.cur.paths = []string{stored}
		return
	}
	if !slices.Contains(f.cur.paths, stored) {
		f.cur.paths = append(f.cur.paths, stored)
	}
}

// setProvider changes the platform and moves the port to one the platform accepts.
func (f *form) setProvider(p account.Provider) {
	old := f.cur.provider
	if p == old {
		return
	}
	f.cur.provider = p
	switch {
	case p == account.Custom:
		f.cur.portText = strconv.Itoa(f.cur.port)
	case old == account.Custom:
		if n, err := strconv.Atoi(strings.TrimSpace(f.cur.portText)); err == nil && n == 443 && p.Supports443() {
			f.cur.port = 443
		} else {
			f.cur.port = 22
		}
	case !p.Supports443():
		f.cur.port = 22
	}
}

func (f *form) chooseColor(color string) {
	f.cur.color = color
	f.hex = hexOf(color)
	f.hexShown = f.hex
}

// syncHex takes what was typed into the hex field as the color.
func (f *form) syncHex() {
	if f.hex != f.hexShown {
		f.hexShown = f.hex
		f.cur.color = strings.TrimSpace(f.hex)
	}
}

// defaultRewrites are the rules the platform's own addresses need to reach the alias, in the order they are added.
func (f *form) defaultRewrites() []account.Rewrite {
	if f.custom() {
		return nil
	}
	return account.DefaultRewrites(f.cur.provider, f.cur.alias)
}

// nextDefaultRewrite is the first default rule the form does not have yet.
func (f *form) nextDefaultRewrite() (account.Rewrite, bool) {
	for _, r := range f.defaultRewrites() {
		if !slices.ContainsFunc(f.cur.rewrites, func(have account.Rewrite) bool {
			return have.From == r.From && have.To == r.To
		}) {
			return r, true
		}
	}
	return account.Rewrite{}, false
}

// account builds the identity to save from base, which should be the latest stored version, so that fields the
// form does not edit (when it was last used) are not overwritten with old values. ok is false when the port of
// a custom platform is not a number.
func (f *form) account(base account.Account, env paths.Env) (a account.Account, portOK bool) {
	v := f.cur
	trim := strings.TrimSpace
	a = base
	a.Label, a.Username, a.GitUserName, a.GitUserEmail = trim(v.label), trim(v.username), trim(v.gitName), trim(v.email)
	a.SSHAlias, a.Notes, a.Provider, a.Color = trim(v.alias), v.notes, v.provider, v.color
	portOK = true
	if v.provider == account.Custom {
		a.HostName = trim(v.hostName)
		n, err := strconv.Atoi(trim(v.portText))
		a.SSHPort, portOK = n, err == nil
	} else {
		a.HostName, a.SSHPort = "", v.port
	}
	key := ""
	if raw := trim(v.keyPath); raw != "" {
		key = env.Tilde(env.Expand(raw))
	}
	if key != a.KeyPath {
		// The fingerprint belongs to the old key.
		a.KeyFingerprint = ""
	}
	a.KeyPath = key
	a.Scope = account.Scope{Global: v.global}
	if !v.global {
		a.Scope.Paths = []string{}
		for _, p := range v.paths {
			if n := gitdir.Normalize(p, env); n != "" {
				a.Scope.Paths = append(a.Scope.Paths, n)
			}
		}
	}
	a.URLRewrites = []account.Rewrite{}
	for _, r := range v.rewrites {
		r.From, r.To = trim(r.From), trim(r.To)
		if r.From != "" || r.To != "" {
			a.URLRewrites = append(a.URLRewrites, r)
		}
	}
	return a, portOK
}
