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

// shows reports whether the frame shows text, either as one text or as the pieces of a path line side by side,
// whose spaces are padding and not text.
func shows(tt *ui.Tester, text string) bool {
	flat := func(s string) string { return strings.ReplaceAll(s, " ", "") }
	return tt.HasText(text) || strings.Contains(flat(strings.Join(tt.Texts(), "")), flat(text))
}

func want(t *testing.T, tt *ui.Tester, texts ...string) {
	t.Helper()
	for _, text := range texts {
		if !shows(tt, text) {
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

// same compares sizes, which differ in the last bits of a float.
func same(a, b float32) bool { return a-b < 0.01 && b-a < 0.01 }

// openMenu opens the card's "⋯" menu and returns its entries, "-" for a separator.
func openMenu(t *testing.T, s *Screen, tt *ui.Tester) []string {
	t.Helper()
	click(t, tt, s.text.More)
	return tt.Menu()
}

// choose chooses an entry of the open menu.
func choose(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	if err := tt.ChooseMenuItem(label); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
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
	want(t, tt, "No identity matches this folder", "Bind to", "~/work/", "~/work/client/", "New Identity", "~/scratch/notes")
	wantNot(t, tt, "adds another", "More", "Unbind")
	if _, ok := tt.Find("New Identity for This Folder"); !ok {
		t.Error("the long label names the short button")
	}
	// Rows are equally high, whatever they say.
	_, h1 := size(t, tt, "Bind to work")
	_, h2 := size(t, tt, "Bind to client")
	if !same(h1, h2) {
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
	click(t, tt, "New Identity for This Folder")
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
	want(t, tt, "New Identity for This Folder")
	wantNot(t, tt, "Bind to")
}

func TestMatchedByAParentOffersRebindButNotUnbind(t *testing.T) {
	s, tt, r := card(t, matched(api, "work", false), work(), client(), personal())
	want(t, tt, "work", s.text.More)
	wantNot(t, tt, "Unbind", "Undo", s.openLabel(), "Copy Path")
	// The menu opens the folder, copies its path and chooses another, then lists the others, and has no Unbind:
	// only a parent folder covers this one.
	if got := openMenu(t, s, tt); !slices.Equal(got, []string{s.openLabel(), "Copy Path", "Choose Another Folder…", "-", "Rebind to", "client", "personal"}) {
		t.Errorf("menu %q", got)
	}
	choose(t, tt, s.openLabel())
	if !slices.Equal(r.folders, []string{api}) {
		t.Fatalf("open: %v", r.folders)
	}
	openMenu(t, s, tt)
	choose(t, tt, "Copy Path")
	if !slices.Equal(r.copied, []string{api}) {
		t.Errorf("copied %q", r.copied)
	}
	want(t, tt, "Copied")
	// With nobody to rebind to and nothing to unbind, the menu keeps the folder's entries and no separator.
	s, tt, _ = card(t, matched(api, "work", false), work())
	if got := openMenu(t, s, tt); !slices.Equal(got, []string{s.openLabel(), "Copy Path", "Choose Another Folder…"}) {
		t.Errorf("menu %q", got)
	}
}

func TestExactMatchOffersUnbind(t *testing.T) {
	s, tt, r := card(t, matched(api, "work", true), work(), client())
	wantNot(t, tt, "Unbind")
	wo, hClone := size(t, tt, s.text.CloneToggle)
	wm, hMore := size(t, tt, s.text.More)
	wc, hClear := size(t, tt, s.text.Clear)
	if !same(hClone, hMore) || !same(hMore, hClear) || !same(wo, wm) || !same(wm, wc) {
		t.Errorf("the icon buttons are the same size: %v x %v, %v x %v, %v x %v", wo, hClone, wm, hMore, wc, hClear)
	}
	// Unbind is in the menu, below the identities, only when the identity lists this very folder.
	if got := openMenu(t, s, tt); !slices.Equal(got[4:], []string{"Rebind to", "client", "-", "Unbind"}) {
		t.Errorf("Unbind comes last: %q", got)
	}
	choose(t, tt, "Unbind")
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
	if got := openMenu(t, s, tt); !slices.Equal(got[4:], []string{"Rebind to", "client", "personal", "-", "Unbind"}) {
		t.Fatalf("menu %q", got)
	}
	choose(t, tt, "client")
	if len(r.ran) != 1 || r.ran[0].Kind != bind.KindRebind || r.ran[0].From != "work" || r.ran[0].To != "client" {
		t.Fatalf("ran %+v", r.ran)
	}
	if s.confirmTo != "" {
		t.Error("a scoped identity needs no confirmation")
	}
	// The heading is not a choice.
	s2, tt2, r2 := card(t, matched(api, "work", true), work(), client())
	openMenu(t, s2, tt2)
	if err := tt2.ChooseMenuItem("Rebind to"); err == nil {
		tt2.Frame()
	}
	if len(r2.ran) != 0 {
		t.Fatalf("ran %+v", r2.ran)
	}
}

func TestRebindToAGlobalIdentityAsksFirst(t *testing.T) {
	s, tt, r := card(t, matched(api, "work", true), work(), personal())
	openMenu(t, s, tt)
	choose(t, tt, "personal")
	if len(r.ran) != 0 || s.confirmTo != "personal" {
		t.Fatalf("the menu closes and the confirmation opens: %+v %q", r.ran, s.confirmTo)
	}
	want(t, tt, "personal will apply only here", "Cancel", "Rebind")
	wantNot(t, tt, "Global identity")
	// One line of text above the buttons.
	_, line := size(t, tt, "Cancel")
	if _, h := size(t, tt, "personal will apply only here"); h > line {
		t.Errorf("the question is one line: %v", h)
	}
	within(t, tt, "personal will apply only here", "Cancel", "Rebind")
	cw, ch := size(t, tt, "Cancel")
	rw, rh := size(t, tt, "Rebind")
	if !same(cw, rw) || !same(ch, rh) {
		t.Errorf("both buttons are the same size: %v x %v, %v x %v", cw, ch, rw, rh)
	}
	click(t, tt, "Cancel")
	if s.confirmTo != "" || len(r.ran) != 0 {
		t.Fatal("Cancel closes the block")
	}
	wantNot(t, tt, "personal will apply only here")
	openMenu(t, s, tt)
	choose(t, tt, "personal")
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if s.confirmTo != "" || r.hidden != 0 {
		t.Fatalf("Escape cancels the confirmation: %q %d", s.confirmTo, r.hidden)
	}
	openMenu(t, s, tt)
	choose(t, tt, "personal")
	click(t, tt, "Rebind")
	if len(r.ran) != 1 || r.ran[0].Kind != bind.KindRebind || r.ran[0].To != "personal" || r.ran[0].From != "work" {
		t.Fatalf("ran %+v", r.ran)
	}
}

func TestCoveredBindChangesNothingAndSaysWhy(t *testing.T) {
	s, tt, r := card(t, matched(api, "client", false), work(), client())
	openMenu(t, s, tt)
	choose(t, tt, "work")
	s.OpDone(bind.Outcome{Op: r.ran[0], CoveredID: "work", CoveredBy: "~/work/", Result: matched(api, "client", false)})
	tt.Frame()
	// The card keeps its answer; its context line says "Covered by …" for a moment, the full sentence is a tooltip.
	want(t, tt, "Covered by ~/work/")
	wantNot(t, tt, "Undo", "Nothing changed")
	// The status sits in the answer row before the buttons, whole.
	status, _ := tt.Find("Covered by ~/work/")
	more, _ := tt.Find(s.text.More)
	if status.X+status.W > more.X || status.Y+status.H < more.Y || status.Y > more.Y+more.H {
		t.Errorf("status %v, more %v", status, more)
	}
	within(t, tt, "Covered by ~/work/")
	if tip := hover(t, tt, "Covered by ~/work/"); !tip("work already covers this folder with ~/work/. Nothing changed.") {
		t.Errorf("the full sentence is the tooltip: %q", tt.Texts())
	}
	if s.note != "work already covers this folder with ~/work/. Nothing changed." {
		t.Errorf("tooltip %q", s.note)
	}
	s.now = func() time.Time { return time.Now().Add(2 * time.Second) }
	tt.Frame()
	wantNot(t, tt, "Covered by")
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
	want(t, tt, "2 identities cover this folder", "Git reads them in order; the last one wins.", "Used", "Overridden",
		"Use Only client Here", "Remove ~/work/ from work", s.text.More)
	first, _ := tt.Find("~/work/")
	second, _ := tt.Find("~/work/client/")
	if first.Y >= second.Y {
		t.Errorf("git's read order from top to bottom: %v %v", first, second)
	}
	wantNot(t, tt, "Rebind to")
	// The pin, the minus and the info mark are icon buttons the size of the clear button.
	wc, hc := size(t, tt, "Clear")
	for _, name := range []string{"Use Only client Here", "Remove ~/work/ from work", "Git reads them in order; the last one wins."} {
		if w, h := size(t, tt, name); !same(w, wc) || !same(h, hc) {
			t.Errorf("%s is %v x %v, the clear button %v x %v", name, w, h, wc, hc)
		}
	}
	// Each sits at the end of its row: the minus by the overridden identity, the pin by the one Git uses.
	pin, _ := tt.Find("Use Only client Here")
	minus, _ := tt.Find("Remove ~/work/ from work")
	if !same(pin.X, minus.X) || minus.Y >= pin.Y || pin.Y <= second.Y-pin.H || minus.Y <= first.Y-minus.H {
		t.Errorf("pin %v minus %v rows %v %v", pin, minus, first, second)
	}
	within(t, tt, "2 identities cover this folder", "Use Only client Here", "Remove ~/work/ from work")
	click(t, tt, "Use Only client Here")
	if len(r.ran) != 1 || r.ran[0].Kind != bind.KindUseOnly || r.ran[0].To != "client" {
		t.Fatalf("ran %+v", r.ran)
	}
	s.OpDone(bind.Outcome{Op: r.ran[0], Undo: []account.Account{client()}, Result: res})
	tt.Frame()
	click(t, tt, "Remove ~/work/ from work")
	if len(r.ran) != 2 || r.ran[1].Kind != bind.KindRemovePath || r.ran[1].From != "work" || r.ran[1].Path != "~/work/" {
		t.Fatalf("ran %+v", r.ran)
	}
	// The winner with its own path is an explicit choice: no overlap card, and after "Use only" it goes away.
	res.Exact = true
	_, tt, _ = card(t, res, work(), client())
	wantNot(t, tt, "cover this folder", "Use Only client Here", "Remove ~/work/ from work")
	want(t, tt, s.text.More)
	s.OpDone(bind.Outcome{Op: r.ran[1], Result: res})
	tt.Frame()
	wantNot(t, tt, "cover this folder")
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
	want(t, tt, "Bound to client ·", "Undo", "More")
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
	openMenu(t, s, tt)
	choose(t, tt, "client")
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
	openMenu(t, s, tt)
	choose(t, tt, "Unbind")
	s.OpDone(bind.Outcome{Op: r.ran[0], Err: fmt.Errorf("writing: %w", errors.New("permission denied"))})
	tt.Frame()
	want(t, tt, "Could not change the bindings: permission denied", "More")
	if s.Result().ID != "work" || s.busy {
		t.Error("a failed request leaves the card as it was and ends the wait")
	}
	// The buttons work again.
	openMenu(t, s, tt)
	choose(t, tt, "Unbind")
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
	want(t, tt, "More")

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
	want(t, tt, "~/Work/Notes")
	if got := openMenu(t, ws, tt); got[0] != "Open in Explorer" {
		t.Errorf("menu %q", got)
	}
}

func TestBindingTextInChinese(t *testing.T) {
	r := &recorder{}
	s := New(Text("zh-Hans"), env, r.actions())
	s.SetAccounts([]account.Account{work(), client()}, nil, "")
	tt := ui.NewTester(s.View, 340, 480)
	s.Drop([]string{notes})
	s.SetResult(notes, noMatch(), nil)
	tt.Frame()
	want(t, tt, "没有身份匹配这个目录", "绑定到", "~/work/", "新建身份", "新建身份并绑定这个文件夹")
	wantNot(t, tt, "会再加一条")
	res := matched("/home/ada/work/client/app", "client", false, match.Cover{ID: "work", Path: "~/work/"}, match.Cover{ID: "client", Path: "~/work/client/"})
	s.Drop([]string{res.Root})
	s.SetResult(res.Root, res, nil)
	tt.Frame()
	want(t, tt, "2 个身份覆盖这里", "这里只用 client", "从 work 去掉 ~/work/", "生效", "被覆盖", "更多", "清除")
	if got := openMenu(t, s, tt); !slices.Equal(got[:3], []string{s.openLabel(), "复制路径", "选择其他文件夹…"}) {
		t.Errorf("menu %q", got)
	}
	tt.CloseMenu()
	s.OpDone(bind.Outcome{Op: bind.Op{Dir: res.Root}, CoveredID: "work", CoveredBy: "~/work/", Result: res})
	tt.Frame()
	want(t, tt, "已由 ~/work/ 覆盖")
	if s.note != "work 的 ~/work/ 已经覆盖这里，没有改动" {
		t.Errorf("tooltip %q", s.note)
	}
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

func TestBindRowsStayUniformWhateverTheyHoldToSay(t *testing.T) {
	long := identity("a-very-long-identity-name-that-cannot-fit-in-one-line", "", "x@example.com", account.GitHub, 0,
		"~/a/very/long/path/that/would/wrap/on/a/narrow/panel/with/any/font/at/all/")
	_, tt, _ := card(t, noMatch(), work(), long, client())
	_, h := size(t, tt, "Bind to work")
	for _, name := range []string{"Bind to client", "Bind to " + long.Label} {
		if _, got := size(t, tt, name); !same(got, h) {
			t.Errorf("%s is %v high, the first row %v", name, got, h)
		}
	}
}

// band is how high the plain band around y is at x in the last frame: from the first row above y of another color
// (a border or hairline) to the first such row below it, or to the bottom of the window.
func band(tt *ui.Tester, x, y int) int {
	img := tt.Image()
	ref := img.RGBAAt(x, y)
	top, bottom := y, y
	for top > 0 && img.RGBAAt(x, top) == ref {
		top--
	}
	for bottom < img.Bounds().Max.Y && img.RGBAAt(x, bottom) == ref {
		bottom++
	}
	return bottom - top
}

func TestCardHeaderIsThe36DIPContextLine(t *testing.T) {
	_, tt, _ := card(t, noMatch(), work(), client())
	clear, _ := tt.Find("Clear")
	// Left of the clear button, from the card's top edge to its hairline.
	header := band(tt, int(clear.X)-2, int(clear.Y+clear.H/2))
	if d := header - 36; d > 1 || d < -1 {
		t.Errorf("the card header is %d high, want 36", header)
	}
}

func TestMatchedCardPutsTheAnswerFirstAndTheDetailsUnderIt(t *testing.T) {
	s, tt, _ := card(t, matched(api, "work", false), work(), client())
	name, _ := tt.Find("work")
	path, _ := tt.Find("work/api")
	email, _ := tt.Find("bob@example.com")
	alias, _ := tt.Find(" · github-work")
	if !(name.Y < path.Y && path.Y < email.Y) {
		t.Errorf("name %v, folder %v, email %v from top to bottom", name, path, email)
	}
	if d := email.X - name.X; d > 1 || d < -1 {
		t.Errorf("the details start under the name: %v %v", email, name)
	}
	if d := (alias.Y + alias.H/2) - (email.Y + email.H/2); d > 1 || d < -1 || alias.X < email.X+email.W-1 {
		t.Errorf("the alias follows the email on its line: %v %v", alias, email)
	}
	if tip := hover(t, tt, "work"); !tip(s.text.UsesHere) {
		t.Errorf("the name's tooltip says what it means: %q", tt.Texts())
	}
	wantNot(t, tt, "Folders", "Alias", "Email")
}

func TestOverlapShowsTheFolderOnce(t *testing.T) {
	res := matched("/home/ada/work/client/app", "client", false, match.Cover{ID: "work", Path: "~/work/"}, match.Cover{ID: "client", Path: "~/work/client/"})
	s, tt, _ := card(t, res, work(), client())
	if n := count(tt, "Clear"); n != 1 {
		t.Errorf("%d clear buttons", n)
	}
	if n := count(tt, "work/client/app"); n != 1 {
		t.Errorf("the folder shows %d times: %q", n, tt.Texts())
	}
	// The overlap card comes first and starts with its title; the matched card's context line is under it.
	title, _ := tt.Find("2 identities cover this folder")
	path, _ := tt.Find("work/client/app")
	if title.Y >= path.Y {
		t.Errorf("title %v, path %v", title, path)
	}
	click(t, tt, "Clear")
	if s.Result() != nil {
		t.Error("the one clear button clears both cards")
	}
	wantNot(t, tt, "cover this folder")
}
