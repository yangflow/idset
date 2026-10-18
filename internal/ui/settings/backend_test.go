package settings

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/apply"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/store"
	"github.com/yangflow/idset/internal/sysexec"
)

// noGit is a git with nothing configured: every call exits 1, as git config does for a missing key.
func noGit(context.Context, []string, string, ...string) (sysexec.Result, error) {
	return sysexec.Result{ExitCode: 1}, nil
}

// importDisk is a home with two hosts in ~/.ssh/config and an empty store under it.
func importDisk(t *testing.T) Backend {
	t.Helper()
	env := paths.Env{Home: t.TempDir()}
	if err := os.MkdirAll(filepath.Join(env.Home, ".ssh"), 0o700); err != nil {
		t.Fatal(err)
	}
	config := "Host gh-work\n  HostName github.com\n  IdentityFile ~/.ssh/id_work\n\n" +
		"Host gl-own\n  HostName gitlab.com\n  IdentityFile ~/.ssh/id_own\n"
	if err := os.WriteFile(filepath.Join(env.Home, ".ssh", "config"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	target := apply.DefaultTarget(env)
	now := func() time.Time { return base }
	return Backend{Store: store.Open(target.Dir, now, store.WithRegenerate(apply.Regenerate(env, target, now))), Env: env}
}

func TestBackendImportsWhatItScanned(t *testing.T) {
	b := importDisk(t)
	cands, err := b.Scan(context.Background(), noGit)
	if err != nil || len(cands) != 2 {
		t.Fatalf("%+v, %v", cands, err)
	}
	list := []account.Account{cands[0].Account, cands[1].Account}
	added, err := b.Import(list, base.Add(1500*time.Millisecond))
	if err != nil || len(added) != 2 {
		t.Fatalf("%+v, %v", added, err)
	}
	got := b.Store.Accounts()
	if len(got) != 2 || got[0].ID != added[0].ID || got[1].ID != added[1].ID || got[0].ID == got[1].ID {
		t.Fatalf("stored %+v, added %+v", got, added)
	}
	if got[0].Color != account.ColorPresets[0] || got[1].Color != account.ColorPresets[1] {
		t.Errorf("colors %q, %q", got[0].Color, got[1].Color)
	}
	if want := base.Add(time.Second); !got[0].CreatedAt.Equal(want) || !got[0].UpdatedAt.Equal(want) {
		t.Errorf("times %v, %v", got[0].CreatedAt, got[0].UpdatedAt)
	}
	backups, err := b.Store.Backups()
	if err != nil || len(backups) == 0 {
		t.Errorf("backups %d, %v", len(backups), err)
	}

	again, err := b.Scan(context.Background(), noGit)
	if err != nil || len(again) != 2 || !again[0].Exists || !again[1].Exists {
		t.Fatalf("after importing %+v, %v", again, err)
	}
}

func TestBackendImportStopsAtTheFirstFailure(t *testing.T) {
	b := importDisk(t)
	cands, err := b.Scan(context.Background(), noGit)
	if err != nil || len(cands) != 2 {
		t.Fatalf("%+v, %v", cands, err)
	}
	// The second gh-work takes an alias already taken, so validation refuses it and gl-own is not tried.
	added, err := b.Import([]account.Account{cands[0].Account, cands[0].Account, cands[1].Account}, base)
	if err == nil || len(added) != 1 || len(b.Store.Accounts()) != 1 {
		t.Fatalf("added %+v, %v", added, err)
	}
}

func TestBackendImportCountsTheSavedWhenRegeneratingFails(t *testing.T) {
	b := importDisk(t)
	cands, err := b.Scan(context.Background(), noGit)
	if err != nil {
		t.Fatal(err)
	}
	fail := func([]account.Account) error { return errors.New("disk full") }
	b.Store = store.Open(apply.DefaultTarget(b.Env).Dir, time.Now, store.WithRegenerate(fail))
	added, err := b.Import([]account.Account{cands[0].Account, cands[1].Account}, base)
	var regenerate *store.RegenerateError
	if !errors.As(err, &regenerate) || len(added) != 1 || len(b.Store.Accounts()) != 1 {
		t.Fatalf("added %+v, %v", added, err)
	}
}
