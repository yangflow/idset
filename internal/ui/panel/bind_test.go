package panel

import (
	"errors"
	"fmt"
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
	notes = "/home/ada/scratch/notes"
	api   = "/home/ada/work/api"
)

func work() account.Account {
	return identity("work", "github-work", "bob@example.com", account.GitHub, 0, "~/work/", "~/src/new/")
}
func client() account.Account {
	return identity("client", "", "c@example.com", account.GitHub, 0, "~/work/client/")
}
func personal() account.Account {
	return identity("personal", "github-personal", "p@example.com", account.GitHub, 0)
}

// card shows a result for a dropped folder.
func card(t *testing.T, res match.Result, list ...account.Account) (*Screen, *ui.Tester, *recorder) {
	t.Helper()
	s, tt, r := screen(t, list...)
	dir := res.Root
	if dir == "" {
		dir = res.Dir
	}
	s.Drop([]string{dir})
	res.Dir = dir
	s.SetResult(dir, res, nil)
	tt.Frame()
	return s, tt, r
}

func matched(root, id string, exact bool, covering ...match.Cover) match.Result {
	if covering == nil {
		covering = []match.Cover{{ID: id}}
	}
	return match.Result{Kind: match.Matched, Root: root, ID: id, Exact: exact, Covering: covering}
}

func want(t *testing.T, tt *ui.Tester, texts ...string) {
	t.Helper()
	for _, text := range texts {
		if !tt.HasText(text) {
			t.Errorf("missing %q in %q", text, tt.Texts())
		}
	}
}

func wantNot(t *testing.T, tt *ui.Tester, texts ...string) {
	t.Helper()
	for _, text := range texts {
		if tt.HasText(text) {
			t.Errorf("unexpected %q in %q", text, tt.Texts())
		}
	}
}

func click(t *testing.T, tt *ui.Tester, text string) {
	t.Helper()
	if err := tt.Click(text); err != nil {
		t.Fatalf("%s: %v", text, err)
	}
}

func size(t *testing.T, tt *ui.Tester, text string) (w, h float32) {
	t.Helper()
	r, ok := tt.Find(text)
	if !ok {
		t.Fatalf("no %q in %q", text, tt.Texts())
	}
	return r.W, r.H
}

// fakeClock is a clock the test moves.
type fakeClock struct{ at time.Time }

func (f *fakeClock) now() time.Time { return f.at }

func newClock(s *Screen) *fakeClock {
	f := &fakeClock{at: time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)}
	s.now = f.now
	return f
}

func noMatch() match.Result { return match.Result{Kind: match.NoMatch, Root: notes} }

func TestNoMatchOffersEveryIdentityAndANewOne(t *testing.T) {
	s, tt, r := card(t, noMatch(), work(), client())
	want(t, tt, "No identity matches this folder", "Bind to", "Has ~/work/ · adds another", "Has ~/work/client/ · adds another",
		"New Identity and Bind Here", "~/scratch/notes")
	if tt.HasText("Rebind to") || tt.HasText("Unbind") {
		t.Error("no match, nothing to rebind or unbind")
	}
	// Rows are equally high, whatever they say.
	_, h1 := size(t, tt, "Bind to work")
	_, h2 := size(t, tt, "Bind to client")
	if h1 != h2 {
		t.Errorf("bind rows differ in height: %v %v", h1, h2)
	}
	click(t, tt, "Bind to client")
	want1 := bind.Op{Kind: bind.KindBind, To: "client", Dir: notes, Root: notes}
	if len(r.ran) != 1 || r.ran[0].Kind != want1.Kind || r.ran[0].To != want1.To || r.ran[0].Dir != notes || r.ran[0].Root != notes {
		t.Fatalf("ran %+v", r.ran)
	}
	// While the owner works, no second request goes out.
	click(t, tt, "Bind to work")
	if len(r.ran) != 1 || !s.busy {
		t.Fatalf("a second request while busy: %+v", r.ran)
	}
}

func TestNewIdentityForTheFolder(t *testing.T) {
	_, tt, r := card(t, noMatch(), work())
	click(t, tt, "New Identity and Bind Here")
	if len(r.boundNew) != 1 || r.boundNew[0] != notes || r.newDraft != 0 {
		t.Fatalf("new identity for the folder: %v", r.boundNew)
	}
}

func TestNoMatchWithoutIdentitiesStillOffersANewOne(t *testing.T) {
	r := &recorder{}
	s := New(Text("en"), env, r.actions())
	s.SetAccounts(nil, nil, "")
	tt := ui.NewTester(s.View, 340, 480)
	s.Drop([]string{notes})
	s.SetResult(notes, noMatch(), nil)
	tt.Frame()
	want(t, tt, "New Identity and Bind Here")
	wantNot(t, tt, "Bind to")
}

