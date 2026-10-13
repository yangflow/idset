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
	"github.com/yangflow/idset/internal/apply"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/project"
)

// rig is a store wired to a temporary home, with a fixed generation time so that unchanged content means
// unchanged bytes.
type rig struct {
	t      *testing.T
	target apply.Target
	s      *Store
}

func newRig(t *testing.T) *rig {
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	env := paths.Env{Home: home, Windows: runtime.GOOS == "windows"}
	target := apply.DefaultTarget(env)
	fixed := func() time.Time { return time.Date(2026, 8, 22, 8, 0, 0, 0, time.UTC) }
	s := Open(target.Dir, newClock().now, WithRegenerate(apply.Regenerate(env, target, fixed)))
	return &rig{t: t, target: target, s: s}
}

func (r *rig) read(path string) string {
	r.t.Helper()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	if err != nil {
		r.t.Fatal(err)
	}
	return string(data)
}

func (r *rig) managed(name string) string { return r.read(filepath.Join(r.target.Dir, name)) }

func (r *rig) subs() []string {
	m, _ := filepath.Glob(filepath.Join(r.target.Dir, "gitconfig-*.managed"))
	return m
}

func (r *rig) checkBlocks() {
	r.t.Helper()
	ssh, git := r.read(r.target.SSHConfig), r.read(r.target.GitConfig)
	if !strings.HasPrefix(ssh, "# --- idset managed ---\n") || !strings.Contains(ssh, project.SSHFile) {
		r.t.Errorf("ssh config lacks the block:\n%s", ssh)
	}
	if !strings.HasSuffix(git, "# --- idset managed end ---\n") || !strings.Contains(git, project.GitFile) {
		r.t.Errorf("git config lacks the block:\n%s", git)
	}
}

func TestEveryChangeRegenerates(t *testing.T) {
	r := newRig(t)
	if r.managed(project.SSHFile) != "" {
		t.Fatal("opening must not generate")
	}
	personal, work := identity("personal", true), identity("work", false)
	personal.GitUserName, work.GitUserName = "Alice", "Bob"

	if err := r.s.Add(personal); err != nil {
		t.Fatal(err)
	}
	r.checkBlocks()
	if got := r.managed(project.SSHFile); !strings.Contains(got, "Host github-personal") {
		t.Fatalf("add: ssh managed:\n%s", got)
	}
	if got := r.managed(project.GitFile); !strings.Contains(got, "name = Alice") {
		t.Fatalf("add: git managed:\n%s", got)
	}

	if err := r.s.Add(work); err != nil {
		t.Fatal(err)
	}
	if got := r.managed(project.SubFile(work.ID)); len(r.subs()) != 1 || !strings.Contains(got, "work@example.com") {
		t.Fatalf("add: sub files %v\n%s", r.subs(), got)
	}

	work.GitUserEmail = "bob@example.com"
	if err := r.s.Update(work); err != nil {
		t.Fatal(err)
	}
	if got := r.managed(project.SubFile(work.ID)); !strings.Contains(got, "bob@example.com") {
		t.Fatalf("update: sub file:\n%s", got)
	}

	if err := r.s.Delete(work.ID); err != nil {
		t.Fatal(err)
	}
	if len(r.subs()) != 0 || strings.Contains(r.managed(project.SSHFile), "github-work") {
		t.Fatalf("delete left files %v", r.subs())
	}

	if err := r.s.ReplaceAll([]account.Account{personal, work}); err != nil {
		t.Fatal(err)
	}
	if len(r.subs()) != 1 || !strings.Contains(r.managed(project.SSHFile), "github-work") {
		t.Fatalf("replace all: %v", r.subs())
	}

	b, _ := r.s.Backups()
	// The oldest backup was taken by the second Add and holds only the personal identity.
	if err := r.s.Restore(b[len(b)-1].Path); err != nil {
		t.Fatal(err)
	}
	if len(r.subs()) != 0 || strings.Contains(r.managed(project.SSHFile), "github-work") {
		t.Fatalf("restore: %v", r.subs())
	}
	r.checkBlocks()
}

