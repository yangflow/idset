package settings

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/apply"
	"github.com/yangflow/idset/internal/paths"
)

var env = paths.Env{Home: "/home/ada"}

var base = time.Date(2026, 10, 12, 9, 7, 0, 0, time.UTC)

type recorder struct {
	languages      []string
	logins         []bool
	loadedLogin    int
	loadedBackups  int
	restores       []string
	deletes        []string
	loadedIncludes int
	removes        int
	generated      []KeyRequest
	loads          []string
	uses           [][3]string
	drafts         []KeyDraft
	reveals        []string
	scans          int
	imports        [][]account.Account
	opened         []string
}

func (r *recorder) actions() Actions {
	return Actions{
		Language:       func(choice string) { r.languages = append(r.languages, choice) },
		SetLogin:       func(on bool) { r.logins = append(r.logins, on) },
		LoadLogin:      func() { r.loadedLogin++ },
		LoadBackups:    func() { r.loadedBackups++ },
		Restore:        func(path string) { r.restores = append(r.restores, path) },
		DeleteBackup:   func(path string) { r.deletes = append(r.deletes, path) },
		LoadIncludes:   func() { r.loadedIncludes++ },
		RemoveIncludes: func() { r.removes++ },
		Generate:       func(req KeyRequest) { r.generated = append(r.generated, req) },
		LoadKey:        func(keyPath string) { r.loads = append(r.loads, keyPath) },
		UseKey:         func(id, keyPath, fp string) { r.uses = append(r.uses, [3]string{id, keyPath, fp}) },
		NewIdentity:    func(d KeyDraft) { r.drafts = append(r.drafts, d) },
		Reveal:         func(keyPath string) { r.reveals = append(r.reveals, keyPath) },
		Scan:           func() { r.scans++ },
		Import:         func(list []account.Account) { r.imports = append(r.imports, list) },
		OpenAccounts:   func(id string) { r.opened = append(r.opened, id) },
	}
}

func identity(label, color string, global bool) account.Account {
	a := account.Account{ID: label + "-id", Label: label, Color: color, GitUserEmail: label + "@example.com"}
	if global {
		a.Scope.Global = true
	} else {
		a.Scope.Paths = []string{"~/work/"}
	}
	return a
}

// sampleBackups are the backups of the mockup: 4, 3, 2 and 1 identities, the third one unreadable.
func sampleBackups() []Backup {
	four := []account.Account{identity("work", "blue", false), identity("personal", "green", true),
		identity("client", "orange", false), identity("", "purple", false)}
	return []Backup{
		{Path: "/b/4", Time: base, Size: 2458, Accounts: four},
		{Path: "/b/3", Time: base.Add(-26 * time.Hour), Size: 1946, Accounts: four[:3]},
		{Path: "/b/2", Time: base.Add(-50 * time.Hour), Size: 205, Err: errors.New("invalid character")},
		{Path: "/b/1", Time: base.Add(-80 * time.Hour), Size: 717, Accounts: four[:1]},
	}
}

type rig struct {
	s   *Screen
	tt  *ui.Tester
	rec *recorder
}

func open(t *testing.T, lang string, req Request) *rig {
	t.Helper()
	rec := &recorder{}
	s := New(Text(lang), env, rec.actions())
	s.zone = time.UTC
	s.now = func() time.Time { return base }
	s.SetSystem("en")
	s.Open(req)
	r := &rig{s: s, tt: ui.NewTester(s.View, 720, 1400), rec: rec}
	r.tt.Frame()
	return r
}

func (r *rig) frame() { r.tt.Frame() }

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
	r.frame()
}

// clickRow clicks the first button labeled label at the height of the row showing the time stamp.
func (r *rig) clickRow(t *testing.T, label, stamp string) {
	t.Helper()
	button, ok := r.tt.Find(label)
	row, ok2 := r.tt.Find(stamp)
	if !ok || !ok2 {
		t.Fatalf("no %q or %q in %q", label, stamp, r.tt.Texts())
	}
	r.tt.ClickAt(button.X+button.W/2, row.Y+row.H)
	r.frame()
}

