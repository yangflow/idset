package settings

import (
	"os"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/apply"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/store"
)

// Backend is the work behind the window's actions. Every method blocks on disk, so the owner runs them off the
// UI thread and hands the outcome back to the Screen.
type Backend struct {
	Store *store.Store
	Env   paths.Env
}

// Backups lists the backups, newest first, each read to count its identities. A backup that cannot be read or
// decoded carries the reason in Err; it is still listed.
func (b Backend) Backups() ([]Backup, error) {
	list, err := b.Store.Backups()
	if err != nil {
		return nil, err
	}
	out := make([]Backup, 0, len(list))
	for _, item := range list {
		entry := Backup{Path: item.Path, Time: item.Time}
		if info, err := os.Lstat(item.Path); err == nil {
			entry.Size = info.Size()
		}
		data, err := os.ReadFile(item.Path)
		if err == nil {
			entry.Accounts, err = account.Decode(data)
		}
		entry.Err = err
		out = append(out, entry)
	}
	return out, nil
}

// Restore restores a backup, which backs up the current file first.
func (b Backend) Restore(path string) error { return b.Store.Restore(path) }

// Delete deletes a backup.
func (b Backend) Delete(path string) error { return store.DeleteBackup(path) }

// Includes looks for the Include block in the user's two files.
func (b Backend) Includes() apply.IncludeState { return apply.Includes(apply.DefaultTarget(b.Env)) }

// RemoveIncludes removes the Include blocks and reports the files before and after, for the message of a
// partial failure.
func (b Backend) RemoveIncludes() (before, after apply.IncludeState, err error) {
	target := apply.DefaultTarget(b.Env)
	before = apply.Includes(target)
	err = apply.RemoveIncludes(target)
	return before, apply.Includes(target), err
}
