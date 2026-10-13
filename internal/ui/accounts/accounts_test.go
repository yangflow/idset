package accounts

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/keyfile"
	"github.com/yangflow/idset/internal/store"
	"github.com/yangflow/idset/internal/ui/locale"
)

func (r *rig) want(t *testing.T, texts ...string) {
	t.Helper()
	for _, text := range texts {
		if !r.tt.HasText(text) {
			t.Errorf("missing %q in %q", text, r.tt.Texts())
		}
	}
}

func (r *rig) wantNot(t *testing.T, texts ...string) {
	t.Helper()
	for _, text := range texts {
		if r.tt.HasText(text) {
			t.Errorf("unexpected %q in %q", text, r.tt.Texts())
		}
	}
}

func (r *rig) click(t *testing.T, text string) {
	t.Helper()
	if err := r.tt.Click(text); err != nil {
		t.Fatal(err)
	}
}

// type replaces the value of the row with the given label by typing into it.
func (r *rig) typeInto(t *testing.T, label, text string) {
	t.Helper()
	box, ok := r.tt.Find(label)
	if !ok {
		t.Fatalf("no row %q in %q", label, r.tt.Texts())
	}
	r.tt.ClickAt(box.X+230, box.Y+box.H/2)
	r.tt.Key(ui.Cmd, ui.KeyA)
	r.tt.Type(text)
}

func (r *rig) form() *form { return r.s.selectedForm() }

// saveSucceeds answers the Save the window asked for as the owner does: the stored list now holds a.
func (r *rig) saveSucceeds(t *testing.T, err error) account.Account {
	t.Helper()
	if len(r.rec.saves) == 0 {
		t.Fatal("no save was asked for")
	}
	call := r.rec.saves[len(r.rec.saves)-1]
	list := slices.Clone(r.s.list)
	if i := slices.IndexFunc(list, func(a account.Account) bool { return a.ID == call.a.ID }); i >= 0 {
		list[i] = call.a
	} else {
		list = append(list, call.a)
	}
	r.s.SaveDone(call.a.ID, list, err)
	r.tt.Frame()
	return call.a
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
	if len(translations) != len(locale.Languages) {
		t.Error("the catalog has a language the interface does not")
	}
	if Text("zh-CN").Save != "保存" || Text("fr").Save != "Save" {
		t.Error("locale selection")
	}
}

func TestLoadingAndReadOnly(t *testing.T) {
	rec := &recorder{}
	s := New(Text("en"), env, rec.actions())
	tt := ui.NewTester(s.View, 760, 520)
	if !tt.HasText("Reading configuration…") || tt.HasText("No identities yet") {
		t.Fatalf("loading: %q", tt.Texts())
	}
	s.Open(Request{ID: personalID})
	s.SetAccounts(sample(), nil, "/home/ada/.config/idset/accounts.json")
	tt.Frame()
	if s.Selected() != personalID {
		t.Fatalf("the request made while loading must apply: %q", s.Selected())
	}
	s.SetAccounts(nil, errors.New("accounts.json cannot be read"), "/home/ada/.config/idset/accounts.json")
	tt.Frame()
	for _, want := range []string{"accounts.json cannot be read", "~/.config/idset/accounts.json"} {
		if !tt.HasText(want) {
			t.Errorf("read-only: missing %q in %q", want, tt.Texts())
		}
	}
	s.newDraft()
	if s.Draft() != "" || len(rec.prefilled) != 0 {
		t.Error("no new identity while accounts.json cannot be read")
	}
}

func TestOpeningSelectsInOrder(t *testing.T) {
	list := sample()
	// Storage order, not display order, decides: miscellaneous was added first.
	stored := []account.Account{list[3], list[0], list[1], list[2]}
	cases := []struct {
		name string
		req  Request
		want string
	}{
		{"nothing asked", Request{}, miscID},
		{"an identity asked", Request{ID: clientID}, clientID},
		{"an unknown identity asked", Request{ID: "nope"}, miscID},
	}
	for _, c := range cases {
		r := open(t, "en", stored, c.req)
		if r.s.Selected() != c.want {
			t.Errorf("%s: selected %q, want %q", c.name, r.s.Selected(), c.want)
		}
	}
	r := open(t, "en", stored, Request{New: true, ID: clientID})
	if r.s.Selected() != r.s.Draft() || r.s.Draft() == "" {
		t.Errorf("a new draft beats an identity: %q %q", r.s.Selected(), r.s.Draft())
	}
	// Asking again without a preference keeps what is selected; asking for another identity moves there.
	r.s.Open(Request{})
	if r.s.Selected() != r.s.Draft() {
		t.Error("an empty request must not change the selection")
	}
	r.s.Open(Request{ID: workID})
	if r.s.Selected() != workID {
		t.Error("an identity request selects it")
	}
	r.s.Open(Request{New: true})
	if len(r.rec.prefilled) != 1 {
		t.Errorf("one draft at a time, prefilled %d times", len(r.rec.prefilled))
	}
}