func TestMatchedByAParentOffersRebindButNotUnbind(t *testing.T) {
	_, tt, r := card(t, matched(api, "work", false), work(), client(), personal())
	want(t, tt, "work", "Open in Finder", "Rebind to")
	wantNot(t, tt, "Unbind", "Undo")
	click(t, tt, "Open in Finder")
	if !slices.Equal(r.folders, []string{api}) {
		t.Fatalf("open: %v", r.folders)
	}
}

func TestExactMatchOffersUnbind(t *testing.T) {
	s, tt, r := card(t, matched(api, "work", true), work(), client())
	_, hOpen := size(t, tt, "Open in Finder")
	_, hRebind := size(t, tt, "Rebind to")
	_, hUnbind := size(t, tt, "Unbind")
	if hOpen != hRebind || hRebind != hUnbind {
		t.Errorf("the card's actions are equally high: %v %v %v", hOpen, hRebind, hUnbind)
	}
	click(t, tt, "Unbind")
	if len(r.ran) != 1 || r.ran[0].Kind != bind.KindUnbind || r.ran[0].From != "work" || r.ran[0].Dir != api || r.ran[0].Root != api {
		t.Fatalf("ran %+v", r.ran)
	}
	// Unbinding leaves no undo bar, however it comes back.
	s.OpDone(bind.Outcome{Op: r.ran[0], Result: matched(api, "client", false)})
	tt.Frame()
	wantNot(t, tt, "Undo")
	if s.Result().ID != "client" {
		t.Error("the card shows the folder looked at again")
	}
}

func TestRebindMenuListsTheOtherIdentities(t *testing.T) {
	s, tt, r := card(t, matched(api, "work", true), work(), client(), personal())
	click(t, tt, "Rebind to")
	want(t, tt, "Rebind to client", "Rebind to personal")
	wantNot(t, tt, "Rebind to work")
	if !s.menuOpen {
		t.Fatal("menu open")
	}
	// Escape closes the menu and leaves the panel open.
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if s.menuOpen || r.hidden != 0 {
		t.Fatalf("Escape closes the menu first: open %v hidden %d", s.menuOpen, r.hidden)
	}
	click(t, tt, "Rebind to")
	click(t, tt, "Rebind to client")
	if len(r.ran) != 1 || r.ran[0].Kind != bind.KindRebind || r.ran[0].From != "work" || r.ran[0].To != "client" {
		t.Fatalf("ran %+v", r.ran)
	}
	if s.menuOpen || s.confirmTo != "" {
		t.Error("choosing closes the menu; a scoped identity needs no confirmation")
	}
	// The menu sits under the card's actions, right-aligned to them, and is as wide as the mockup says.
	s2, tt2, _ := card(t, matched(api, "work", true), work(), client())
	click(t, tt2, "Rebind to")
	entry, _ := tt2.Find("Rebind to client")
	if entry.H <= 0 || s2.menuOpen != true {
		t.Fatalf("entry %v", entry)
	}
	copyBtn, _ := tt2.Find("Clear result")
	if right := entry.X + entry.W; right >= copyBtn.X+copyBtn.W {
		t.Errorf("the menu stays inside the card: %v %v", entry, copyBtn)
	}
}

func TestRebindToAGlobalIdentityAsksFirst(t *testing.T) {
	s, tt, r := card(t, matched(api, "work", true), work(), personal())
	click(t, tt, "Rebind to")
	click(t, tt, "Rebind to personal")
	if len(r.ran) != 0 || s.confirmTo != "personal" || s.menuOpen {
		t.Fatalf("the menu closes and the confirmation opens: %+v %q", r.ran, s.confirmTo)
	}
	want(t, tt, "After binding, personal applies only to this folder, not globally.", "Global identity", "Cancel", "Rebind")
	cw, ch := size(t, tt, "Cancel")
	rw, rh := size(t, tt, "Rebind")
	if cw != rw || ch != rh {
		t.Errorf("both buttons are the same size: %v x %v, %v x %v", cw, ch, rw, rh)
	}
	click(t, tt, "Cancel")
	if s.confirmTo != "" || len(r.ran) != 0 {
		t.Fatal("Cancel closes the block")
	}
	wantNot(t, tt, "Global identity")
	click(t, tt, "Rebind to")
	click(t, tt, "Rebind to personal")
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if s.confirmTo != "" || r.hidden != 0 {
		t.Fatalf("Escape cancels the confirmation: %q %d", s.confirmTo, r.hidden)
	}
	click(t, tt, "Rebind to")
	click(t, tt, "Rebind to personal")
	click(t, tt, "Rebind")
	if len(r.ran) != 1 || r.ran[0].Kind != bind.KindRebind || r.ran[0].To != "personal" || r.ran[0].From != "work" {
		t.Fatalf("ran %+v", r.ran)
	}
}