// clickDialogPrimary clicks the button right of Cancel in the open dialog; the two share its width, so it is one button width and a gap from Cancel's center.
func (r *rig) clickDialogPrimary(t *testing.T) {
	t.Helper()
	box, ok := r.tt.Find("Cancel")
	if !ok {
		t.Fatal("no dialog")
	}
	r.tt.ClickAt(box.X+box.W/2+150, box.Y+box.H/2)
	r.frame()
}

// loaded opens the Backups pane with the sample backups.
func loaded(t *testing.T, lang string) *rig {
	t.Helper()
	r := open(t, lang, Request{Pane: Backups})
	r.s.SetBackups(sampleBackups(), nil)
	r.frame()
	return r
}

func TestTextFollowsLocale(t *testing.T) {
	for loc, want := range map[string]string{"": "General", "en_US": "General", "zh_CN": "通用", "zh-Hans": "通用", "zh_TW": "通用", "fr": "General"} {
		if got := Text(loc).General; got != want {
			t.Errorf("Text(%q).General = %q, want %q", loc, got, want)
		}
	}
}

func TestCatalogsHaveEveryKey(t *testing.T) {
	for lang, c := range translations {
		v := reflect.ValueOf(c)
		for i := range v.NumField() {
			name := v.Type().Field(i).Name
			if strings.HasSuffix(name, "Pre") || strings.HasSuffix(name, "Post") {
				continue
			}
			if v.Field(i).String() == "" {
				t.Errorf("%s: %s is empty", lang, v.Type().Field(i).Name)
			}
		}
	}
	if len(translations) != 2 {
		t.Errorf("languages = %d, want en and zh-Hans only", len(translations))
	}
}

func TestTabsSwitchByClickAndArrowKeys(t *testing.T) {
	r := open(t, "en", Request{})
	if r.s.Tab() != General || r.rec.loadedLogin != 1 {
		t.Fatalf("tab %v, login loads %d", r.s.Tab(), r.rec.loadedLogin)
	}
	r.want(t, "Language", "Startup")
	r.click(t, "Backups")
	if r.s.Tab() != Backups || r.rec.loadedBackups != 1 {
		t.Fatalf("tab %v, backup loads %d", r.s.Tab(), r.rec.loadedBackups)
	}
	r.want(t, "Reading backups…")
	r.tt.Key(0, ui.KeyRight)
	r.frame()
	if r.s.Tab() != Config || r.rec.loadedIncludes != 1 {
		t.Fatalf("tab %v after Right, include loads %d", r.s.Tab(), r.rec.loadedIncludes)
	}
	r.want(t, "Include", "~/.ssh/config", "~/.gitconfig")
	r.tt.Key(0, ui.KeyLeft)
	r.tt.Key(0, ui.KeyLeft)
	r.frame()
	if r.s.Tab() != Import || r.rec.scans != 1 {
		t.Fatalf("tab %v after Left twice, scans %d", r.s.Tab(), r.rec.scans)
	}
	r.tt.Key(0, ui.KeyLeft)
	r.tt.Key(0, ui.KeyLeft)
	r.frame()
	if r.s.Tab() != General {
		t.Fatalf("tab %v after Left four times", r.s.Tab())
	}
	r.tt.Key(0, ui.KeyEnd)
	r.frame()
	if r.s.Tab() != Config {
		t.Fatalf("tab %v after End", r.s.Tab())
	}
}

func TestOpenChoosesThePane(t *testing.T) {
	r := open(t, "en", Request{Pane: Backups})
	if r.s.Tab() != Backups {
		t.Fatalf("tab %v", r.s.Tab())
	}
	r.s.Open(Request{Pane: Last})
	if r.s.Tab() != Backups {
		t.Fatalf("Last moved the tab to %v", r.s.Tab())
	}
	r.s.Open(Request{Pane: General})
	if r.s.Tab() != General {
		t.Fatalf("tab %v", r.s.Tab())
	}
	before := r.rec.loadedLogin
	r.s.Open(Request{Pane: Last})
	r.frame()
	if r.rec.loadedLogin != before+1 {
		t.Errorf("opening again did not read the login item again: %d", r.rec.loadedLogin)
	}
}

