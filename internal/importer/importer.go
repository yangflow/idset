// Package importer finds identities in the user's existing ~/.ssh/config and git configuration: it reads them, one
// level of Include and includeIf deep, and turns each SSH Host into a candidate identity with its port, platform,
// key, signature and rewrite rules. It only reads; the files are read and git is run through injectable functions,
// and tests use fakes.
package importer

import (
	"cmp"
	"context"
	"errors"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/diagnose"
	"github.com/yangflow/idset/internal/gitdir"
	"github.com/yangflow/idset/internal/match"
	"github.com/yangflow/idset/internal/paths"
)

// Candidate is an identity that can be imported. Account has everything read from the configuration and a Label
// of the Host name; ID, color and times are set when it is added. Exists marks an identity Idset already has,
// which is not checked by default; Limited marks a global identity imported as limited, without paths, because
// there can be only one global identity.
type Candidate struct {
	Account account.Account
	Exists  bool
	Limited bool
}

// Source is where Scan reads from: the files, through ReadFile and Glob, which take paths of the file system, and
// git, the program Git run through Run.
type Source struct {
	Env      paths.Env
	ReadFile func(name string) ([]byte, error)
	Glob     func(pattern string) ([]string, error)
	Run      diagnose.Run
	Git      string
}

// Scan reads ~/.ssh/config with the files it includes and the global git configuration with the files its
// includeIf lines point at, and returns the candidates for list. A missing ~/.ssh/config or global git
// configuration has nothing in it; an included file that cannot be read is skipped. The error is for
// ~/.ssh/config being unreadable and git not running or timing out.
func Scan(ctx context.Context, src Source, list []account.Account) ([]Candidate, error) {
	cfg, err := ReadSSH(src)
	if err != nil {
		return nil, err
	}
	g, err := ReadGit(ctx, src)
	if err != nil {
		return nil, err
	}
	return Recognize(Hosts(cfg), g, list, src.Env), nil
}

// ReadSSH reads ~/.ssh/config and the files of its Include lines: ~ expanded, relative paths in ~/.ssh/, patterns
// matched in order. Included files are not followed further.
func ReadSSH(src Source) (Config, error) {
	data, err := src.ReadFile(src.Env.Native("~/.ssh/config"))
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	cfg := Config{Lines: ParseSSH(string(data)), Included: map[int][][]Line{}}
	for i, args := range Includes(cfg.Lines) {
		for _, arg := range args {
			matches, err := src.Glob(src.Env.Native(includePath(arg, src.Env)))
			if err != nil {
				continue
			}
			slices.Sort(matches)
			for _, m := range matches {
				if data, err := src.ReadFile(m); err == nil {
					cfg.Included[i] = append(cfg.Included[i], ParseSSH(string(data)))
				}
			}
		}
	}
	return cfg, nil
}

func includePath(arg string, env paths.Env) string {
	p := env.Expand(arg)
	if !absolute(p) {
		p = env.Expand("~/.ssh/" + p)
	}
	return p
}

func absolute(p string) bool {
	return strings.HasPrefix(p, "/") || len(p) >= 2 && p[1] == ':' && (p[0]|0x20 >= 'a' && p[0]|0x20 <= 'z')
}

// ReadGit reads the global configuration with `git config --global --list -z --show-origin`, so that git finds
// the file wherever it is, and each includeIf file with `git config --file <path> --list -z`. git reads a file
// given this way without following its includes. A non-zero exit, such as for a missing file, reads as empty.
func ReadGit(ctx context.Context, src Source) (Gitconfig, error) {
	var g Gitconfig
	res, err := src.Run(ctx, nil, src.Git, "config", "--global", "--list", "-z", "--show-origin")
	if err != nil {
		return g, err
	}
	if res.ExitCode != 0 {
		return g, nil
	}
	entries := ParseGit(res.Stdout, true)
	collect(entries, &g.Global, &g)
	for _, inc := range IncludeIfs(entries) {
		p := src.Env.Expand(inc.Path)
		if !absolute(p) {
			p = path.Join(path.Dir(paths.Slash(inc.Origin)), p)
		}
		res, err := src.Run(ctx, nil, src.Git, "config", "--file", src.Env.Native(p), "--list", "-z")
		if err != nil {
			return g, err
		}
		if res.ExitCode != 0 {
			continue
		}
		id := Identity{Cond: inc.Cond, Origin: inc.Origin}
		collect(ParseGit(res.Stdout, false), &id, &g)
		g.Included = append(g.Included, id)
	}
	return g, nil
}

type kind int

const (
	none kind = iota
	global
	limited
)

