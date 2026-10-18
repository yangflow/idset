package settings

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/keygen"
)

// defaultKeyName is the file name the Keys form starts with.
const defaultKeyName = "id_idset"

// newIdentity is the choice of the identity picker that makes a new identity for the key.
const newIdentity = "\x00new"

type agentState int

const (
	agentIdle agentState = iota
	agentBusy
	agentLoaded
	agentMissing
	agentFailed
)

// keys is the state of the Keys pane: the form until a key is generated, then the result.
type keys struct {
	kind     int
	platform string
	name     string
	comment  string
	// fresh: the comment has not been filled in yet, for want of identities.
	fresh bool
	// target is the identity the form was filled in for from the diagnostics; it gets the key once generated.
	target string

	busy bool
	err  string

	key      *keygen.Key
	made     KeyRequest
	provider account.Provider
	copied   time.Time

	agent    agentState
	agentErr string

	used    string
	useBusy bool
	useErr  string

	pickOpen bool
	pick     string
}

// resetKeys puts the form back to its defaults.
func (s *Screen) resetKeys() {
	s.keys = keys{name: defaultKeyName, platform: account.GitHub.DisplayName(), fresh: true}
	if len(s.accounts) > 0 {
		s.keys.comment, s.keys.fresh = s.accounts[0].GitUserEmail, false
	}
}

func (s *Screen) identity(id string) (account.Account, bool) {
	if i := slices.IndexFunc(s.accounts, func(a account.Account) bool { return a.ID == id }); i >= 0 {
		return s.accounts[i], true
	}
	return account.Account{}, false
}

// keysFor fills the form in for an identity whose key is missing: its key's file name, its email as the comment and
// its platform. When its key path does not lie directly in ~/.ssh/ with a valid name, the plain form shows.
func (s *Screen) keysFor(id string) {
	if s.keys.busy {
		return
	}
	s.resetKeys()
	a, ok := s.identity(id)
	if !ok {
		return
	}
	name := keygen.NameOf(s.env, a.KeyPath)
	if name == "" {
		return
	}
	s.keys.name, s.keys.comment, s.keys.target = name, a.GitUserEmail, id
	s.keys.platform = s.providerName(a.Provider)
}

func (s *Screen) providerName(p account.Provider) string {
	if p == account.Custom {
		return s.text.Custom
	}
	return p.DisplayName()
}

// formProvider is the platform the form names.
func (s *Screen) formProvider() account.Provider {
	for _, p := range account.Providers {
		if s.providerName(p) == s.keys.platform {
			return p
		}
	}
	return account.GitHub
}

// nameInvalid: the file name is not one Generate accepts.
func (s *Screen) nameInvalid() bool { return !keygen.ValidName(s.keys.name) }

func (s *Screen) generate() {
	if s.keys.busy || s.nameInvalid() {
		return
	}
	req := KeyRequest{Type: keygen.Ed25519, Name: s.keys.name, Comment: keygen.Comment(s.keys.comment)}
	if s.keys.kind == 1 {
		req.Type = keygen.RSA
	}
	s.keys.busy, s.keys.err, s.keys.made, s.keys.provider = true, "", req, s.formProvider()
	s.act.Generate(req)
}

// GenerateDone reports Generate. When the form was filled in for an identity whose key path is the new key, the
// key goes to it straight away.
func (s *Screen) GenerateDone(key keygen.Key, err error) {
	s.keys.busy = false
	var exists *keygen.ExistsError
	switch {
	case err == nil:
	case errors.As(err, &exists):
		s.keys.err = fmt.Sprintf(s.text.KExists, exists.Path)
		return
	case errors.Is(err, keygen.ErrName):
		s.keys.err = s.text.KInvalid
		return
	case keygen.NotFound(err):
		s.keys.err = s.text.KNoKeygen
		return
	default:
		s.keys.err = fmt.Sprintf(s.text.KFail, cause(err))
		return
	}
	s.keys.key = &key
	s.keys.agent, s.keys.used, s.keys.useErr = agentIdle, "", ""
	s.announce = s.text.Created
	if a, ok := s.identity(s.keys.target); ok && !s.readOnly && s.env.Equal(s.env.Tilde(strings.TrimSpace(a.KeyPath)), key.Path) {
		s.use(a.ID)
	}
}

