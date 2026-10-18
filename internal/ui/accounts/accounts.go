// Package accounts is the account window: the identity list on the left, the editor of the selected identity on
// the right, and the confirmation for deleting one. State changes on the UI thread only; saving, deleting and
// the commands behind them run in the owner's background work (see Backend) and report back through SaveDone,
// DeleteDone and the other setters.
package accounts

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
	"github.com/yangflow/idset/internal/gitdir"
	"github.com/yangflow/idset/internal/keyfile"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/store"
	"github.com/yangflow/idset/internal/ui/locale"
)

//go:embed copy.json
var catalog []byte

// Copy is the window's text in one language. Main reads the window title from it.
type Copy struct {
	Window, AddIdentity, Identities, Scoped, Global, Unnamed, NewTag, Loading, NoneTitle, NoneHint, Create, ImportExisting,
	GroupIdentity, GroupSSH, GroupScope, GroupRewrites, GroupNotes, GroupTimes,
	Name, Platform, Username, GitName, Email, Alias, Key, Port, Host, Custom, ChooseFile, ChooseFolder, Path, ChooseKeyTitle,
	ChooseFolderTitle, RemovePath, RemoveRewrite, ColorLabel, Hex, Applies, ScopeGlobal, ScopeGitdir, ScopeHint,
	AddPath, From, To, PushOnly, AddRewrite, AddDefault, NeedAlias, FromGit, Created, Updated, Used, Never,
	Blue, Green, Orange, Red, Purple, Yellow,
	Delete, Revert, Save, Saving, Saved, Dirty, DraftStatus, Writing, WriteFailed, RegenFailed, KeyNotDeleted,
	ProblemsOne, ProblemsMany, ReadOnly, ReadOnlyHint, DelTitle, DelBody, PrivateKey, Folders, None, DelNote, AlsoKey,
	KeyNote, Cancel, ReasonNoKey, ReasonShared, ReasonSymlink, ReasonMissing, ReasonNotFile, DeleteFailed,
	ErrAlias, ErrDuplicateAlias, ErrReservedAlias, ErrMissingHost, ErrBadHost, ErrBadPort, ErrNewLine, ErrQuote,
	ErrSecondGlobal, ErrBadColor, ErrOther string
}

var translations = func() map[string]Copy {
	var values map[string]Copy
	if err := json.Unmarshal(catalog, &values); err != nil {
		panic(err)
	}
	return values
}()

// Text returns the copy for a system locale.
func Text(systemLocale string) Copy { return translations[locale.Language(systemLocale)] }

// Actions are what the window asks its owner to do. All of them return at once: the owner works in the
// background and reports back on the UI thread.
type Actions struct {
	// Save stores a (added when isNew) and answers with SaveDone.
	Save func(a account.Account, isNew bool)
	// Delete removes an identity, and its private key when deleteKey is set, and answers with DeleteDone.
	Delete func(id string, deleteKey bool)
	// InspectKey checks whether the private key of an identity can be deleted, and answers with SetKeyReason.
	InspectKey func(id string)
	// Prefill reads `git config --global` for a new identity and answers with SetPrefill.
	Prefill func(id string)
	// ChooseKey and ChooseFolder show the system pickers; the answers are SetKeyPath and SetFolder.
	ChooseKey    func(id string)
	ChooseFolder func(id string, index int)
	// OpenImport opens the settings window on Import.
	OpenImport func()
}

// Request says what the window should select when it opens.
type Request struct {
	// New asks for a new draft; otherwise ID asks for that identity. Neither selects the first identity in
	// storage order, unless one is selected already.
	New bool
	ID  string
	// Folder, with New, scopes the draft to this folder: the folder of a result card to bind a new identity to.
	// A draft that is open already takes the folder as one more path.
	Folder string
	// Key, with New, fills the draft in for a key generated in the settings window. A draft that is open already
	// takes it too.
	Key *Key
}

// Key is a generated key a new identity starts from: its platform, the email of its comment (may be empty), its
// stored path and its fingerprint.
type Key struct {
	Provider    account.Provider
	Email       string
	Path        string
	Fingerprint string
}

// flashFor is how long "Written" stays on the Save button.
const flashFor = 1200 * time.Millisecond

type confirm struct {
	id       string
	checking bool
	reason   keyfile.Reason
	other    string
	alsoKey  bool
	err      string
}

// Screen holds the window's state.
type Screen struct {
	text Copy
	env  paths.Env
	act  Actions
	now  func() time.Time
	goos string

	loading  bool
	readErr  error
	dataPath string
	// list is the stored identities in storage order.
	list    []account.Account
	pending *Request

	// forms are the edits in progress, by identity ID; the draft of a new identity is one of them.
	forms   map[string]*form
	draftID string
	// draftFolder is the folder the draft was opened for, "" when it was not opened for one.
	draftFolder string
	sel         string
	splitW      float32

	busy bool
	// problems are the checks the last save attempt failed, for the form they were made on, until it changes.
	problems      []account.Problem
	problemsID    string
	problemsState values
	notice        string
	savedUntil    time.Time
	announce      string

	confirmOpen bool
	confirm     *confirm
}

