package keygen

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/sysexec"
)

const (
	pubKey      = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGVvBRLiy1IB3trYVbYfC8KOIqOu/SBSTpKehjxdeEqt ada@example.com"
	fingerprint = "SHA256:GaZxSsN21z2W8DOaH9fsWhWZ2d8xBcx1tRPgPmuE4qc"
)

func TestValidName(t *testing.T) {
	for name, want := range map[string]bool{
		"id_idset": true, "id_ed25519.work": true, "A-1_b.c": true,
		"": false, ".hidden": false, "a..b": false, "a/b": false, `a\b`: false, "a b": false, "ключ": false, "a:b": false,
	} {
		if got := ValidName(name); got != want {
			t.Errorf("ValidName(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestComment(t *testing.T) {
	if got := Comment("  ada@example.com\nwork\t "); got != "ada@example.com work" {
		t.Errorf("Comment = %q", got)
	}
}

func TestNameOf(t *testing.T) {
	env := paths.Env{Home: "/home/ada"}
	for keyPath, want := range map[string]string{
		"~/.ssh/id_work":           "id_work",
		"/home/ada/.ssh/id_work":   "id_work",
		"~/.ssh/keys/id_work":      "",
		"~/keys/id_work":           "",
		"~/.ssh/.hidden":           "",
		"":                         "",
		" ~/.ssh/id_client ":       "id_client",
		"/elsewhere/.ssh/id_other": "",
	} {
		if got := NameOf(env, keyPath); got != want {
			t.Errorf("NameOf(%q) = %q, want %q", keyPath, got, want)
		}
	}
}

// keygenFake stands in for ssh-keygen: it records the arguments and writes the two files the real one would, the
// private one with a placeholder.
type keygenFake struct {
	args [][]string
	pub  string
	res  sysexec.Result
	err  error
}

func (f *keygenFake) run(_ context.Context, _ []string, _ string, args ...string) (sysexec.Result, error) {
	f.args = append(f.args, args)
	if f.err != nil || f.res.ExitCode != 0 {
		return f.res, f.err
	}
	out := args[len(args)-1]
	if err := os.WriteFile(out, []byte("placeholder"), 0o600); err != nil {
		return sysexec.Result{}, err
	}
	if f.pub != "" {
		if err := os.WriteFile(out+".pub", []byte(f.pub+"\n"), 0o644); err != nil {
			return sysexec.Result{}, err
		}
	}
	return f.res, nil
}

func home(t *testing.T) paths.Env {
	return paths.Env{Home: t.TempDir(), Windows: runtime.GOOS == "windows"}
}

func TestGenerate(t *testing.T) {
	env := home(t)
	f := &keygenFake{pub: pubKey}
	key, err := Generate(context.Background(), f.run, "ssh-keygen", env, Ed25519, "id_idset", "ada@example.com\n")
	if err != nil {
		t.Fatal(err)
	}
	want := Key{Path: "~/.ssh/id_idset", Public: pubKey, Fingerprint: fingerprint}
	if key != want {
		t.Errorf("key = %+v, want %+v", key, want)
	}
	private := filepath.Join(env.Home, ".ssh", "id_idset")
	args := []string{"-q", "-t", "ed25519", "-N", "", "-C", "ada@example.com", "-f", private}
	if !reflect.DeepEqual(f.args, [][]string{args}) {
		t.Errorf("args = %q, want %q", f.args, args)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(env.Home, ".ssh"))
		if err != nil || info.Mode().Perm() != 0o700 {
			t.Errorf("~/.ssh mode = %v, %v", info.Mode(), err)
		}
	}
}

func TestGenerateRSA(t *testing.T) {
	env := home(t)
	f := &keygenFake{pub: pubKey}
	if _, err := Generate(context.Background(), f.run, "ssh-keygen", env, RSA, "id_rsa_idset", ""); err != nil {
		t.Fatal(err)
	}
	if got := f.args[0][:5]; !reflect.DeepEqual(got, []string{"-q", "-t", "rsa", "-b", "4096"}) {
		t.Errorf("args = %q", got)
	}
}

func TestGenerateNeverOverwrites(t *testing.T) {
	for _, existing := range []string{"id_idset", "id_idset.pub"} {
		env := home(t)
		dir := filepath.Join(env.Home, ".ssh")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, existing), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		f := &keygenFake{pub: pubKey}
		_, err := Generate(context.Background(), f.run, "ssh-keygen", env, Ed25519, "id_idset", "")
		var exists *ExistsError
		if !errors.As(err, &exists) || exists.Path != "~/.ssh/"+existing {
			t.Errorf("%s: err = %v", existing, err)
		}
		if len(f.args) != 0 {
			t.Errorf("%s: ssh-keygen ran", existing)
		}
	}
}

func TestGenerateRefusesSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic links need privileges on Windows")
	}
	env := home(t)
	dir := filepath.Join(env.Home, ".ssh")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(env.Home, "gone"), filepath.Join(dir, "id_idset")); err != nil {
		t.Fatal(err)
	}
	f := &keygenFake{pub: pubKey}
	var exists *ExistsError
	if _, err := Generate(context.Background(), f.run, "ssh-keygen", env, Ed25519, "id_idset", ""); !errors.As(err, &exists) {
		t.Errorf("err = %v", err)
	}
}

