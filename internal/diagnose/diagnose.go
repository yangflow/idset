// Package diagnose checks that each identity can reach its platform over SSH, the way `git push` does, and says
// what stands in the way when it cannot: the connectivity check, the state of the key, the problem bar of each
// identity and the summary of the panel. The system commands are injectable; tests use a fake.
package diagnose

import (
	"context"
	"errors"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/sysexec"
)

// Run runs a system command with extra environment variables. A non-zero exit is in the result, not the error.
type Run func(ctx context.Context, env []string, name string, args ...string) (sysexec.Result, error)

// System runs the commands of the machine, each limited to timeout.
func System(timeout time.Duration) Run {
	return func(ctx context.Context, env []string, name string, args ...string) (sysexec.Result, error) {
		return sysexec.Run(ctx, timeout, env, name, args...)
	}
}

// Tools are the ssh programs to run. They sit in one directory, so that ssh-add talks to the agent of the ssh
// that connects.
type Tools struct{ SSH, Add, Keygen string }

// FindTools looks for ssh: on Windows the program of git's core.sshCommand first, else the first on PATH; on
// macOS PATH, then the places Homebrew and the system install it, since an app started from Finder has a short
// PATH. ssh-add and ssh-keygen are taken from the same directory. When nothing is found, the bare names are kept and
// running them reports the missing command.
func FindTools(goos, sshCommand string, lookPath func(string) (string, error), exists func(string) bool) Tools {
	exe := ""
	if goos == "windows" {
		exe = ".exe"
	}
	bare := Tools{SSH: "ssh" + exe, Add: "ssh-add" + exe, Keygen: "ssh-keygen" + exe}
	in := func(dir string) Tools {
		return Tools{SSH: path.Join(dir, bare.SSH), Add: path.Join(dir, bare.Add), Keygen: path.Join(dir, bare.Keygen)}
	}
	if goos == "windows" {
		if words := strings.Fields(sshCommand); len(words) > 0 {
			prog := paths.Slash(strings.Trim(words[0], `"'`))
			if strings.EqualFold(path.Base(strings.TrimSuffix(strings.ToLower(prog), ".exe")), "ssh") && exists(prog) {
				return in(path.Dir(prog))
			}
		}
	}
	if found, err := lookPath(bare.SSH); err == nil {
		return in(path.Dir(paths.Slash(found)))
	}
	if goos == "darwin" {
		for _, dir := range []string{"/opt/homebrew/bin", "/usr/local/bin", "/usr/bin"} {
			if exists(path.Join(dir, bare.SSH)) {
				return in(dir)
			}
		}
	}
	return bare
}

// Failure is why a connectivity check failed.
type Failure int

const (
	// Rejected: the platform refused the key.
	Rejected Failure = iota + 1
	// NoHost: the host name of the alias does not resolve.
	NoHost
	// Refused: the host refused the connection.
	Refused
	// TimedOut: the connection or the command ran out of time.
	TimedOut
	// HostKey: the host key does not match known_hosts.
	HostKey
	// NoKeyFile: ssh says the key file does not exist.
	NoKeyFile
	// NoCommand: ssh is not installed or not found.
	NoCommand
	// Other: none of the above; Detail has the first line of the output.
	Other
)

// Check is the outcome of `ssh -T` for one identity. User is the account name the platform greeted, empty for
// custom. At is when the check finished. Key is the state of the key, gathered only after a failure.
type Check struct {
	ID      string
	OK      bool
	User    string
	Failure Failure
	Detail  string
	At      time.Time
	Key     KeyState
}

// Checkable reports whether an identity has an alias, the only identities the check runs for.
func Checkable(a account.Account) bool { return strings.TrimSpace(a.SSHAlias) != "" }

