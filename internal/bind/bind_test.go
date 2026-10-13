package bind

import (
	"reflect"
	"testing"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/paths"
)

var (
	unix = paths.Env{Home: "/home/ada"}
	win  = paths.Env{Home: `C:\Users\Ada`, Windows: true}
	now  = time.Date(2026, 8, 22, 8, 0, 0, 0, time.UTC)
)

func scoped(id string, p ...string) account.Account {
	return account.Account{ID: id, Label: id, Scope: account.Scope{Paths: p}}
}

func global(id string) account.Account {
	return account.Account{ID: id, Label: id, Scope: account.Scope{Global: true}}
}

func paths_(t *testing.T, list []account.Account, id string) []string {
	t.Helper()
	for _, a := range list {
		if a.ID == id {
			return a.Scope.Paths
		}
	}
	t.Fatalf("no %s", id)
	return nil
}

func TestBindAppendsTheStoredFormAndCountsAsAUse(t *testing.T) {
	list := []account.Account{scoped("work", "~/work/"), global("home")}
	plan, err := Bind(list, "work", "/home/ada/scratch/notes", unix, now)
	if err != nil {
		t.Fatal(err)
	}
	if got := plan.Changes[0].Scope.Paths; !reflect.DeepEqual(got, []string{"~/work/", "~/scratch/notes/"}) {
		t.Fatalf("paths %v", got)
	}
	if plan.Changes[0].LastUsedAt == nil || !plan.Changes[0].LastUsedAt.Equal(now) {
		t.Fatal("binding is a use")
	}
	if !reflect.DeepEqual(plan.Before[0].Scope.Paths, []string{"~/work/"}) || plan.Before[0].LastUsedAt != nil {
		t.Fatal("Before must be the identity as it was")
	}
	if !reflect.DeepEqual(list[0].Scope.Paths, []string{"~/work/"}) {
		t.Fatal("the list must not change")
	}
}

func TestBindAlreadyCoveredChangesNothing(t *testing.T) {
	list := []account.Account{scoped("work", "~/work/")}
	for _, root := range []string{"/home/ada/work/api", "/home/ada/work"} {
		plan, err := Bind(list, "work", root, unix, now)
		if err != nil || !plan.Empty() || plan.CoveredID != "work" || plan.CoveredBy != "~/work/" {
			t.Fatalf("%s: %+v %v", root, plan, err)
		}
	}
	if plan, _ := Bind(list, "work", "/home/ada/workshop", unix, now); plan.Empty() {
		t.Fatal("a sibling folder with the same prefix is not covered")
	}
}

func TestBindingAGlobalIdentityMakesItScopedToOnePath(t *testing.T) {
	list := []account.Account{global("home"), scoped("work", "~/work/")}
	plan, err := Bind(list, "home", "/home/ada/work/api", unix, now)
	if err != nil {
		t.Fatal(err)
	}
	got := plan.Changes[0]
	if got.Scope.Global || !reflect.DeepEqual(got.Scope.Paths, []string{"~/work/api/"}) {
		t.Fatalf("scope %+v", got.Scope)
	}
	if !plan.Before[0].Scope.Global {
		t.Fatal("Before keeps the global scope")
	}
}

func TestBindUnknownIdentity(t *testing.T) {
	if _, err := Bind(nil, "x", "/home/ada/a", unix, now); err != ErrNotFound {
		t.Fatalf("%v", err)
	}
}

func TestUnbindRemovesOnlyTheExactPath(t *testing.T) {
	list := []account.Account{scoped("work", "~/work/", "~/work/api/", "/home/ada/work/api", "~/src/")}
	plan, err := Unbind(list, "work", "/home/ada/work/api", unix, now)
	if err != nil {
		t.Fatal(err)
	}
	if got := plan.Changes[0].Scope.Paths; !reflect.DeepEqual(got, []string{"~/work/", "~/src/"}) {
		t.Fatalf("paths %v", got)
	}
	if plan, _ := Unbind(list, "work", "/home/ada/work/other", unix, now); !plan.Empty() {
		t.Fatal("a parent's path is not the folder")
	}
	only := []account.Account{scoped("work", "~/work/api/")}
	plan, _ = Unbind(only, "work", "/home/ada/work/api", unix, now)
	if got := plan.Changes[0].Scope.Paths; got == nil || len(got) != 0 {
		t.Fatalf("an identity without paths keeps an empty list, not null: %#v", got)
	}
}

func TestHasExact(t *testing.T) {
	a := scoped("work", "~/work/", "~/work/api/")
	if !HasExact(a, "/home/ada/work/api", unix) || HasExact(a, "/home/ada/work/web", unix) || HasExact(global("g"), "/home/ada", unix) {
		t.Fatal("HasExact")
	}
}

