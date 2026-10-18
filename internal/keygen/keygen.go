// Package keygen generates a new SSH key pair in ~/.ssh/ with ssh-keygen and loads it into ssh-agent with ssh-add.
// It never overwrites a file, never opens the private key and reads only the public key next to it. The commands
// are injectable; tests use a fake.
package keygen

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"

	"github.com/yangflow/idset/internal/diagnose"
	"github.com/yangflow/idset/internal/paths"
)

// Type is the kind of key to generate.
type Type int

const (
	Ed25519 Type = iota
	RSA
)

// Dir is the stored form of the folder new keys go in.
const Dir = "~/.ssh/"

// ValidName reports whether name may be the file name of a new key: letters, digits, ".", "_" and "-" only, not
// starting with "." and without "..", so that the key lands in ~/.ssh/ itself and is not hidden.
func ValidName(name string) bool {
	if name == "" || strings.HasPrefix(name, ".") || strings.Contains(name, "..") {
		return false
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// Comment is the comment of a new key on one line: control characters become spaces and the ends are trimmed.
func Comment(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < ' ' || r == 0x7f {
			return ' '
		}
		return r
	}, s)
	return strings.TrimSpace(s)
}

// NameOf is the file name of a stored key path that lies directly in ~/.ssh/ with a valid name, else "".
func NameOf(env paths.Env, keyPath string) string {
	p := env.Tilde(strings.TrimSpace(keyPath))
	if !strings.HasPrefix(p, Dir) {
		return ""
	}
	if name := p[len(Dir):]; ValidName(name) {
		return name
	}
	return ""
}

// Key is a generated key: Path is the stored form of the private key, Public the text of the .pub file and
// Fingerprint its SHA256 fingerprint. Public and Fingerprint are empty when the .pub file could not be read.
type Key struct {
	Path, Public, Fingerprint string
}

// ErrName is returned by Generate for a file name ValidName refuses.
var ErrName = errors.New("invalid key file name")

// ExistsError is returned by Generate when the private key or the .pub file already exists. Path is its stored
// form.
type ExistsError struct{ Path string }

func (e *ExistsError) Error() string { return e.Path + " already exists" }

// Generate creates ~/.ssh/<name> and its .pub file with ssh-keygen, without a passphrase. Neither file may exist
// beforehand. A missing ssh-keygen is reported as exec.ErrNotFound.
func Generate(ctx context.Context, run diagnose.Run, keygen string, env paths.Env, t Type, name, comment string) (Key, error) {
	if !ValidName(name) {
		return Key{}, ErrName
	}
	stored := Dir + name
	private := env.Native(stored)
	for _, p := range []string{stored, stored + ".pub"} {
		if _, err := os.Lstat(env.Native(p)); err == nil {
			return Key{}, &ExistsError{Path: p}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return Key{}, err
		}
	}
	if err := os.MkdirAll(env.Native(Dir), 0o700); err != nil {
		return Key{}, err
	}
	args := []string{"-q", "-t", "ed25519"}
	if t == RSA {
		args = []string{"-q", "-t", "rsa", "-b", "4096"}
	}
	args = append(args, "-N", "", "-C", Comment(comment), "-f", private)
	res, err := run(ctx, nil, keygen, args...)
	if err != nil {
		return Key{}, fmt.Errorf("ssh-keygen: %w", err)
	}
	if res.ExitCode != 0 {
		return Key{}, errors.New(firstLine(res.Stderr + "\n" + res.Stdout))
	}
	key := Key{Path: stored}
	if text, err := os.ReadFile(private + ".pub"); err == nil {
		key.Public = strings.TrimSpace(string(text))
		key.Fingerprint, _ = diagnose.Fingerprint(key.Public)
	}
	return key, nil
}

// NotFound reports whether err says the command does not exist.
func NotFound(err error) bool { return errors.Is(err, exec.ErrNotFound) }

// ErrNoAgent is returned by Load when no ssh-agent answers.
var ErrNoAgent = errors.New("ssh-agent is not running")

// Load adds the key at the stored path keyPath to ssh-agent. The key has no passphrase, so ssh-add asks nothing;
// SSH_ASKPASS_REQUIRE=never makes sure it never waits for one.
func Load(ctx context.Context, run diagnose.Run, add string, env paths.Env, keyPath string) error {
	res, err := run(ctx, []string{"SSH_ASKPASS_REQUIRE=never"}, add, env.Native(keyPath))
	if err != nil {
		return err
	}
	out := res.Stderr + "\n" + res.Stdout
	switch {
	case res.ExitCode == 0:
		return nil
	case res.ExitCode == 2 || strings.Contains(strings.ToLower(out), "could not open a connection to your authentication agent") ||
		strings.Contains(strings.ToLower(out), "error connecting to agent"):
		return ErrNoAgent
	}
	return errors.New(firstLine(out))
}

// firstLine is the first line of output that is not blank.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return "exit status non-zero"
}
