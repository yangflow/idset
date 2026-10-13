// Package keyfile decides whether an identity's private key may be deleted along with it, and deletes it. It
// looks at the path with Lstat only and never opens the key: Idset does not read private keys.
package keyfile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/paths"
)

// Reason says why a private key cannot be deleted; None means it can.
type Reason int

const (
	None    Reason = iota
	NoPath         // the identity has no key path
	Shared         // another identity uses the same key
	Symlink        // the key path is a symbolic link
	Missing        // the file does not exist
	NotFile        // the path is not a regular file
)

// Inspect checks the key of the identity with the given ID against the rest of list. The second result is the
// label of the other identity for Shared.
func Inspect(env paths.Env, list []account.Account, id string) (Reason, string) {
	var self account.Account
	found := false
	for _, a := range list {
		if a.ID == id {
			self, found = a, true
		}
	}
	if !found || strings.TrimSpace(self.KeyPath) == "" {
		return NoPath, ""
	}
	for _, a := range list {
		if a.ID != id && strings.TrimSpace(a.KeyPath) != "" && same(env, a.KeyPath, self.KeyPath) {
			return Shared, a.Label
		}
	}
	info, err := os.Lstat(env.Native(self.KeyPath))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return Missing, ""
	case err != nil:
		return NotFile, ""
	case info.Mode()&fs.ModeSymlink != 0:
		return Symlink, ""
	case !info.Mode().IsRegular():
		return NotFile, ""
	}
	return None, ""
}

func same(env paths.Env, a, b string) bool {
	return env.Equal(path.Clean(env.Expand(a)), path.Clean(env.Expand(b)))
}

// Remove deletes the private key at keyPath, checked again just before: only the key file itself goes, and the
// .pub file next to it stays. remaining is the identity list after the owner was deleted, so that a key another
// identity still uses is kept.
func Remove(env paths.Env, remaining []account.Account, keyPath string) error {
	const probeID = "removing"
	probe := append(append([]account.Account{}, remaining...), account.Account{ID: probeID, KeyPath: keyPath})
	if reason, other := Inspect(env, probe, probeID); reason != None {
		return &RefusedError{Reason: reason, Other: other}
	}
	return os.Remove(env.Native(keyPath))
}

// RefusedError is returned by Remove when the key may not be deleted.
type RefusedError struct {
	Reason Reason
	Other  string
}

func (e *RefusedError) Error() string {
	switch e.Reason {
	case NoPath:
		return "the identity has no key path"
	case Shared:
		return fmt.Sprintf("another identity (%s) uses the same key", e.Other)
	case Symlink:
		return "the key path is a symbolic link"
	case Missing:
		return "the key file does not exist"
	}
	return "the key path is not a regular file"
}
