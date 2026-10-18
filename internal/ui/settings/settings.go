// Package settings is the settings window: a row of tabs over four panes. General holds the interface language and
// the login item, Keys generates a new SSH key and puts it to use, Backups lists the snapshots of accounts.json to
// restore or delete, and Configuration removes the Include blocks from the user's own files. State changes on the UI thread only; everything that touches the disk,
// the system or a command runs in the owner's background work (see Actions and Backend) and reports back through
// the setters and the Done methods.
package settings

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/apply"
	"github.com/yangflow/idset/internal/keygen"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/ui/locale"
)

//go:embed copy.json
var catalog []byte

// Copy is the window's text in one language. Main reads the window title from it.
type Copy struct {
	Window, General, Backups, Config, Tabs,
	Language, FollowSystem, Current, LangNote, SystemNote, LangEnglish, LangChinese,
	Startup, Login, LoginNote, Approve, LoginFail,
	Restore, Restoring, Restored, Delete, Expand, Collapse, Cancel,
	OneIdentity, Identities, Unreadable, More, Global, Scoped, Unnamed,
	BackupNote, ReadOnlyBanner, RestoreFailed, DeleteFailed, ListFailed, Loading,
	EmptyTitle, EmptyBody, ConfirmTitle, ConfirmBody, ConfirmNote,
	Include, Present, Absent, Remove, Removed, IncludeNote, RemovedNote, RemoveFail, RemoveFailPartial,
	RemoveTitle, RemoveBody, Removes, Keeps, MarkerPre, MarkerPost, Kept1Post, Kept2Pre, Kept2Post, Kept3, RemoveButton,
	Keys, KNew, KType, KPlatform, KFile, KComment, Custom, KFormNote, Generate, Generating, KInvalid, KExists, KNoKeygen, KFail, KDiag,
	Created, PubKey, CopyPub, Copied, NoPub, KFiles, Priv, Pub, Fp, Pass, None, PassNote, RevealFinder, RevealExplorer, RevealFiles,
	KNext, AddOn, OpenSettings, AddCustom, AddCustomHint, Agent, Load, Loaded, AgentFail, LoadFail, UseFor, Choose, Change,
	NextNote, NextCustom, UsedNote, UseFail, Another, UseTitle, UseBody, NowKey, NoKey, NewID, NewIDSub, NewIDSubPlain, UseNote, UseButton string
}

var translations = func() map[string]Copy {
	var values map[string]Copy
	if err := json.Unmarshal(catalog, &values); err != nil {
		panic(err)
	}
	return values
}()

// Text returns the copy for a locale: a system locale such as "zh_TW", or a UI language such as "zh-Hans".
func Text(loc string) Copy { return translations[locale.Language(loc)] }

// Pane is a tab of the window.
type Pane int

// Last, in a Request, keeps the pane the window shows. The others are the tabs in order.
const (
	Last Pane = iota
	General
	Keys
	Backups
	Config
)

const paneCount = 4

// Request says what the window should show when it opens. For, with the Keys pane, is the ID of an identity whose
// private key is missing: the form is filled in for it, and the new key goes to it.
type Request struct {
	Pane Pane
	For  string
}

// Language choices as stored: "" follows the system.
const (
	FollowSystem = ""
	English      = "en"
	Chinese      = "zh-Hans"
)

// Backup is one snapshot of accounts.json as the window shows it. Err is why it cannot be read, nil otherwise;
// Accounts are the identities it holds.
type Backup struct {
	Path     string
	Time     time.Time
	Size     int64
	Err      error
	Accounts []account.Account
}

// Actions are what the window asks its owner to do. All of them return at once: the owner works in the
// background and reports back on the UI thread.
type Actions struct {
	// Language stores the choice and applies it to every open screen; the owner then calls SetText here too.
	Language func(choice string)
	// SetLogin turns the login item on or off and answers with LoginDone.
	SetLogin func(on bool)
	// LoadLogin reads the login item and answers with SetLogin's counterpart, SetLoginState.
	LoadLogin func()
	// LoadBackups lists the backups and answers with SetBackups.
	LoadBackups func()
	// Restore restores a backup and answers with RestoreDone.
	Restore func(path string)
	// DeleteBackup deletes a backup and answers with DeleteDone.
	DeleteBackup func(path string)
	// LoadIncludes looks for the Include blocks and answers with SetIncludes.
	LoadIncludes func()
	// RemoveIncludes removes the Include blocks and answers with RemoveDone.
	RemoveIncludes func()
	// Generate generates a key pair and answers with GenerateDone.
	Generate func(req KeyRequest)
	// LoadKey adds the key at a stored path to ssh-agent and answers with LoadDone.
	LoadKey func(keyPath string)
	// UseKey writes a key path and fingerprint to an identity, regenerates the configuration and answers with UseDone.
	UseKey func(id, keyPath, fingerprint string)
	// NewIdentity opens the accounts window on a new identity filled in with the key.
	NewIdentity func(draft KeyDraft)
	// Reveal shows a file, by its stored path, in the file manager.
	Reveal func(keyPath string)
}

