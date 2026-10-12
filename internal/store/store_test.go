package store

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/fsutil"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time { c.t = c.t.Add(time.Second); return c.t }

func newClock() *clock { return &clock{t: time.Date(2026, 8, 22, 8, 0, 0, 0, time.UTC)} }

func identity(label string, global bool) account.Account {
	a := account.New(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))
	a.Label, a.SSHAlias, a.KeyPath = label, "github-"+label, "~/.ssh/id_"+label
	a.GitUserEmail = label + "@example.com"
	if !global {
		a.Scope = account.Scope{Paths: []string{"~/" + label + "/"}}
	}
	return a
}

func TestMissingFileIsEmptyAndNotCreated(t *testing.T) {
	dir := t.TempDir()
	s := Open(dir, newClock().now)
	if s.ReadOnly() != nil || len(s.Accounts()) != 0 {
		t.Fatal("missing file must be an empty, writable list")
	}
	if _, err := os.Stat(s.Path()); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("opening must not create the file")
	}
}

func TestAddBacksUpOnlyAnExistingFile(t *testing.T) {
	dir := t.TempDir()
	s := Open(dir, newClock().now)
	if err := s.Add(identity("personal", true)); err != nil {
		t.Fatal(err)
	}
	if b, _ := s.Backups(); len(b) != 0 {
		t.Fatalf("first add backed up: %v", b)
	}
	if err := s.Add(identity("work", false)); err != nil {
		t.Fatal(err)
	}
	b, _ := s.Backups()
	if len(b) != 1 {
		t.Fatalf("second add: %v", b)
	}
	old, _ := os.ReadFile(b[0].Path)
	if !strings.Contains(string(old), "personal") || strings.Contains(string(old), "work@") {
		t.Fatalf("backup holds the file before the add:\n%s", old)
	}
	if runtime.GOOS != "windows" {
		if info, _ := os.Stat(s.Path()); info.Mode().Perm() != 0o600 {
			t.Errorf("mode %v", info.Mode())
		}
	}
	reopened := Open(dir, newClock().now)
	if got := reopened.Accounts(); len(got) != 2 || got[0].Label != "personal" || got[1].Label != "work" {
		t.Fatalf("reopened: %+v", got)
	}
}

func TestUpdateDeleteTouch(t *testing.T) {
	c := newClock()
	s := Open(t.TempDir(), c.now)
	a := identity("work", false)
	if err := s.Add(a); err != nil {
		t.Fatal(err)
	}
	a.Notes = "client"
	if err := s.Update(a); err != nil {
		t.Fatal(err)
	}
	got := s.Accounts()[0]
	if got.Notes != "client" || !got.UpdatedAt.Equal(c.t) {
		t.Fatalf("update: %+v", got)
	}
	updated := got.UpdatedAt
	if err := s.Touch(a.ID); err != nil {
		t.Fatal(err)
	}
	got = s.Accounts()[0]
	if got.LastUsedAt == nil || !got.LastUsedAt.Equal(c.t) || !got.UpdatedAt.Equal(updated) {
		t.Fatalf("touch: %+v", got)
	}
	if err := s.Touch("unknown"); err != nil {
		t.Fatal(err)
	}
	if b, _ := s.Backups(); len(b) != 0 {
		t.Fatalf("update and touch must not back up: %v", b)
	}
	if err := s.Update(identity("nobody", false)); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if err := s.Add(a); !errors.Is(err, ErrDuplicate) {
		t.Fatal(err)
	}
	if err := s.Delete(a.ID); err != nil || len(s.Accounts()) != 0 {
		t.Fatal(err)
	}
}

func TestInvalidChangesAreRejected(t *testing.T) {
	s := Open(t.TempDir(), newClock().now)
	if err := s.Add(identity("personal", true)); err != nil {
		t.Fatal(err)
	}
	var p account.Problems
	if err := s.Add(identity("other", true)); !errors.As(err, &p) || p[0].Code != account.SecondGlobal {
		t.Fatalf("second global: %v", err)
	}
	if len(s.Accounts()) != 1 {
		t.Fatal("a rejected change must not be kept")
	}
}