func TestGenerateFailures(t *testing.T) {
	env := home(t)
	if _, err := Generate(context.Background(), (&keygenFake{}).run, "ssh-keygen", env, Ed25519, "../id", ""); !errors.Is(err, ErrName) {
		t.Errorf("bad name: err = %v", err)
	}
	missing := &keygenFake{err: fmt.Errorf("start: %w", exec.ErrNotFound)}
	if _, err := Generate(context.Background(), missing.run, "ssh-keygen", env, Ed25519, "id_a", ""); !NotFound(err) {
		t.Errorf("missing ssh-keygen: err = %v", err)
	}
	failed := &keygenFake{res: sysexec.Result{ExitCode: 1, Stderr: "\nSaving key failed: Permission denied\n"}}
	if _, err := Generate(context.Background(), failed.run, "ssh-keygen", env, Ed25519, "id_b", ""); err == nil || err.Error() != "Saving key failed: Permission denied" {
		t.Errorf("failed ssh-keygen: err = %v", err)
	}
	noPub := &keygenFake{}
	key, err := Generate(context.Background(), noPub.run, "ssh-keygen", env, Ed25519, "id_c", "")
	if err != nil || key != (Key{Path: "~/.ssh/id_c"}) {
		t.Errorf("no .pub: key = %+v, err = %v", key, err)
	}
}

func TestLoad(t *testing.T) {
	env := paths.Env{Home: "/home/ada"}
	var got []string
	var gotEnv []string
	run := func(res sysexec.Result, err error) func(context.Context, []string, string, ...string) (sysexec.Result, error) {
		return func(_ context.Context, e []string, name string, args ...string) (sysexec.Result, error) {
			got, gotEnv = append([]string{name}, args...), e
			return res, err
		}
	}
	if err := Load(context.Background(), run(sysexec.Result{}, nil), "ssh-add", env, "~/.ssh/id_idset"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"ssh-add", filepath.FromSlash("/home/ada/.ssh/id_idset")}; !reflect.DeepEqual(got, want) {
		t.Errorf("ran %q, want %q", got, want)
	}
	if !reflect.DeepEqual(gotEnv, []string{"SSH_ASKPASS_REQUIRE=never"}) {
		t.Errorf("env = %q", gotEnv)
	}
	for _, res := range []sysexec.Result{
		{ExitCode: 2, Stderr: "Could not open a connection to your authentication agent."},
		{ExitCode: 1, Stderr: "Error connecting to agent: No such file or directory"},
	} {
		if err := Load(context.Background(), run(res, nil), "ssh-add", env, "~/.ssh/id_idset"); !errors.Is(err, ErrNoAgent) {
			t.Errorf("%q: err = %v", res.Stderr, err)
		}
	}
	if err := Load(context.Background(), run(sysexec.Result{ExitCode: 1, Stderr: "Invalid key"}, nil), "ssh-add", env, "~/.ssh/id_idset"); err == nil || err.Error() != "Invalid key" {
		t.Errorf("err = %v", err)
	}
}