func TestLanguageChoiceAppliesAndRelabels(t *testing.T) {
	r := open(t, "en", Request{})
	r.want(t, "Follow System", "Current: English", "Following the system now: English.")
	r.click(t, "简体中文")
	if !slices.Equal(r.rec.languages, []string{Chinese}) {
		t.Fatalf("languages = %q", r.rec.languages)
	}
	r.want(t, "Applies immediately.")
	r.wantNot(t, "Current: English", "Following the system now")
	r.s.SetText(Text(Chinese))
	r.frame()
	r.want(t, "通用", "界面语言", "跟随系统", "登录时启动")
	r.wantNot(t, "Language", "Follow System")
	r.click(t, "English")
	r.click(t, "跟随系统")
	if !slices.Equal(r.rec.languages, []string{Chinese, English, FollowSystem}) {
		t.Fatalf("languages = %q", r.rec.languages)
	}
	r.s.SetSystem("zh_CN")
	r.frame()
	r.want(t, "当前：简体中文", "现在跟随系统：简体中文。")
}

// The chosen language row is marked by a check, not an accent fill, and is exposed to assistive technology as a radio.
func TestChosenLanguageRowHasNoFill(t *testing.T) {
	r := open(t, "en", Request{})
	r.click(t, "English")
	row, ok := r.tt.Find("English")
	if !ok {
		t.Fatal("no English row")
	}
	img := r.tt.Image()
	scale := float32(img.Bounds().Dx()) / 720.0
	c := img.RGBAAt(int((row.X+row.W+200)*scale), int((row.Y+row.H/2)*scale))
	if c.B > 200 && c.R < 100 {
		t.Errorf("chosen row filled with %v", c)
	}
}

func TestChoosingTheChosenLanguageDoesNothing(t *testing.T) {
	r := open(t, "en", Request{})
	r.s.SetLanguage(English)
	r.frame()
	r.click(t, "English")
	if len(r.rec.languages) != 0 {
		t.Errorf("languages = %q", r.rec.languages)
	}
}

// toggleLogin clicks the switch, which sits at the end of the Open at login row.
func (r *rig) toggleLogin(t *testing.T) {
	t.Helper()
	box, ok := r.tt.Find("Open at login")
	if !ok {
		t.Fatal("no login row")
	}
	r.tt.ClickAt(box.X+box.W+14, box.Y+box.H/2)
	r.frame()
}

func TestLoginSwitch(t *testing.T) {
	r := open(t, "en", Request{})
	r.s.SetLoginState(false)
	r.toggleLogin(t)
	if !slices.Equal(r.rec.logins, []bool{true}) {
		t.Fatalf("logins = %v", r.rec.logins)
	}
	r.toggleLogin(t)
	if len(r.rec.logins) != 1 {
		t.Errorf("the switch changed again while the first call was pending: %v", r.rec.logins)
	}
	r.s.LoginDone(true, nil)
	r.frame()
	if !r.s.login || r.s.loginNote != loginPlain {
		t.Errorf("login %v, note %v", r.s.login, r.s.loginNote)
	}
	r.toggleLogin(t)
	if !slices.Equal(r.rec.logins, []bool{true, false}) {
		t.Fatalf("logins = %v", r.rec.logins)
	}
	r.s.LoginDone(false, nil)
	if r.s.login {
		t.Error("the switch stayed on")
	}
}

func TestLoginNeedsApproval(t *testing.T) {
	r := open(t, "en", Request{})
	r.toggleLogin(t)
	r.s.LoginDone(true, errors.New("Idset is not allowed: approve it in Login Items"))
	r.frame()
	if r.s.login {
		t.Error("the switch stays off until the user approves")
	}
	r.want(t, "Needs approval: turn on Idset")
	r.wantNot(t, "Could not change the login item")
}

