// Package apply is the only place that writes generated configuration: the managed files in the Idset directory
// and the marked blocks in the user's ~/.ssh/config and ~/.gitconfig.
package apply

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/fsutil"
	"github.com/yangflow/idset/internal/include"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/project"
)

// Target is where files go, all absolute.
type Target struct {
	Dir       string // the Idset directory, ~/.config/idset
	SSHConfig string // ~/.ssh/config
	GitConfig string // ~/.gitconfig
}

// DefaultTarget is the standard layout under the home directory, the same on every platform.
func DefaultTarget(env paths.Env) Target {
	return Target{
		Dir:       env.Native(project.Dir),
		SSHConfig: env.Native("~/.ssh/config"),
		GitConfig: env.Native("~/.gitconfig"),
	}
}

// Regenerate returns the function a store calls after each change: it projects the list at the time now reports
// and writes it to t. Nothing runs until the function is called.
func Regenerate(env paths.Env, t Target, now func() time.Time) func([]account.Account) error {
	return func(list []account.Account) error {
		return Write(project.Project(list, env, now()), t)
	}
}

// Write writes the managed files, deletes per-identity files that are no longer generated, and makes sure both
// marked blocks are present and current.
func Write(out project.Output, t Target) error {
	if err := fsutil.WriteAtomic(filepath.Join(t.Dir, project.SSHFile), []byte(out.SSH), 0o600, 0o700); err != nil {
		return err
	}
	if err := fsutil.WriteAtomic(filepath.Join(t.Dir, project.GitFile), []byte(out.Git), 0o600, 0o700); err != nil {
		return err
	}
	stale, err := filepath.Glob(filepath.Join(t.Dir, "gitconfig-*.managed"))
	if err != nil {
		return err
	}
	for _, p := range stale {
		if _, keep := out.Subs[filepath.Base(p)]; !keep {
			if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
		}
	}
	for name, content := range out.Subs {
		if err := fsutil.WriteAtomic(filepath.Join(t.Dir, name), []byte(content), 0o600, 0o700); err != nil {
			return err
		}
	}
	if err := edit(t.SSHConfig, func(c string) (string, bool, error) {
		return include.EnsureTop(c, include.SSHBlock(project.Dir+"/"+project.SSHFile))
	}); err != nil {
		return err
	}
	return edit(t.GitConfig, func(c string) (string, bool, error) {
		return include.EnsureBottom(c, include.GitBlock(project.Dir+"/"+project.GitFile))
	})
}

// RemoveIncludes strips the marked blocks from both user files, leaving everything else as it was.
func RemoveIncludes(t Target) error {
	if err := edit(t.SSHConfig, include.Strip); err != nil {
		return err
	}
	return edit(t.GitConfig, include.Strip)
}

// edit rewrites a user file through change, following a symbolic link to the real file and writing nothing when
// the content stays the same. Line endings are kept as the file has them.
func edit(path string, change func(string) (string, bool, error)) error {
	real, err := fsutil.Resolve(path)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(real)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	content := string(data)
	crlf := strings.Contains(content, "\r\n")
	if crlf {
		content = strings.ReplaceAll(content, "\r\n", "\n")
	}
	next, changed, err := change(content)
	if err != nil || !changed {
		return err
	}
	if crlf {
		next = strings.ReplaceAll(next, "\n", "\r\n")
	}
	return fsutil.WriteAtomic(real, []byte(next), 0o600, 0o700)
}
