package diagnose

import (
	"slices"
	"strings"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/include"
	"github.com/yangflow/idset/internal/match"
	"github.com/yangflow/idset/internal/remote"
)

// Problem is the problem bar under an identity: what went wrong, in the order it is looked for, each with its fixed
// next step.
type Problem int

const (
	// MissingCommand: ssh is not installed or not found. Next step: install OpenSSH.
	MissingCommand Problem = iota + 1
	// MissingKey: the identity names a key file that does not exist. Next step: generate a key.
	MissingKey
	// LockedKey: the agent runs, the key has a passphrase and the agent does not hold it. Next step: unlock with the
	// keychain on macOS, the ssh-add command elsewhere.
	LockedKey
	// Unreachable: the host does not resolve, refuses, times out or has another host key. No next step.
	Unreachable
	// Denied: any other failure. Next step: copy the public key and open the platform's SSH settings.
	Denied
	// HTTPSRemote: the check passed, the identity is the current match, and its origin is https that no rewrite of
	// the identity turns into SSH. Next step: add the platform's default rewrites.
	HTTPSRemote
)

// ProblemOf is the problem bar for a check, if any. matched says whether the identity is the one the current
// result card matched, and origin is that card's remote URL.
func ProblemOf(a account.Account, c Check, matched bool, origin string) (Problem, bool) {
	if c.OK {
		if matched && remote.IsHTTPS(origin) && !rewritten(a, origin) {
			return HTTPSRemote, true
		}
		return 0, false
	}
	switch {
	case c.Failure == NoCommand:
		return MissingCommand, true
	case c.Key.HasPath && !c.Key.Exists:
		return MissingKey, true
	case c.Key.Agent && c.Key.Passphrase && c.Key.Fingerprint != "" && !c.Key.InAgent:
		return LockedKey, true
	case c.Failure == NoHost || c.Failure == Refused || c.Failure == TimedOut || c.Failure == HostKey:
		return Unreachable, true
	}
	return Denied, true
}

// Retryable reports whether the bar has "Check again": every problem but an https remote, which no check changes.
func (p Problem) Retryable() bool { return p != HTTPSRemote }

// rewritten reports whether one of the identity's own rewrites, fetch or push, matches the start of origin: git
// then already talks to the alias.
func rewritten(a account.Account, origin string) bool {
	o := strings.ToLower(strings.TrimSpace(origin))
	for _, r := range a.URLRewrites {
		if from := strings.ToLower(strings.TrimSpace(r.From)); from != "" && strings.HasPrefix(o, from) {
			return true
		}
	}
	return false
}

// Code names an entry of the diagnostics summary.
type Code string

const (
	// NET001: an identity failed its check.
	NET001 Code = "NET001"
	// SSH001: the user's own ~/.ssh/config has a Host with the name of an Idset alias.
	SSH001 Code = "SSH001"
	// GIT001: the identity check of the current match found something.
	GIT001 Code = "GIT001"
	// GIT002: accounts.json has more than one global identity.
	GIT002 Code = "GIT002"
)

// Entry is one line of the summary. ID is the identity it is about: the failed one, the one whose alias is
// shadowed, the match, or for GIT002 the global identity in effect. Alias is the alias for NET001 and SSH001.
// Regenerate is set for GIT001 on a scoped identity, whose fix is to write the managed files again.
type Entry struct {
	Code       Code
	Severity   match.Severity
	ID         string
	Alias      string
	Regenerate bool
}

// Input is what the summary is made of. Checks are the current checks; UserHosts the Host names of the user's own
// ~/.ssh/config; Matched the identity of the current result card, if any, with its identity check findings.
type Input struct {
	List      []account.Account
	Checks    []Check
	UserHosts []string
	Matched   string
	Findings  []match.Finding
}

// Summarize lists the entries of the summary, errors first, then in the order of the codes and the list.
func Summarize(in Input) []Entry {
	byID := map[string]account.Account{}
	for _, a := range in.List {
		byID[a.ID] = a
	}
	var out []Entry
	for _, c := range in.Checks {
		if a, ok := byID[c.ID]; ok && !c.OK {
			out = append(out, Entry{Code: NET001, Severity: match.Error, ID: c.ID, Alias: a.SSHAlias})
		}
	}
	var globals []account.Account
	for _, a := range in.List {
		if a.Scope.Global {
			globals = append(globals, a)
		}
	}
	if len(globals) > 1 {
		out = append(out, Entry{Code: GIT002, Severity: match.Error, ID: globals[len(globals)-1].ID})
	}
	if a, ok := byID[in.Matched]; ok && len(in.Findings) > 0 {
		out = append(out, Entry{Code: GIT001, Severity: match.Worst(in.Findings), ID: a.ID, Regenerate: !a.Scope.Global})
	}
	for _, a := range in.List {
		if !Checkable(a) {
			continue
		}
		for _, h := range in.UserHosts {
			if strings.EqualFold(h, a.SSHAlias) {
				out = append(out, Entry{Code: SSH001, Severity: match.Warning, ID: a.ID, Alias: a.SSHAlias})
				break
			}
		}
	}
	slices.SortStableFunc(out, func(a, b Entry) int { return int(b.Severity) - int(a.Severity) })
	return out
}

// Worst is the highest severity of the summary, or 0 when it is empty: the tray icon shows it.
func Worst(entries []Entry) match.Severity {
	var worst match.Severity
	for _, e := range entries {
		worst = max(worst, e.Severity)
	}
	return worst
}

// UserHosts lists the exact host names of the Host lines in the user's ~/.ssh/config outside Idset's marker
// block. Patterns with wildcards and negations match more than one name and are left out.
func UserHosts(config string) []string {
	var out []string
	inBlock := false
	for _, line := range strings.Split(config, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == include.Begin:
			inBlock = true
			continue
		case line == include.End:
			inBlock = false
			continue
		case inBlock || line == "" || strings.HasPrefix(line, "#"):
			continue
		}
		key, rest := cutKeyword(line)
		if !strings.EqualFold(key, "Host") {
			continue
		}
		for _, name := range strings.Fields(rest) {
			name = strings.Trim(name, `"`)
			if name != "" && !strings.ContainsAny(name, "*?!") {
				out = append(out, name)
			}
		}
	}
	return out
}

// cutKeyword splits an ssh_config line into its keyword and arguments; "=" may stand between them.
func cutKeyword(line string) (string, string) {
	i := strings.IndexAny(line, " \t=")
	if i < 0 {
		return line, ""
	}
	rest := strings.TrimLeft(line[i:], " \t")
	rest = strings.TrimPrefix(rest, "=")
	return line[:i], strings.TrimSpace(rest)
}