func TestCoveredBindChangesNothingAndSaysWhy(t *testing.T) {
	s, tt, r := card(t, matched(api, "client", false), work(), client())
	click(t, tt, "Rebind to")
	click(t, tt, "Rebind to work")
	s.OpDone(bind.Outcome{Op: r.ran[0], CoveredID: "work", CoveredBy: "~/work/", Result: matched(api, "client", false)})
	tt.Frame()
	want(t, tt, "work already covers this folder with ~/work/. Nothing changed.")
	wantNot(t, tt, "Undo")
	if got := tt.Announcements(); !slices.Contains(got, "work already covers this folder with ~/work/. Nothing changed.") {
		t.Errorf("announced %v", got)
	}
	if s.busy {
		t.Error("the wait is over")
	}
}

func TestOverlapListsGitsOrderAndOffersToSettleIt(t *testing.T) {
	res := matched("/home/ada/work/client/app", "client", false, match.Cover{ID: "work", Path: "~/work/"}, match.Cover{ID: "client", Path: "~/work/client/"})
	s, tt, r := card(t, res, work(), client())
	want(t, tt, "2 identities cover this folder", "Git reads them in this order; the last one wins.", "Used", "Overridden",
		"Use only client here", "Remove ~/work/ from work", "Open in Finder")
	first, _ := tt.Find("~/work/")
	second, _ := tt.Find("~/work/client/")
	if first.Y >= second.Y {
		t.Errorf("git's read order from top to bottom: %v %v", first, second)
	}
	_, hOnly := size(t, tt, "Use only client here")
	_, hRemove := size(t, tt, "Remove ~/work/ from work")
	if hOnly != hRemove {
		t.Errorf("the stacked actions are equally high: %v %v", hOnly, hRemove)
	}
	click(t, tt, "Use only client here")
	if len(r.ran) != 1 || r.ran[0].Kind != bind.KindUseOnly || r.ran[0].To != "client" {
		t.Fatalf("ran %+v", r.ran)
	}
	s.OpDone(bind.Outcome{Op: r.ran[0], Undo: []account.Account{client()}, Result: res})
	tt.Frame()
	click(t, tt, "Remove ~/work/ from work")
	if len(r.ran) != 2 || r.ran[1].Kind != bind.KindRemovePath || r.ran[1].From != "work" || r.ran[1].Path != "~/work/" {
		t.Fatalf("ran %+v", r.ran)
	}
	// The winner with its own path needs no "Use only".
	res.Exact = true
	_, tt, _ = card(t, res, work(), client())
	wantNot(t, tt, "Use only client here")
	want(t, tt, "Remove ~/work/ from work")
	// One identity covering is no overlap.
	_, tt, _ = card(t, matched(api, "work", false, match.Cover{ID: "work", Path: "~/work/"}), work(), client())
	wantNot(t, tt, "Overridden", "Used")
}

func TestUndoBarCountsDownFromAnAbsoluteDeadline(t *testing.T) {
	s, tt, r := card(t, noMatch(), work(), client())
	clock := newClock(s)
	click(t, tt, "Bind to client")
	before := []account.Account{client()}
	s.OpDone(bind.Outcome{Op: r.ran[0], Undo: before, Result: matched(notes, "client", true)})
	tt.Frame()
	want(t, tt, "Bound to client ·", "Undo", "Unbind")
	if got := tt.Announcements(); !slices.Contains(got, "Bound to client") {
		t.Errorf("announced %v", got)
	}
	if s.undoLeft() != 5 {
		t.Fatalf("steps %d", s.undoLeft())
	}
	// Closing and opening the panel reloads the list; the deadline stays.
	s.SetAccounts([]account.Account{work(), client()}, nil, "")
	for _, c := range []struct {
		after time.Duration
		steps int
	}{{500 * time.Millisecond, 5}, {1500 * time.Millisecond, 4}, {2500 * time.Millisecond, 3}, {3500 * time.Millisecond, 2}, {4500 * time.Millisecond, 1}, {5 * time.Second, 0}} {
		clock.at = time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC).Add(c.after)
		if s.undoLeft() != c.steps {
			t.Errorf("after %v: %d steps, want %d", c.after, s.undoLeft(), c.steps)
		}
	}
	tt.Frame()
	wantNot(t, tt, "Undo", "Bound to client ·")
	s.Undo()
	if len(r.ran) != 1 {
		t.Fatalf("an expired undo does nothing: %+v", r.ran)
	}
}