func TestTouchDoesNotRegenerate(t *testing.T) {
	r := newRig(t)
	a := identity("personal", true)
	if err := r.s.Add(a); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(r.target.Dir, project.SSHFile)); err != nil {
		t.Fatal(err)
	}
	if err := r.s.Touch(a.ID); err != nil {
		t.Fatal(err)
	}
	if r.managed(project.SSHFile) != "" {
		t.Fatal("touch regenerated")
	}
}

func TestUnchangedContentKeepsBytes(t *testing.T) {
	r := newRig(t)
	if err := r.s.Add(identity("personal", true)); err != nil {
		t.Fatal(err)
	}
	managed := []string{filepath.Join(r.target.Dir, project.SSHFile), filepath.Join(r.target.Dir, project.GitFile)}
	user := []string{r.target.SSHConfig, r.target.GitConfig}
	before, stamps := map[string]string{}, map[string]time.Time{}
	for _, f := range append(managed, user...) {
		before[f] = r.read(f)
		info, _ := os.Stat(f)
		stamps[f] = info.ModTime()
	}
	if err := r.s.ReplaceAll(r.s.Accounts()); err != nil {
		t.Fatal(err)
	}
	for f, want := range before {
		if r.read(f) != want {
			t.Errorf("%s changed", f)
		}
	}
	for _, f := range user { // the user files are not even rewritten
		if info, _ := os.Stat(f); !info.ModTime().Equal(stamps[f]) {
			t.Errorf("%s rewritten", f)
		}
	}
}

func TestUserContentSurvivesRegeneration(t *testing.T) {
	r := newRig(t)
	os.MkdirAll(filepath.Dir(r.target.SSHConfig), 0o700)
	os.WriteFile(r.target.SSHConfig, []byte("Host box\n\tHostName 192.0.2.1\n"), 0o600)
	os.WriteFile(r.target.GitConfig, []byte("[core]\n\teditor = vim\n"), 0o600)
	if err := r.s.Add(identity("personal", true)); err != nil {
		t.Fatal(err)
	}
	r.checkBlocks()
	if !strings.HasSuffix(r.read(r.target.SSHConfig), "Host box\n\tHostName 192.0.2.1\n") ||
		!strings.HasPrefix(r.read(r.target.GitConfig), "[core]\n\teditor = vim\n") {
		t.Fatal("user content lost")
	}
}

func TestReadOnlyWritesNothing(t *testing.T) {
	r := newRig(t)
	os.MkdirAll(r.target.Dir, 0o700)
	os.WriteFile(r.s.Path(), []byte("{"), 0o600)
	s := Open(r.target.Dir, newClock().now, WithRegenerate(func([]account.Account) error {
		t.Error("regenerate called in the read-only state")
		return nil
	}))
	var ro *ReadOnlyError
	if err := s.Add(identity("personal", true)); !errors.As(err, &ro) {
		t.Fatalf("add: %v", err)
	}
	if err := s.ReplaceAll(nil); !errors.As(err, &ro) {
		t.Fatalf("replace: %v", err)
	}
	bad := filepath.Join(t.TempDir(), "bad")
	os.WriteFile(bad, []byte("{"), 0o600)
	if err := s.Restore(bad); !errors.As(err, &ro) {
		t.Fatalf("restoring a damaged backup must stay read-only: %v", err)
	}
	if r.managed(project.SSHFile) != "" || r.read(r.target.SSHConfig) != "" {
		t.Fatal("files written")
	}
}

func TestRegenerateFailureIsReturnedAndChangeStands(t *testing.T) {
	boom := errors.New("disk full")
	dir := t.TempDir()
	s := Open(dir, newClock().now, WithRegenerate(func([]account.Account) error { return boom }))
	err := s.Add(identity("personal", true))
	var re *RegenerateError
	if !errors.As(err, &re) || !errors.Is(err, boom) {
		t.Fatalf("got %v", err)
	}
	if got := Open(dir, newClock().now).Accounts(); len(got) != 1 {
		t.Fatalf("the saved list must stand: %+v", got)
	}
}

func TestRejectedChangeDoesNotRegenerate(t *testing.T) {
	calls := 0
	s := Open(t.TempDir(), newClock().now, WithRegenerate(func([]account.Account) error { calls++; return nil }))
	a := identity("personal", true)
	if err := s.Add(a); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(a); !errors.Is(err, ErrDuplicate) {
		t.Fatal(err)
	}
	if err := s.Delete("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("regenerate ran %d times", calls)
	}
}