// Connect runs `ssh -T` to the alias of a, through the user's whole SSH configuration, without prompts.
func Connect(ctx context.Context, run Run, tools Tools, a account.Account) Check {
	c := Check{ID: a.ID}
	res, err := run(ctx, nil, tools.SSH, "-T", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", "git@"+a.SSHAlias)
	switch {
	case errors.Is(err, sysexec.ErrTimeout):
		c.Failure = TimedOut
		return c
	case errors.Is(err, exec.ErrNotFound):
		c.Failure, c.Detail = NoCommand, path.Base(paths.Slash(tools.SSH))
		return c
	case err != nil:
		c.Failure, c.Detail = Other, firstLine(err.Error())
		return c
	}
	out := res.Stdout + "\n" + res.Stderr
	if a.Provider != account.Custom {
		if user, ok := greeting(a.Provider, out); ok {
			c.OK, c.User = true, user
			return c
		}
	}
	if f, ok := classify(out); ok {
		c.Failure = f
		return c
	}
	if a.Provider == account.Custom {
		c.OK = true
		return c
	}
	c.Failure, c.Detail = Other, firstLine(out)
	return c
}

var (
	greetings = map[account.Provider]*regexp.Regexp{
		account.GitHub: regexp.MustCompile(`(?m)^Hi ([^\s!]+)!`),
		account.GitLab: regexp.MustCompile(`(?m)^Welcome to GitLab, @([^\s!]+)!`),
		account.Gitea:  regexp.MustCompile(`(?m)^Hi there, ([^\s!]+)!`),
	}
	username = regexp.MustCompile(`^[A-Za-z0-9._-]{1,39}$`)
)

// greeting finds the platform's banner and the user name in it. A name that a platform would not allow is not
// taken as success, so that a forged banner does not pass.
func greeting(p account.Provider, out string) (string, bool) {
	re := greetings[p]
	if re == nil {
		return "", false
	}
	m := re.FindStringSubmatch(out)
	if m == nil || !username.MatchString(m[1]) {
		return "", false
	}
	return m[1], true
}

var failures = []struct {
	text    string
	failure Failure
}{
	{"permission denied", Rejected},
	{"could not resolve hostname", NoHost},
	{"connection refused", Refused},
	{"timed out", TimedOut},
	{"host key verification failed", HostKey},
	{"no such identity", NoKeyFile},
	{"no such file", NoKeyFile},
}

// classify sorts the output of a failed check, ignoring case.
func classify(out string) (Failure, bool) {
	l := strings.ToLower(out)
	for _, f := range failures {
		if strings.Contains(l, f.text) {
			return f.failure, true
		}
	}
	return 0, false
}

// firstLine is the first line that is not blank, at most 80 characters.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			if r := []rune(line); len(r) > 80 {
				return string(r[:80])
			}
			return line
		}
	}
	return ""
}

// CacheTTL is how long a failed check stands before it runs again on its own.
const CacheTTL = 10 * time.Minute

// Cache keeps the checks in memory. A success stands until the identity changes or a check is forced; a failure
// stands for CacheTTL. It is used from one goroutine.
type Cache struct {
	checks map[string]cached
}

type cached struct {
	sig   string
	check Check
}

// signature is what a check depends on: an edit to any of it makes the old check worthless.
func signature(a account.Account) string {
	return strings.Join([]string{string(a.Provider), a.SSHAlias, a.HostName, strconv.Itoa(a.SSHPort), a.KeyPath,
		a.KeyFingerprint}, "\x00")
}

// Get is the check kept for a, if it is still about the same identity.
func (c *Cache) Get(a account.Account) (Check, bool) {
	e, ok := c.checks[a.ID]
	if !ok || e.sig != signature(a) {
		return Check{}, false
	}
	return e.check, true
}

// Due lists the identities to check at now: those with an alias, no check, a changed identity, or a failure
// older than CacheTTL. With force set, every identity with an alias is due.
func (c *Cache) Due(list []account.Account, now time.Time, force bool) []account.Account {
	var due []account.Account
	for _, a := range list {
		if !Checkable(a) {
			continue
		}
		check, ok := c.Get(a)
		if force || !ok || (!check.OK && now.Sub(check.At) >= CacheTTL) {
			due = append(due, a)
		}
	}
	return due
}

// Put keeps checks of identities in list and drops those of identities that are gone or lost their alias.
func (c *Cache) Put(list []account.Account, checks []Check) {
	if c.checks == nil {
		c.checks = map[string]cached{}
	}
	byID := map[string]account.Account{}
	for _, a := range list {
		if Checkable(a) {
			byID[a.ID] = a
		}
	}
	for id := range c.checks {
		if _, ok := byID[id]; !ok {
			delete(c.checks, id)
		}
	}
	for _, check := range checks {
		if a, ok := byID[check.ID]; ok {
			c.checks[check.ID] = cached{sig: signature(a), check: check}
		}
	}
}

// Checks are the kept checks of the identities in list that are still current, in list order.
func (c *Cache) Checks(list []account.Account) []Check {
	var out []Check
	for _, a := range list {
		if check, ok := c.Get(a); ok && Checkable(a) {
			out = append(out, check)
		}
	}
	return out
}

// Runner checks identities with the system's commands.
type Runner struct {
	Run   Run
	Tools Tools
	Env   paths.Env
	Now   func() time.Time
}

// CheckAll checks the identities at once, gathers the key state of those that failed, and returns the checks in
// the order of list.
func (r Runner) CheckAll(ctx context.Context, list []account.Account) []Check {
	out := make([]Check, len(list))
	var wg sync.WaitGroup
	for i, a := range list {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := Connect(ctx, r.Run, r.Tools, a)
			if !c.OK && c.Failure != NoCommand {
				c.Key = InspectKey(ctx, r.Run, r.Tools, r.Env, a)
			}
			c.At = r.Now()
			out[i] = c
		}()
	}
	wg.Wait()
	return out
}