func TestLoginFailure(t *testing.T) {
	r := open(t, "en", Request{})
	r.toggleLogin(t)
	r.s.LoginDone(true, &wrapErr{errors.New("boom")})
	r.frame()
	if r.s.login {
		t.Error("the switch stayed on after a failure")
	}
	r.want(t, "Could not change the login item: boom")
	r.wantNot(t, "Needs approval: turn on Idset")
	r.s.SetLoginState(true)
	r.toggleLogin(t)
	r.s.LoginDone(false, errors.New("unregister: boom"))
	if !r.s.login {
		t.Error("turning it off failed, the switch must go back on")
	}
}

func TestLoginStateIsKeptWhileChanging(t *testing.T) {
	r := open(t, "en", Request{})
	r.toggleLogin(t)
	r.s.SetLoginState(false)
	if !r.s.login {
		t.Error("an old reading overwrote the switch being changed")
	}
}

func TestBackupListShowsRowsNewestFirst(t *testing.T) {
	r := loaded(t, "en")
	r.want(t, "2026-10-12 09:07", "2026-10-11 07:07", "4 identities · work, personal, client +1 · 2.4 KB",
		"3 identities · work, personal, client · 1.9 KB", "1 identity · work · 0.7 KB", "Unreadable backup · 0.2 KB",
		"Backed up before adding an identity")
	texts := r.tt.Texts()
	if slices.Index(texts, "2026-10-12 09:07") > slices.Index(texts, "2026-10-11 07:07") {
		t.Errorf("not newest first: %q", texts)
	}
	r.wantNot(t, "No backups yet", "work (unnamed)")
}

func TestBackupRowExpands(t *testing.T) {
	r := loaded(t, "en")
	r.wantNot(t, "Global")
	r.click(t, "Show Identities")
	r.want(t, "personal", "personal@example.com", "Global", "(unnamed)", "Hide Identities")
	r.click(t, "Hide Identities")
	r.wantNot(t, "Global", "Hide Identities")
}

func TestExpandedRowsFollowDeletedBackups(t *testing.T) {
	r := loaded(t, "en")
	r.click(t, "Show Identities")
	r.s.SetBackups(sampleBackups()[1:], nil)
	if len(r.s.expanded) != 0 {
		t.Errorf("expanded = %v", r.s.expanded)
	}
}

func TestUnreadableBackupCannotBeRestoredOrExpanded(t *testing.T) {
	r := loaded(t, "en")
	r.clickRow(t, "Restore", "2026-10-10 07:07")
	r.clickRow(t, "Show Identities", "2026-10-10 07:07")
	if r.s.confirmOpen || len(r.s.expanded) != 0 {
		t.Fatalf("confirm %v, expanded %v", r.s.confirmOpen, r.s.expanded)
	}
	r.wantNot(t, "Restore backup")
	r.s.askRestore("/b/2")
	if r.s.confirmOpen {
		t.Error("the unreadable backup asked for a restore")
	}
	r.clickRow(t, "Delete", "2026-10-10 07:07")
	if !slices.Equal(r.rec.deletes, []string{"/b/2"}) {
		t.Errorf("an unreadable backup can still be deleted: %v", r.rec.deletes)
	}
}

func TestRestoreAsksFirstAndCancelLeavesEverything(t *testing.T) {
	r := loaded(t, "en")
	r.clickRow(t, "Restore", "2026-10-11 07:07")
	r.want(t, "Restore backup 2026-10-11 07:07?", "Identities not in this backup are removed.")
	if !r.tt.Focused("Cancel") {
		t.Error("Cancel does not have the initial focus")
	}
	if len(r.rec.restores) != 0 {
		t.Fatal("restored without asking")
	}
	r.click(t, "Cancel")
	r.wantNot(t, "Restore backup")
	if len(r.rec.restores) != 0 || r.s.restoring != "" {
		t.Errorf("Cancel restored: %v", r.rec.restores)
	}
}