func TestListShowsEveryIdentityInPanelOrder(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: workID})
	texts := r.tt.Texts()
	order := []string{"work", "personal", "client", "(unnamed)"}
	last := -1
	for _, name := range order {
		i := slices.Index(texts, name)
		if i < 0 || i < last {
			t.Fatalf("order of %v in %q", order, texts)
		}
		last = i
	}
	r.want(t, "Identities", "4", "bob@example.com", "bob@client.example", "nobody@example.com", "Add Identity")
	r.click(t, "personal")
	if r.s.Selected() != personalID {
		t.Fatalf("clicking a row selects it: %q", r.s.Selected())
	}
	r.want(t, "Global", "Used when no folder rule matches.")
}

func TestEmptyStateOffersNewOnly(t *testing.T) {
	r := open(t, "en", []account.Account{}, Request{})
	r.want(t, "No identities yet", "New")
	// No toolbar and no sidebar: the centered button is the only way to add.
	r.wantNot(t, "Import", "Revert", "Save", "Add Identity")
	r.click(t, "New")
	if r.s.Draft() == "" || !slices.Equal(r.rec.prefilled, []string{r.s.Draft()}) {
		t.Fatalf("the draft must ask for the signature: %v", r.rec.prefilled)
	}
	r.tt.Frame()
	r.want(t, "New identity, not saved yet", "Revert", "Save")
	r.wantNot(t, "No identities yet")
}

func TestGlobalIdentity(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: personalID})
	r.want(t, "personal", "Global", "Identity", "SSH", "Scope", "URL rewrites", "Notes", "Times",
		"Used when no folder rule matches.", "Add Rewrite", "Add Platform Default Rewrite", "Created", "Updated", "Last used", "2026-10-11")
	r.wantNot(t, "Add Path")
	f := r.form()
	if !f.cur.global || f.scope != 0 || f.dirty() {
		t.Errorf("form %+v", f.cur)
	}
	want := values{label: "personal", username: "bob-example", gitName: "Bob", email: "bob@example.net", alias: "github-personal",
		keyPath: "~/.ssh/id_ed25519", provider: account.GitHub, port: 443, portText: "443", global: true, paths: []string{},
		rewrites: []account.Rewrite{}, color: "green"}
	if !reflect.DeepEqual(f.cur, want) {
		t.Errorf("form %+v", f.cur)
	}
}

func TestScopedIdentityListsEveryDetailGroup(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: workID})
	r.want(t, "Scoped", "Identity", "SSH", "Scope", "URL rewrites", "Notes", "Times", "Name", "Platform", "Username", "Git name", "Email",
		"Alias", "Private key", "Port", "Choose File…", "Applies to", "Add Path", "Push only", "Add Rewrite", "Created", "Updated", "Last used")
	f := r.form()
	if f.cur.label != "work" || f.cur.username != "bob-work" || f.cur.gitName != "Bob" || f.cur.alias != "github-work" ||
		f.cur.keyPath != "~/.ssh/id_ed25519_work" || f.hex != "#3A7BD5" || f.cur.notes != "Employer repositories." ||
		!slices.Equal(f.cur.paths, []string{"~/work/", "~/src/new/", "~/clients/acme/"}) || len(f.cur.rewrites) != 2 {
		t.Errorf("form %+v", f.cur)
	}
	if n := strings.Count(strings.Join(r.tt.Texts(), "|"), "Remove this path"); n != 3 {
		t.Errorf("one remove button per path, got %d", n)
	}
	r.wantNot(t, "Used when no folder rule matches.", "Host name", "New identity, not saved yet")
	if r.form().cur.port != 443 {
		t.Error("port 443 is chosen")
	}
	// Delete exists for a saved identity, Revert and Save wait for a change.
	r.want(t, "Delete")
	if f := r.form(); f.dirty() {
		t.Error("a fresh form is not dirty")
	}
	r.wantNot(t, "Unsaved changes")
}