func TestUndoBeforeExpiryPutsTheIdentitiesBack(t *testing.T) {
	s, tt, r := card(t, noMatch(), work(), client())
	clock := newClock(s)
	click(t, tt, "Bind to client")
	before := []account.Account{client()}
	s.OpDone(bind.Outcome{Op: r.ran[0], Undo: before, Result: matched(notes, "client", true)})
	tt.Frame()
	clock.at = clock.at.Add(4900 * time.Millisecond)
	tt.Frame()
	click(t, tt, "Undo")
	if len(r.ran) != 2 || r.ran[1].Kind != bind.KindUndo || len(r.ran[1].Before) != 1 || r.ran[1].Before[0].ID != "client" || r.ran[1].Dir != notes {
		t.Fatalf("ran %+v", r.ran)
	}
	if !s.busy {
		t.Error("waiting for the undo")
	}
	// The folder is as it was: no match, with "Undone" next to the clear button for a moment.
	s.OpDone(bind.Outcome{Op: r.ran[1], Result: noMatch()})
	tt.Frame()
	want(t, tt, "Undone", "No identity matches this folder")
	wantNot(t, tt, "Bound to client ·")
	if got := tt.Announcements(); !slices.Contains(got, "Undone") {
		t.Errorf("announced %v", got)
	}
	clock.at = clock.at.Add(1300 * time.Millisecond)
	tt.Frame()
	wantNot(t, tt, "Undone")
}

func TestARebindOrUseOnlyIsUndoneWithBothIdentities(t *testing.T) {
	s, tt, r := card(t, matched(api, "work", true), work(), client())
	newClock(s)
	click(t, tt, "Rebind to")
	click(t, tt, "Rebind to client")
	before := []account.Account{work(), client()}
	s.OpDone(bind.Outcome{Op: r.ran[0], Undo: before, Result: matched(api, "client", true)})
	tt.Frame()
	want(t, tt, "Bound to client ·")
	click(t, tt, "Undo")
	if len(r.ran[1].Before) != 2 {
		t.Fatalf("both identities go back: %+v", r.ran[1])
	}
}

func TestAnotherDropReplacesTheBar(t *testing.T) {
	s, tt, r := card(t, noMatch(), work(), client())
	newClock(s)
	click(t, tt, "Bind to client")
	s.OpDone(bind.Outcome{Op: r.ran[0], Undo: []account.Account{client()}, Result: matched(notes, "client", true)})
	s.Drop([]string{"/home/ada/other"})
	s.SetResult("/home/ada/other", match.Result{Kind: match.NotRepo, Dir: "/home/ada/other"}, nil)
	tt.Frame()
	wantNot(t, tt, "Undo", "Bound to client ·")
	if s.undo != nil {
		t.Error("the bar goes with the card")
	}
}

func TestResultOfAnotherFolderOnlyEndsTheWait(t *testing.T) {
	s, tt, r := card(t, noMatch(), work(), client())
	click(t, tt, "Bind to client")
	s.Drop([]string{"/home/ada/other"})
	s.SetResult("/home/ada/other", match.Result{Kind: match.NotRepo, Dir: "/home/ada/other"}, nil)
	s.OpDone(bind.Outcome{Op: r.ran[0], Undo: []account.Account{client()}, Result: matched(notes, "client", true)})
	if s.busy || s.undo != nil || s.Result().Kind != match.NotRepo {
		t.Fatalf("busy %v undo %v result %+v", s.busy, s.undo, s.Result())
	}
}

func TestFailuresShowInTheCard(t *testing.T) {
	s, tt, r := card(t, matched(api, "work", true), work(), client())
	click(t, tt, "Unbind")
	s.OpDone(bind.Outcome{Op: r.ran[0], Err: fmt.Errorf("writing: %w", errors.New("permission denied"))})
	tt.Frame()
	want(t, tt, "Could not change the bindings: permission denied", "Unbind")
	if s.Result().ID != "work" || s.busy {
		t.Error("a failed request leaves the card as it was and ends the wait")
	}
	// The buttons work again.
	click(t, tt, "Unbind")
	if len(r.ran) != 2 {
		t.Fatalf("ran %+v", r.ran)
	}
	// The list is saved but the configuration is not: the change stands, undo stays, and the card says so.
	s.OpDone(bind.Outcome{Op: r.ran[1], Err: &store.RegenerateError{Err: errors.New("disk full")}, Result: matched(api, "work", false)})
	tt.Frame()
	want(t, tt, "Saved, but the generated configuration could not be rewritten: disk full.")
	if s.Result().Exact {
		t.Error("the change stands")
	}
	// A git that fails while looking again.
	s.OpDone(bind.Outcome{Op: bind.Op{Dir: api}, ResultErr: errors.New("no git")})
	tt.Frame()
	if !slices.ContainsFunc(tt.Texts(), func(s string) bool { return strings.HasPrefix(s, "Could not run git") }) {
		t.Errorf("%q", tt.Texts())
	}
}