func TestRestoreDialogClosesWithEscape(t *testing.T) {
	r := loaded(t, "en")
	r.clickRow(t, "Restore", "2026-10-11 07:07")
	r.want(t, "Restore backup")
	r.tt.Key(0, ui.KeyEscape)
	r.frame()
	r.wantNot(t, "Restore backup")
	if len(r.rec.restores) != 0 {
		t.Errorf("Escape restored: %v", r.rec.restores)
	}
}

func TestRestoreListsTheIdentitiesAfterwards(t *testing.T) {
	r := loaded(t, "en")
	r.clickRow(t, "Restore", "2026-10-11 07:07")
	r.want(t, "work", "personal", "client")
	r.wantNot(t, "(unnamed)")
}

func TestOtherRowsCannotExpandWhileRestoring(t *testing.T) {
	r := loaded(t, "en")
	r.clickRow(t, "Restore", "2026-10-11 07:07")
	r.clickDialogPrimary(t)
	r.clickRow(t, "Show Identities", "2026-10-12 09:07")
	if len(r.s.expanded) != 0 {
		t.Errorf("expanded = %v", r.s.expanded)
	}
}

func TestRestoreRunsAndShowsProgress(t *testing.T) {
	r := loaded(t, "en")
	r.clickRow(t, "Restore", "2026-10-11 07:07")
	r.clickDialogPrimary(t)
	if !slices.Equal(r.rec.restores, []string{"/b/3"}) {
		t.Fatalf("restores = %v", r.rec.restores)
	}
	r.want(t, "Restoring…")
	r.clickRow(t, "Delete", "2026-10-12 09:07")
	r.clickRow(t, "Restore", "2026-10-12 09:07")
	if len(r.rec.deletes) != 0 || len(r.rec.restores) != 1 || r.s.confirmOpen {
		t.Errorf("deletes %v, restores %v, confirm %v: other actions must wait", r.rec.deletes, r.rec.restores, r.s.confirmOpen)
	}
}

func TestRestoreSucceedsAndTheFeedbackFades(t *testing.T) {
	r := loaded(t, "en")
	r.clickRow(t, "Restore", "2026-10-11 07:07")
	r.clickDialogPrimary(t)
	r.s.RestoreDone("/b/3", nil)
	r.frame()
	r.want(t, "Restored")
	r.wantNot(t, "Restoring…")
	if got := r.tt.Announcements(); !slices.Contains(got, "Restored") {
		t.Errorf("announcements = %q", got)
	}
	r.s.now = func() time.Time { return base.Add(flashFor + time.Millisecond) }
	r.frame()
	r.wantNot(t, "Restored")
}

func TestRestoreFailureIsInline(t *testing.T) {
	r := loaded(t, "en")
	r.clickRow(t, "Restore", "2026-10-11 07:07")
	r.clickDialogPrimary(t)
	r.s.RestoreDone("/b/3", &wrapErr{errors.New("no space left")})
	r.frame()
	r.want(t, "Could not restore: no space left")
	r.wantNot(t, "Restoring…", "Restored")
	r.clickRow(t, "Restore", "2026-10-11 07:07")
	r.clickDialogPrimary(t)
	r.wantNot(t, "Could not restore")
}

func TestDeleteNeedsNoConfirmation(t *testing.T) {
	r := loaded(t, "en")
	r.clickRow(t, "Delete", "2026-10-11 07:07")
	if !slices.Equal(r.rec.deletes, []string{"/b/3"}) {
		t.Fatalf("deletes = %v", r.rec.deletes)
	}
	r.wantNot(t, "Delete backup")
	r.s.DeleteDone("/b/3", nil)
	r.frame()
	r.wantNot(t, "2026-10-11 07:07")
	r.want(t, "2026-10-12 09:07", "2026-10-09 01:07")
}

func TestDeleteFailureIsInline(t *testing.T) {
	r := loaded(t, "en")
	r.clickRow(t, "Delete", "2026-10-11 07:07")
	r.s.DeleteDone("/b/3", &wrapErr{errors.New("permission denied")})
	r.frame()
	r.want(t, "Could not delete: permission denied", "2026-10-11 07:07")
}