func TestCustomPlatformHasHostAndPort(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: clientID})
	r.want(t, "Host name", "Port")
	r.wantNot(t, "Add Platform Default Rewrite")
	if f := r.form(); f.cur.hostName != "git.client.example" || f.cur.portText != "22" {
		t.Errorf("form %+v", f.cur)
	}
	// Port is a text value for a custom platform, two radios otherwise.
	if r.tt.HasText("443") {
		t.Errorf("a custom port is not a radio choice: %q", r.tt.Texts())
	}
	r.s.selectID(workID)
	r.tt.Frame()
	r.want(t, "443", "Add Platform Default Rewrite")
	r.wantNot(t, "Host name")
}

func TestEditingMakesTheFormDirtyAndRevertGoesBack(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: workID})
	r.typeInto(t, "Username", "bob-new")
	r.tt.Frame()
	if f := r.form(); f.cur.username != "bob-new" || !f.dirty() {
		t.Fatalf("typed username %q", r.form().cur.username)
	}
	r.want(t, "Unsaved changes")
	r.click(t, "Revert")
	if f := r.form(); f.cur.username != "bob-work" || f.dirty() {
		t.Fatalf("revert: %q", f.cur.username)
	}
	r.wantNot(t, "Unsaved changes")
	// The edits stay while another identity is looked at.
	r.typeInto(t, "Username", "bob-new")
	r.click(t, "personal")
	r.click(t, "work")
	if r.form().cur.username != "bob-new" {
		t.Error("edits are kept when the selection moves away and back")
	}
}

func TestChoicesChangeTheForm(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: personalID})
	// Scope: global to folders gives one default path; back to global drops the paths.
	r.click(t, "Limited to folders")
	if f := r.form(); f.cur.global || !slices.Equal(f.cur.paths, []string{"~/"}) {
		t.Fatalf("scope: %+v", f.cur)
	}
	r.want(t, "Add Path")
	r.click(t, "Remove this path")
	if len(r.form().cur.paths) != 1 {
		t.Error("the last path cannot be removed")
	}
	r.click(t, "Add Path")
	if len(r.form().cur.paths) != 2 {
		t.Error("Add Path adds a path")
	}
	r.click(t, "Global")
	if f := r.form(); !f.cur.global || len(f.cur.paths) != 0 {
		t.Fatalf("back to global: %+v", f.cur)
	}
	// Port radios.
	r.click(t, "22")
	if r.form().cur.port != 22 {
		t.Error("choosing 22")
	}
	// Color choices.
	r.click(t, "Red")
	if f := r.form(); f.cur.color != "red" || f.hex != "#D64545" {
		t.Errorf("red: %q %q", f.cur.color, f.hex)
	}
	// Platform: Gitea has no port 443, Custom keeps the port as text.
	r.form().setProvider(account.Gitea)
	if f := r.form(); f.cur.port != 22 {
		t.Errorf("gitea port %d", f.cur.port)
	}
	r.form().setProvider(account.Custom)
	if f := r.form(); f.cur.portText != "22" {
		t.Errorf("custom port text %q", f.cur.portText)
	}
	r.form().cur.portText = "443"
	r.form().setProvider(account.GitHub)
	if f := r.form(); f.cur.port != 443 || !f.dirty() {
		t.Errorf("github port %d", f.cur.port)
	}
}

func TestPlatformSelectChangesProvider(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: workID})
	r.click(t, "GitHub")
	r.click(t, "Gitea")
	if f := r.form(); f.cur.provider != account.Gitea || f.cur.port != 22 {
		t.Fatalf("provider %q port %d", f.cur.provider, f.cur.port)
	}
}

