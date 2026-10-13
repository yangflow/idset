package accounts

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/keyfile"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/store"
	"github.com/yangflow/idset/internal/sysexec"
)

// GitConfig reads one key of `git config --global`; a key that is not set is "" without an error.
type GitConfig func(ctx context.Context, key string) (string, error)

// SystemGitConfig runs the git on PATH, each call limited to timeout.
func SystemGitConfig(timeout time.Duration) GitConfig {
	return func(ctx context.Context, key string) (string, error) {
		res, err := sysexec.Run(ctx, timeout, nil, "git", "config", "--global", "--get", key)
		if err != nil {
			return "", err
		}
		if res.ExitCode != 0 {
			return "", nil
		}
		first, _, _ := strings.Cut(res.Stdout, "\n")
		return strings.TrimSpace(first), nil
	}
}

// Backend is the work behind the window's actions. Every method blocks on disk or on a command, so the owner
// runs them off the UI thread and hands the outcome back to the Screen.
type Backend struct {
	Store *store.Store
	Env   paths.Env
	Git   GitConfig
}

// Save adds a new identity or replaces an existing one.
func (b Backend) Save(a account.Account, isNew bool) error {
	if isNew {
		return b.Store.Add(a)
	}
	return b.Store.Update(a)
}

// DeleteResult is the outcome of Delete. Err is the identity's deletion: a *store.RegenerateError means it was
// deleted but the configuration was not rewritten. KeyErr is the private key's removal, which never undoes the
// deletion of the identity.
type DeleteResult struct {
	Err, KeyErr error
}

// Delete removes the identity and regenerates the configuration, and only then, when deleteKey is set, removes
// the private key.
func (b Backend) Delete(id string, deleteKey bool) DeleteResult {
	var keyPath string
	for _, a := range b.Store.Accounts() {
		if a.ID == id {
			keyPath = a.KeyPath
		}
	}
	var res DeleteResult
	res.Err = b.Store.Delete(id)
	var regenerate *store.RegenerateError
	if res.Err != nil && !errors.As(res.Err, &regenerate) {
		return res
	}
	if deleteKey && strings.TrimSpace(keyPath) != "" {
		res.KeyErr = keyfile.Remove(b.Env, b.Store.Accounts(), keyPath)
	}
	return res
}

// InspectKey says whether the private key of an identity may be deleted with it, looking at the path only.
func (b Backend) InspectKey(id string) (keyfile.Reason, string) {
	return keyfile.Inspect(b.Env, b.Store.Accounts(), id)
}

// Prefill reads the signature from `git config --global`; whatever cannot be read is "".
func (b Backend) Prefill(ctx context.Context) (name, email string) {
	name, _ = b.Git(ctx, "user.name")
	email, _ = b.Git(ctx, "user.email")
	return name, email
}
