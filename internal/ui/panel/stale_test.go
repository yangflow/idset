package panel

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/bind"
	"github.com/yangflow/idset/internal/match"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/store"
)

const (
	moved   = "~/Documents/archive/2023/legacy-services/payments-gateway/"
	renamed = "~/projects/api-old/"
	newRoot = "/home/ada/projects/api"
)

// staleList is an identity whose path is gone, and two others whose rows the bind list shows too.
func staleList(old string) []account.Account {
	return []account.Account{
		identity("work", "github-work", "bob@example.com", account.GitHub, 0, old, "~/src/new/"),
		client(),
		identity("misc", "", "m@example.com", account.GitHub, 0, moved),
	}
}

// tallCard is card in a window high enough to show the whole panel, so that no row is cut short.
func tallCard(t *testing.T, res match.Result, list ...account.Account) *ui.Tester {
	t.Helper()
	r := &recorder{}
	s := New(Text("en"), env, r.actions())
	s.SetAccounts(list, nil, "")
	tt := ui.NewTester(s.View, 340, 1000)
	s.Drop([]string{res.Root})
	res.Dir = res.Root
	s.SetResult(res.Root, res, nil)
	tt.Frame()
	return tt
}

func staleResult(root, old string) match.Result {
	return match.Result{Kind: match.NoMatch, Root: root, Stale: &match.Stale{ID: "work", Path: old}}
}

func TestStalePromptOnARenamedFolder(t *testing.T) {
	_, tt, _ := card(t, staleResult(newRoot, renamed), staleList(renamed)...)
	want(t, tt, "work’s folder is gone", "~/projects/api-old", "~/projects/api", "Update", "Ignore")
	if count(tt, "Old")+count(tt, "New") != 0 {
		t.Errorf("no labels before the paths: %q", tt.Texts())
	}
	// The short words name the long actions, and the old path comes above the new one.
	update, _ := tt.Find("Change to This Path")
	ignore, _ := tt.Find("Keep Old Path")
	if update.Y != ignore.Y || update.X >= ignore.X {
		t.Errorf("Update, then Ignore, on one line: %v %v", update, ignore)
	}
	old, _ := tt.Find("api-old")
	head, _ := tt.Find("’s folder is gone")
	if old.Y <= head.Y || old.Y >= update.Y {
		t.Errorf("header, paths, buttons: %v %v %v", head, old, update)
	}
	within(t, tt, "’s folder is gone", "api-old", "Change to This Path", "Keep Old Path")
	// The no-match card below is unchanged: its sentence, the bind list and the new identity.
	want(t, tt, "No identity matches this folder", "Bind to", "Bind to client", "New Identity and Bind Here")
	wantNot(t, tt, "Updated")
}

func TestStalePromptInChinese(t *testing.T) {
	r := &recorder{}
	s := New(Text("zh-Hans"), env, r.actions())
	s.SetAccounts(staleList(renamed), nil, "")
	tt := ui.NewTester(s.View, 340, 480)
	s.Drop([]string{newRoot})
	s.SetResult(newRoot, staleResult(newRoot, renamed), nil)
	tt.Frame()
	want(t, tt, "work 的路径已不在", "更新", "忽略", "改成这次的路径", "保留旧路径", "没有身份匹配这个目录")
	within(t, tt, " 的路径已不在", "更新", "忽略")
	if got := tt.Announcements(); !slices.ContainsFunc(got, func(a string) bool { return strings.Contains(a, "work 的路径已不在") }) {
		t.Errorf("the prompt is announced: %q", got)
	}
}

// stackHeights are the heights the prompt's lines take, for any path: from the header to the buttons, and the
// buttons.
func stackHeights(t *testing.T, tt *ui.Tester) (paths, buttons float32) {
	t.Helper()
	b, ok := tt.Find("Change to This Path")
	if !ok {
		t.Fatalf("no button in %q", tt.Texts())
	}
	k, _ := tt.Find("Keep Old Path")
	if b.Y != k.Y {
		t.Errorf("the two buttons sit on one line: %v %v", b.Y, k.Y)
	}
	h, _ := tt.Find("’s folder is gone")
	return b.Y - h.Y, k.H
}