// signature is the commit signature of a candidate and where it applies.
type signature struct {
	name, email string
	kind        kind
	paths       []string
}

type candidate struct {
	acct account.Account
	sig  signature
}

// Recognize turns the hosts into candidates, in the order of the hosts: a host without a key file is left out, as
// is a host of an unknown server that nothing marks as git's (User git, a signature or a rewrite rule). Hosts with
// the same key, platform and a compatible signature become one candidate. See docs/product.md, "导入".
func Recognize(hosts []Host, g Gitconfig, list []account.Account, env paths.Env) []Candidate {
	var cands []candidate
	seen := map[string]bool{}
	for _, h := range hosts {
		if seen[strings.ToLower(h.Name)] {
			continue
		}
		seen[strings.ToLower(h.Name)] = true
		c, ok := recognize(h, g, env)
		if !ok {
			continue
		}
		if i := slices.IndexFunc(cands, func(o candidate) bool { return compatible(o, c, env) }); i >= 0 {
			cands[i] = merge(cands[i], c, env)
			continue
		}
		cands = append(cands, c)
	}
	out := make([]Candidate, len(cands))
	hasGlobal := slices.ContainsFunc(list, func(a account.Account) bool { return a.Scope.Global })
	for i, c := range cands {
		out[i] = Candidate{Account: c.account(), Exists: exists(c.acct, list, env)}
		if out[i].Exists || !out[i].Account.Scope.Global {
			continue
		}
		if hasGlobal {
			out[i].Account.Scope, out[i].Limited = account.Scope{Paths: []string{}}, true
		}
		hasGlobal = true
	}
	return out
}

func recognize(h Host, g Gitconfig, env paths.Env) (candidate, bool) {
	key := h.Options["identityfile"]
	if key == "" || strings.EqualFold(key, "none") {
		return candidate{}, false
	}
	key = keyPath(key, env)
	host := strings.ReplaceAll(h.Options["hostname"], "%h", h.Name)
	if host == "" {
		host = h.Name
	}
	port, _ := strconv.Atoi(h.Options["port"])
	a := account.Account{Label: h.Name, KeyPath: key, URLRewrites: []account.Rewrite{}}
	a.Provider, a.SSHPort = platform(host, port)
	if a.Provider == account.Custom {
		a.HostName = host
	}
	a.SSHAlias = h.Name
	if !validAlias(a) {
		a.SSHAlias = ""
	}
	for _, r := range g.Rules {
		if strings.Contains(r.To, "@"+h.Name+":") || strings.Contains(r.To, "@"+h.Name+"/") {
			a.URLRewrites = append(a.URLRewrites, account.Rewrite{From: r.From, To: r.To, Push: r.Push})
		}
	}
	sig := sign(key, g, env)
	if sig.kind == none && a.Provider != account.Custom && strings.EqualFold(h.Name, a.Provider.CanonicalHost()) &&
		(g.Global.Name != "" || g.Global.Email != "") {
		sig = signature{name: g.Global.Name, email: g.Global.Email, kind: global}
	}
	if a.Provider == account.Custom && sig.kind == none && len(a.URLRewrites) == 0 && h.Options["user"] != "git" {
		return candidate{}, false
	}
	return candidate{acct: a, sig: sig}, true
}

// keyPath is the stored form of an IdentityFile: %d and ~ for the home directory, slashes, ~ again for it.
func keyPath(p string, env paths.Env) string {
	p = strings.ReplaceAll(p, "%d", "~")
	return env.Tilde(env.Expand(p))
}

// platform recognizes the platform and port of a host: a platform's port 443 host, or port 443 where the
// platform serves it, is port 443; port 22 or none is 22; any other port, or a host no platform has, is custom.
func platform(host string, port int) (account.Provider, int) {
	if port == 0 {
		port = 22
	}
	for _, p := range account.Providers {
		if p == account.Custom {
			continue
		}
		on443 := account.Account{Provider: p, SSHPort: 443}.RealHost()
		switch {
		case p.Supports443() && strings.EqualFold(host, on443):
			return p, 443
		case strings.EqualFold(host, p.CanonicalHost()) && (port == 22 || port == 443 && p.Supports443()):
			return p, port
		}
	}
	return account.Custom, port
}

// validAlias reports whether a's alias may be saved: a host named after a platform's or its own real host keeps
// that host's name in its own Host line and comes in without an alias.
func validAlias(a account.Account) bool {
	a.ID = "00000000-0000-4000-8000-000000000000"
	a.Color = account.ColorPresets[0]
	return !slices.ContainsFunc(account.Validate([]account.Account{a}), func(p account.Problem) bool {
		return p.Field == "sshAlias"
	})
}

