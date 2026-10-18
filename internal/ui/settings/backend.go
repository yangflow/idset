package settings

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/apply"
	"github.com/yangflow/idset/internal/diagnose"
	"github.com/yangflow/idset/internal/importer"
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

// Scan reads the user's SSH and Git configuration for identities to import, running git through run. Without a
// store yet, every candidate is new.
func (b Backend) Scan(ctx context.Context, run diagnose.Run) ([]importer.Candidate, error) {
	var list []account.Account
	if b.Store != nil {
		list = b.Store.Accounts()
	}
	src := importer.Source{Env: b.Env, ReadFile: os.ReadFile, Glob: filepath.Glob, Run: run, Git: "git"}
	return importer.Scan(ctx, src, list)
}

// Import adds the identities one by one, each with its own backup and regeneration, giving each a new ID, the
// first color no identity uses yet and now as its times. It stops at the first that fails and returns those
// added, as stored. An identity saved whose regeneration failed counts as added.
func (b Backend) Import(list []account.Account, now time.Time) ([]account.Account, error) {
	now = now.UTC().Truncate(time.Second)
	var added []account.Account
	for _, a := range list {
		a.ID, a.Color, a.CreatedAt, a.UpdatedAt = account.NewID(), freeColor(b.Store.Accounts()), now, now
		err := b.Store.Add(a)
		var regen *store.RegenerateError
		if err == nil || errors.As(err, &regen) {
			added = append(added, a)
		}
		if err != nil {
			return added, err
		}
	}
	return added, nil
}

// freeColor is the first preset no identity in list uses, else the first preset.
func freeColor(list []account.Account) string {
	for _, c := range account.ColorPresets {
		if !slices.ContainsFunc(list, func(a account.Account) bool { return a.Color == c }) {
			return c
		}
	}
	return account.ColorPresets[0]
}