// KeyRequest is the key the Keys form asks for.
type KeyRequest struct {
	Type    keygen.Type
	Name    string
	Comment string
}

// KeyDraft is what a new identity starts with when it is made for a generated key.
type KeyDraft struct {
	Provider    account.Provider
	Email       string
	KeyPath     string
	Fingerprint string
}

// flashFor is how long "Restored" shows on a row.
const flashFor = 1200 * time.Millisecond

type loginNote int

const (
	loginPlain loginNote = iota
	loginApprove
	loginFailed
)

type removal int

const (
	removalNone removal = iota
	removalDone
	removalFailed
)

// Screen holds the window's state.
type Screen struct {
	text   Copy
	env    paths.Env
	act    Actions
	now    func() time.Time
	zone   *time.Location
	system string

	tab   int
	shown int

	language string

	login      bool
	loginBusy  bool
	loginNote  loginNote
	loginError string

	readOnly  bool
	loaded    bool
	backups   []Backup
	listErr   string
	expanded  map[string]bool
	restoring string
	restored  string
	until     time.Time
	failed    map[string]string
	deleting  bool

	confirmOpen bool
	confirmPath string

	includes     apply.IncludeState
	includesRead bool
	removing     bool
	removal      removal
	removeErr    string
	removeOpen   bool

	accounts []account.Account
	goos     string
	keys     keys

	announce string
}

// New returns a window on the General pane.
func New(text Copy, env paths.Env, act Actions) *Screen {
	s := &Screen{text: text, env: env, act: act, now: time.Now, zone: time.Local, shown: -1, goos: runtime.GOOS,
		expanded: map[string]bool{}, failed: map[string]string{}}
	s.resetKeys()
	return s
}

// SetText changes the language of the window. The platform of the Keys form keeps its choice.
func (s *Screen) SetText(text Copy) {
	p := s.formProvider()
	s.text = text
	s.keys.platform = s.providerName(p)
}

// SetSystem tells the language the system asks for, a UI language such as "zh-Hans", for the "Current" hint.
func (s *Screen) SetSystem(language string) { s.system = locale.Language(language) }

// SetLanguage shows the stored language choice: FollowSystem, English or Chinese.
func (s *Screen) SetLanguage(choice string) { s.language = choice }

// Open shows the pane the request asks for.
func (s *Screen) Open(req Request) {
	s.shown = -1
	if req.Pane >= General && int(req.Pane) <= paneCount {
		s.tab = int(req.Pane) - 1
	}
	if req.Pane == Keys && req.For != "" {
		s.keysFor(req.For)
	}
}

// Tab is the pane being shown.
func (s *Screen) Tab() Pane { return Pane(s.tab + 1) }

// SetReadOnly says whether accounts.json is read-only, which the Backups pane tells in a banner; the Keys pane then
// writes no identity.
func (s *Screen) SetReadOnly(readOnly bool) { s.readOnly = readOnly }

// SetAccounts gives the identities the Keys pane fills its form from and offers the new key to.
func (s *Screen) SetAccounts(list []account.Account) {
	s.accounts = list
	if s.keys.fresh && s.keys.comment == "" && len(list) > 0 {
		s.keys.comment, s.keys.fresh = list[0].GitUserEmail, false
	}
}

// entered asks for what the pane about to be shown needs.
func (s *Screen) entered() {
	switch Pane(s.tab + 1) {
	case General:
		s.act.LoadLogin()
	case Backups:
		s.act.LoadBackups()
	case Config:
		s.act.LoadIncludes()
	}
}

// chooseLanguage applies a language choice at once; the owner stores it and re-labels every screen.
func (s *Screen) chooseLanguage(choice string) {
	if choice == s.language {
		return
	}
	s.language = choice
	s.act.Language(choice)
}

// SetLoginState shows whether the login item is on.
func (s *Screen) SetLoginState(on bool) {
	if !s.loginBusy {
		s.login = on
	}
}

func (s *Screen) toggleLogin() {
	if s.loginBusy {
		return
	}
	s.loginBusy, s.loginNote = true, loginPlain
	s.act.SetLogin(s.login)
}

// LoginDone reports SetLogin(want). An error that names Login Items means macOS wants the user's approval: the
// switch stays off and the note says where to give it. Any other error puts the switch back.
func (s *Screen) LoginDone(want bool, err error) {
	s.loginBusy = false
	switch {
	case err == nil:
		s.login, s.loginNote = want, loginPlain
	case want && strings.Contains(err.Error(), "Login Items"):
		s.login, s.loginNote = false, loginApprove
	default:
		s.login, s.loginNote, s.loginError = !want, loginFailed, cause(err)
	}
}

// SetBackups shows the listed backups, newest first, or why they could not be listed.
func (s *Screen) SetBackups(list []Backup, err error) {
	s.loaded = true
	s.backups = list
	s.listErr = ""
	if err != nil {
		s.listErr = fmt.Sprintf(s.text.ListFailed, cause(err))
	}
	for path := range s.expanded {
		if !slices.ContainsFunc(list, func(b Backup) bool { return b.Path == path }) {
			delete(s.expanded, path)
		}
	}
}

