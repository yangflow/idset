package diagnose

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"os"
	"strings"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/paths"
)

// KeyState is what is known about the key of an identity whose check failed. The private key is never opened:
// ssh-keygen and ssh-add look at it, Idset reads only the public key next to it.
type KeyState struct {
	// HasPath: the identity names a key; without one ssh tries its default keys and nothing else is gathered.
	HasPath bool
	// Exists: the private key file exists. When it does not, no command runs.
	Exists bool
	// Agent: ssh-agent answered; InAgent: it holds the key, known only when Fingerprint is.
	Agent, InAgent bool
	// Passphrase: the private key is encrypted.
	Passphrase bool
	// Fingerprint is the SHA256 fingerprint: the stored one, else the one of the .pub file, else empty.
	Fingerprint string
}

// InspectKey gathers the key state of a.
func InspectKey(ctx context.Context, run Run, tools Tools, env paths.Env, a account.Account) KeyState {
	var k KeyState
	if strings.TrimSpace(a.KeyPath) == "" {
		return k
	}
	k.HasPath = true
	key := env.Native(strings.TrimSpace(a.KeyPath))
	if info, err := os.Stat(key); err != nil || !info.Mode().IsRegular() {
		return k
	}
	k.Exists = true
	k.Fingerprint = strings.TrimSpace(a.KeyFingerprint)
	if k.Fingerprint == "" {
		if text, err := os.ReadFile(key + ".pub"); err == nil {
			k.Fingerprint, _ = Fingerprint(string(text))
		}
	}
	// ssh-add -l exits 0 with keys, 1 for an agent without keys, 2 when there is no agent.
	if res, err := run(ctx, nil, tools.Add, "-l", "-E", "sha256"); err == nil && (res.ExitCode == 0 || res.ExitCode == 1) {
		k.Agent = true
		for _, line := range strings.Split(res.Stdout, "\n") {
			if f := strings.Fields(line); len(f) >= 2 && k.Fingerprint != "" && f[1] == k.Fingerprint {
				k.InAgent = true
			}
		}
	}
	// With an empty passphrase ssh-keygen prints the public key, which is dropped; an encrypted key fails.
	if res, err := run(ctx, nil, tools.Keygen, "-y", "-P", "", "-f", key); err == nil && res.ExitCode != 0 &&
		strings.Contains(strings.ToLower(res.Stdout+res.Stderr), "incorrect passphrase") {
		k.Passphrase = true
	}
	return k
}

// Fingerprint is the SHA256 fingerprint of a public key line, "SHA256:" and the unpadded base64 of the hash, as
// ssh-keygen -l prints it.
func Fingerprint(pub string) (string, error) {
	f := strings.Fields(pub)
	if len(f) < 2 {
		return "", errors.New("not a public key")
	}
	blob, err := base64.StdEncoding.DecodeString(f[1])
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(blob)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:]), nil
}

// ErrNoPublicKey is returned by PublicKey when the .pub file is missing or empty.
var ErrNoPublicKey = errors.New("the public key file is missing or empty")

// PublicKey is the text of the .pub file next to the key of a, for the clipboard.
func PublicKey(env paths.Env, a account.Account) (string, error) {
	if strings.TrimSpace(a.KeyPath) == "" {
		return "", ErrNoPublicKey
	}
	text, err := os.ReadFile(env.Native(strings.TrimSpace(a.KeyPath)) + ".pub")
	if errors.Is(err, os.ErrNotExist) {
		return "", ErrNoPublicKey
	}
	if err != nil {
		return "", err
	}
	if s := strings.TrimSpace(string(text)); s != "" {
		return s, nil
	}
	return "", ErrNoPublicKey
}

// Unlock loads the key of a into the agent with the passphrase in the macOS keychain. SSH_ASKPASS_REQUIRE=never
// keeps ssh-add from asking: without a stored passphrase it fails instead.
func Unlock(ctx context.Context, run Run, tools Tools, env paths.Env, a account.Account) error {
	res, err := run(ctx, []string{"SSH_ASKPASS_REQUIRE=never"}, tools.Add, "--apple-use-keychain",
		env.Native(strings.TrimSpace(a.KeyPath)))
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return errors.New(firstLine(res.Stderr + "\n" + res.Stdout))
	}
	return nil
}

// UnlockCommand is the ssh-add command to run in a terminal where the keychain cannot unlock the key. The path is
// quoted for the shell when it needs it.
func UnlockCommand(env paths.Env, a account.Account) string {
	p := env.Expand(strings.TrimSpace(a.KeyPath))
	if env.Windows {
		p = strings.ReplaceAll(p, "/", `\`)
		if strings.ContainsAny(p, " '\"") {
			p = `"` + p + `"`
		}
		return "ssh-add " + p
	}
	if t := env.Tilde(p); strings.HasPrefix(t, "~/") && plain(t[2:]) {
		return "ssh-add " + t
	}
	if plain(p) {
		return "ssh-add " + p
	}
	return "ssh-add '" + strings.ReplaceAll(p, "'", `'\''`) + "'"
}

// plain reports whether a path needs no quoting in a POSIX shell.
func plain(p string) bool {
	for _, r := range p {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("/._-+@%:,", r)) {
			return false
		}
	}
	return p != ""
}