func TestRebindMovesTheFolderInOnePlan(t *testing.T) {
	list := []account.Account{scoped("work", "~/work/api/", "~/src/"), scoped("client", "~/c/"), global("home")}
	plan, err := Rebind(list, "work", "client", "/home/ada/work/api", unix, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Changes) != 2 || len(plan.Before) != 2 {
		t.Fatalf("both identities in one plan: %+v", plan)
	}
	if got := paths_(t, plan.Changes, "work"); !reflect.DeepEqual(got, []string{"~/src/"}) {
		t.Fatalf("work %v", got)
	}
	if got := paths_(t, plan.Changes, "client"); !reflect.DeepEqual(got, []string{"~/c/", "~/work/api/"}) {
		t.Fatalf("client %v", got)
	}
	for _, a := range plan.Changes {
		if a.ID == "work" && a.LastUsedAt != nil {
			t.Fatal("the identity left is not used")
		}
		if a.ID == "client" && a.LastUsedAt == nil {
			t.Fatal("the identity bound is used")
		}
	}
	// From a parent's cover: nothing to unbind, only the bind.
	plan, _ = Rebind(list, "work", "client", "/home/ada/src/x", unix, now)
	if len(plan.Changes) != 1 || plan.Changes[0].ID != "client" {
		t.Fatalf("%+v", plan)
	}
	// To a global identity: it becomes scoped, the old one loses the path, still one plan.
	plan, _ = Rebind(list, "work", "home", "/home/ada/work/api", unix, now)
	if len(plan.Changes) != 2 || paths_(t, plan.Changes, "home")[0] != "~/work/api/" {
		t.Fatalf("%+v", plan)
	}
	// A target that covers the folder already changes nothing, the origin included.
	plan, _ = Rebind(list, "client", "work", "/home/ada/src/x", unix, now)
	if !plan.Empty() || plan.CoveredID != "work" || plan.CoveredBy != "~/src/" {
		t.Fatalf("%+v", plan)
	}
}

func TestOverlapListsGitsReadOrderWithTheWinnerLast(t *testing.T) {
	list := []account.Account{scoped("work", "~/work/"), scoped("client", "~/work/client/"), global("home"), scoped("misc", "~/misc/")}
	hits := Overlap(list, "/home/ada/work/client/app", unix)
	want := []Hit{{"work", "~/work/"}, {"client", "~/work/client/"}}
	if !reflect.DeepEqual(hits, want) {
		t.Fatalf("%+v", hits)
	}
	if got := Overlap(list, "/home/ada/work/web", unix); len(got) != 1 {
		t.Fatalf("one identity is no overlap: %+v", got)
	}
	// One identity with two covering paths appears once, at its last position.
	list = append(list, scoped("both", "~/", "~/work/client/app/"))
	hits = Overlap(list, "/home/ada/work/client/app", unix)
	if len(hits) != 3 || hits[2].ID != "both" || hits[2].Path != "~/work/client/app/" {
		t.Fatalf("%+v", hits)
	}
}

func TestUseOnlyAddsTheExactPathToTheWinner(t *testing.T) {
	list := []account.Account{scoped("work", "~/work/"), scoped("client", "~/work/client/")}
	plan, err := UseOnly(list, "client", "/home/ada/work/client/app", unix, now)
	if err != nil {
		t.Fatal(err)
	}
	if got := plan.Changes[0].Scope.Paths; !reflect.DeepEqual(got, []string{"~/work/client/", "~/work/client/app/"}) {
		t.Fatalf("%v", got)
	}
	if plan, _ := UseOnly(plan.Changes, "client", "/home/ada/work/client/app", unix, now); !plan.Empty() {
		t.Fatal("it already has the path")
	}
}

func TestRemovePathTakesTheNormalizedOrStoredForm(t *testing.T) {
	list := []account.Account{scoped("work", "~/work", "~/src/")}
	plan, err := RemovePath(list, "work", "~/work/", unix, now)
	if err != nil || !reflect.DeepEqual(plan.Changes[0].Scope.Paths, []string{"~/src/"}) {
		t.Fatalf("%+v %v", plan, err)
	}
	if plan, _ := RemovePath(list, "work", "~/other/", unix, now); !plan.Empty() {
		t.Fatal("an unknown path changes nothing")
	}
	if plan, _ := RemovePath(list, "work", "", unix, now); !plan.Empty() {
		t.Fatal("a blank path changes nothing")
	}
}

func TestWindowsPathForms(t *testing.T) {
	list := []account.Account{scoped("work", "~/Work/"), global("home")}
	root := `c:\users\ada\work\API`
	if plan, _ := Bind(list, "work", root, win, now); !plan.Empty() || plan.CoveredBy != "~/Work/" {
		t.Fatalf("case-insensitive cover: %+v", plan)
	}
	plan, err := Bind(list, "home", root, win, now)
	if err != nil || !reflect.DeepEqual(plan.Changes[0].Scope.Paths, []string{"~/work/API/"}) {
		t.Fatalf("stored form %+v %v", plan, err)
	}
	list = []account.Account{scoped("work", "~/work/API/", `C:\Users\Ada\work\web`)}
	for _, root := range []string{`C:\Users\Ada\Work\api`, `c:/users/ada/work/WEB`} {
		plan, err := Unbind(list, "work", root, win, now)
		if err != nil || len(plan.Changes[0].Scope.Paths) != 1 {
			t.Fatalf("%s: %+v %v", root, plan, err)
		}
	}
	plan, _ = RemovePath(list, "work", `C:\Users\Ada\WORK\web`, win, now)
	if plan.Empty() {
		t.Fatal("remove by a Windows path form")
	}
	hits := Overlap([]account.Account{scoped("a", "~/work/"), scoped("b", "c:/users/ada/work/api/")}, `C:\Users\Ada\work\api`, win)
	if len(hits) != 2 || hits[1].ID != "b" {
		t.Fatalf("%+v", hits)
	}
}