func (s *Screen) toggle(path string) {
	if s.expanded[path] {
		delete(s.expanded, path)
		return
	}
	s.expanded[path] = true
}

func (s *Screen) backup(path string) *Backup {
	if i := slices.IndexFunc(s.backups, func(b Backup) bool { return b.Path == path }); i >= 0 {
		return &s.backups[i]
	}
	return nil
}

// askRestore opens the confirmation for a backup.
func (s *Screen) askRestore(path string) {
	if s.restoring != "" || s.backup(path) == nil || s.backup(path).Err != nil {
		return
	}
	s.confirmPath, s.confirmOpen = path, true
}

// confirmRestore starts the restoring the dialog asks for.
func (s *Screen) confirmRestore() {
	path := s.confirmPath
	s.confirmOpen = false
	if s.restoring != "" || s.backup(path) == nil {
		return
	}
	s.restoring, s.restored = path, ""
	delete(s.failed, path)
	s.act.Restore(path)
}

// RestoreDone reports Restore.
func (s *Screen) RestoreDone(path string, err error) {
	s.restoring = ""
	if err != nil {
		s.failed[path] = fmt.Sprintf(s.text.RestoreFailed, cause(err))
		return
	}
	s.restored, s.until = path, s.now().Add(flashFor)
	s.announce = s.text.Restored
}

func (s *Screen) deleteBackup(path string) {
	if s.restoring != "" || s.deleting {
		return
	}
	s.deleting = true
	delete(s.failed, path)
	s.act.DeleteBackup(path)
}

// DeleteDone reports DeleteBackup.
func (s *Screen) DeleteDone(path string, err error) {
	s.deleting = false
	if err != nil {
		s.failed[path] = fmt.Sprintf(s.text.DeleteFailed, cause(err))
		return
	}
	s.backups = slices.DeleteFunc(s.backups, func(b Backup) bool { return b.Path == path })
	delete(s.expanded, path)
	delete(s.failed, path)
}

// SetIncludes shows where the Include block is.
func (s *Screen) SetIncludes(state apply.IncludeState) {
	s.includes, s.includesRead = state, true
}

func (s *Screen) askRemove() {
	if s.removing || !s.includesRead {
		return
	}
	s.removeOpen = true
}

func (s *Screen) confirmRemove() {
	s.removeOpen = false
	if s.removing {
		return
	}
	s.removing, s.removal, s.removeErr = true, removalNone, ""
	s.act.RemoveIncludes()
}

// RemoveDone reports RemoveIncludes: before and after are the state of the files around the attempt, err what
// went wrong. The files are written in order, ~/.ssh/config first, so a block that is gone from it while the
// other write failed was removed already.
func (s *Screen) RemoveDone(before, after apply.IncludeState, err error) {
	s.removing = false
	s.includes, s.includesRead = after, true
	if err == nil {
		s.removal = removalDone
		s.announce = s.text.Removed
		return
	}
	s.removal = removalFailed
	const ssh, git = "~/.ssh/config", "~/.gitconfig"
	if after.SSH.Present || after.SSH.Err != nil {
		s.removeErr = fmt.Sprintf(s.text.RemoveFail, ssh, cause(err))
		return
	}
	if before.SSH.Present {
		s.removeErr = fmt.Sprintf(s.text.RemoveFailPartial, git, cause(err), ssh)
		return
	}
	s.removeErr = fmt.Sprintf(s.text.RemoveFail, git, cause(err))
}

// cause is the short cause of an error: the innermost, without the path the operation names.
func cause(err error) string {
	for {
		inner := errors.Unwrap(err)
		if inner == nil {
			return err.Error()
		}
		err = inner
	}
}

func (s *Screen) systemName() string {
	if s.system == Chinese {
		return s.text.LangChinese
	}
	return s.text.LangEnglish
}

func (s *Screen) label(a account.Account) string {
	if strings.TrimSpace(a.Label) == "" {
		return s.text.Unnamed
	}
	return a.Label
}

// identityCount is "N identities", "1 identity" in English.
func (s *Screen) identityCount(n int) string {
	if n == 1 {
		return fmt.Sprintf(s.text.OneIdentity, n)
	}
	return fmt.Sprintf(s.text.Identities, n)
}

// names are the first three labels and how many more there are: "work, personal, client +2".
func (s *Screen) names(list []account.Account) string {
	var shown []string
	for _, a := range list[:min(len(list), 3)] {
		shown = append(shown, s.label(a))
	}
	out := strings.Join(shown, ", ")
	if len(list) > 3 {
		out += " " + fmt.Sprintf(s.text.More, len(list)-3)
	}
	return out
}

// size is a file size in KB with one decimal, 0.1 KB at least.
func size(n int64) string { return fmt.Sprintf("%.1f KB", max(float64(n)/1024, 0.1)) }

// stamp is the time of a backup in local time.
func (s *Screen) stamp(t time.Time) string { return t.In(s.zone).Format("2006-01-02 15:04") }