func TestStaleCardHasTheSameHeightForAnyPath(t *testing.T) {
	short := tallCard(t, staleResult(newRoot, renamed), staleList(renamed)...)
	long := tallCard(t, staleResult("/home/ada/projects/payments-gateway", moved), staleList(moved)...)
	a1, a2 := stackHeights(t, short)
	b1, b2 := stackHeights(t, long)
	if !same(a1, b1) || !same(a2, b2) {
		t.Errorf("short %v %v, long %v %v", a1, a2, b1, b2)
	}
	// The bind rows are as high as each other, long path or not.
	_, h1 := size(t, long, "Bind to work")
	_, h2 := size(t, long, "Bind to misc")
	_, h3 := size(t, long, "Bind to client")
	if !same(h1, h2) || !same(h2, h3) {
		t.Errorf("bind rows differ in height: %v %v %v", h1, h2, h3)
	}
	// The card below starts at the same distance from the buttons in both.
	s1, _ := short.Find("No identity matches this folder")
	s2, _ := long.Find("No identity matches this folder")
	k1, _ := short.Find("Keep Old Path")
	k2, _ := long.Find("Keep Old Path")
	if !same(s1.Y-k1.Y, s2.Y-k2.Y) {
		t.Errorf("the no-match card moves with the path: %v %v", s1.Y-k1.Y, s2.Y-k2.Y)
	}
}

func TestLongPathsKeepTheirStartAndLastFolder(t *testing.T) {
	tt := tallCard(t, staleResult("/home/ada/projects/payments-gateway", moved), staleList(moved)...)
	// A path is one line made of its start, its middle and its last folder, side by side with the words around it;
	// only the middle gives way. The old path of the prompt, the bind rows and the identity list all do it.
	texts := tt.Texts()
	for _, line := range [][]string{
		{"’s folder is gone", "~/", "Documents/archive/2023/legacy-services/", "payments-gateway", "~/", "projects/", "payments-gateway", "Update"},
		{"Bind to work", "work", "~/", "Documents/archive/2023/legacy-services/", "payments-gateway/", "Bind"},
		{"m@example.com", "~/", "Documents/archive/2023/legacy-services/", "payments-gateway/", "work"},
		{"bob@example.com", "~/", "Documents/archive/2023/legacy-services/", "payments-gateway/", "+1"},
	} {
		at := slices.Index(texts, line[0])
		if at < 0 {
			t.Errorf("no %q in %q", line[0], texts)
			continue
		}
		for _, piece := range line {
			if i := slices.Index(texts[at:], piece); i < 0 {
				t.Errorf("%q has no %q after %q", texts, piece, line[0])
				break
			}
			at += slices.Index(texts[at:], piece)
		}
	}
	// Short paths have no middle.
	if !slices.Contains(texts, "work/") {
		t.Errorf("a short path is its start and its last folder: %q", texts)
	}
}

func TestChangeToThisPathAsksForOneChange(t *testing.T) {
	s, tt, r := card(t, staleResult(newRoot, renamed), staleList(renamed)...)
	click(t, tt, "Change to This Path")
	if len(r.ran) != 1 {
		t.Fatalf("ran %+v", r.ran)
	}
	got := r.ran[0]
	if got.Kind != bind.KindFixPath || got.From != "work" || got.Path != renamed || got.Dir != newRoot || got.Root != newRoot {
		t.Fatalf("%+v", got)
	}
	// While the owner works, nothing else goes out.
	tt.Frame()
	click(t, tt, "Keep Old Path")
	click(t, tt, "Change to This Path")
	if len(r.ran) != 1 || !s.busy {
		t.Fatalf("a second request while busy: %+v", r.ran)
	}
}

func TestAfterChangingThePathTheFolderMatchesWithUpdated(t *testing.T) {
	fixed := identity("work", "github-work", "bob@example.com", account.GitHub, 0, "~/projects/api/", "~/src/new/")
	s, tt, r := card(t, staleResult(newRoot, renamed), staleList(renamed)...)
	s.goos = "darwin"
	clock := newClock(s)
	click(t, tt, "Change to This Path")
	s.SetAccounts([]account.Account{fixed, client()}, nil, "")
	s.OpDone(bind.Outcome{Op: r.ran[0], Result: matched(newRoot, "work", true)})
	tt.Frame()
	want(t, tt, "Updated", "Git uses this identity here", "More", s.openLabel())
	wantNot(t, tt, "’s folder is gone", "Change to This Path", "Keep Old Path", "Undo", "Bound to work")
	if got := tt.Announcements(); !slices.Contains(got, "Updated") {
		t.Errorf("announced %q", got)
	}
	// Like "Copied": a moment, then gone at once.
	clock.at = clock.at.Add(updatedFor + time.Millisecond)
	tt.Frame()
	wantNot(t, tt, "Updated")
	want(t, tt, "Git uses this identity here")
	if s.busy {
		t.Error("the wait is over")
	}
}

