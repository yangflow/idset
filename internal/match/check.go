package match

import (
	"context"
	"path"
	"strings"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/paths"
)

// FindingKind is what the identity check found wrong in a repository.
type FindingKind int

const (
	// NoEmail: the repository has no user.email.
	NoEmail FindingKind = iota + 1
	// OtherEmail: its user.email belongs to another identity, named by Owner.
	OtherEmail
	// UnknownEmail: its user.email belongs to no identity.
	UnknownEmail
	// OtherKey: core.sshCommand picks a key other than the identity's.
	OtherKey
)

// Severity ranks a finding: a warning may be fine, an error means the wrong identity signs or pushes.
type Severity int

const (
	Warning Severity = iota + 1
	Error
)

// Finding is one disagreement between the repository's own configuration and the matched identity. Value is the
// email or key path as the repository has it, Owner the ID of the identity that email belongs to.
type Finding struct {
	Kind  FindingKind
	Value string
	Owner string
}

// Severity is Error for the kinds that make another identity act, Warning otherwise.
func (f Finding) Severity() Severity {
	if f.Kind == OtherEmail || f.Kind == OtherKey {
		return Error
	}
	return Warning
}

// Worst is the highest severity among findings, or 0 for none.
func Worst(findings []Finding) Severity {
	var worst Severity
	for _, f := range findings {
		worst = max(worst, f.Severity())
	}
	return worst
}

// Check reads the effective user.email and core.sshCommand of the repository at root and compares them with the
// identity a. A value that git cannot be asked for gives no finding: the check never turns a match into a failure.
func Check(ctx context.Context, git Git, root string, a account.Account, list []account.Account, env paths.Env) []Finding {
	var email, command string
	var emailOK, hasEmail, hasCommand bool
	if strings.TrimSpace(a.GitUserEmail) != "" {
		out, code, err := git(ctx, root, "config", "--get", "user.email")
		emailOK = err == nil && (code == 0 || code == 1)
		email, hasEmail = out, code == 0
	}
	if strings.TrimSpace(a.KeyPath) != "" {
		out, code, err := git(ctx, root, "config", "--get", "core.sshCommand")
		command, hasCommand = out, err == nil && code == 0
	}
	var findings []Finding
	if emailOK {
		if f, bad := compareEmail(a, list, email, hasEmail); bad {
			findings = append(findings, f)
		}
	}
	if hasCommand {
		if f, bad := compareKey(a, command, env); bad {
			findings = append(findings, f)
		}
	}
	return findings
}

// sameEmail compares addresses ignoring case and surrounding white space.
func sameEmail(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func compareEmail(a account.Account, list []account.Account, email string, set bool) (Finding, bool) {
	email = strings.TrimSpace(email)
	switch {
	case !set || email == "":
		return Finding{Kind: NoEmail}, true
	case sameEmail(email, a.GitUserEmail):
		return Finding{}, false
	}
	for _, other := range list {
		if other.ID != a.ID && sameEmail(email, other.GitUserEmail) {
			return Finding{Kind: OtherEmail, Value: email, Owner: other.ID}, true
		}
	}
	return Finding{Kind: UnknownEmail, Value: email}, true
}

func compareKey(a account.Account, command string, env paths.Env) (Finding, bool) {
	keys := IdentityFiles(command, env.Windows)
	if len(keys) == 0 {
		return Finding{}, false
	}
	for _, k := range keys {
		if samePath(k, a.KeyPath, env) {
			return Finding{}, false
		}
	}
	return Finding{Kind: OtherKey, Value: keys[0]}, true
}

// samePath compares two key paths as text: it never touches the files. ~ and the home directory are the same, and
// on Windows case and the kind of slash do not matter.
func samePath(a, b string, env paths.Env) bool {
	norm := func(p string) string { return env.Tilde(path.Clean(env.Expand(strings.TrimSpace(p)))) }
	return env.Equal(norm(a), norm(b))
}

// IdentityFiles lists the key files a core.sshCommand names, in order: "-i path", "-ipath", and
// "-o IdentityFile=path" (also "-oIdentityFile=path" and with a space for "="). git hands the command to a shell;
// it is read here with the shell's quoting but without running anything. With windows set, a backslash outside
// single quotes only escapes a quote or a space, so that "C:\Users\ada\.ssh\id" stays one path.
func IdentityFiles(command string, windows bool) []string {
	words := splitWords(command, windows)
	var out []string
	for i := 0; i < len(words); i++ {
		w := words[i]
		next := func() (string, bool) {
			if i+1 < len(words) {
				i++
				return words[i], true
			}
			return "", false
		}
		switch {
		case w == "-i":
			if v, ok := next(); ok {
				out = append(out, v)
			}
		case w == "-o":
			if v, ok := next(); ok {
				if p, ok := identityOption(v); ok {
					out = append(out, p)
				}
			}
		case strings.HasPrefix(w, "-o"):
			if p, ok := identityOption(w[2:]); ok {
				out = append(out, p)
			}
		case strings.HasPrefix(w, "-i") && len(w) > 2:
			out = append(out, w[2:])
		}
	}
	return out
}

// identityOption reads the path of an "IdentityFile=path" or "IdentityFile path" option.
func identityOption(opt string) (string, bool) {
	key, value, found := strings.Cut(opt, "=")
	if !found {
		key, value, found = strings.Cut(opt, " ")
	}
	value = strings.TrimSpace(value)
	if !found || !strings.EqualFold(strings.TrimSpace(key), "IdentityFile") || value == "" {
		return "", false
	}
	return strings.Trim(value, `"'`), true
}

// splitWords cuts a command into words the way a POSIX shell does: white space separates, single quotes keep
// everything, double quotes keep everything but a backslash before a quote, a backslash or $ ` , and a backslash
// outside quotes escapes the next character.
func splitWords(s string, windows bool) []string {
	var (
		words []string
		cur   strings.Builder
		in    bool
		quote rune
	)
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case quote == '\'':
			if r == '\'' {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\\' && i+1 < len(rs) && escapes(rs[i+1], quote, windows):
			i++
			cur.WriteRune(rs[i])
			in = true
		case quote == '"':
			if r == '"' {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote, in = r, true
		case r == ' ' || r == '\t' || r == '\n':
			if in {
				words = append(words, cur.String())
				cur.Reset()
				in = false
			}
		default:
			cur.WriteRune(r)
			in = true
		}
	}
	if in {
		words = append(words, cur.String())
	}
	return words
}

// escapes says whether a backslash before next is an escape in the given quoting.
func escapes(next, quote rune, windows bool) bool {
	switch {
	case windows:
		return next == '"' || (quote == 0 && (next == ' ' || next == '\''))
	case quote == '"':
		return strings.ContainsRune("\"\\$`", next)
	}
	return true
}
