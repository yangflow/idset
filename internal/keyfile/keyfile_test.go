package keyfile

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/paths"
)

func setup(t *testing.T) (paths.Env, string) {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(home, ".ssh"), 0o700); err != nil {
		t.Fatal(err)
	}
	return paths.Env{Home: home, Windows: runtime.GOOS == "windows"}, home
}

func write(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("not a real key"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func identity(id, label, key string) account.Account {
	return account.Account{ID: id, Label: label, KeyPath: key}
}

func TestInspectReasons(t *testing.T) {
	env, home := setup(t)
	write(t, filepath.Join(home, ".ssh", "id_work"))
	write(t, filepath.Join(home, ".ssh", "id_shared"))
	if err := os.Mkdir(filepath.Join(home, ".ssh", "dir"), 0o700); err != nil {
		t.Fatal(err)
	}
	list := []account.Account{
		identity("a", "work", "~/.ssh/id_work"),
		identity("b", "none", ""),
		identity("c", "one", "~/.ssh/id_shared"),
		identity("d", "two", filepath.ToSlash(filepath.Join(home, ".ssh", "id_shared"))),
		identity("e", "gone", "~/.ssh/id_gone"),
		identity("f", "dir", "~/.ssh/dir"),
	}
	cases := map[string]Reason{"a": None, "b": NoPath, "c": Shared, "d": Shared, "e": Missing, "f": NotFile, "zz": NoPath}
	for id, want := range cases {
		if got, _ := Inspect(env, list, id); got != want {
			t.Errorf("%s: %d, want %d", id, got, want)
		}
	}
	if _, other := Inspect(env, list, "c"); other != "two" {
		t.Errorf("shared with %q", other)
	}
}

func TestInspectSymlink(t *testing.T) {
	env, home := setup(t)
	target := filepath.Join(home, ".ssh", "real")
	write(t, target)
	if err := os.Symlink(target, filepath.Join(home, ".ssh", "link")); err != nil {
		t.Skip("symbolic links are not available:", err)
	}
	list := []account.Account{identity("a", "linked", "~/.ssh/link")}
	if got, _ := Inspect(env, list, "a"); got != Symlink {
		t.Fatalf("got %d", got)
	}
	if err := Remove(env, nil, "~/.ssh/link"); err == nil {
		t.Fatal("a symbolic link must not be removed")
	}
	if _, err := os.Lstat(target); err != nil {
		t.Fatal("the target must stay")
	}
}

func TestRemoveDeletesOnlyThePrivateKey(t *testing.T) {
	env, home := setup(t)
	key := filepath.Join(home, ".ssh", "id_work")
	write(t, key)
	write(t, key+".pub")
	if err := Remove(env, nil, "~/.ssh/id_work"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(key); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the key must be gone: %v", err)
	}
	if _, err := os.Lstat(key + ".pub"); err != nil {
		t.Fatalf("the .pub file must stay: %v", err)
	}
	var refused *RefusedError
	if err := Remove(env, nil, "~/.ssh/id_work"); !errors.As(err, &refused) || refused.Reason != Missing {
		t.Fatalf("a second removal: %v", err)
	}
	write(t, key)
	err := Remove(env, []account.Account{identity("x", "other", "~/.ssh/id_work")}, "~/.ssh/id_work")
	if !errors.As(err, &refused) || refused.Reason != Shared || refused.Error() == "" {
		t.Fatalf("a shared key: %v", err)
	}
}