// sign finds the signature of the git identities whose core.sshCommand uses key: the global one first, then the
// includeIf ones in order. Further identities with the same signature add their gitdir paths.
func sign(key string, g Gitconfig, env paths.Env) signature {
	var sig signature
	found := false
	for _, id := range append([]Identity{g.Global}, g.Included...) {
		if !slices.ContainsFunc(match.IdentityFiles(id.SSHCommand, env.Windows), func(p string) bool {
			return env.Equal(keyPath(p, env), key)
		}) {
			continue
		}
		s := signature{name: id.Name, email: id.Email, kind: global}
		if id.Cond != "" {
			s.kind = limited
			if p := gitdirPath(id, env); p != "" {
				s.paths = []string{p}
			}
		}
		switch {
		case !found:
			sig, found = s, true
		case s.name == sig.name && strings.EqualFold(s.email, sig.email) && s.kind == limited && sig.kind == limited:
			sig.paths = union(sig.paths, s.paths, env)
		}
	}
	return sig
}

// gitdirPath is the stored path of a gitdir or gitdir/i condition, or "" for any other condition.
func gitdirPath(id Identity, env paths.Env) string {
	var p string
	switch {
	case strings.HasPrefix(id.Cond, "gitdir:"):
		p = id.Cond[len("gitdir:"):]
	case strings.HasPrefix(id.Cond, "gitdir/i:"):
		p = id.Cond[len("gitdir/i:"):]
	default:
		return ""
	}
	p = strings.TrimSuffix(p, "**")
	if strings.HasPrefix(p, "./") {
		p = path.Join(path.Dir(paths.Slash(id.Origin)), p)
	}
	return gitdir.Normalize(p, env)
}

func union(a, b []string, env paths.Env) []string {
	out := slices.Clone(a)
	for _, p := range b {
		if !slices.ContainsFunc(out, func(q string) bool { return env.Equal(p, q) }) {
			out = append(out, p)
		}
	}
	return out
}

// compatible reports whether two candidates are one identity: the same key on the same platform and server, and
// signatures that agree where both are set and apply in the same way.
func compatible(a, b candidate, env paths.Env) bool {
	same := func(x, y string, eq func(string, string) bool) bool { return x == "" || y == "" || eq(x, y) }
	return env.Equal(a.acct.KeyPath, b.acct.KeyPath) && a.acct.Provider == b.acct.Provider &&
		strings.EqualFold(a.acct.HostName, b.acct.HostName) &&
		(a.acct.Provider != account.Custom || a.acct.SSHPort == b.acct.SSHPort) &&
		same(a.sig.name, b.sig.name, func(x, y string) bool { return x == y }) &&
		same(a.sig.email, b.sig.email, strings.EqualFold) &&
		(a.sig.kind == none || b.sig.kind == none || a.sig.kind == b.sig.kind)
}

// merge adds b to a: a keeps its own values and takes from b what it lacks.
func merge(a, b candidate, env paths.Env) candidate {
	if a.acct.SSHAlias == "" {
		a.acct.SSHAlias = b.acct.SSHAlias
	}
	a.sig.name = cmp.Or(a.sig.name, b.sig.name)
	a.sig.email = cmp.Or(a.sig.email, b.sig.email)
	if a.sig.kind == none {
		a.sig.kind = b.sig.kind
	}
	a.sig.paths = union(a.sig.paths, b.sig.paths, env)
	for _, r := range b.acct.URLRewrites {
		if !slices.Contains(a.acct.URLRewrites, r) {
			a.acct.URLRewrites = append(a.acct.URLRewrites, r)
		}
	}
	return a
}

func (c candidate) account() account.Account {
	a := c.acct
	a.GitUserName, a.GitUserEmail = c.sig.name, c.sig.email
	switch c.sig.kind {
	case global:
		a.Scope = account.Scope{Global: true}
	default:
		a.Scope = account.Scope{Paths: append([]string{}, c.sig.paths...)}
	}
	return a
}

// exists reports whether Idset already has the identity: one with the same alias, or, for a candidate without an
// alias, one with the same key on the same platform.
func exists(a account.Account, list []account.Account, env paths.Env) bool {
	return slices.ContainsFunc(list, func(o account.Account) bool {
		if a.SSHAlias != "" {
			return strings.EqualFold(o.SSHAlias, a.SSHAlias)
		}
		return o.Provider == a.Provider && env.Equal(env.Tilde(env.Expand(o.KeyPath)), a.KeyPath)
	})
}