func TestHandEditedProblemsDoNotBlockOtherChanges(t *testing.T) {
	dir := t.TempDir()
	data, _ := account.Encode([]account.Account{identity("a", true), identity("b", true)})
	if err := os.WriteFile(filepath.Join(dir, FileName), data, 0o600); err != nil {
		t.Fatal(err)
	}
	s := Open(dir, newClock().now)
	if err := s.Add(identity("c", false)); err != nil {
		t.Fatalf("unrelated add: %v", err)
	}
	if err := s.Add(identity("d", true)); err == nil {
		t.Fatal("a third global identity is a new problem")
	}
}

func TestDamagedFileIsReadOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	damaged := []byte(`{"version":1,"accounts":[{"label":"x"`)
	if err := os.WriteFile(path, damaged, 0o600); err != nil {
		t.Fatal(err)
	}
	s := Open(dir, newClock().now)
	var ro *ReadOnlyError
	if !errors.As(s.ReadOnly(), &ro) {
		t.Fatalf("read-only: %v", s.ReadOnly())
	}
	for _, err := range []error{s.Add(identity("a", true)), s.Touch("x"), s.ReplaceAll(nil)} {
		if !errors.As(err, &ro) {
			t.Fatalf("write in read-only state: %v", err)
		}
	}
	if got, _ := os.ReadFile(path); string(got) != string(damaged) {
		t.Fatal("damaged file was overwritten")
	}
}

func TestRestoreRepairsAndBacksUpFirst(t *testing.T) {
	dir := t.TempDir()
	s := Open(dir, newClock().now)
	if err := s.Add(identity("personal", true)); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(identity("work", false)); err != nil {
		t.Fatal(err)
	}
	b, _ := s.Backups()
	good := b[0].Path
	if err := os.WriteFile(s.Path(), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	s = Open(dir, newClock().now)
	if err := s.Restore(good); err != nil {
		t.Fatal(err)
	}
	if s.ReadOnly() != nil || len(s.Accounts()) != 1 {
		t.Fatalf("restored: %+v %v", s.Accounts(), s.ReadOnly())
	}
	after, _ := s.Backups()
	if len(after) != 2 {
		t.Fatalf("restore must back up the current file: %v", after)
	}
	if data, _ := os.ReadFile(after[0].Path); string(data) != "{" {
		t.Fatalf("newest backup is the damaged file: %q", data)
	}
}

func TestBackupNamesOrderAndRetention(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, FileName)
	if err := os.WriteFile(src, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	backups := filepath.Join(dir, "backups")
	at := time.Date(2026, 8, 22, 8, 0, 0, 0, time.FixedZone("x", 8*3600))
	var names []string
	for range 3 {
		p, err := Snapshot(src, backups, at)
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, filepath.Base(p))
	}
	want := []string{"accounts.json.20260822-000000", "accounts.json.20260822-000000-1", "accounts.json.20260822-000000-2"}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("names %v", names)
		}
	}
	os.WriteFile(filepath.Join(backups, "accounts.json.bak"), nil, 0o600)
	os.WriteFile(filepath.Join(backups, "accounts.json.20260822-00000"), nil, 0o600)
	os.WriteFile(filepath.Join(backups, "other.json.20260822-000000"), nil, 0o600)
	list, _ := List(backups, FileName)
	if len(list) != 3 || filepath.Base(list[0].Path) != want[2] || filepath.Base(list[2].Path) != want[0] {
		t.Fatalf("list %v", list)
	}
	for i := range 12 {
		if _, err := Snapshot(src, backups, at.Add(time.Duration(i+1)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	list, _ = List(backups, FileName)
	if len(list) != KeepBackups || !list[0].Time.Equal(at.Add(12*time.Minute)) {
		t.Fatalf("kept %d, newest %v", len(list), list[0].Time)
	}
}

func TestBackupsRefuseSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic links need privileges on Windows")
	}
	dir := t.TempDir()
	real := filepath.Join(dir, "real.json")
	os.WriteFile(real, []byte("{}"), 0o600)
	link := filepath.Join(dir, FileName)
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Snapshot(link, filepath.Join(dir, "b"), time.Now()); !errors.Is(err, fsutil.ErrSymlink) {
		t.Fatalf("snapshot of a link: %v", err)
	}
	if err := restoreFile(real, link); !errors.Is(err, fsutil.ErrSymlink) {
		t.Fatalf("restore over a link: %v", err)
	}
	if err := DeleteBackup(link); !errors.Is(err, fsutil.ErrSymlink) {
		t.Fatalf("delete a link: %v", err)
	}
}