// New returns a window that shows the loading state until SetAccounts.
func New(text Copy, env paths.Env, act Actions) *Screen {
	return &Screen{text: text, env: env, act: act, now: time.Now, goos: runtime.GOOS, loading: true,
		forms: map[string]*form{}, splitW: 232}
}

// SetText changes the language of the window.
func (s *Screen) SetText(text Copy) { s.text = text }

// Open selects what the request asks for, as soon as the list is loaded.
func (s *Screen) Open(req Request) {
	if s.loading {
		s.pending = &req
		return
	}
	s.apply(req)
}

func (s *Screen) apply(req Request) {
	switch {
	case req.New:
		s.newDraft(req.Folder)
		if req.Key != nil {
			s.draftKey(*req.Key)
		}
	case req.ID != "" && s.stored(req.ID) != nil:
		s.sel = req.ID
	case s.selectedForm() == nil:
		s.selectFirst()
	}
}

// SetAccounts shows a freshly loaded list, or the read-only state when readErr is set. Edits in progress stay.
func (s *Screen) SetAccounts(list []account.Account, readErr error, dataPath string) {
	s.loading, s.list, s.readErr, s.dataPath = false, list, readErr, dataPath
	s.refresh()
	if s.pending != nil {
		req := *s.pending
		s.pending = nil
		s.apply(req)
	}
	if s.selectedForm() == nil {
		s.selectFirst()
	}
}

// refresh drops the forms of identities that are gone and reloads those without edits.
func (s *Screen) refresh() {
	for id, f := range s.forms {
		if f.isNew {
			continue
		}
		a := s.stored(id)
		switch {
		case a == nil:
			delete(s.forms, id)
		case !f.dirty():
			s.forms[id] = newForm(*a, false)
		default:
			f.base = *a
		}
	}
}

func (s *Screen) stored(id string) *account.Account {
	if i := slices.IndexFunc(s.list, func(a account.Account) bool { return a.ID == id }); i >= 0 {
		return &s.list[i]
	}
	return nil
}

// selectFirst selects the draft if there is one, otherwise the first identity in storage order.
func (s *Screen) selectFirst() {
	switch {
	case s.draftID != "":
		s.sel = s.draftID
	case len(s.list) > 0:
		s.sel = s.list[0].ID
	default:
		s.sel = ""
	}
}

// selectedForm is the form of the selected identity, created on first use; nil when nothing is selected.
func (s *Screen) selectedForm() *form {
	if s.sel == "" {
		return nil
	}
	if f, ok := s.forms[s.sel]; ok {
		return f
	}
	if a := s.stored(s.sel); a != nil {
		f := newForm(*a, false)
		s.forms[s.sel] = f
		return f
	}
	return nil
}

func (s *Screen) empty() bool { return len(s.list) == 0 && s.draftID == "" }

// newDraft selects the draft, creating it with the defaults of a new identity and asking for the signature of
// `git config --global` to be filled in. With a folder the draft applies to that folder only.
func (s *Screen) newDraft(folder string) {
	if s.readErr != nil {
		return
	}
	stored := gitdir.Normalize(folder, s.env)
	if s.draftID == "" {
		a := account.New(s.now())
		a.Color = s.freeColor()
		if stored != "" {
			a.Scope = account.Scope{Paths: []string{stored}}
			s.draftFolder = folder
		}
		s.draftID = a.ID
		s.forms[a.ID] = newForm(a, true)
		s.act.Prefill(a.ID)
	} else if f := s.forms[s.draftID]; stored != "" && f != nil {
		f.addFolder(stored)
		s.draftFolder = folder
	}
	s.sel = s.draftID
	s.notice = ""
}

// draftKey puts a generated key into the open draft: its path and fingerprint, its platform, and its email when the
// draft has none yet.
func (s *Screen) draftKey(k Key) {
	f := s.forms[s.draftID]
	if f == nil {
		return
	}
	f.setProvider(k.Provider)
	f.platform = s.providerName(k.Provider)
	f.cur.keyPath = k.Path
	// The fingerprint is kept on save while the key path stays the one of base.
	f.base.KeyPath, f.base.KeyFingerprint = k.Path, k.Fingerprint
	if k.Email != "" && f.cur.email == "" {
		f.cur.email = k.Email
	}
}

// freeColor is the first preset no identity uses, or the first preset.
func (s *Screen) freeColor() string {
	for _, c := range account.ColorPresets {
		if !slices.ContainsFunc(s.list, func(a account.Account) bool { return a.Color == c }) {
			return c
		}
	}
	return account.ColorPresets[0]
}