// another goes back to the form for one more key.
func (s *Screen) another() {
	if s.keys.useBusy || s.keys.agent == agentBusy {
		return
	}
	s.resetKeys()
}

func (s *Screen) copyPublic() {
	s.keys.copied = s.now().Add(flashFor)
	s.announce = s.text.Copied
}

func (s *Screen) loadKey() {
	if s.keys.key == nil || s.keys.agent == agentBusy || s.keys.agent == agentLoaded {
		return
	}
	s.keys.agent, s.keys.agentErr = agentBusy, ""
	s.act.LoadKey(s.keys.key.Path)
}

// LoadDone reports LoadKey.
func (s *Screen) LoadDone(err error) {
	switch {
	case err == nil:
		s.keys.agent = agentLoaded
		s.announce = s.text.Loaded
	case errors.Is(err, keygen.ErrNoAgent):
		s.keys.agent = agentMissing
	default:
		s.keys.agent, s.keys.agentErr = agentFailed, fmt.Sprintf(s.text.LoadFail, cause(err))
	}
}

// openPick opens the identity picker on the identity the key went to, else the first without a key, else the
// first, else a new identity.
func (s *Screen) openPick() {
	if s.keys.key == nil || s.keys.useBusy || s.readOnly {
		return
	}
	s.keys.pick = newIdentity
	switch {
	case s.keys.used != "":
		s.keys.pick = s.keys.used
	case len(s.accounts) > 0:
		s.keys.pick = s.accounts[0].ID
		if i := slices.IndexFunc(s.accounts, func(a account.Account) bool { return strings.TrimSpace(a.KeyPath) == "" }); i >= 0 {
			s.keys.pick = s.accounts[i].ID
		}
	}
	s.keys.pickOpen = true
}

func (s *Screen) confirmPick() {
	s.keys.pickOpen = false
	if s.keys.key == nil {
		return
	}
	if s.keys.pick == newIdentity {
		s.act.NewIdentity(KeyDraft{Provider: s.keys.provider, Email: s.draftEmail(), KeyPath: s.keys.key.Path,
			Fingerprint: s.keys.key.Fingerprint})
		return
	}
	if _, ok := s.identity(s.keys.pick); ok {
		s.use(s.keys.pick)
	}
}

// draftEmail is the comment of the new key when it reads as an email address, for a new identity.
func (s *Screen) draftEmail() string {
	if c := s.keys.made.Comment; strings.Contains(c, "@") && !strings.ContainsAny(c, " \t") {
		return c
	}
	return ""
}

func (s *Screen) use(id string) {
	if s.keys.useBusy || s.readOnly {
		return
	}
	s.keys.useBusy, s.keys.useErr = true, ""
	s.act.UseKey(id, s.keys.key.Path, s.keys.key.Fingerprint)
}

// UseDone reports UseKey.
func (s *Screen) UseDone(id string, err error) {
	s.keys.useBusy = false
	if err != nil {
		s.keys.useErr = fmt.Sprintf(s.text.UseFail, cause(err))
		return
	}
	s.keys.used = id
	if a, ok := s.identity(id); ok {
		s.announce = fmt.Sprintf(s.text.UsedNote, s.label(a))
	}
}

// revealLabel names the platform's file manager on the button that shows the key.
func (s *Screen) revealLabel() string {
	switch s.goos {
	case "darwin":
		return s.text.RevealFinder
	case "windows":
		return s.text.RevealExplorer
	}
	return s.text.RevealFiles
}
