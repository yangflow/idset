// Package panel is the window the tray icon opens: the identity list and the drop zone that shows which identity
// applies in a folder. State changes on the UI thread only; disk and git work happen in the callbacks' owner.
package panel

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/bind"
	"github.com/yangflow/idset/internal/match"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/ui/locale"
)

//go:embed copy.json
var catalog []byte

// Copy is the panel's text in one language. Main reads the tray texts from it too.
type Copy struct {
	HidePanel, Quit, Loading, DropLine, DropHint, Choose, ChooseShort, ChooseTitle, Empty, EmptyHint, Filter, NoneMatch,
	Unnamed, NoAlias, NoEmail, ReadOnly, ReadOnlyShort, ReadOnlyHint, CopyPath, Copied, Clear, Checking, NoMatch, NotRepo, NotFolder,
	GitFailed, Matched, TooltipIdle, TooltipMatch, TooltipNoMatch, Open, Custom, UsesHere, Alias, Email, Folders, Global,
	NoFolders, Identities, Reload, Settings, SettingsMenu, RestoreBackup, RevealFinder, RevealExplorer, RevealFiles, AddIdentity,
	OpenFinder, OpenExplorer, OpenFiles, More, BindTo, Bind, BindLabel, NewShort, NewBound, Cancel, Rebind, RebindTo,
	Unbind, OverlapTitle, OverlapHint, Used, Overridden, OnlyUse, RemoveFrom, Has, ConfirmShort, Covered, CoveredBy,
	BoundTo, Undo, Undone, BindFailed, BindRegenFailed, StaleHead, Then, StaleUpdate, StaleIgnore, StaleFix, StaleKeep,
	Updated, FindNoEmail, FindNoEmailTip, FindCommits, FindOwner, FindOtherTip, FindUnknownTip, FindPushes, FindKeyTip, HTTPSRow, HTTPSTip, CopyCmd, CopyCmdTip, CloneLabel, CloneTip, CloneHint, CloneCopy,
	CloneToggle, ConnOK, ConnOKPlain, ConnPending, ConnFail, Retry, Recheck, NoSSH, NoSSHWindows, NoSSHMac, NoSSHLinux, NoKey, NoKeyTip, GenerateKey, Locked, LockedTip, Unlock, UnlockFailed, UnlockCmdTip, CopyUnlock, TimedOut, TimedOutTip, NoHost, NoHostTip, Refused, RefusedTip, HostKey, HostKeyTip, Denied, DeniedTip, CopyPub, OpenKeys, OpenKeysTip, NoPub, NoPubTip, HTTPSBarTip, AddRewrite, AddRewriteTip, AllOK, OneError, Errors, OneWarning, Warnings, Net001, Net001How, CopyDebug, Ssh001, Ssh001How, Git001, Git001How, Git001Global, Regenerate, Git002, Git002How string
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

// Actions are what the panel asks its owner to do. Resolve and ChooseFolder run in the background and report
// back through SetResult, Run through OpDone.
type Actions struct {
	Resolve      func(dir string)
	ChooseFolder func()
	// Run binds, unbinds, rebinds or undoes, and looks at the folder again.
	Run func(op bind.Op)
	// OpenAccounts opens the accounts window on an identity; NewIdentity opens it on a new draft, and
	// NewIdentityFor on a new draft that applies to a folder only.
	OpenAccounts   func(id string)
	NewIdentity    func()
	NewIdentityFor func(folder string)
	// Open opens a folder in the system's file manager.
	Open   func(path string)
	Reload func()
	Reveal func(path string)
	// OpenSettings opens the settings window on the pane it showed last; OpenBackups opens it on Backups.
	OpenSettings func()
	OpenBackups  func()
	CopyText     func(string)
	// CloneCopied reports a clone command copied for an identity, with the owner prefix of the repository ("acme/",
	// or "" when there is none): the owner remembers the prefix and records a use of the identity.
	CloneCopied func(id, prefix string)
	// CheckAgain checks identities again now, all of them when ids is empty; the owner reports through
	// SetDiagnostics. Unlock loads the key of an identity into the agent with the keychain and reports through
	// UnlockDone; CopyPublicKey copies the .pub file of its key and reports through PublicKeyDone.
	CheckAgain    func(ids []string)
	Unlock        func(id string)
	CopyPublicKey func(id string)
	// AddRewrites adds the default rewrites of an identity's platform that it does not have yet; Regenerate writes
	// the managed files again. Both look at the folder again when they are done.
	AddRewrites func(id string)
	Regenerate  func()
	// GenerateKey opens the Keys pane of the settings window, filled in for an identity whose key is missing.
	GenerateKey func(id string)
	Hide        func()
	Quit        func()
}

// FilterThreshold is how many identities make the filter appear.
const FilterThreshold = 3

// Screen holds the panel's state.
type Screen struct {
	text Copy
	env  paths.Env
	act  Actions

	loading  bool
	readErr  error
	dataPath string
	accounts []account.Account

	checking  string
	lastDir   string
	result    *match.Result
	resultErr error
	// resultSeq counts the results shown; the view announces each one once, when announcedSeq catches up.
	resultSeq    int
	announcedSeq int

	// clone is the helper on the matched card, row the one under the open identity row, and prefixes are the
	// remembered owner prefixes by identity.
	matchClone, rowClone cloneField
	openRow              string
	prefixes             map[string]string
	httpsUntil           time.Time
	query                string
	providers            []account.Provider
	copiedUntil          time.Time
	now                  func() time.Time
	goos                 string
	diag                 diagState

	binding
}

// New returns a panel that shows the loading state until SetAccounts.
func New(text Copy, env paths.Env, act Actions) *Screen {
	return &Screen{text: text, env: env, act: act, loading: true, now: time.Now, goos: runtime.GOOS}
}

// SetText changes the language of the panel.
func (s *Screen) SetText(text Copy) { s.text = text }

// SetAccounts shows a freshly loaded list, or the read-only state when readErr is set.
func (s *Screen) SetAccounts(list []account.Account, readErr error, dataPath string) {
	s.loading, s.accounts, s.readErr, s.dataPath = false, account.Sorted(list), readErr, dataPath
	s.providers = slices.DeleteFunc(s.providers, func(p account.Provider) bool {
		return !slices.Contains(account.ProvidersInUse(s.accounts), p)
	})
}

// SetPrefixes gives the remembered owner prefixes by identity.
func (s *Screen) SetPrefixes(p map[string]string) { s.prefixes = p }

// Drop takes the first of the dropped paths and asks for it to be resolved.
func (s *Screen) Drop(dropped []string) {
	if len(dropped) == 0 || s.loading || s.readErr != nil {
		return
	}
	s.checking, s.lastDir, s.result, s.resultErr = dropped[0], dropped[0], nil, nil
	s.binding = binding{busy: s.busy}
	s.act.Resolve(dropped[0])
}

// SetResult shows the outcome of resolving dir; a result for a folder that is no longer the latest is dropped.
func (s *Screen) SetResult(dir string, r match.Result, err error) {
	if dir != s.checking {
		return
	}
	s.checking = ""
	s.resultSeq++
	s.matchClone, s.httpsUntil = cloneField{}, time.Time{}
	if err != nil {
		s.result, s.resultErr = nil, err
		return
	}
	s.result, s.resultErr = &r, nil
	s.matchClone.value = cloneSeed(r, s.prefixes[r.ID])
}

// Result is the latest outcome, nil when there is none.
func (s *Screen) Result() *match.Result { return s.result }

// unresolved is the sentence for a result without an identity.
func (s *Screen) unresolved(k match.Kind) string {
	return map[match.Kind]string{match.NoMatch: s.text.NoMatch, match.NotRepo: s.text.NotRepo, match.NotFolder: s.text.NotFolder}[k]
}

// resultMessage says the latest result in one sentence for screen readers, or "" when there is none.
func (s *Screen) resultMessage() string {
	switch {
	case s.resultErr != nil:
		return s.text.GitFailed + s.text.Then + s.resultErr.Error()
	case s.result == nil:
		return ""
	case s.result.Kind == match.Matched:
		if a, ok := s.find(s.result.ID); ok {
			msg := fmt.Sprintf(s.text.Matched, s.label(a))
			for _, f := range s.result.Findings {
				msg += s.text.Then + s.describe(f, a).sentence()
			}
			return msg
		}
		return ""
	}
	if a, ok := s.stale(); ok {
		return s.unresolved(s.result.Kind) + s.text.Then + s.label(a) + s.text.StaleHead
	}
	return s.unresolved(s.result.Kind)
}

func (s *Screen) clear() {
	s.checking, s.result, s.resultErr = "", nil, nil
	s.binding = binding{busy: s.busy}
}

func (s *Screen) find(id string) (account.Account, bool) {
	i := slices.IndexFunc(s.accounts, func(a account.Account) bool { return a.ID == id })
	if i < 0 {
		return account.Account{}, false
	}
	return s.accounts[i], true
}

func (s *Screen) label(a account.Account) string {
	if strings.TrimSpace(a.Label) == "" {
		return s.text.Unnamed
	}
	return a.Label
}

// revealLabel names the platform's file manager on the reveal button.
func (s *Screen) revealLabel() string {
	switch s.goos {
	case "darwin":
		return s.text.RevealFinder
	case "windows":
		return s.text.RevealExplorer
	}
	return s.text.RevealFiles
}

// Tooltip is the tray icon's tooltip for the current result.
func (s *Screen) Tooltip() string {
	switch {
	case s.result == nil:
		return s.text.TooltipIdle
	case s.result.Kind == match.Matched:
		if a, ok := s.find(s.result.ID); ok {
			return fmt.Sprintf(s.text.TooltipMatch, s.label(a))
		}
	}
	return s.text.TooltipNoMatch
}

func (s *Screen) visible() []account.Account {
	return account.Filter(s.accounts, s.query, s.providers)
}

func (s *Screen) toggleProvider(p account.Provider) {
	if i := slices.Index(s.providers, p); i >= 0 {
		s.providers = slices.Delete(s.providers, i, i+1)
	} else {
		s.providers = append(s.providers, p)
	}
}