func TestRewrites(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: personalID})
	r.wantNot(t, "Push only", "Enter an alias")
	r.click(t, "Add Platform Default Rewrite")
	r.click(t, "Add Platform Default Rewrite")
	want := []account.Rewrite{
		{From: "https://github.com/", To: "git@github-personal:"},
		{From: "git@github.com:", To: "git@github-personal:"},
	}
	if got := r.form().cur.rewrites; !slices.Equal(got, want) {
		t.Fatalf("defaults: %+v", got)
	}
	r.tt.Frame()
	// Both defaults are there: the button has nothing more to add.
	r.click(t, "Add Platform Default Rewrite")
	if len(r.form().cur.rewrites) != 2 {
		t.Error("a default rule is added once")
	}
	r.click(t, "Add Rewrite")
	if len(r.form().cur.rewrites) != 3 {
		t.Error("Add Rewrite adds an empty rule")
	}
	r.click(t, "Remove this rewrite")
	if len(r.form().cur.rewrites) != 2 {
		t.Error("Remove deletes one rule")
	}
	r.form().cur.rewrites[0].Push = true
	a, _ := r.form().account(*r.s.stored(personalID), env)
	// The empty rule that "Remove" left is not saved.
	if len(a.URLRewrites) != 1 || !a.URLRewrites[0].Push {
		t.Errorf("saved rules %+v", a.URLRewrites)
	}
	// Without an alias the platform default cannot be added, and the reason is shown.
	r.form().cur.alias = ""
	r.tt.Frame()
	r.want(t, "Enter an alias to add the platform default rewrite.")
	if _, ok := r.form().nextDefaultRewrite(); ok {
		t.Error("no default rule without an alias")
	}
	before := len(r.form().cur.rewrites)
	r.click(t, "Add Platform Default Rewrite")
	if len(r.form().cur.rewrites) != before {
		t.Error("the disabled button must not add a rule")
	}
}

func TestDraftPrefillFillsOnlyEmptyFields(t *testing.T) {
	r := open(t, "en", sample(), Request{New: true})
	id := r.s.Draft()
	if !slices.Equal(r.rec.prefilled, []string{id}) {
		t.Fatalf("prefill asked: %v", r.rec.prefilled)
	}
	if f := r.form(); f.cur.color != "red" || !f.cur.global || f.cur.provider != account.GitHub || f.cur.port != 443 {
		t.Errorf("the draft starts as %+v", f.cur)
	}
	r.tt.Frame()
	r.want(t, "New", "New identity, not saved yet")
	r.wantNot(t, "From git config --global", "Times", "Delete")
	// The user starts typing the email before git answers.
	r.form().cur.email = "typed@example.com"
	r.s.SetPrefill(id, "Ada", "git@example.com")
	f := r.form()
	if f.cur.gitName != "Ada" || f.cur.email != "typed@example.com" {
		t.Fatalf("only empty fields are filled: %q %q", f.cur.gitName, f.cur.email)
	}
	r.tt.Frame()
	if n := strings.Count(strings.Join(r.tt.Texts(), "|"), "From git config --global"); n != 1 {
		t.Errorf("one hint for the one prefilled field, got %d in %q", n, r.tt.Texts())
	}
	// Editing the prefilled value takes the hint away.
	f.cur.gitName = "Ada L"
	r.tt.Frame()
	r.wantNot(t, "From git config --global")
	// An answer for a draft that is gone, or for a saved identity, is ignored.
	r.s.SetPrefill("nope", "X", "x@example.com")
	r.s.SetPrefill(workID, "X", "x@example.com")
	if r.s.forms[workID] != nil && r.s.forms[workID].cur.gitName == "X" {
		t.Error("prefill only goes into a draft")
	}
}

func TestRevertDiscardsTheDraft(t *testing.T) {
	r := open(t, "en", sample(), Request{New: true})
	r.click(t, "Revert")
	if r.s.Draft() != "" || r.s.Selected() != workID {
		t.Fatalf("draft %q selected %q", r.s.Draft(), r.s.Selected())
	}
	r = open(t, "en", []account.Account{}, Request{New: true})
	r.click(t, "Revert")
	r.want(t, "No identities yet")
}

