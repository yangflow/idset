package bind

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/match"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/store"
)

type rig struct {
	t           *testing.T
	env         paths.Env
	store       *store.Store
	regenerated int
	regenErr    error
	runner      Runner
	dir         string
}

// uid is the valid ID of a test identity, which the tests name by its label.
func uid(name string) string {
	return "00000000-0000-4000-8000-" + fmt.Sprintf("%012x", []byte(name))[:12]
}

// valid makes a bare test identity one the store accepts.
func valid(a account.Account) account.Account {
	full := account.New(now)
	full.ID, full.Label, full.Scope = uid(a.ID), a.ID, a.Scope
	return full
}

func newRig(t *testing.T, list ...account.Account) *rig {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r := &rig{t: t, env: paths.Env{Home: home, Windows: runtime.GOOS == "windows"}}
	clock := time.Date(2026, 8, 22, 8, 0, 0, 0, time.UTC)
	tick := func() time.Time { clock = clock.Add(time.Second); return clock }
	r.store = store.Open(filepath.Join(home, ".config", "idset"), tick, store.WithRegenerate(func([]account.Account) error {
		r.regenerated++
		return r.regenErr
	}))
	for _, a := range list {
		if err := r.store.Add(valid(a)); err != nil {
			t.Fatal(err)
		}
	}
	r.regenerated = 0
	r.dir = filepath.Join(home, "scratch", "notes")
	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		t.Fatal(err)
	}
	git := func(_ context.Context, dir string, args ...string) (string, int, error) {
		if args[0] == "rev-parse" {
			return dir, 0, nil
		}
		return "", 1, nil
	}
	r.runner = Runner{Store: r.store, Env: r.env, Git: git, Now: tick}
	return r
}

func (r *rig) run(op Op) Outcome {
	r.t.Helper()
	op.Dir, op.Root = r.dir, paths.Slash(r.dir)
	op.From, op.To = uid(op.From), uid(op.To)
	return r.runner.Run(context.Background(), op)
}

func (r *rig) account(id string) account.Account {
	for _, a := range r.store.Accounts() {
		if a.ID == uid(id) {
			return a
		}
	}
	r.t.Fatalf("no %s", id)
	return account.Account{}
}

func TestBindWritesOnceAndMatchesAgain(t *testing.T) {
	r := newRig(t, scoped("work", "~/work/"), scoped("client", "~/c/"))
	out := r.run(Op{Kind: KindBind, To: "client"})
	if out.Err != nil || r.regenerated != 1 {
		t.Fatalf("%v, regenerated %d", out.Err, r.regenerated)
	}
	if got := r.account("client").Scope.Paths; !reflect.DeepEqual(got, []string{"~/c/", "~/scratch/notes/"}) {
		t.Fatalf("%v", got)
	}
	if r.account("client").LastUsedAt == nil || r.account("work").LastUsedAt != nil {
		t.Fatal("the bound identity is used")
	}
	if out.Result.Kind != match.Matched || out.Result.ID != uid("client") {
		t.Fatalf("the folder is matched again: %+v", out.Result)
	}
	if len(out.Undo) != 1 || out.Undo[0].ID != uid("client") || len(out.Undo[0].Scope.Paths) != 1 {
		t.Fatalf("undo %+v", out.Undo)
	}
}

func TestCoveredBindWritesNothing(t *testing.T) {
	r := newRig(t, scoped("work", "~/scratch/"))
	before := r.store.Accounts()
	out := r.run(Op{Kind: KindBind, To: "work"})
	if out.Err != nil || r.regenerated != 0 || out.CoveredID != uid("work") || out.CoveredBy != "~/scratch/" || out.Undo != nil {
		t.Fatalf("%+v regenerated %d", out, r.regenerated)
	}
	if !reflect.DeepEqual(r.store.Accounts(), before) {
		t.Fatal("no change")
	}
	if out.Result.ID != uid("work") {
		t.Fatalf("the folder is still looked at: %+v", out.Result)
	}
}