// SetPrefill fills the draft's signature from git config, only into fields that are still empty.
func (s *Screen) SetPrefill(id, name, email string) {
	f, ok := s.forms[id]
	if !ok || !f.isNew {
		return
	}
	if name != "" && f.cur.gitName == "" {
		f.cur.gitName, f.prefName = name, name
	}
	if email != "" && f.cur.email == "" {
		f.cur.email, f.prefEmail = email, email
	}
}

// SetKeyPath takes the file chosen for an identity's private key.
func (s *Screen) SetKeyPath(id, path string) {
	if f, ok := s.forms[id]; ok && strings.TrimSpace(path) != "" {
		f.cur.keyPath = s.env.Tilde(path)
	}
}

// SetFolder takes the folder chosen for the index-th path of an identity.
func (s *Screen) SetFolder(id string, index int, path string) {
	if f, ok := s.forms[id]; ok && index >= 0 && index < len(f.cur.paths) && strings.TrimSpace(path) != "" {
		f.cur.paths[index] = gitdir.Normalize(path, s.env)
	}
}

func (s *Screen) selectID(id string) {
	if s.busy {
		return
	}
	s.sel, s.notice = id, ""
}

// save checks the form and hands it to the owner.
func (s *Screen) save() {
	f := s.selectedForm()
	if f == nil || s.busy || s.readErr != nil || (!f.isNew && !f.dirty()) {
		return
	}
	base := f.base
	if a := s.stored(f.id); a != nil {
		base = *a
	}
	a, portOK := f.account(base, s.env)
	s.notice = ""
	if problems := s.check(a, f.isNew, portOK); len(problems) > 0 {
		s.problems, s.problemsID, s.problemsState = problems, f.id, f.cur.clone()
		return
	}
	s.problems = nil
	s.busy = true
	s.act.Save(a, f.isNew)
}

// check returns the problems the change introduces for a; problems that accounts.json already had by hand
// editing do not stop other changes, as in the store.
func (s *Screen) check(a account.Account, isNew, portOK bool) []account.Problem {
	next := slices.Clone(s.list)
	if i := slices.IndexFunc(next, func(o account.Account) bool { return o.ID == a.ID }); i >= 0 {
		next[i] = a
	} else {
		next = append(next, a)
	}
	before := map[account.Problem]bool{}
	for _, p := range account.Validate(s.list) {
		before[p] = true
	}
	var out []account.Problem
	for _, p := range account.Validate(next) {
		switch {
		case before[p]:
		case p.ID == a.ID:
			out = append(out, p)
		case p.Code == account.SecondGlobal:
			// The check blames the identity that comes later in the list; tell this form about the other one.
			out = append(out, account.Problem{ID: a.ID, Field: "scope", Code: p.Code, Other: s.labelOfID(next, p.ID)})
		case p.Code == account.DuplicateAlias:
			out = append(out, account.Problem{ID: a.ID, Field: "sshAlias", Code: p.Code, Other: s.labelOfID(next, p.ID)})
		}
	}
	if !portOK && a.Provider == account.Custom && !slices.ContainsFunc(out, func(p account.Problem) bool { return p.Field == "sshPort" }) {
		out = append(out, account.Problem{ID: a.ID, Field: "sshPort", Code: account.BadPort})
	}
	return out
}