func TestSaveWithAShortcutAndTheResultFlash(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: workID})
	clock := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	r.s.now = func() time.Time { return clock }
	r.tt.Key(ui.Cmd, ui.KeyS)
	if len(r.rec.saves) != 0 {
		t.Fatal("nothing to save yet")
	}
	r.typeInto(t, "Alias", "  github-work2 ")
	r.typeInto(t, "Private key", " /home/ada/.ssh/id_new ")
	r.form().cur.paths[0] = "/home/ada/work"
	r.tt.Frame()
	r.tt.Key(ui.Cmd, ui.KeyS)
	if len(r.rec.saves) != 1 || r.rec.saves[0].isNew {
		t.Fatalf("saves: %+v", r.rec.saves)
	}
	got := r.rec.saves[0].a
	if got.SSHAlias != "github-work2" || got.KeyPath != "~/.ssh/id_new" || got.Scope.Paths[0] != "~/work/" {
		t.Errorf("the saved identity is trimmed and in stored form: %q %q %v", got.SSHAlias, got.KeyPath, got.Scope.Paths)
	}
	if got.KeyFingerprint != "" || got.LastUsedAt == nil || !got.CreatedAt.Equal(base) {
		t.Errorf("a changed key drops the fingerprint, other fields stay: %+v", got)
	}
	// While the store works the form is disabled, with a spinner.
	r.tt.Frame()
	r.want(t, "Writing…", "Saving")
	r.wantNot(t, "Unsaved changes", "Save")
	r.tt.Key(ui.Cmd, ui.KeyS)
	if len(r.rec.saves) != 1 {
		t.Error("a second save while saving")
	}
	r.saveSucceeds(t, nil)
	r.want(t, "Written")
	r.wantNot(t, "Writing…", "Unsaved changes", "Saving", "Save")
	if got := r.tt.Announcements(); !slices.Contains(got, "Written") {
		t.Errorf("the save must be announced: %v", got)
	}
	if f := r.form(); f.dirty() || f.cur.alias != "github-work2" {
		t.Error("the form shows what was stored")
	}
	clock = clock.Add(2 * time.Second)
	r.tt.Frame()
	r.wantNot(t, "Written")
	r.want(t, "Save")
}

func TestSaveFailureStaysDirtyAndSaysWhy(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: workID})
	r.typeInto(t, "Username", "bob-new")
	r.click(t, "Save")
	r.saveSucceeds(t, fmt.Errorf("write: %w", errors.New("permission denied")))
	r.want(t, "Could not write ~/.config/idset/accounts.json: permission denied")
	if f := r.form(); !f.dirty() || f.cur.username != "bob-new" {
		t.Error("the edits must stay after a failed save")
	}
	if r.s.busy {
		t.Error("saving again must be possible")
	}
	r.click(t, "Save")
	if len(r.rec.saves) != 2 {
		t.Error("Save stays available")
	}
}

func TestSavedButConfigurationNotRegenerated(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: workID})
	r.typeInto(t, "Username", "bob-new")
	r.click(t, "Save")
	r.saveSucceeds(t, &store.RegenerateError{Err: errors.New("permission denied")})
	r.want(t, "Saved, but the generated configuration could not be rewritten: permission denied.")
	r.wantNot(t, "Written")
	if f := r.form(); f.dirty() || f.cur.username != "bob-new" {
		t.Error("the identity is saved, the form is clean")
	}
}

func TestInvalidFormIsNotSaved(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: workID})
	r.click(t, "Global")
	r.click(t, "Save")
	if len(r.rec.saves) != 0 {
		t.Fatal("an invalid identity must not be sent to the store")
	}
	r.want(t, "1 problem to fix first", "There is already a global identity. Change personal to limited first.")
	if len(r.s.problems) == 0 || r.s.problemsID != workID {
		t.Error("the problem belongs to the form")
	}
	// Fixing the form takes the messages away.
	r.click(t, "Limited to folders")
	r.wantNot(t, "problem to fix first", "There is already a global identity")
	r.click(t, "Save")
	if len(r.rec.saves) != 1 {
		t.Errorf("a valid form is saved: %d", len(r.rec.saves))
	}
}

func TestEveryKindOfProblemHasAMessage(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: clientID})
	f := r.form()
	f.cur.alias = "github-work"
	f.cur.hostName = ""
	f.cur.portText = "http"
	f.cur.label = "two\nlines"
	f.cur.color = "teal"
	f.cur.keyPath = `~/.ssh/"id"`
	r.click(t, "Save")
	r.tt.Frame()
	r.want(t, "6 problems to fix first", "Another identity (work) already uses this alias.", "Enter the host name.",
		"Use port 22 or 443 for this platform, or 1 to 65535 for a custom one.", "Use a single line here.",
		"Use one of the six colors or a hex value such as #3A7BD5.", "Double quotes are not allowed here.")
	f.cur.alias = "github.com"
	f.cur.hostName = "bad host!"
	r.click(t, "Save")
	r.want(t, "This is the real host name of a platform; choose another alias.", "The host name has characters it cannot have.")
	if len(r.rec.saves) != 0 {
		t.Error("nothing is saved")
	}
}

func TestProblemsInTheFileDoNotBlockOtherChanges(t *testing.T) {
	list := sample()
	list[1].Scope = account.Scope{Global: true}
	list[2].Scope = account.Scope{Global: true}
	r := open(t, "en", list, Request{ID: workID})
	r.typeInto(t, "Username", "bob-new")
	r.click(t, "Save")
	if len(r.rec.saves) != 1 {
		t.Fatalf("a hand-made second global identity must not stop an unrelated save: %q", r.tt.Texts())
	}
}

