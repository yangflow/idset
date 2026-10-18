package diagnose

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/sysexec"
)

const (
	pubKey      = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGVvBRLiy1IB3trYVbYfC8KOIqOu/SBSTpKehjxdeEqt ada@example.com"
	fingerprint = "SHA256:GaZxSsN21z2W8DOaH9fsWhWZ2d8xBcx1tRPgPmuE4qc"
)

var tools = Tools{SSH: "/usr/bin/ssh", Add: "/usr/bin/ssh-add", Keygen: "/usr/bin/ssh-keygen"}

// call is one command a fake Run was asked to run.
type call struct {
	env  []string
	name string
	args []string
}

// fake answers each program with a fixed result and records what it ran.
type fake struct {
	mu      sync.Mutex
	calls   []call
	answers map[string]func(args []string) (sysexec.Result, error)
}

func (f *fake) run(_ context.Context, env []string, name string, args ...string) (sysexec.Result, error) {
	f.mu.Lock()
	f.calls = append(f.calls, call{env, name, args})
	f.mu.Unlock()
	if answer := f.answers[filepath.Base(name)]; answer != nil {
		return answer(args)
	}
	return sysexec.Result{}, fmt.Errorf("unexpected %s", name)
}

func (f *fake) ran(name string) bool {
	for _, c := range f.calls {
		if filepath.Base(c.name) == name {
			return true
		}
	}
	return false
}

func answer(stdout, stderr string, code int) func([]string) (sysexec.Result, error) {
	return func([]string) (sysexec.Result, error) {
		return sysexec.Result{Stdout: stdout, Stderr: stderr, ExitCode: code}, nil
	}
}

func identity(id string, p account.Provider) account.Account {
	return account.Account{ID: id, Provider: p, SSHAlias: "gh-" + id, SSHPort: 22, KeyPath: "~/.ssh/id_" + id}
}