func TestKeepOldPathOnlyHidesThePrompt(t *testing.T) {
	s, tt, r := card(t, staleResult(newRoot, renamed), staleList(renamed)...)
	click(t, tt, "Keep Old Path")
	tt.Frame()
	if len(r.ran) != 0 || s.busy {
		t.Fatalf("no data changes: %+v", r.ran)
	}
	wantNot(t, tt, "’s folder is gone", "Change to This Path", "Keep Old Path", "Updated")
	want(t, tt, "No identity matches this folder", "Bind to client")
	// A request on the card does not bring it back, even when it fails.
	click(t, tt, "Bind to client")
	s.OpDone(bind.Outcome{Op: r.ran[0], Err: errors.New("denied")})
	tt.Frame()
	wantNot(t, tt, "’s folder is gone")
	// Dropping the folder again is a new result, and asks again.
	s.Drop([]string{newRoot})
	s.SetResult(newRoot, staleResult(newRoot, renamed), nil)
	tt.Frame()
	want(t, tt, "’s folder is gone")
}

func TestStaleFailuresShowOnceInTheCard(t *testing.T) {
	s, tt, r := card(t, staleResult(newRoot, renamed), staleList(renamed)...)
	click(t, tt, "Change to This Path")
	s.OpDone(bind.Outcome{Op: r.ran[0], Err: errors.New("denied")})
	tt.Frame()
	n := 0
	for _, text := range tt.Texts() {
		if text == "Could not change the bindings: denied" {
			n++
		}
	}
	if n != 1 || s.busy {
		t.Fatalf("shown %d times in %q", n, tt.Texts())
	}
	want(t, tt, "Change to This Path", "Keep Old Path")
	// The buttons work again.
	click(t, tt, "Change to This Path")
	if len(r.ran) != 2 {
		t.Fatalf("ran %+v", r.ran)
	}
	// The list is saved but the configuration is not: the folder matches now, and the matched card says so.
	s.OpDone(bind.Outcome{Op: r.ran[1], Err: &store.RegenerateError{Err: errors.New("disk full")}, Result: matched(newRoot, "work", true)})
	tt.Frame()
	want(t, tt, "Saved, but the generated configuration could not be rewritten: disk full.", "Updated")
}

func TestNoPromptWithoutAnIdentityOrAMatchResult(t *testing.T) {
	_, tt, _ := card(t, staleResult(newRoot, renamed), client())
	wantNot(t, tt, "’s folder is gone")
	want(t, tt, "No identity matches this folder")
	other := match.Result{Kind: match.NotRepo, Root: newRoot, Stale: &match.Stale{ID: "work", Path: renamed}}
	_, tt, _ = card(t, other, staleList(renamed)...)
	wantNot(t, tt, "’s folder is gone")
}

func TestStalePromptOnWindowsPaths(t *testing.T) {
	winEnv := paths.Env{Home: `C:\Users\Ada`, Windows: true}
	r := &recorder{}
	s := New(Text("en"), winEnv, r.actions())
	s.goos = "windows"
	old := "~/Projects/Api-Old/"
	s.SetAccounts([]account.Account{identity("work", "", "bob@example.com", account.GitHub, 0, old, "D:/keep/")}, nil, "")
	tt := ui.NewTester(s.View, 340, 480)
	root := `C:/Users/Ada/Projects/Api`
	s.Drop([]string{root})
	s.SetResult(root, staleResult(root, old), nil)
	tt.Frame()
	want(t, tt, "~/Projects/Api-Old", "~/Projects/Api", "Change to This Path")
	click(t, tt, "Change to This Path")
	if len(r.ran) != 1 || r.ran[0].Path != old || r.ran[0].Root != root {
		t.Fatalf("%+v", r.ran)
	}
	// Another drive keeps its letter in the old path.
	s.Drop([]string{"E:/src/api"})
	s.SetResult("E:/src/api", staleResult("E:/src/api", "D:/code/api/"), nil)
	tt.Frame()
	want(t, tt, "D:/code/api", "E:/src/api")
}