func TestSaveNewIdentity(t *testing.T) {
	r := open(t, "en", sample(), Request{New: true})
	id := r.s.Draft()
	r.s.SetPrefill(id, "Ada", "ada@example.com")
	r.typeInto(t, "Name", "side")
	// The list has a global identity already.
	r.click(t, "Limited to folders")
	r.click(t, "Save")
	if len(r.rec.saves) != 1 || !r.rec.saves[0].isNew || r.rec.saves[0].a.ID != id || r.rec.saves[0].a.Label != "side" {
		t.Fatalf("saves: %+v", r.rec.saves)
	}
	if a := r.rec.saves[0].a; a.GitUserName != "Ada" || a.GitUserEmail != "ada@example.com" || a.CreatedAt.IsZero() {
		t.Errorf("new identity %+v", a)
	}
	r.saveSucceeds(t, nil)
	if r.s.Draft() != "" || r.s.Selected() != id || r.form().isNew {
		t.Fatalf("the draft is now an identity: draft %q selected %q", r.s.Draft(), r.s.Selected())
	}
	r.want(t, "Written", "Delete", "Times", "side")
	r.wantNot(t, "New identity, not saved yet")
}

func TestPickers(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: workID})
	r.click(t, "Choose File…")
	if !slices.Equal(r.rec.keys, []string{workID}) {
		t.Fatalf("key picker: %v", r.rec.keys)
	}
	r.s.SetKeyPath(workID, "/home/ada/.ssh/id_other")
	if r.form().cur.keyPath != "~/.ssh/id_other" {
		t.Errorf("chosen key %q", r.form().cur.keyPath)
	}
	r.s.SetKeyPath(workID, "")
	if r.form().cur.keyPath != "~/.ssh/id_other" {
		t.Error("an empty answer changes nothing")
	}
	r.tt.Frame()
	buttons := 0
	for _, name := range r.tt.Texts() {
		if name == "Choose folder" {
			buttons++
		}
	}
	if err := r.tt.Click("Choose folder"); err != nil {
		t.Fatal(err)
	}
	if len(r.rec.folders) != 1 || r.rec.folders[0] != (folderCall{workID, 0}) {
		t.Fatalf("folder picker: %v (%d buttons)", r.rec.folders, buttons)
	}
	r.s.SetFolder(workID, 1, "/home/ada/projects")
	r.s.SetFolder(workID, 9, "/home/ada/ignored")
	if got := r.form().cur.paths; !slices.Equal(got, []string{"~/work/", "~/projects/", "~/clients/acme/"}) {
		t.Errorf("chosen folder: %v", got)
	}
	r.s.SetFolder("nope", 0, "/x")
}

func TestSelectionFollowsTheStoreAfterDeletionOutside(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: clientID})
	r.s.SetAccounts(sample()[:2], nil, "/home/ada/.config/idset/accounts.json")
	r.tt.Frame()
	if r.s.Selected() != workID {
		t.Errorf("selected %q", r.s.Selected())
	}
	// Edits survive a reload, and a clean form follows the new data.
	r.typeInto(t, "Username", "bob-new")
	list := sample()[:2]
	list[1].GitUserName = "Robert"
	r.s.SetAccounts(list, nil, "/home/ada/.config/idset/accounts.json")
	if r.s.forms[workID].cur.username != "bob-new" {
		t.Error("edits in progress stay")
	}
	r.s.selectID(personalID)
	r.s.SetAccounts(list, nil, "/home/ada/.config/idset/accounts.json")
	if r.s.selectedForm().cur.gitName != "Robert" {
		t.Error("a form without edits follows the file")
	}
}

func TestChinese(t *testing.T) {
	r := open(t, "zh-Hans", sample(), Request{ID: workID})
	r.want(t, "添加身份", "身份", "限定", "署名", "邮箱", "别名", "私钥", "端口", "范围", "地址改写", "备注", "时间", "删除", "还原", "保存", "适用于")
	r.click(t, "删除")
	r.want(t, "删除身份 work？", "同时删除私钥", "磁盘上的项目文件夹不会被改动。")
}

func TestDarkScheme(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: workID})
	r.tt.SetDark(true)
	r.want(t, "work", "Scoped")
}