func TestEmptyList(t *testing.T) {
	r := open(t, "en", Request{Pane: Backups})
	r.s.SetBackups(nil, nil)
	r.frame()
	r.want(t, "No backups yet", "Backups are made before adding an identity")
	r.wantNot(t, "Idset backs up accounts.json before")
}

func TestListFailure(t *testing.T) {
	r := open(t, "en", Request{Pane: Backups})
	r.s.SetBackups(nil, &wrapErr{errors.New("denied")})
	r.frame()
	r.want(t, "Could not read the backups: denied")
	r.wantNot(t, "No backups yet")
}

func TestReadOnlyBanner(t *testing.T) {
	r := loaded(t, "en")
	r.wantNot(t, "Restore a backup to carry on.")
	r.s.SetReadOnly(true)
	r.frame()
	r.want(t, "accounts.json cannot be read; Idset is read-only. Restore a backup.")
	r.s.SetReadOnly(false)
	r.frame()
	r.wantNot(t, "Restore a backup to carry on.")
}

func TestConfigurationShowsTheBlocksAndRemovesThem(t *testing.T) {
	r := open(t, "en", Request{Pane: Config})
	r.want(t, "Remove Include…")
	r.wantNot(t, "Include block present", "No Include block")
	r.s.SetIncludes(apply.IncludeState{SSH: apply.FileState{Present: true}, Git: apply.FileState{Present: true}})
	r.frame()
	if got := countText(r.tt, "Include block present"); got != 2 {
		t.Fatalf("present rows = %d in %q", got, r.tt.Texts())
	}
	r.click(t, "Remove Include…")
	r.want(t, "Remove Include?", "Will be removed", "Left as is", "The Idset marker block in", "~/.ssh/config",
		"~/.gitconfig", "accounts.json", "and your identities", "Managed files in", "~/.config/idset/")
	if !r.tt.Focused("Cancel") {
		t.Error("Cancel does not have the initial focus")
	}
	r.click(t, "Cancel")
	if r.rec.removes != 0 {
		t.Fatal("removed after Cancel")
	}
	r.click(t, "Remove Include…")
	r.tt.Key(0, ui.KeyEscape)
	r.frame()
	if r.rec.removes != 0 {
		t.Fatal("removed after Escape")
	}
	r.click(t, "Remove Include…")
	r.clickDialogPrimary(t)
	if r.rec.removes != 1 {
		t.Fatalf("removes = %d", r.rec.removes)
	}
	r.s.RemoveDone(apply.IncludeState{}, apply.IncludeState{}, nil)
	r.frame()
	if got := countText(r.tt, "No Include block"); got != 2 {
		t.Fatalf("absent rows = %d in %q", got, r.tt.Texts())
	}
	r.want(t, "Removed", "Removed. They return when an identity changes.")
	if got := countText(r.tt, "Removed"); got != 1 {
		t.Errorf("Removed shown %d times in %q", got, r.tt.Texts())
	}
	r.click(t, "Removed")
	r.wantNot(t, "Remove Include?")
	if got := r.tt.Announcements(); !slices.Contains(got, "Removed") {
		t.Errorf("announcements = %q", got)
	}
}

func countText(tt *ui.Tester, text string) int {
	n := 0
	for _, s := range tt.Texts() {
		if s == text {
			n++
		}
	}
	return n
}

func TestRemoveFailureNamesTheFile(t *testing.T) {
	both := apply.IncludeState{SSH: apply.FileState{Present: true}, Git: apply.FileState{Present: true}}
	gitOnly := apply.IncludeState{Git: apply.FileState{Present: true}}
	cases := []struct {
		name          string
		before, after apply.IncludeState
		want          string
	}{
		{"first file", both, both, "Could not write ~/.ssh/config: permission denied."},
		{"second file", both, gitOnly, "Could not write ~/.gitconfig: permission denied. The block in ~/.ssh/config was already removed."},
		{"only the second had a block", gitOnly, gitOnly, "Could not write ~/.gitconfig: permission denied."},
	}
	for _, c := range cases {
		r := open(t, "en", Request{Pane: Config})
		r.s.SetIncludes(c.before)
		r.click(t, "Remove Include…")
		r.clickDialogPrimary(t)
		r.s.RemoveDone(c.before, c.after, &wrapErr{errors.New("permission denied")})
		r.frame()
		r.want(t, c.want)
		r.wantNot(t, "Removed. They return when an identity changes.")
	}
}