// SaveDone reports the owner's Save: list is the stored list after it. The list stands when err is a
// *store.RegenerateError: the identity is saved, only the configuration was not rewritten.
func (s *Screen) SaveDone(id string, list []account.Account, err error) {
	s.busy = false
	var regenerate *store.RegenerateError
	saved := err == nil || errors.As(err, &regenerate)
	if !saved {
		var problems account.Problems
		if f := s.forms[id]; f != nil && errors.As(err, &problems) {
			s.problems, s.problemsID, s.problemsState = problems, id, f.cur.clone()
			return
		}
		s.notice = fmt.Sprintf(s.text.WriteFailed, s.env.Tilde(s.dataPath), cause(err))
		return
	}
	s.list = list
	a := s.stored(id)
	if a == nil {
		return
	}
	if id == s.draftID {
		s.draftID, s.draftFolder = "", ""
	}
	s.forms[id] = newForm(*a, false)
	s.sel, s.problems = id, nil
	if regenerate != nil {
		s.notice = fmt.Sprintf(s.text.RegenFailed, cause(regenerate.Err))
		return
	}
	s.notice = ""
	s.savedUntil = s.now().Add(flashFor)
	s.announce = s.text.Saved
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

// revert drops the edits of the selected identity; for a draft that is the draft itself.
func (s *Screen) revert() {
	f := s.selectedForm()
	if f == nil || s.busy {
		return
	}
	s.notice, s.problems = "", nil
	if f.isNew {
		delete(s.forms, f.id)
		s.draftID, s.draftFolder = "", ""
		s.selectFirst()
		return
	}
	if a := s.stored(f.id); a != nil {
		s.forms[f.id] = newForm(*a, false)
	}
}

// askDelete opens the confirmation for the selected identity and asks for its private key to be inspected.
func (s *Screen) askDelete() {
	f := s.selectedForm()
	if f == nil || f.isNew || s.busy || s.readErr != nil {
		return
	}
	s.confirm = &confirm{id: f.id, checking: true}
	s.confirmOpen = true
	s.act.InspectKey(f.id)
}

// SetKeyReason reports InspectKey. A key that cannot be deleted unchecks the box.
func (s *Screen) SetKeyReason(id string, reason keyfile.Reason, other string) {
	if c := s.confirm; c != nil && c.id == id {
		c.checking, c.reason, c.other = false, reason, other
		if reason != keyfile.None {
			c.alsoKey = false
		}
	}
}

// confirmDelete starts the deletion the dialog asks for.
func (s *Screen) confirmDelete() {
	c := s.confirm
	if c == nil || c.checking || s.busy {
		return
	}
	c.err = ""
	s.busy = true
	s.act.Delete(c.id, c.alsoKey && c.reason == keyfile.None)
}

// DeleteDone reports the owner's Delete. When the identity could not be deleted the dialog stays and says why;
// otherwise it is gone, and a failure of the configuration or of the private key is told in the status line.
func (s *Screen) DeleteDone(id string, list []account.Account, res DeleteResult) {
	s.busy = false
	err, keyErr := res.Err, res.KeyErr
	var regenerate *store.RegenerateError
	if err != nil && !errors.As(err, &regenerate) && !errors.Is(err, store.ErrNotFound) {
		if s.confirm != nil {
			s.confirm.err = fmt.Sprintf(s.text.DeleteFailed, cause(err))
		}
		return
	}
	s.list = list
	delete(s.forms, id)
	s.confirmOpen, s.confirm = false, nil
	s.selectFirst()
	var parts []string
	if regenerate != nil {
		parts = append(parts, fmt.Sprintf(s.text.RegenFailed, cause(regenerate.Err)))
	}
	if keyErr != nil {
		parts = append(parts, fmt.Sprintf(s.text.KeyNotDeleted, cause(keyErr)))
	}
	s.notice = strings.Join(parts, " ")
}

// problemTexts are the messages of the failed checks of the selected form on the given fields.
func (s *Screen) problemTexts(f *form, fields ...string) []string {
	var out []string
	for _, p := range s.problems {
		if s.problemsID == f.id && slices.Contains(fields, p.Field) {
			out = append(out, s.problemText(p))
		}
	}
	return out
}

func (s *Screen) problemText(p account.Problem) string {
	other := p.Other
	if strings.TrimSpace(other) == "" {
		other = s.text.Unnamed
	}
	switch p.Code {
	case account.BadAlias:
		return s.text.ErrAlias
	case account.DuplicateAlias:
		return fmt.Sprintf(s.text.ErrDuplicateAlias, other)
	case account.ReservedAlias:
		return s.text.ErrReservedAlias
	case account.MissingHost:
		return s.text.ErrMissingHost
	case account.BadHost:
		return s.text.ErrBadHost
	case account.BadPort:
		return s.text.ErrBadPort
	case account.NewLine:
		return s.text.ErrNewLine
	case account.Quote:
		return s.text.ErrQuote
	case account.SecondGlobal:
		return fmt.Sprintf(s.text.ErrSecondGlobal, other)
	case account.BadColor:
		return s.text.ErrBadColor
	}
	return s.text.ErrOther
}

func (s *Screen) labelOfID(list []account.Account, id string) string {
	for _, a := range list {
		if a.ID == id {
			return a.Label
		}
	}
	return ""
}

func (s *Screen) label(a account.Account) string { return s.labelOf(a.Label) }

func (s *Screen) labelOf(label string) string {
	if strings.TrimSpace(label) == "" {
		return s.text.Unnamed
	}
	return label
}

func (s *Screen) providerName(p account.Provider) string {
	if p == account.Custom {
		return s.text.Custom
	}
	return p.DisplayName()
}

// Selected is the ID of the selected identity or draft, "" when none is.
func (s *Screen) Selected() string { return s.sel }

// DraftFolder is the folder the draft with this ID was opened for, "" for any other identity.
func (s *Screen) DraftFolder(id string) string {
	if id == "" || id != s.draftID {
		return ""
	}
	return s.draftFolder
}

// Draft is the ID of the draft, "" when there is none.
func (s *Screen) Draft() string { return s.draftID }