func TestKeyReasonsTextsExist(t *testing.T) {
	s := New(Text("en"), env, (&recorder{}).actions())
	for reason, want := range map[keyfile.Reason]string{
		keyfile.NoPath: "This identity has no key path.", keyfile.Shared: "Another identity (x) uses the same key.",
		keyfile.Symlink: "The key path is a symbolic link; Idset will not delete it.", keyfile.Missing: "The key file does not exist.",
		keyfile.NotFile: "The key path is not a regular file.", keyfile.None: "",
	} {
		if got := s.keyReason(&confirm{reason: reason, other: "x"}); got != want {
			t.Errorf("%d: %q", reason, got)
		}
	}
}

func TestAddIdentityIsAtTheBottomOfTheSidebar(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: workID})
	r.tt.SetSize(760, 520)
	r.tt.Frame()
	add, ok := r.tt.Find("Add Identity")
	if !ok {
		t.Fatalf("no add button in %q", r.tt.Texts())
	}
	if add.X > 232 || add.Y < 520-60 {
		t.Errorf("the add button sits at the bottom of the source list (232 wide): %v", add)
	}
	r.click(t, "Add Identity")
	if r.s.Draft() == "" || r.s.Selected() != r.s.Draft() {
		t.Fatal("the bar adds a draft")
	}
	// A long list scrolls above the bar, which does not move.
	var many []account.Account
	for i := range 20 {
		a := sample()[0]
		a.ID = fmt.Sprintf("00000000-0000-0000-0000-%012d", i)
		a.Label, a.SSHAlias = fmt.Sprintf("many-%d", i), fmt.Sprintf("alias-%d", i)
		many = append(many, a)
	}
	r = open(t, "en", many, Request{})
	r.tt.SetSize(760, 520)
	r.tt.Frame()
	before, _ := r.tt.Find("Add Identity")
	r.tt.Scroll(100, 200, 0, 3000)
	after, _ := r.tt.Find("Add Identity")
	if before != after || before.Y < 520-60 {
		t.Errorf("the add bar is fixed: %v %v", before, after)
	}
}

func TestDetailScrollsUnderAFixedFooterAtTheDefaultSize(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: workID})
	r.tt.SetSize(760, 520)
	r.tt.Frame()
	notes, ok := r.tt.Find("Notes")
	if !ok || notes.Y < 520 {
		t.Fatalf("the Notes group starts below the fold at 760x520: %v %v", notes, ok)
	}
	revert, _ := r.tt.Find("Revert")
	delBefore, _ := r.tt.Find("Delete")
	if revert.Y+revert.H > 520 || delBefore.Y+delBefore.H > 520 {
		t.Fatalf("the footer is inside the window: %v %v", revert, delBefore)
	}
	header, _ := r.tt.Find("Identity")
	for i := 0; i < 3; i++ {
		r.tt.Scroll(500, 250, 0, 2000)
	}
	notes, _ = r.tt.Find("Notes")
	if notes.Y > 520-40 {
		t.Errorf("scrolling brings the last groups into view: %v", notes)
	}
	if got, _ := r.tt.Find("Revert"); got != revert {
		t.Errorf("the footer stays: %v %v", got, revert)
	}
	// The header is the first thing in the scrolling content, so it scrolls away with it, as in the mockup.
	if got, ok := r.tt.Find("Identity"); ok && got.Y >= header.Y {
		t.Errorf("the header scrolls with the content: %v %v", got, header)
	}
}

func TestLongKeyPathIsShortenedInTheMiddleUntilEdited(t *testing.T) {
	long := "~/.ssh/keys/projects/example/id_ed25519_personal_laptop"
	if got := middle(long, keyShown); len([]rune(got)) != keyShown || !strings.HasSuffix(got, "personal_laptop") || !strings.Contains(got, "…") {
		t.Fatalf("middle: %q", got)
	}
	if middle("~/.ssh/id_work", keyShown) != "~/.ssh/id_work" {
		t.Error("a short path is left alone")
	}
	list := sample()
	list[0].KeyPath = long
	r := open(t, "en", list, Request{ID: workID})
	// The tooltip carries the whole path.
	r.want(t, middle(long, keyShown))
	r.click(t, middle(long, keyShown))
	r.tt.Frame()
	r.wantNot(t, middle(long, keyShown))
	if !r.form().keyEditing {
		t.Fatal("clicking the shortened path edits it")
	}
	// Typing keeps the input, even while the text is short again.
	r.tt.Key(ui.Cmd, ui.KeyA)
	r.tt.Type("~/.ssh/id_short")
	r.tt.Frame()
	if r.form().cur.keyPath != "~/.ssh/id_short" || !r.form().keyEditing {
		t.Errorf("typed %q editing %v", r.form().cur.keyPath, r.form().keyEditing)
	}
}

