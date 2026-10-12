// Package panel is the window the tray icon opens: the identity list and the drop zone that shows which identity
// applies in a folder. State changes on the UI thread only; disk and git work happen in the callbacks' owner.
package panel

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/match"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/ui/locale"
)

//go:embed copy.json
var catalog []byte

// Copy is the panel's text in one language. Main reads the tray texts from it too.
type Copy struct {
	Hide, Quit, Loading, DropTitle, DropHint, DropAgain, Choose, ChooseTitle, Empty, EmptyHint, Filter, NoneMatch,
	Unnamed, NoAlias, NoEmail, ReadOnly, ReadOnlyHint, ScopeGlobal, ScopePaths, CopyPath, Copied, Clear, Checking,
	NoMatch, NotRepo, NotFolder, GitFailed, TooltipIdle, TooltipMatch, TooltipNoMatch, Open, Custom string
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
// back through SetResult.
type Actions struct {
	Resolve      func(dir string)
	ChooseFolder func()
	CopyText     func(string)
	Hide         func()
	Quit         func()
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
	result    *match.Result
	resultErr error

	query       string
	providers   []account.Provider
	copiedUntil time.Time
	now         func() time.Time
}

// New returns a panel that shows the loading state until SetAccounts.
func New(text Copy, env paths.Env, act Actions) *Screen {
	return &Screen{text: text, env: env, act: act, loading: true, now: time.Now}
}

// SetAccounts shows a freshly loaded list, or the read-only state when readErr is set.
func (s *Screen) SetAccounts(list []account.Account, readErr error, dataPath string) {
	s.loading, s.accounts, s.readErr, s.dataPath = false, account.Sorted(list), readErr, dataPath
	s.providers = slices.DeleteFunc(s.providers, func(p account.Provider) bool {
		return !slices.Contains(account.ProvidersInUse(s.accounts), p)
	})
}

// Drop takes the first of the dropped paths and asks for it to be resolved.
func (s *Screen) Drop(dropped []string) {
	if len(dropped) == 0 || s.loading || s.readErr != nil {
		return
	}
	s.checking, s.result, s.resultErr = dropped[0], nil, nil
	s.act.Resolve(dropped[0])
}

// SetResult shows the outcome of resolving dir; a result for a folder that is no longer the latest is dropped.
func (s *Screen) SetResult(dir string, r match.Result, err error) {
	if dir != s.checking {
		return
	}
	s.checking = ""
	if err != nil {
		s.result, s.resultErr = nil, err
		return
	}
	s.result, s.resultErr = &r, nil
}

// Result is the latest outcome, nil when there is none.
func (s *Screen) Result() *match.Result { return s.result }

func (s *Screen) clear() { s.checking, s.result, s.resultErr = "", nil, nil }

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