func TestFindTools(t *testing.T) {
	none := func(string) (string, error) { return "", exec.ErrNotFound }
	onPath := func(string) (string, error) { return "/usr/local/bin/ssh", nil }
	exists := func(set ...string) func(string) bool {
		return func(p string) bool {
			for _, s := range set {
				if s == p {
					return true
				}
			}
			return false
		}
	}
	for _, c := range []struct {
		name, goos, command string
		look                func(string) (string, error)
		exists              func(string) bool
		want                Tools
	}{
		{"path", "linux", "", onPath, exists(),
			Tools{"/usr/local/bin/ssh", "/usr/local/bin/ssh-add", "/usr/local/bin/ssh-keygen"}},
		{"homebrew after path", "darwin", "", none, exists("/opt/homebrew/bin/ssh"),
			Tools{"/opt/homebrew/bin/ssh", "/opt/homebrew/bin/ssh-add", "/opt/homebrew/bin/ssh-keygen"}},
		{"system on macOS", "darwin", "", none, exists("/usr/bin/ssh"),
			Tools{"/usr/bin/ssh", "/usr/bin/ssh-add", "/usr/bin/ssh-keygen"}},
		{"missing", "linux", "", none, exists(), Tools{"ssh", "ssh-add", "ssh-keygen"}},
		{"git's ssh on Windows", "windows", `"C:\Windows\System32\OpenSSH\ssh.exe" -o IdentitiesOnly=yes`, onPath,
			exists("C:/Windows/System32/OpenSSH/ssh.exe"),
			Tools{"C:/Windows/System32/OpenSSH/ssh.exe", "C:/Windows/System32/OpenSSH/ssh-add.exe",
				"C:/Windows/System32/OpenSSH/ssh-keygen.exe"}},
		{"plink is not ssh", "windows", `C:/PuTTY/plink.exe`, func(string) (string, error) {
			return `C:\Program Files\Git\usr\bin\ssh.exe`, nil
		}, exists("C:/PuTTY/plink.exe"),
			Tools{"C:/Program Files/Git/usr/bin/ssh.exe", "C:/Program Files/Git/usr/bin/ssh-add.exe",
				"C:/Program Files/Git/usr/bin/ssh-keygen.exe"}},
		{"missing on Windows", "windows", "", none, exists(), Tools{"ssh.exe", "ssh-add.exe", "ssh-keygen.exe"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := FindTools(c.goos, c.command, c.look, c.exists); got != c.want {
				t.Errorf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestConnectArguments(t *testing.T) {
	f := &fake{answers: map[string]func([]string) (sysexec.Result, error){"ssh": answer("", "Hi ada!", 1)}}
	Connect(context.Background(), f.run, tools, identity("work", account.GitHub))
	want := call{nil, "/usr/bin/ssh", []string{"-T", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", "git@gh-work"}}
	if len(f.calls) != 1 || !reflect.DeepEqual(f.calls[0], want) {
		t.Errorf("ran %+v, want %+v", f.calls, want)
	}
}

func TestConnect(t *testing.T) {
	long := strings.Repeat("x", 100)
	for _, c := range []struct {
		name     string
		provider account.Provider
		stdout   string
		stderr   string
		code     int
		err      error
		want     Check
	}{
		{"github", account.GitHub, "",
			"Hi ada-lovelace! You've successfully authenticated, but GitHub does not provide shell access.\n", 1, nil,
			Check{OK: true, User: "ada-lovelace"}},
		{"gitlab", account.GitLab, "Welcome to GitLab, @ada.l!\n", "", 0, nil, Check{OK: true, User: "ada.l"}},
		{"gitea", account.Gitea, "Hi there, ada_l! You've successfully authenticated with the key named work\n", "", 1,
			nil, Check{OK: true, User: "ada_l"}},
		{"banner after a warning", account.GitHub, "",
			"Warning: Permanently added 'github.com' to the list of known hosts.\nHi ada!", 1, nil,
			Check{OK: true, User: "ada"}},
		{"forged name", account.GitHub, "", "Hi " + strings.Repeat("a", 40) + "!", 1, nil,
			Check{Failure: Other, Detail: "Hi " + strings.Repeat("a", 40) + "!"}},
		{"gitea banner is not github's", account.GitHub, "Hi there, ada!", "", 1, nil,
			Check{Failure: Other, Detail: "Hi there, ada!"}},
		{"denied", account.GitHub, "", "git@github.com: Permission denied (publickey).\n", 255, nil,
			Check{Failure: Rejected}},
		{"no host", account.GitHub, "",
			"ssh: Could not resolve hostname github.com: nodename nor servname provided\n", 255, nil,
			Check{Failure: NoHost}},
		{"refused", account.GitLab, "", "ssh: connect to host gitlab.com port 22: Connection refused\n", 255, nil,
			Check{Failure: Refused}},
		{"connect timeout", account.GitHub, "", "ssh: connect to host github.com port 22: Operation timed out\n", 255,
			nil, Check{Failure: TimedOut}},
		{"host key", account.GitHub, "", "Host key verification failed.\n", 255, nil, Check{Failure: HostKey}},
		{"no identity", account.GitHub, "",
			"Warning: Identity file /home/ada/.ssh/id_work not accessible: No such file or directory.\n", 255, nil,
			Check{Failure: NoKeyFile}},
		{"other", account.GitHub, "", "\n  " + long + "\nmore\n", 255, nil,
			Check{Failure: Other, Detail: strings.Repeat("x", 80)}},
		{"killed", account.GitHub, "", "", -1, fmt.Errorf("ssh: %w", sysexec.ErrTimeout), Check{Failure: TimedOut}},
		{"no ssh", account.GitHub, "", "", 0, &exec.Error{Name: "ssh", Err: exec.ErrNotFound},
			Check{Failure: NoCommand, Detail: "ssh"}},
		{"custom without failure", account.Custom, "Welcome to our forge\n", "", 1, nil, Check{OK: true}},
		{"custom denied", account.Custom, "", "Permission denied (publickey).", 255, nil, Check{Failure: Rejected}},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := &fake{answers: map[string]func([]string) (sysexec.Result, error){"ssh": func([]string) (sysexec.Result, error) {
				return sysexec.Result{Stdout: c.stdout, Stderr: c.stderr, ExitCode: c.code}, c.err
			}}}
			c.want.ID = "work"
			if got := Connect(context.Background(), f.run, tools, identity("work", c.provider)); got != c.want {
				t.Errorf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestCache(t *testing.T) {
	start := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
	ok, failed, plain := identity("ok", account.GitHub), identity("failed", account.GitHub), identity("plain", account.GitHub)
	plain.SSHAlias = ""
	list := []account.Account{ok, failed, plain}
	var c Cache
	ids := func(l []account.Account) []string {
		var out []string
		for _, a := range l {
			out = append(out, a.ID)
		}
		return out
	}
	if got := ids(c.Due(list, start, false)); !reflect.DeepEqual(got, []string{"ok", "failed"}) {
		t.Fatalf("due at first %v", got)
	}
	c.Put(list, []Check{{ID: "ok", OK: true, At: start}, {ID: "failed", Failure: Rejected, At: start}})
	if got := c.Due(list, start.Add(CacheTTL-time.Second), false); len(got) != 0 {
		t.Errorf("due before the failure is old: %v", ids(got))
	}
	if got := ids(c.Due(list, start.Add(CacheTTL), false)); !reflect.DeepEqual(got, []string{"failed"}) {
		t.Errorf("due after the TTL %v, want only the failure", got)
	}
	if got := ids(c.Due(list, start.Add(time.Hour), false)); !reflect.DeepEqual(got, []string{"failed"}) {
		t.Errorf("a success ran again on its own: %v", got)
	}
	if got := ids(c.Due(list, start, true)); !reflect.DeepEqual(got, []string{"ok", "failed"}) {
		t.Errorf("forced %v", got)
	}
	edited := ok
	edited.KeyPath = "~/.ssh/id_other"
	if _, kept := c.Get(edited); kept {
		t.Error("a check of the old key stands for the new one")
	}
	if got := ids(c.Due([]account.Account{edited}, start, false)); !reflect.DeepEqual(got, []string{"ok"}) {
		t.Errorf("an edited identity is not due: %v", got)
	}
	edited.Notes, edited.Label = "renamed", "renamed"
	if _, kept := c.Get(ok); !kept {
		t.Error("the check is gone")
	}
	c.Put([]account.Account{ok}, nil)
	if got := c.Checks(list); len(got) != 1 || got[0].ID != "ok" {
		t.Errorf("checks after the failed identity went away %+v", got)
	}
}

func TestCheckAll(t *testing.T) {
	at := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
	f := &fake{answers: map[string]func([]string) (sysexec.Result, error){
		"ssh": func(args []string) (sysexec.Result, error) {
			if args[len(args)-1] == "git@gh-ok" {
				return sysexec.Result{Stderr: "Hi ada!", ExitCode: 1}, nil
			}
			return sysexec.Result{Stderr: "Permission denied (publickey).", ExitCode: 255}, nil
		},
	}}
	r := Runner{Run: f.run, Tools: tools, Env: paths.Env{Home: t.TempDir()}, Now: func() time.Time { return at }}
	got := r.CheckAll(context.Background(), []account.Account{identity("ok", account.GitHub), identity("bad", account.GitHub)})
	want := []Check{
		{ID: "ok", OK: true, User: "ada", At: at},
		{ID: "bad", Failure: Rejected, At: at, Key: KeyState{HasPath: true}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if f.ran("ssh-add") || f.ran("ssh-keygen") {
		t.Error("ran a key command for a missing key")
	}
}

// keyDir writes a stand-in private key and, unless pub is empty, its public key, and returns the env and the
// identity that uses them.
func keyDir(t *testing.T, pub string) (paths.Env, account.Account) {
	t.Helper()
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".ssh", "id_work"), []byte("stand-in"), 0o600); err != nil {
		t.Fatal(err)
	}
	if pub != "" {
		if err := os.WriteFile(filepath.Join(home, ".ssh", "id_work.pub"), []byte(pub+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return paths.Env{Home: home}, identity("work", account.GitHub)
}

func TestInspectKey(t *testing.T) {
	agentWith := func(fp string) func([]string) (sysexec.Result, error) {
		return answer("256 "+fp+" ada@example.com (ED25519)\n", "", 0)
	}
	encrypted := answer("", `Load key "id_work": incorrect passphrase supplied to decrypt private key`, 255)
	for _, c := range []struct {
		name   string
		pub    string
		stored string
		add    func([]string) (sysexec.Result, error)
		keygen func([]string) (sysexec.Result, error)
		want   KeyState
	}{
		{"locked", pubKey, "", agentWith("SHA256:other"), encrypted,
			KeyState{HasPath: true, Exists: true, Agent: true, Passphrase: true, Fingerprint: fingerprint}},
		{"in agent", pubKey, "", agentWith(fingerprint), encrypted,
			KeyState{HasPath: true, Exists: true, Agent: true, InAgent: true, Passphrase: true, Fingerprint: fingerprint}},
		{"stored fingerprint wins", pubKey, "SHA256:stored", agentWith("SHA256:stored"), answer("ssh-ed25519 AAAA", "", 0),
			KeyState{HasPath: true, Exists: true, Agent: true, InAgent: true, Fingerprint: "SHA256:stored"}},
		{"empty agent", pubKey, "", answer("The agent has no identities.\n", "", 1), answer("", "", 0),
			KeyState{HasPath: true, Exists: true, Agent: true, Fingerprint: fingerprint}},
		{"no agent", pubKey, "", answer("", "Could not open a connection to your authentication agent.\n", 2), encrypted,
			KeyState{HasPath: true, Exists: true, Passphrase: true, Fingerprint: fingerprint}},
		{"no public key", "", "", agentWith(fingerprint), encrypted,
			KeyState{HasPath: true, Exists: true, Agent: true, Passphrase: true}},
	} {
		t.Run(c.name, func(t *testing.T) {
			env, a := keyDir(t, c.pub)
			a.KeyFingerprint = c.stored
			f := &fake{answers: map[string]func([]string) (sysexec.Result, error){"ssh-add": c.add, "ssh-keygen": c.keygen}}
			if got := InspectKey(context.Background(), f.run, tools, env, a); got != c.want {
				t.Errorf("got %+v, want %+v", got, c.want)
			}
			key := filepath.Join(env.Home, ".ssh", "id_work")
			wantKeygen := []string{"-y", "-P", "", "-f", key}
			for _, call := range f.calls {
				if filepath.Base(call.name) == "ssh-keygen" && !reflect.DeepEqual(call.args, wantKeygen) {
					t.Errorf("ssh-keygen %v, want %v", call.args, wantKeygen)
				}
			}
		})
	}
	t.Run("no key path", func(t *testing.T) {
		f := &fake{}
		a := identity("work", account.GitHub)
		a.KeyPath = ""
		if got := InspectKey(context.Background(), f.run, tools, paths.Env{Home: t.TempDir()}, a); got != (KeyState{}) {
			t.Errorf("got %+v", got)
		}
		if len(f.calls) != 0 {
			t.Errorf("ran %+v", f.calls)
		}
	})
	t.Run("a folder is not a key", func(t *testing.T) {
		env, a := keyDir(t, "")
		a.KeyPath = "~/.ssh"
		f := &fake{}
		if got := InspectKey(context.Background(), f.run, tools, env, a); got != (KeyState{HasPath: true}) {
			t.Errorf("got %+v", got)
		}
	})
}

func TestFingerprint(t *testing.T) {
	if got, err := Fingerprint(pubKey); err != nil || got != fingerprint {
		t.Errorf("got %q, %v, want %q", got, err, fingerprint)
	}
	for _, bad := range []string{"", "ssh-ed25519", "ssh-ed25519 not*base64"} {
		if _, err := Fingerprint(bad); err == nil {
			t.Errorf("%q has a fingerprint", bad)
		}
	}
}

func TestPublicKey(t *testing.T) {
	env, a := keyDir(t, pubKey)
	if got, err := PublicKey(env, a); err != nil || got != pubKey {
		t.Errorf("got %q, %v", got, err)
	}
	if err := os.WriteFile(filepath.Join(env.Home, ".ssh", "id_work.pub"), []byte(" \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := PublicKey(env, a); !errors.Is(err, ErrNoPublicKey) {
		t.Errorf("empty file: %v", err)
	}
	env, a = keyDir(t, "")
	if _, err := PublicKey(env, a); !errors.Is(err, ErrNoPublicKey) {
		t.Errorf("missing file: %v", err)
	}
	a.KeyPath = ""
	if _, err := PublicKey(env, a); !errors.Is(err, ErrNoPublicKey) {
		t.Errorf("no key path: %v", err)
	}
}

func TestUnlock(t *testing.T) {
	env := paths.Env{Home: "/home/ada"}
	a := identity("work", account.GitHub)
	f := &fake{answers: map[string]func([]string) (sysexec.Result, error){"ssh-add": answer("", "Identity added: id_work\n", 0)}}
	if err := Unlock(context.Background(), f.run, tools, env, a); err != nil {
		t.Fatal(err)
	}
	want := call{[]string{"SSH_ASKPASS_REQUIRE=never"}, "/usr/bin/ssh-add",
		[]string{"--apple-use-keychain", filepath.FromSlash("/home/ada/.ssh/id_work")}}
	if !reflect.DeepEqual(f.calls, []call{want}) {
		t.Errorf("ran %+v, want %+v", f.calls, want)
	}
	f.answers["ssh-add"] = answer("", "\nCould not add identity\n", 1)
	if err := Unlock(context.Background(), f.run, tools, env, a); err == nil || err.Error() != "Could not add identity" {
		t.Errorf("failure %v", err)
	}
}

func TestUnlockCommand(t *testing.T) {
	for _, c := range []struct {
		name string
		env  paths.Env
		key  string
		want string
	}{
		{"tilde", paths.Env{Home: "/home/ada"}, "~/.ssh/id_work", "ssh-add ~/.ssh/id_work"},
		{"outside home", paths.Env{Home: "/home/ada"}, "/keys/id_work", "ssh-add /keys/id_work"},
		{"space", paths.Env{Home: "/home/ada"}, "~/My Keys/id_work", `ssh-add '/home/ada/My Keys/id_work'`},
		{"quote", paths.Env{Home: "/home/ada"}, "/keys/ada's", `ssh-add '/keys/ada'\''s'`},
		{"windows", paths.Env{Home: "C:/Users/ada", Windows: true}, "~/.ssh/id_work", `ssh-add C:\Users\ada\.ssh\id_work`},
		{"windows space", paths.Env{Home: "C:/Users/ada", Windows: true}, "~/My Keys/id_work",
			`ssh-add "C:\Users\ada\My Keys\id_work"`},
	} {
		t.Run(c.name, func(t *testing.T) {
			a := identity("work", account.GitHub)
			a.KeyPath = c.key
			if got := UnlockCommand(c.env, a); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

// TestInspectKeyWithSystemCommands runs the real ssh-keygen and ssh-add on keys made for the test, so that a change
// in their output shows up here.
func TestInspectKeyWithSystemCommands(t *testing.T) {
	keygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		t.Skip("no ssh-keygen")
	}
	sys := FindTools(runtime.GOOS, "", exec.LookPath, func(string) bool { return false })
	sys.Keygen = keygen
	home := t.TempDir()
	env := paths.Env{Home: home}
	run := System(10 * time.Second)
	for _, c := range []struct {
		name, passphrase string
		want             bool
	}{{"plain", "", false}, {"encrypted", "correct horse", true}} {
		t.Run(c.name, func(t *testing.T) {
			key := filepath.Join(home, "id_"+c.name)
			res, err := run(context.Background(), nil, keygen, "-q", "-t", "ed25519", "-N", c.passphrase, "-C", "ada@example.com", "-f", key)
			if err != nil || res.ExitCode != 0 {
				t.Fatalf("ssh-keygen: %v %s", err, res.Stderr)
			}
			a := account.Account{ID: c.name, KeyPath: "~/id_" + c.name}
			got := InspectKey(context.Background(), run, sys, env, a)
			if !got.Exists || got.Passphrase != c.want {
				t.Errorf("got %+v, want passphrase %v", got, c.want)
			}
			out, err := run(context.Background(), nil, keygen, "-l", "-E", "sha256", "-f", key+".pub")
			if err != nil || len(strings.Fields(out.Stdout)) < 2 || strings.Fields(out.Stdout)[1] != got.Fingerprint {
				t.Errorf("fingerprint %q, ssh-keygen says %q", got.Fingerprint, out.Stdout)
			}
		})
	}
}
