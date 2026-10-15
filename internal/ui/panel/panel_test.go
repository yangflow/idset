package panel

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/bind"
	"github.com/yangflow/idset/internal/match"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/ui/locale"
)

var env = paths.Env{Home: "/home/ada"}

type recorder struct {
	resolved []string
	copied   []string
	chosen   int
	hidden   int
	quit     int
	reloaded int
	revealed []string
	opened   []string
	newDraft int
	ran      []bind.Op
	folders  []string
	boundNew []string
}

func (r *recorder) actions() Actions {
	return Actions{
		Resolve:        func(d string) { r.resolved = append(r.resolved, d) },
		ChooseFolder:   func() { r.chosen++ },
		CopyText:       func(s string) { r.copied = append(r.copied, s) },
		Hide:           func() { r.hidden++ },
		Quit:           func() { r.quit++ },
		Reload:         func() { r.reloaded++ },
		Reveal:         func(p string) { r.revealed = append(r.revealed, p) },
		OpenAccounts:   func(id string) { r.opened = append(r.opened, id) },
		NewIdentity:    func() { r.newDraft++ },
		Run:            func(op bind.Op) { r.ran = append(r.ran, op) },
		NewIdentityFor: func(f string) { r.boundNew = append(r.boundNew, f) },
		Open:           func(p string) { r.folders = append(r.folders, p) },
	}
}

func identity(label, alias, email string, p account.Provider, used int, scope ...string) account.Account {
	a := account.Account{ID: label, Label: label, SSHAlias: alias, GitUserEmail: email, Provider: p, Color: "blue"}
	if used > 0 {
		at := time.Date(2026, 8, 1, used, 0, 0, 0, time.UTC)
		a.LastUsedAt = &at
	}
	if len(scope) > 0 {
		a.Scope = account.Scope{Paths: scope}
	} else {
		a.Scope = account.Scope{Global: true}
	}
	return a
}

func screen(t *testing.T, list ...account.Account) (*Screen, *ui.Tester, *recorder) {
	t.Helper()
	r := &recorder{}
	s := New(Text("en"), env, r.actions())
	if list != nil {
		s.SetAccounts(list, nil, "/home/ada/.config/idset/accounts.json")
	}
	return s, ui.NewTester(s.View, 340, 480), r
}

func TestEveryLanguageHasEveryText(t *testing.T) {
	for _, lang := range locale.Languages {
		v := reflect.ValueOf(translations[lang])
		for i := range v.NumField() {
			if v.Field(i).String() == "" {
				t.Errorf("%s has no %s", lang, v.Type().Field(i).Name)
			}
		}
	}
	if Text("zh-CN").Quit != "退出" || Text("fr").Quit != "Quit" {
		t.Error("locale selection")
	}
}

func TestLoadingAndReadOnly(t *testing.T) {
	s, tt, rec := screen(t)
	if !tt.HasText("Reading configuration…") || tt.HasText("Drop a project folder here") {
		t.Fatalf("loading: %v", tt.Texts())
	}
	s.goos = "darwin"
	s.SetAccounts(nil, fmt.Errorf("accounts.json cannot be read: %w", errors.New("unexpected end of JSON input")), "/home/ada/.config/idset/accounts.json")
	tt.Frame()
	for _, want := range []string{"accounts.json cannot be read", "~/.config/idset/accounts.json", "unexpected end of JSON input", "Show in Finder"} {
		if !tt.HasText(want) {
			t.Fatalf("missing %q: %v", want, tt.Texts())
		}
	}
	if tt.HasText("Drop a project folder here") {
		t.Fatal("nothing else shows while read-only")
	}
	if err := tt.Click("Read Again"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Show in Finder"); err != nil {
		t.Fatal(err)
	}
	if rec.reloaded != 1 || !slices.Equal(rec.revealed, []string{"/home/ada/.config/idset/accounts.json"}) {
		t.Fatalf("actions: %d %v", rec.reloaded, rec.revealed)
	}
	s.Drop([]string{"/home/ada/x"})
	if s.checking != "" {
		t.Fatal("no drops while read-only")
	}
	for goos, want := range map[string]string{"windows": "Show in Explorer", "linux": "Show in File Manager"} {
		s.goos = goos
		tt.Frame()
		if !tt.HasText(want) {
			t.Errorf("%s: %v", goos, tt.Texts())
		}
	}
}

func TestEmpty(t *testing.T) {
	_, tt, r := screen(t, []account.Account{}...)
	s := New(Text("en"), env, r.actions())
	s.SetAccounts(nil, nil, "")
	tt = ui.NewTester(s.View, 340, 480)
	if !tt.HasText("No identities yet") || !tt.HasText("Drop a project folder here") {
		t.Fatalf("empty: %v", tt.Texts())
	}
	if err := tt.Click("Choose Folder…"); err != nil || r.chosen != 1 {
		t.Fatalf("choose: %v %d", err, r.chosen)
	}
	if err := tt.Click("Add Identity"); err != nil || r.newDraft != 1 {
		t.Fatalf("add: %v %d", err, r.newDraft)
	}
	if tt.HasText("Import") {
		t.Error("importing is not offered yet")
	}
}