func TestPrefillHintSitsAtTheEndOfAStandardRow(t *testing.T) {
	for _, lang := range []string{"en", "zh-Hans"} {
		r := open(t, lang, sample(), Request{New: true})
		x := r.s.text
		r.tt.SetSize(760, 1500)
		r.tt.Frame()
		// Row heights depend on the platform's fonts, so compare with the same rows before the prefill.
		user, _ := r.tt.Find(x.Username)
		name, _ := r.tt.Find(x.GitName)
		email, _ := r.tt.Find(x.Email)
		plainName, plainEmail := name.Y-user.Y, email.Y-name.Y
		r.s.SetPrefill(r.s.Draft(), "Ada", "ada@example.com")
		r.tt.Frame()
		user, _ = r.tt.Find(x.Username)
		name, _ = r.tt.Find(x.GitName)
		email, _ = r.tt.Find(x.Email)
		if a, b := name.Y-user.Y, email.Y-name.Y; math.Abs(float64(a-plainName)) > 0.5 || math.Abs(float64(b-plainEmail)) > 0.5 {
			t.Errorf("%s: rows with a hint are as high as without it: %v %v, want %v %v", lang, a, b, plainName, plainEmail)
		}
		hint, ok := r.tt.Find(x.FromGit)
		if !ok || hint.X < 500 || hint.Y < name.Y-8 || hint.Y > name.Y+16 {
			t.Errorf("%s: the hint is at the end of the Git name row: %v %v", lang, hint, name)
		}
		// A long value gives way to the hint instead of overlapping it.
		r.form().cur.gitName = strings.Repeat("Ada Lovelace ", 20)
		r.form().prefName = r.form().cur.gitName
		r.tt.Frame()
		long, _ := r.tt.Find(x.FromGit)
		if long.X < 500 || long.W != hint.W {
			t.Errorf("%s: the hint keeps its place and size: %v %v", lang, long, hint)
		}
		// Editing takes the hint away, from that field only.
		r.form().cur.gitName = "Ada L"
		r.tt.Frame()
		if n := strings.Count(strings.Join(r.tt.Texts(), "|"), x.FromGit); n != 1 {
			t.Errorf("%s: one hint left, for the email: %d", lang, n)
		}
		r.form().cur.email = "other@example.com"
		r.tt.Frame()
		r.wantNot(t, x.FromGit)
	}
}

func TestBottomBarsLineUpAndFooterButtonsNeverResize(t *testing.T) {
	for _, lang := range []string{"en", "zh-Hans"} {
		r := open(t, lang, sample(), Request{ID: workID})
		r.tt.SetSize(760, 520)
		x := r.s.text
		r.tt.Frame()
		add, _ := r.tt.Find(x.AddIdentity)
		del, _ := r.tt.Find(x.Delete)
		// The add bar's button is 24 high, centered in the bar; the footer's Delete text in its own bar.
		if got, want := add.Y+12, del.Y+del.H/2; got < want-1 || got > want+1 {
			t.Errorf("%s: the bars' centers differ: %v vs %v", lang, got, want)
		}
		r.typeInto(t, x.Username, "bob-new")
		revert, _ := r.tt.Find(x.Revert)
		if got, _ := r.tt.Find(x.Delete); got != del {
			t.Errorf("%s: Delete moved: %v %v", lang, got, del)
		}
		r.click(t, x.Save)
		r.tt.Frame()
		for _, name := range []string{"saving", "written"} {
			want := x.Saving
			if name == "written" {
				r.saveSucceeds(t, nil)
				want = x.Saved
			}
			if !r.tt.HasText(want) {
				t.Fatalf("%s %s: missing %q in %q", lang, name, want, r.tt.Texts())
			}
			if got, _ := r.tt.Find(x.Revert); got != revert {
				t.Errorf("%s %s: Revert moved: %v %v", lang, name, got, revert)
			}
			if got, _ := r.tt.Find(x.Delete); got != del {
				t.Errorf("%s %s: Delete moved: %v %v", lang, name, got, del)
			}
		}
	}
}
