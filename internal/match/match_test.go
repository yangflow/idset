package match

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/paths"
)

func identity(id string, scope ...string) account.Account {
	a := account.Account{ID: id}
	if len(scope) == 0 {
		a.Scope = account.Scope{Global: true}
	} else {
		a.Scope = account.Scope{Paths: scope}
	}
	return a
}

func TestIdentity(t *testing.T) {
	env := paths.Env{Home: "/home/ada"}
	work := identity("work", "~/work/")
	client := identity("client", "~/work/client/")
	global := identity("personal")
	for _, c := range []struct {
		list []account.Account
		root string
		kind Kind
		id   string
	}{
		{[]account.Account{client, work}, "/home/ada/work/client/app", Matched, "client"},
		{[]account.Account{client, work}, "/home/ada/work/api", Matched, "work"},
		{[]account.Account{client, work}, "/home/ada/src/x", NoMatch, ""},
		{[]account.Account{global, work}, "/home/ada/src/x", Matched, "personal"},
		{[]account.Account{global, work, identity("second")}, "/home/ada/src/x", Matched, "second"},
		{[]account.Account{global, identity("empty", " ")}, "/home/ada/work/x", Matched, "personal"},
	} {
		kind, id := Identity(c.list, c.root, env)
		if kind != c.kind || id != c.id {
			t.Errorf("%s: %v %q, want %v %q", c.root, kind, id, c.kind, c.id)
		}
	}
}

type fakeGit map[string]struct {
	out  string
	code int
}

func (f fakeGit) run(_ context.Context, _ string, args ...string) (string, int, error) {
	r, ok := f[strings.Join(args, " ")]
	if !ok {
		return "", 1, nil
	}
	return r.out, r.code, nil
}

func TestResolve(t *testing.T) {
	dir := t.TempDir()
	env := paths.Env{Home: "/home/ada"}
	list := []account.Account{identity("work", "~/work/")}
	git := fakeGit{
		"rev-parse --show-toplevel":      {"/home/ada/work/api", 0},
		"config --get remote.origin.url": {"", 1},
		"ls-remote --get-url origin":     {"https://github.com/example-org/api.git", 0},
	}
	r, err := Resolve(context.Background(), git.run, dir, list, env)
	if err != nil {
		t.Fatal(err)
	}
	want := Result{Kind: Matched, Dir: paths.Slash(dir), Root: "/home/ada/work/api", Origin: "https://github.com/example-org/api.git", ID: "work",
		Covering: []Cover{{ID: "work", Path: "~/work/"}}}
	if !reflect.DeepEqual(r, want) {
		t.Fatalf("%+v", r)
	}

	git["ls-remote --get-url origin"] = struct {
		out  string
		code int
	}{"origin", 0}
	if r, _ := Resolve(context.Background(), git.run, dir, list, env); r.Origin != "" {
		t.Errorf("an echoed remote name is not a URL: %q", r.Origin)
	}

	git["rev-parse --show-toplevel"] = struct {
		out  string
		code int
	}{"", 128}
	if r, _ := Resolve(context.Background(), git.run, dir, list, env); r.Kind != NotRepo {
		t.Errorf("not a repo: %+v", r)
	}

	file := filepath.Join(dir, "f")
	os.WriteFile(file, nil, 0o600)
	for _, p := range []string{file, filepath.Join(dir, "missing")} {
		if r, err := Resolve(context.Background(), git.run, p, list, env); err != nil || r.Kind != NotFolder {
			t.Errorf("%s: %+v %v", p, r, err)
		}
	}

	broken := func(context.Context, string, ...string) (string, int, error) { return "", 0, errors.New("no git") }
	if _, err := Resolve(context.Background(), broken, dir, list, env); err == nil {
		t.Error("a git that cannot run must be an error")
	}
}

func TestSystemGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir, _ := filepath.EvalSymlinks(t.TempDir())
	git := SystemGit(10 * time.Second)
	if _, code, err := git(context.Background(), dir, "init", "-q"); err != nil || code != 0 {
		t.Fatal(code, err)
	}
	git(context.Background(), dir, "remote", "add", "origin", "git@example.com:o/r.git")
	sub := filepath.Join(dir, "sub")
	os.Mkdir(sub, 0o755)
	r, err := Resolve(context.Background(), git, sub, []account.Account{identity("g")}, paths.Env{Home: dir})
	if err != nil {
		t.Fatal(err)
	}
	if r.Kind != Matched || !strings.EqualFold(r.Root, paths.Slash(dir)) || r.Origin != "git@example.com:o/r.git" {
		t.Fatalf("%+v", r)
	}
}

func TestCoveringListsGitsReadOrderWithTheWinnerLast(t *testing.T) {
	env := paths.Env{Home: "/home/ada"}
	list := []account.Account{identity("work", "~/work/"), identity("client", "~/work/client/"), identity("home"), identity("misc", "~/misc/")}
	want := []Cover{{"work", "~/work/"}, {"client", "~/work/client/"}}
	if got := Covering(list, "/home/ada/work/client/app", env); !reflect.DeepEqual(got, want) {
		t.Fatalf("%+v", got)
	}
	if got := Covering(list, "/home/ada/work/web", env); len(got) != 1 {
		t.Fatalf("one identity is no overlap: %+v", got)
	}
	if got := Covering(list, "/home/ada/src", env); len(got) != 0 {
		t.Fatalf("a global identity covers nothing by path: %+v", got)
	}
	// One identity with two covering paths appears once, where its last one is read.
	list = append(list, identity("both", "~/", "~/work/client/app/"))
	got := Covering(list, "/home/ada/work/client/app", env)
	if len(got) != 3 || got[2] != (Cover{"both", "~/work/client/app/"}) {
		t.Fatalf("%+v", got)
	}
	win := paths.Env{Home: `C:\Users\Ada`, Windows: true}
	got = Covering([]account.Account{identity("a", "~/work/"), identity("b", "c:/users/ada/work/api/")}, `C:\Users\Ada\work\api`, win)
	if len(got) != 2 || got[1].ID != "b" {
		t.Fatalf("Windows path forms: %+v", got)
	}
}

func TestResolveSaysWhetherTheMatchedIdentityHasTheFolderItself(t *testing.T) {
	dir := t.TempDir()
	env := paths.Env{Home: "/home/ada"}
	git := fakeGit{"rev-parse --show-toplevel": {"/home/ada/work/api", 0}}
	for _, c := range []struct {
		scope []string
		exact bool
	}{{[]string{"~/work/"}, false}, {[]string{"~/work/", "~/work/api/"}, true}} {
		r, err := Resolve(context.Background(), git.run, dir, []account.Account{identity("work", c.scope...)}, env)
		if err != nil || r.Exact != c.exact {
			t.Errorf("%v: %+v %v", c.scope, r, err)
		}
	}
	r, _ := Resolve(context.Background(), git.run, dir, []account.Account{identity("home")}, env)
	if r.Exact || len(r.Covering) != 0 {
		t.Errorf("a global identity has no path: %+v", r)
	}
}