func TestIdentityRowsAndAddButtonOpenTheAccountsWindow(t *testing.T) {
	work := identity("work", "github-work", "bob@example.com", account.GitHub, 2, "~/work/")
	home := identity("home", "", "", account.GitHub, 5)
	_, tt, r := screen(t, work, home)
	if err := tt.Click("work"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.opened, []string{"work"}) || r.newDraft != 0 {
		t.Fatalf("a row opens its identity: %v %d", r.opened, r.newDraft)
	}
	if err := tt.Click("home"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.opened, []string{"work", "home"}) {
		t.Fatalf("opened %v", r.opened)
	}
	if err := tt.Click("Add Identity"); err != nil {
		t.Fatal(err)
	}
	if r.newDraft != 1 || len(r.opened) != 2 {
		t.Fatalf("Add Identity opens a new draft: %v %d", r.opened, r.newDraft)
	}
}

func TestListIsSortedAndFilterAppearsAtThree(t *testing.T) {
	work := identity("work", "github-work", "bob@example.com", account.GitHub, 2, "~/work/")
	home := identity("home", "", "", account.GitHub, 5)
	_, tt, _ := screen(t, work, home)
	texts := tt.Texts()
	if slices.Index(texts, "home") > slices.Index(texts, "work") {
		t.Fatalf("most recently used first: %v", texts)
	}
	for _, want := range []string{"Identities", "2", "bob@example.com", "~/work/", "no email", "Global"} {
		if !shows(tt, want) {
			t.Fatalf("missing %q: %v", want, texts)
		}
	}
	if tt.HasText("Filter identities") {
		t.Fatal("no filter for two identities")
	}
	_, tt, _ = screen(t, work, home, identity("client", "gitlab-client", "c@example.com", account.GitLab, 0))
	if !tt.HasText("GitHub") || !tt.HasText("GitLab") {
		t.Fatalf("provider chips: %v", tt.Texts())
	}
	if err := tt.Click("GitLab"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("work") || !tt.HasText("client") {
		t.Fatalf("chip filter: %v", tt.Texts())
	}
	if err := tt.Click("GitLab"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("work") {
		t.Fatal("chip toggles off")
	}
}

func TestFilterQuery(t *testing.T) {
	s, tt, _ := screen(t,
		identity("Café", "a", "a@example.com", account.GitHub, 0),
		identity("work", "b", "b@example.com", account.GitHub, 0),
		identity("home", "c", "c@example.com", account.GitHub, 0))
	s.query = "cafe"
	tt.Frame()
	if !tt.HasText("Café") || tt.HasText("work") {
		t.Fatalf("query: %v", tt.Texts())
	}
	s.query = "nothing"
	tt.Frame()
	if !tt.HasText("No matching identities") {
		t.Fatalf("no match: %v", tt.Texts())
	}
}

func TestDropAndMatchedResult(t *testing.T) {
	work := identity("work", "github-work", "bob@example.com", account.GitHub, 0, "~/work/", "~/src/new/")
	s, tt, r := screen(t, work)
	s.Drop([]string{"/home/ada/work/api", "/home/ada/other"})
	if !slices.Equal(r.resolved, []string{"/home/ada/work/api"}) {
		t.Fatalf("resolve: %v", r.resolved)
	}
	tt.Frame()
	if !tt.HasText("Checking…") || !tt.HasText("~/work/api") || !tt.HasText("Drop another folder, or") || tt.HasText("Clear result") {
		t.Fatalf("checking: %v", tt.Texts())
	}
	s.SetResult("/home/ada/elsewhere", match.Result{Kind: match.NoMatch}, nil)
	if s.Result() != nil {
		t.Fatal("a stale result must be ignored")
	}
	s.SetResult("/home/ada/work/api", match.Result{Kind: match.Matched, Root: "/home/ada/work/api", ID: "work"}, nil)
	tt.Frame()
	for _, want := range []string{"W", "work", "Git uses this identity here", "Alias", "github-work", "Email", "bob@example.com",
		"Folders", "~/work/", "~/src/new/", "~/work/api", "~/work/ +1"} {
		if !shows(tt, want) {
			t.Fatalf("missing %q: %v", want, tt.Texts())
		}
	}
	if s.Tooltip() != "Idset · work" {
		t.Errorf("tooltip %q", s.Tooltip())
	}
	if got := tt.Announcements(); !slices.Equal(got, []string{"Git uses work here"}) {
		t.Fatalf("the result must be announced once: %v", got)
	}
	tt.Frame()
	if got := tt.Announcements(); len(got) != 0 {
		t.Fatalf("announced again: %v", got)
	}
	if err := tt.Click("Copy the full path"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.copied, []string{"/home/ada/work/api"}) || !tt.HasText("Copied") {
		t.Fatalf("copy: %v %v", r.copied, tt.Texts())
	}
	if !slices.Contains(tt.Announcements(), "Copied") {
		t.Fatal("the copy must be announced")
	}
	s.now = func() time.Time { return time.Now().Add(2 * time.Second) }
	tt.Frame()
	if tt.HasText("Copied") {
		t.Fatal("the acknowledgement must go away")
	}
	if err := tt.Click("Clear result"); err != nil {
		t.Fatal(err)
	}
	if s.Result() != nil || s.Tooltip() != "Idset" || !tt.HasText("Drop a project folder here") {
		t.Fatalf("clear: %v", tt.Texts())
	}
}

func TestUnresolvedResults(t *testing.T) {
	s, tt, _ := screen(t, identity("work", "w", "w@example.com", account.GitHub, 0, "~/work/"))
	for kind, want := range map[match.Kind]string{
		match.NoMatch: "No identity matches this folder", match.NotRepo: "Not a Git repository", match.NotFolder: "Only folders can be checked",
	} {
		s.Drop([]string{"/home/ada/scratch"})
		s.SetResult("/home/ada/scratch", match.Result{Kind: kind, Dir: "/home/ada/scratch"}, nil)
		tt.Frame()
		if !tt.HasText(want) || !tt.HasText("~/scratch") {
			t.Errorf("%v: %v", kind, tt.Texts())
		}
		if got := tt.Announcements(); !slices.Equal(got, []string{want}) {
			t.Errorf("%v announced %v", kind, got)
		}
		if kind != match.Matched && s.Tooltip() != "Idset · no match" {
			t.Errorf("tooltip %q", s.Tooltip())
		}
	}
	s.Drop([]string{"/home/ada/scratch"})
	s.SetResult("/home/ada/scratch", match.Result{}, errors.New("exec: \"git\": executable file not found"))
	tt.Frame()
	if !slices.ContainsFunc(tt.Texts(), func(s string) bool { return strings.HasPrefix(s, "Could not run git") }) {
		t.Errorf("git error: %v", tt.Texts())
	}
	if got := tt.Announcements(); len(got) != 1 || !strings.HasPrefix(got[0], "Could not run git") {
		t.Errorf("git error announced %v", got)
	}
}

func TestHideAndQuit(t *testing.T) {
	_, tt, r := screen(t, identity("work", "w", "w@example.com", account.GitHub, 0))
	if tt.HasText("Hide") {
		t.Fatal("the panel has no hide button; the tray icon and Escape hide it")
	}
	tt.Key(0, ui.KeyEscape)
	if err := tt.Click("Quit"); err != nil {
		t.Fatal(err)
	}
	if r.hidden != 1 || r.quit != 1 {
		t.Fatalf("hidden %d quit %d", r.hidden, r.quit)
	}
	tt.SetDark(true)
	if !tt.HasText("work") {
		t.Fatal("dark scheme lost state")
	}
}

func TestChineseCopy(t *testing.T) {
	r := &recorder{}
	s := New(Text("zh-Hans"), env, r.actions())
	s.SetAccounts([]account.Account{identity("", "", "", account.Custom, 0)}, nil, "")
	tt := ui.NewTester(s.View, 340, 480)
	for _, want := range []string{"把项目文件夹拖到这里", "（未命名）", "没有邮箱", "全局", "身份", "退出", "添加身份"} {
		if !tt.HasText(want) {
			t.Errorf("missing %q: %v", want, tt.Texts())
		}
	}
}

func TestManyIdentitiesScrollAndKeepTheFooterReachable(t *testing.T) {
	var list []account.Account
	for i := range 9 {
		list = append(list, identity(fmt.Sprintf("id%d", i), "", "a@example.com", account.GitHub, i+1))
	}
	_, tt, r := screen(t, list...)
	quit, ok := tt.Find("Quit")
	if !ok || quit.Y+quit.H > 480 {
		t.Fatalf("the footer stays inside the panel: %v %v", quit, ok)
	}
	// The footer holds Add Identity, so it is on screen without scrolling, and so is Quit.
	add, ok := tt.Find("Add Identity")
	if !ok || add.Y+add.H > 480 {
		t.Fatalf("Add Identity must not scroll away: %v %v", add, ok)
	}
	if err := tt.Click("Add Identity"); err != nil || r.newDraft != 1 {
		t.Fatalf("add: %v %d", err, r.newDraft)
	}
	// The list is longer than the panel and scrolls under the fixed footer.
	last, _ := tt.Find("id0")
	if last.Y < 480 {
		t.Fatalf("expected the oldest identity below the fold, at %v", last)
	}
	tt.Scroll(170, 240, 0, 2000)
	if got, _ := tt.Find("id0"); got.Y >= 480 {
		t.Fatalf("scrolling brings the last identity into view: %v", got)
	}
	if got, _ := tt.Find("Quit"); got != quit {
		t.Fatalf("the footer does not move: %v %v", got, quit)
	}
}
