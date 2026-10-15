// Package remote turns repository addresses into the ones that go through an identity's SSH alias: the command that
// changes a repository's origin, and the clone command. It is pure: no disk, no git.
package remote

import (
	"regexp"
	"strings"

	"github.com/yangflow/idset/internal/account"
)

// RemoteCommand is `git remote set-url origin <SSH address>` for a repository whose origin is an http(s) address.
// It reports false when the origin is not http(s), the identity has no alias, no rewrite rule of the identity matches
// the origin, or the address has no repository path. Unlike the clone helper it never falls back to the bare path: an
// origin on another host belongs to another server, and pointing it at this alias would push there.
func RemoteCommand(a account.Account, origin string) (string, bool) {
	origin = strings.TrimSpace(origin)
	if !IsHTTPS(origin) {
		return "", false
	}
	addr, ok := address(a, origin, false)
	if !ok {
		return "", false
	}
	return "git remote set-url origin " + addr, true
}

// IsHTTPS reports whether an address starts with https:// or http://, in either case.
func IsHTTPS(origin string) bool {
	l := strings.ToLower(strings.TrimSpace(origin))
	return strings.HasPrefix(l, "https://") || strings.HasPrefix(l, "http://")
}

// CloneCommand is `git clone <SSH address>` for an owner/repo shorthand, an address or a whole git clone command.
// It reports false when the identity has no alias or the input cannot be read.
func CloneCommand(a account.Account, input string) (string, bool) {
	input = strings.TrimSpace(input)
	if a.SSHAlias == "" || input == "" {
		return "", false
	}
	if fields := strings.Fields(input); len(fields) >= 3 && fields[0] == "git" && fields[1] == "clone" {
		input = fields[len(fields)-1]
	}
	if shorthand.MatchString(input) {
		return "git clone " + withGit("git@"+a.SSHAlias+":"+input), true
	}
	if p, ok := strings.CutPrefix(input, "git@"+a.SSHAlias+":"); ok && strings.Trim(p, "/") != "" {
		return "git clone " + input, true
	}
	addr, ok := address(a, input, true)
	if !ok {
		return "", false
	}
	return "git clone " + addr, true
}

// Repo is the owner/repo path of an address, without ".git", or "" when there is none. It is what the clone helper
// starts with.
func Repo(addr string) string {
	word := lastWord(addr)
	if shorthand.MatchString(word) {
		return strings.TrimSuffix(word, ".git")
	}
	return strings.TrimSuffix(pathOf(word), ".git")
}

// Prefix is the owner part of what a clone command or address names, with its slash ("acme/" or "group/sub/"), or
// "" when it names no owner.
func Prefix(text string) string {
	p := Repo(text)
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[:i+1]
	}
	return ""
}

var shorthand = regexp.MustCompile(`^[\w.-]+(/[\w.-]+)+$`)

func lastWord(s string) string {
	f := strings.Fields(s)
	if len(f) == 0 {
		return ""
	}
	return f[len(f)-1]
}

// withGit makes the address end in ".git" after dropping trailing slashes.
func withGit(addr string) string {
	addr = strings.TrimRight(addr, "/")
	if !strings.HasSuffix(addr, ".git") {
		addr += ".git"
	}
	return addr
}

type rule struct{ from, to string }

// rules are the identity's own rewrites, then the platform's defaults for its hosts. Duplicates add nothing.
func rules(a account.Account) []rule {
	var out []rule
	for _, r := range a.URLRewrites {
		out = append(out, rule{r.From, r.To})
	}
	to := "git@" + a.SSHAlias + ":"
	var hosts []string
	for _, h := range []string{a.Provider.CanonicalHost(), a.RealHost(), a.HostName} {
		if h != "" && !contains(hosts, h) {
			hosts = append(hosts, h)
		}
	}
	for _, h := range hosts {
		for _, from := range []string{"https://" + h + "/", "http://" + h + "/", "git@" + h + ":", "ssh://git@" + h + "/"} {
			if !containsRule(out, rule{from, to}) {
				out = append(out, rule{from, to})
			}
		}
	}
	return out
}

func contains(l []string, s string) bool {
	for _, v := range l {
		if v == s {
			return true
		}
	}
	return false
}

func containsRule(l []rule, r rule) bool {
	for _, v := range l {
		if v == r {
			return true
		}
	}
	return false
}

// address rewrites addr with the longest rule that matches its start, else, with fallback set, rebuilds it from its
// path.
func address(a account.Account, addr string, fallback bool) (string, bool) {
	if a.SSHAlias == "" {
		return "", false
	}
	addr = stripUser(addr)
	best := -1
	var rs = rules(a)
	for i, r := range rs {
		if r.from != "" && strings.HasPrefix(strings.ToLower(addr), strings.ToLower(r.from)) &&
			(best < 0 || len(r.from) > len(rs[best].from)) {
			best = i
		}
	}
	if best >= 0 {
		rest := addr[len(rs[best].from):]
		out := withGit(rs[best].to + rest)
		if strings.Trim(rest, "/") == "" || pathOf(out) == "" {
			return "", false
		}
		return out, true
	}
	if !fallback {
		return "", false
	}
	p := pathOf(addr)
	if p == "" {
		return "", false
	}
	return withGit("git@" + a.SSHAlias + ":" + p), true
}

// stripUser drops "user@" or "user:password@" from an http(s) address, which no rule names.
func stripUser(addr string) string {
	for _, scheme := range []string{"https://", "http://"} {
		if len(addr) >= len(scheme) && strings.EqualFold(addr[:len(scheme)], scheme) {
			rest := addr[len(scheme):]
			host, path, _ := strings.Cut(rest, "/")
			if i := strings.LastIndex(host, "@"); i >= 0 {
				host = host[i+1:]
			}
			if path == "" && !strings.Contains(rest, "/") {
				return addr[:len(scheme)] + host
			}
			return addr[:len(scheme)] + host + "/" + path
		}
	}
	return addr
}

// pathOf is the repository path of a URL or user@host:path address: at least two segments, no query, no
// surrounding slashes. "" when there is none.
func pathOf(addr string) string {
	var p string
	if _, rest, ok := strings.Cut(addr, "://"); ok {
		_, p, ok = strings.Cut(rest, "/")
		if !ok {
			return ""
		}
		p, _, _ = strings.Cut(p, "?")
		p, _, _ = strings.Cut(p, "#")
	} else {
		at := strings.Index(addr, "@")
		colon := strings.Index(addr, ":")
		if at < 0 || colon < at {
			return ""
		}
		p = addr[colon+1:]
	}
	p = strings.Trim(p, "/")
	if strings.Count(p, "/") < 1 || strings.Contains(p, "//") || strings.ContainsAny(p, " \t") {
		return ""
	}
	return p
}
