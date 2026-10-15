package match

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/paths"
)

func named(label string, scope ...string) account.Account {
	a := identity(label, scope...)
	a.Label = label
	return a
}

// existing reports every path as gone but the ones listed.
func existing(present ...string) func(string) bool {
	return func(p string) bool {
		for _, q := range present {
			if strings.TrimSuffix(paths.Slash(p), "/") == q {
				return false
			}
		}
		return true
	}
}

func TestFindStaleCandidateRules(t *testing.T) {
	env := paths.Env{Home: "/home/ada"}
	root := "/home/ada/projects/api"
	for _, c := range []struct {
		name    string
		list    []account.Account
		root    string
		present []string
		want    *Stale
	}{
		{"rename: same parent", []account.Account{named("work", "~/projects/api-old/")}, root, nil, &Stale{"work", "~/projects/api-old/"}},
		{"move: same last folder name", []account.Account{named("work", "~/Documents/archive/api/")}, root, nil, &Stale{"work", "~/Documents/archive/api/"}},
		{"move ignores case", []account.Account{named("work", "~/Documents/API")}, root, nil, &Stale{"work", "~/Documents/API"}},
		{"a path that exists is never stale", []account.Account{named("work", "~/projects/api-old/")}, root, []string{"/home/ada/projects/api-old"}, nil},
		{"unrelated gone path", []account.Account{named("work", "~/elsewhere/other/")}, root, nil, nil},
		{"a global identity has no path", []account.Account{named("home")}, root, nil, nil},
		{"the folder's own path is no candidate", []account.Account{named("work", "~/projects/api/")}, root, nil, nil},
		{"parent is /", []account.Account{named("work", "/old/")}, "/srv", nil, nil},
		{"the home directory", []account.Account{named("work", "~/src/ada/")}, "/home/ada", nil, nil},
		{"blank paths are skipped", []account.Account{named("work", "  ", "")}, root, nil, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := FindStale(c.list, c.root, env, existing(c.present...), same)
			if (got == nil) != (c.want == nil) || got != nil && *got != *c.want {
				t.Fatalf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestFindStaleOrder(t *testing.T) {
	env := paths.Env{Home: "/home/ada"}
	root := "/home/ada/projects/api"
	for _, c := range []struct {
		name string
		list []account.Account
		want Stale
	}{
		{"a rename beats a move, whatever the order", []account.Account{named("a", "~/old/api/"), named("b", "~/projects/api-old/")}, Stale{"b", "~/projects/api-old/"}},
		{"an identity named like the folder comes first", []account.Account{named("a", "~/projects/x/"), named("API", "~/projects/y/")}, Stale{"API", "~/projects/y/"}},
		{"then the stored order", []account.Account{named("a", "~/projects/x/"), named("b", "~/projects/y/")}, Stale{"a", "~/projects/x/"}},
		{"then the path", []account.Account{named("a", "~/projects/y/", "~/projects/x/")}, Stale{"a", "~/projects/x/"}},
		{"naming outranks the stored order only within a kind", []account.Account{named("api", "~/old/api/"), named("b", "~/projects/y/")}, Stale{"b", "~/projects/y/"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := FindStale(c.list, root, env, existing(), same); got == nil || *got != c.want {
				t.Fatalf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestFindStaleOnWindows(t *testing.T) {
	env := paths.Env{Home: `C:\Users\Ada`, Windows: true}
	list := []account.Account{named("work", "~/Projects/Api-Old/", "d:/code/API/")}
	got := FindStale(list, `c:\users\ada\projects\api`, env, existing(), same)
	if got == nil || got.Path != "~/Projects/Api-Old/" {
		t.Fatalf("a rename is found with another case and backslashes: %+v", got)
	}
	got = FindStale(list[:1], `D:\work\web`, env, existing(), same)
	if got != nil {
		t.Fatalf("another drive and another name: %+v", got)
	}
	got = FindStale([]account.Account{named("work", "d:/code/API/")}, `E:\src\api`, env, existing(), same)
	if got == nil || got.Path != "d:/code/API/" {
		t.Fatalf("a move to another drive: %+v", got)
	}
	if got := FindStale([]account.Account{named("work", "d:/old/")}, `D:\proj`, env, existing(), same); got != nil {
		t.Fatalf("a folder directly under a drive has no candidate: %+v", got)
	}
	if got := FindStale([]account.Account{named("work", "~/x/")}, `C:\Users\Ada`, env, existing(), same); got != nil {
		t.Fatalf("the home directory: %+v", got)
	}
}

func TestFindStaleHomeThroughASymbolicLink(t *testing.T) {
	env := paths.Env{Home: "/home/ada", RealHome: "/data/ada"}
	if got := FindStale([]account.Account{named("work", "~/x/")}, "/data/ada", env, existing(), same); got != nil {
		t.Fatalf("the resolved home directory is the home directory: %+v", got)
	}
}

func TestGoneAndResolveReadTheDisk(t *testing.T) {
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	env := paths.Env{Home: home, Windows: runtime.GOOS == "windows"}
	root := filepath.Join(home, "projects", "api")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, "projects", "kept"), 0o755); err != nil {
		t.Fatal(err)
	}
	if Gone(root) || !Gone(filepath.Join(home, "projects", "api-old")) {
		t.Fatal("Gone follows the disk")
	}
	git := fakeGit{"rev-parse --show-toplevel": {paths.Slash(root), 0}}
	list := []account.Account{named("work", "~/projects/api-old/", "~/projects/kept/")}
	r, err := Resolve(context.Background(), git.run, root, list, env)
	if err != nil || r.Kind != NoMatch || r.Stale == nil || r.Stale.Path != "~/projects/api-old/" {
		t.Fatalf("%+v %v", r, err)
	}
	if r, _ := Resolve(context.Background(), git.run, root, []account.Account{named("work", "~/projects/kept/")}, env); r.Stale != nil {
		t.Fatalf("an existing path is never stale: %+v", r.Stale)
	}
	if r, _ := Resolve(context.Background(), git.run, root, append(list, named("home")), env); r.Kind != Matched || r.Stale != nil {
		t.Fatalf("only a result without a match asks: %+v", r)
	}
}

// linkedHome is a home directory reached through a symbolic link, as /tmp is on macOS, and its resolved form.
func linkedHome(t *testing.T) (env paths.Env, link, real string) {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	real = filepath.Join(base, "real", "home")
	if err := os.MkdirAll(filepath.Join(real, "projects", "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	link = filepath.Join(base, "link")
	if err := os.Symlink(filepath.Join(base, "real"), link); err != nil {
		t.Skip("symbolic links are not available:", err)
	}
	link = filepath.Join(link, "home")
	return paths.Env{Home: link, Windows: runtime.GOOS == "windows"}, link, real
}

func TestFindStaleThroughASymbolicLinkedHome(t *testing.T) {
	env, _, real := linkedHome(t)
	root := paths.Slash(filepath.Join(real, "projects", "api"))
	for _, c := range []struct{ name, stored string }{
		{"rename", "~/projects/api-old/"},
		{"move", "~/archive/api/"},
	} {
		got := FindStale([]account.Account{named("work", c.stored)}, root, env, Gone, Canon)
		if got == nil || got.Path != c.stored {
			t.Errorf("%s: %+v", c.name, got)
		}
	}
	// The dropped folder in the form the user wrote it, through the link, matches the same way.
	linked := paths.Slash(filepath.Join(env.Home, "projects", "api"))
	if got := FindStale([]account.Account{named("work", "~/projects/api-old/")}, linked, env, Gone, Canon); got == nil {
		t.Error("the folder through the link")
	}
	// A path that exists is never stale, through the link too.
	if got := FindStale([]account.Account{named("work", "~/projects/api/")}, root, env, Gone, Canon); got != nil {
		t.Errorf("%+v", got)
	}
}

func TestFindStaleHomeExclusionThroughASymbolicLink(t *testing.T) {
	env, _, real := linkedHome(t)
	list := []account.Account{named("work", "~/x/home/")}
	if got := FindStale(list, paths.Slash(real), env, Gone, Canon); got != nil {
		t.Errorf("the resolved home directory: %+v", got)
	}
	if got := FindStale(list, paths.Slash(env.Home), env, Gone, Canon); got != nil {
		t.Errorf("the home directory as written: %+v", got)
	}
}

// same resolves nothing: paths that are not on the disk compare the same on every platform.
func same(p string) string { return strings.TrimSuffix(paths.Slash(p), "/") }

func TestIsTopRecognisesEveryRootForm(t *testing.T) {
	for dir, want := range map[string]bool{
		"/": true, "": true, "D:": true, "D:/": true, `d:\`: true, "c:": true, "//server/share": true, `\\server\share\`: true,
		"/home": false, "D:/work": false, "//server/share/dir": false,
	} {
		if got := isTop(dir); got != want {
			t.Errorf("%q: %v", dir, got)
		}
	}
}