func TestRegenerateErrorKeepsTheUndoBar(t *testing.T) {
	s, tt, r := card(t, noMatch(), work(), client())
	newClock(s)
	click(t, tt, "Bind to client")
	s.OpDone(bind.Outcome{Op: r.ran[0], Err: &store.RegenerateError{Err: errors.New("disk full")}, Undo: []account.Account{client()},
		Result: matched(notes, "client", true)})
	tt.Frame()
	want(t, tt, "Saved, but the generated configuration could not be rewritten: disk full.", "Undo")
}

func TestRecheckLooksAtTheFolderAgainOnlyForTheCardsFolder(t *testing.T) {
	s, tt, r := card(t, noMatch(), work())
	s.Recheck("/home/ada/elsewhere")
	if len(r.resolved) != 1 {
		t.Fatalf("another folder: %v", r.resolved)
	}
	s.Recheck(notes)
	if len(r.resolved) != 2 || r.resolved[1] != notes {
		t.Fatalf("resolved %v", r.resolved)
	}
	tt.Frame()
	want(t, tt, "Checking…")
	s.SetResult(notes, matched(notes, "work", true), nil)
	tt.Frame()
	want(t, tt, "Unbind")

	win := paths.Env{Home: `C:\Users\Ada`, Windows: true}
	rec := &recorder{}
	ws := New(Text("en"), win, rec.actions())
	ws.SetAccounts([]account.Account{work()}, nil, "")
	ws.Drop([]string{`C:\Users\Ada\Work\Notes`})
	ws.SetResult(`C:\Users\Ada\Work\Notes`, match.Result{Kind: match.NoMatch, Root: "C:/Users/Ada/Work/Notes"}, nil)
	ws.Recheck(`c:/users/ada/work/notes`)
	if len(rec.resolved) != 2 {
		t.Errorf("Windows paths compare ignoring case and slash form: %v", rec.resolved)
	}
	ws.goos = "windows"
	ws.SetResult(`C:\Users\Ada\Work\Notes`, matched("C:/Users/Ada/Work/Notes", "work", false), nil)
	tt = ui.NewTester(ws.View, 340, 480)
	want(t, tt, "Open in Explorer", "~/Work/Notes")
}

func TestBindingTextInChinese(t *testing.T) {
	r := &recorder{}
	s := New(Text("zh-Hans"), env, r.actions())
	s.SetAccounts([]account.Account{work(), client()}, nil, "")
	tt := ui.NewTester(s.View, 340, 480)
	s.Drop([]string{notes})
	s.SetResult(notes, noMatch(), nil)
	tt.Frame()
	want(t, tt, "没有身份匹配这个目录", "绑定到", "已有 ~/work/ · 会再加一条", "新建身份并绑到这里")
	res := matched("/home/ada/work/client/app", "client", false, match.Cover{ID: "work", Path: "~/work/"}, match.Cover{ID: "client", Path: "~/work/client/"})
	s.Drop([]string{res.Root})
	s.SetResult(res.Root, res, nil)
	tt.Frame()
	want(t, tt, "2 个身份覆盖这里", "这里只用 client", "从 work 去掉 ~/work/", "生效", "被覆盖", "改绑到")
	s.OpDone(bind.Outcome{Op: bind.Op{Dir: res.Root}, CoveredID: "work", CoveredBy: "~/work/", Result: res})
	tt.Frame()
	want(t, tt, "work 的 ~/work/ 已经覆盖这里，没有改动")
}

func TestKeyboardReachesTheBindRows(t *testing.T) {
	_, tt, r := card(t, noMatch(), work())
	for range 10 {
		if tt.Focused("Bind to work") {
			break
		}
		tt.Key(0, ui.KeyTab)
	}
	if !tt.Focused("Bind to work") {
		t.Fatal("Tab reaches the row")
	}
	tt.Key(0, ui.KeyEnter)
	if len(r.ran) != 1 || r.ran[0].To != "work" {
		t.Fatalf("Enter binds: %+v", r.ran)
	}
}