func TestRebindRegeneratesOnce(t *testing.T) {
	r := newRig(t, scoped("work", "~/scratch/notes/"), scoped("client", "~/c/"))
	out := r.run(Op{Kind: KindRebind, From: "work", To: "client"})
	if out.Err != nil || r.regenerated != 1 {
		t.Fatalf("%v, regenerated %d", out.Err, r.regenerated)
	}
	if len(r.account("work").Scope.Paths) != 0 || len(r.account("client").Scope.Paths) != 2 {
		t.Fatalf("%+v %+v", r.account("work").Scope, r.account("client").Scope)
	}
	if out.Result.ID != uid("client") || len(out.Undo) != 2 {
		t.Fatalf("%+v", out)
	}
}

func TestUnbindHasNoUndoAndRemovesOnlyThisPath(t *testing.T) {
	r := newRig(t, scoped("work", "~/scratch/notes/", "~/scratch/"))
	out := r.run(Op{Kind: KindUnbind, From: "work"})
	if out.Err != nil || r.regenerated != 1 || out.Undo != nil {
		t.Fatalf("%+v regenerated %d", out, r.regenerated)
	}
	if got := r.account("work").Scope.Paths; !reflect.DeepEqual(got, []string{"~/scratch/"}) {
		t.Fatalf("%v", got)
	}
	if out.Result.Kind != match.Matched || out.Result.ID != uid("work") {
		t.Fatalf("a parent still covers: %+v", out.Result)
	}
	if r.account("work").LastUsedAt == nil {
		t.Fatal("unbinding is a use")
	}
	if again := r.run(Op{Kind: KindUnbind, From: "work"}); again.Err != nil || r.regenerated != 1 {
		t.Fatalf("a second unbind has nothing to do: %v %d", again.Err, r.regenerated)
	}
}

func TestUseOnlyAndRemovePathAndTheirUndo(t *testing.T) {
	r := newRig(t, scoped("work", "~/"), scoped("client", "~/scratch/"))
	original := r.store.Accounts()
	out := r.run(Op{Kind: KindUseOnly, To: "client"})
	if out.Err != nil || r.regenerated != 1 || len(r.account("client").Scope.Paths) != 2 {
		t.Fatalf("%+v %d", out, r.regenerated)
	}
	undo := r.run(Op{Kind: KindUndo, Before: out.Undo})
	if undo.Err != nil || r.regenerated != 2 || undo.Undo != nil {
		t.Fatalf("%+v %d", undo, r.regenerated)
	}
	if !reflect.DeepEqual(r.store.Accounts(), original) {
		t.Fatal("undo restores the identities exactly, UpdatedAt and LastUsedAt included")
	}
	out = r.run(Op{Kind: KindRemovePath, From: "work", Path: "~/"})
	if out.Err != nil || out.Undo != nil || len(r.account("work").Scope.Paths) != 0 {
		t.Fatalf("%+v", out)
	}
}

func TestRegenerateErrorStandsAndPlainErrorsStop(t *testing.T) {
	r := newRig(t, scoped("work", "~/w/"))
	r.regenErr = errors.New("disk full")
	out := r.run(Op{Kind: KindBind, To: "work"})
	var regen *store.RegenerateError
	if !errors.As(out.Err, &regen) || !out.Written() || len(r.account("work").Scope.Paths) != 2 || out.Result.ID != uid("work") || out.Undo == nil {
		t.Fatalf("the change stands: %+v", out)
	}
	if out := r.run(Op{Kind: KindBind, To: "nobody"}); !errors.Is(out.Err, ErrNotFound) || out.Written() || out.Result.Dir != "" {
		t.Fatalf("an unknown identity: %+v", out)
	}
	if out := r.run(Op{Kind: KindUndo, Before: []account.Account{valid(scoped("gone"))}}); !errors.Is(out.Err, store.ErrNotFound) {
		t.Fatalf("undo of a deleted identity: %+v", out)
	}
}

func TestReadOnlyStoreRefuses(t *testing.T) {
	r := newRig(t, scoped("work", "~/w/"))
	if err := os.WriteFile(r.store.Path(), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	r.store.Reload()
	var ro *store.ReadOnlyError
	if out := r.run(Op{Kind: KindBind, To: "work"}); !errors.As(out.Err, &ro) {
		t.Fatalf("%+v", out)
	}
}
