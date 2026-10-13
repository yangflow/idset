package gitdir

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/paths"
)

var unix = paths.Env{Home: "/home/ada"}

func TestNormalize(t *testing.T) {
	for raw, want := range map[string]string{
		"":                 "",
		"   ":              "",
		" ~/work ":         "~/work/",
		"~/work/":          "~/work/",
		"/home/ada/src":    "~/src/",
		"/opt/repos":       "/opt/repos/",
		"~":                "~/",
		`C:\not\windows\x`: `C:/not/windows/x/`,
	} {
		if got := Normalize(raw, unix); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", raw, got, want)
		}
	}
	win := paths.Env{Home: `C:\Users\Ada`, Windows: true}
	if got := Normalize(`C:\Users\Ada\Work`, win); got != "~/Work/" {
		t.Errorf("Windows: %q", got)
	}
}

func scoped(paths ...string) account.Account {
	return account.Account{Scope: account.Scope{Paths: paths}}
}

func TestBlocksPutTheMostSpecificLast(t *testing.T) {
	list := []account.Account{
		scoped("~/work/client/", "~/work/client"),
		{Scope: account.Scope{Global: true}},
		scoped("~/"),
		scoped("~/work/", "  ", "/opt/x/"),
		scoped("~/src/"),
	}
	var got []string
	for _, b := range Blocks(list, unix) {
		got = append(got, b.Path)
	}
	want := []string{"/opt/x/", "~/", "~/src/", "~/work/", "~/work/client/"}
	if len(got) != len(want) {
		t.Fatalf("blocks %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("blocks %v, want %v", got, want)
		}
	}
}

func TestBlocksBreakTiesByStorageOrder(t *testing.T) {
	list := []account.Account{scoped("~/b/"), scoped("~/a/"), scoped("~/b/")}
	b := Blocks(list, unix)
	if b[0].Index != 0 || b[1].Index != 1 || b[2].Index != 2 {
		t.Fatalf("ties: %+v", b)
	}
}

func TestMatches(t *testing.T) {
	for _, c := range []struct {
		root, path string
		want       bool
	}{
		{"/home/ada/work/api", "~/work/", true},
		{"/home/ada/work", "~/work/", true},
		{"/home/ada/workshop", "~/work/", false},
		{"/home/ada/work/api", "~/", true},
		{"/home/ada/work/api", "", false},
		{"/home/ada/Work/api", "~/work/", false},
	} {
		if got := Matches(c.root, c.path, unix); got != c.want {
			t.Errorf("Matches(%q, %q) = %v", c.root, c.path, got)
		}
	}
	win := paths.Env{Home: `C:\Users\Ada`, Windows: true}
	if !Matches(`c:\users\ada\work\api`, "~/Work/", win) {
		t.Error("Windows ignores case")
	}
	if Keyword(win) != "gitdir/i:" || Keyword(unix) != "gitdir:" {
		t.Error("keywords")
	}
}

func TestMatchesThroughSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic links need privileges on Windows")
	}
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	if err := os.MkdirAll(filepath.Join(real, "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	env := paths.Env{Home: dir}
	if !Matches(filepath.Join(real, "api"), "~/link/", env) || !Matches(filepath.Join(dir, "link", "api"), "~/real/", env) {
		t.Error("a linked path and its target must match each other")
	}
}

func TestMatchingReturnsWriteOrder(t *testing.T) {
	list := []account.Account{scoped("~/work/api/"), scoped("~/"), scoped("~/work/")}
	m := Matching(Blocks(list, unix), "/home/ada/work/api", unix)
	if len(m) != 3 || m[len(m)-1].Index != 0 {
		t.Fatalf("matching: %+v", m)
	}
}

func TestExactIsTheFolderItselfNotAParent(t *testing.T) {
	for _, c := range []struct {
		root, path string
		want       bool
	}{
		{"/home/ada/work/api", "~/work/api/", true},
		{"/home/ada/work/api", "~/work/api", true},
		{"/home/ada/work/api", "/home/ada/work/api/", true},
		{"/home/ada/work/api", "~/work/", false},
		{"/home/ada/work", "~/work/api/", false},
		{"/home/ada/work/api", "", false},
		{"/home/ada/Work/api", "~/work/api/", false},
	} {
		if got := Exact(c.root, c.path, unix); got != c.want {
			t.Errorf("Exact(%q, %q) = %v", c.root, c.path, got)
		}
	}
	win := paths.Env{Home: `C:\Users\Ada`, Windows: true}
	for _, path := range []string{"~/Work/API/", `C:\users\ada\work\api`, "c:/Users/Ada/work/api/"} {
		if !Exact(`c:\users\ada\work\api`, path, win) {
			t.Errorf("Windows ignores case and slash form: %q", path)
		}
	}
	if Exact(`D:\work\api`, "~/work/api/", win) {
		t.Error("another drive is another folder")
	}
}