func TestRemovingBlocksAskingAgain(t *testing.T) {
	r := open(t, "en", Request{Pane: Config})
	r.s.askRemove()
	if r.s.removeOpen {
		t.Error("asked before the files were read")
	}
	r.s.SetIncludes(apply.IncludeState{})
	r.click(t, "Remove Include…")
	r.clickDialogPrimary(t)
	r.s.askRemove()
	if r.s.removeOpen {
		t.Error("asked again while removing")
	}
}

func TestEveryPaneInBothLanguagesAndSchemes(t *testing.T) {
	panes := []struct {
		pane  Pane
		setup func(s *Screen)
		en    []string
		zh    []string
	}{
		{General, nil, []string{"Language", "Open at login"}, []string{"界面语言", "登录时启动"}},
		{Backups, func(s *Screen) { s.SetBackups(sampleBackups(), nil) }, []string{"4 identities", "Unreadable backup"}, []string{"4 个身份", "无法读取的备份"}},
		{Backups, func(s *Screen) { s.SetBackups(nil, nil) }, []string{"No backups yet"}, []string{"还没有备份"}},
		{Import, func(s *Screen) { s.ScanDone(sampleCandidates(), nil) }, []string{"5 identities found", "Import 4 Identities"}, []string{"找到 5 个身份", "导入 4 个身份"}},
		{Import, func(s *Screen) { s.ScanDone(nil, nil) }, []string{"No identities found"}, []string{"没找到身份"}},
		{Config, func(s *Screen) { s.SetIncludes(apply.IncludeState{}) }, []string{"No Include block"}, []string{"没有 Include 标记块"}},
	}
	for _, dark := range []bool{false, true} {
		for _, p := range panes {
			for lang, want := range map[string][]string{"en": p.en, "zh-Hans": p.zh} {
				r := open(t, lang, Request{Pane: p.pane})
				if p.setup != nil {
					p.setup(r.s)
				}
				r.tt.SetDark(dark)
				r.frame()
				r.want(t, want...)
			}
		}
	}
}

func TestHelpers(t *testing.T) {
	s := New(Text("en"), env, (&recorder{}).actions())
	if got := size(0); got != "0.1 KB" {
		t.Errorf("size(0) = %q", got)
	}
	if got := size(2458); got != "2.4 KB" {
		t.Errorf("size = %q", got)
	}
	if s.identityCount(1) != "1 identity" || s.identityCount(0) != "0 identities" {
		t.Errorf("counts: %q %q", s.identityCount(1), s.identityCount(0))
	}
	four := sampleBackups()[0].Accounts
	if got := s.names(four); got != "work, personal, client +1" {
		t.Errorf("names = %q", got)
	}
	if got := s.names(four[:2]); got != "work, personal" {
		t.Errorf("names = %q", got)
	}
	if got := s.names(four[3:]); got != "(unnamed)" {
		t.Errorf("names = %q", got)
	}
	if got := cause(errors.New("plain")); got != "plain" {
		t.Errorf("cause = %q", got)
	}
	if got := cause(errors.Join(errors.New("a"))); got == "" {
		t.Error("cause of a joined error is empty")
	}
	wrapped := &wrapErr{errors.New("inner")}
	if got := cause(wrapped); got != "inner" {
		t.Errorf("cause = %q", got)
	}
	s.system = Chinese
	if s.systemName() != "简体中文" {
		t.Errorf("systemName = %q", s.systemName())
	}
}

type wrapErr struct{ err error }

func (e *wrapErr) Error() string { return "outer: " + e.err.Error() }
func (e *wrapErr) Unwrap() error { return e.err }
