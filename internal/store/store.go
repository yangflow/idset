// Package store keeps accounts.json: it loads the list once, validates every change against the whole list,
// writes atomically and backs up before additions and restores. A file that cannot be read makes the store
// read-only, so that a damaged file is never overwritten.
//
// After every change to the list (not after Touch) the store calls the regenerate function given to Open, which
// rewrites the managed configuration. Every method blocks on disk work, so callers run them off the UI thread.
package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/fsutil"
)

// FileName is the name of the identity list in the Idset directory.
const FileName = "accounts.json"

var (
	ErrNotFound  = errors.New("account not found")
	ErrDuplicate = errors.New("account already exists")
)

// ReadOnlyError explains why the store refuses to write: the file could not be read.
type ReadOnlyError struct{ Err error }

func (e *ReadOnlyError) Error() string { return "accounts.json cannot be read: " + e.Err.Error() }
func (e *ReadOnlyError) Unwrap() error { return e.Err }

// RegenerateError reports that the list was saved but the managed configuration could not be rewritten. The
// change itself stands; the next change retries the regeneration.
type RegenerateError struct{ Err error }

func (e *RegenerateError) Error() string {
	return "regenerating the managed configuration: " + e.Err.Error()
}
func (e *RegenerateError) Unwrap() error { return e.Err }

// Regenerate rewrites the managed configuration from list; see apply.Regenerate.
type Regenerate func(list []account.Account) error

// Option configures Open.
type Option func(*Store)

// WithRegenerate sets the function called after each change to the list. Without it nothing is regenerated.
func WithRegenerate(r Regenerate) Option { return func(s *Store) { s.regenerate = r } }

// Store is safe for concurrent use.
type Store struct {
	path      string
	backupDir string
	now       func() time.Time

	regenerate Regenerate

	mu       sync.Mutex
	accounts []account.Account
	readErr  error
}

// Open loads the list from dir/accounts.json; backups go to dir/backups. A missing file is an empty list and is
// not created until the first change.
func Open(dir string, now func() time.Time, opts ...Option) *Store {
	s := &Store{path: filepath.Join(dir, FileName), backupDir: filepath.Join(dir, "backups"), now: now}
	for _, opt := range opts {
		opt(s)
	}
	s.load()
	return s
}

func (s *Store) load() {
	s.accounts, s.readErr = nil, nil
	data, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err == nil {
		s.accounts, err = account.Decode(data)
	}
	if err != nil {
		s.accounts, s.readErr = nil, &ReadOnlyError{Err: err}
	}
}

// Path is the location of accounts.json.
func (s *Store) Path() string { return s.path }

// BackupDir is where backups are kept.
func (s *Store) BackupDir() string { return s.backupDir }

// ReadOnly returns a *ReadOnlyError when the file could not be read, nil otherwise.
func (s *Store) ReadOnly() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readErr
}

// Accounts returns a copy of the list in storage order.
func (s *Store) Accounts() []account.Account {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.accounts)
}

func (s *Store) index(id string) int {
	return slices.IndexFunc(s.accounts, func(a account.Account) bool { return a.ID == id })
}

// regenerateNow rewrites the managed configuration from the current list; the caller holds s.mu.
func (s *Store) regenerateNow() error {
	if s.regenerate == nil {
		return nil
	}
	if err := s.regenerate(slices.Clone(s.accounts)); err != nil {
		return &RegenerateError{Err: err}
	}
	return nil
}

// commit validates next, writes it and makes it the list, then regenerates when regen is set. backup is taken
// first when the file exists. A *RegenerateError means the list was saved but the configuration was not.
func (s *Store) commit(next []account.Account, backup, regen bool) error {
	if s.readErr != nil {
		return s.readErr
	}
	if problems := introduced(s.accounts, next); len(problems) > 0 {
		return problems
	}
	data, err := account.Encode(next)
	if err != nil {
		return err
	}
	if backup {
		if _, err := os.Lstat(s.path); err == nil {
			if _, err := Snapshot(s.path, s.backupDir, s.now()); err != nil {
				return fmt.Errorf("backup before saving: %w", err)
			}
		}
	}
	if err := fsutil.WriteAtomic(s.path, data, 0o600, 0o700); err != nil {
		return err
	}
	s.accounts = next
	if regen {
		return s.regenerateNow()
	}
	return nil
}

// introduced returns the problems of next that cur does not already have, so that a file edited by hand into a
// state the app would not save (two global identities, say) can still take unrelated changes.
func introduced(cur, next []account.Account) account.Problems {
	before := map[account.Problem]bool{}
	for _, p := range account.Validate(cur) {
		before[p] = true
	}
	var out account.Problems
	for _, p := range account.Validate(next) {
		if !before[p] {
			out = append(out, p)
		}
	}
	return out
}

// Add appends a new identity after backing up the current file.
func (s *Store) Add(a account.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index(a.ID) >= 0 {
		return ErrDuplicate
	}
	return s.commit(append(slices.Clone(s.accounts), a), true, true)
}

// Update replaces the identity with the same ID in place and stamps UpdatedAt.
func (s *Store) Update(a account.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.index(a.ID)
	if i < 0 {
		return ErrNotFound
	}
	a.UpdatedAt = s.now()
	next := slices.Clone(s.accounts)
	next[i] = a
	return s.commit(next, false, true)
}

// Delete removes an identity.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.index(id)
	if i < 0 {
		return ErrNotFound
	}
	return s.commit(slices.Delete(slices.Clone(s.accounts), i, i+1), false, true)
}

// Touch records a use of an identity for sorting; it does not change UpdatedAt. An unknown ID is ignored.
func (s *Store) Touch(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.readErr != nil {
		return s.readErr
	}
	i := s.index(id)
	if i < 0 {
		return nil
	}
	next := slices.Clone(s.accounts)
	t := s.now()
	next[i].LastUsedAt = &t
	return s.commit(next, false, false)
}

// ReplaceAll writes a whole new list, for undo.
func (s *Store) ReplaceAll(list []account.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.commit(slices.Clone(list), false, true)
}

// Backups lists the backups of accounts.json, newest first.
func (s *Store) Backups() ([]Backup, error) { return List(s.backupDir, FileName) }

// Restore backs up the current file, replaces it with backup and reloads. It works in the read-only state too,
// since restoring is how a damaged file is repaired.
func (s *Store) Restore(backup string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := os.Lstat(s.path); err == nil {
		if _, err := Snapshot(s.path, s.backupDir, s.now()); err != nil {
			return fmt.Errorf("backup before restoring: %w", err)
		}
	}
	if err := restoreFile(backup, s.path); err != nil {
		return err
	}
	s.load()
	if s.readErr != nil {
		return s.readErr
	}
	return s.regenerateNow()
}
