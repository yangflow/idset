package account

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Version is the accounts.json format this program reads and writes.
const Version = 1

// ErrNewerVersion is returned for a file written by a newer version of the program.
var ErrNewerVersion = errors.New("accounts.json was written by a newer version of Idset")

// Scope is either global or a list of gitdir paths in stored form. An empty list is valid and matches nothing.
type Scope struct {
	Global bool
	Paths  []string
}

type gitdirScope struct {
	Paths []string `json:"paths"`
}

type scopeJSON struct {
	Global *struct{}    `json:"global,omitempty"`
	Gitdir *gitdirScope `json:"gitdir,omitempty"`
}

func (s Scope) MarshalJSON() ([]byte, error) {
	if s.Global {
		return json.Marshal(scopeJSON{Global: &struct{}{}})
	}
	paths := s.Paths
	if paths == nil {
		paths = []string{}
	}
	return json.Marshal(scopeJSON{Gitdir: &gitdirScope{Paths: paths}})
}

func (s *Scope) UnmarshalJSON(data []byte) error {
	var v scopeJSON
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch {
	case v.Global != nil && v.Gitdir == nil:
		*s = Scope{Global: true}
	case v.Gitdir != nil && v.Global == nil:
		*s = Scope{Paths: v.Gitdir.Paths}
	default:
		return errors.New(`scope must be either "global" or "gitdir"`)
	}
	return nil
}

type file struct {
	Accounts []Account `json:"accounts"`
	Version  int       `json:"version"`
}

// Decode reads accounts.json. Missing provider and port take their defaults; anything else missing or unknown
// is an error, so that a damaged file is never mistaken for an empty one.
func Decode(data []byte) ([]Account, error) {
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	var raw struct {
		Accounts []map[string]json.RawMessage `json:"accounts"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	switch {
	case f.Version > Version:
		return nil, ErrNewerVersion
	case f.Version < 1:
		return nil, errors.New("accounts.json has no version")
	case f.Accounts == nil:
		return nil, errors.New("accounts.json has no account list")
	}
	for i := range f.Accounts {
		a := &f.Accounts[i]
		if !validID(a.ID) {
			return nil, fmt.Errorf("account %d has no valid id", i+1)
		}
		if _, ok := raw.Accounts[i]["scope"]; !ok {
			return nil, fmt.Errorf("account %s has no scope", a.ID)
		}
		if a.Provider == "" {
			a.Provider = GitHub
		}
		if !a.Provider.Valid() {
			return nil, fmt.Errorf("account %s has an unknown provider %q", a.ID, a.Provider)
		}
		if a.SSHPort == 0 {
			a.SSHPort = DefaultPort(a.Provider)
		}
		if a.URLRewrites == nil {
			a.URLRewrites = []Rewrite{}
		}
	}
	return f.Accounts, nil
}

// Encode writes accounts.json: keys in alphabetical order, two-space indent, UTC times to the second.
func Encode(accounts []Account) ([]byte, error) {
	list := make([]Account, len(accounts))
	for i, a := range accounts {
		a.CreatedAt = second(a.CreatedAt)
		a.UpdatedAt = second(a.UpdatedAt)
		if a.LastUsedAt != nil {
			t := second(*a.LastUsedAt)
			a.LastUsedAt = &t
		}
		if a.URLRewrites == nil {
			a.URLRewrites = []Rewrite{}
		}
		list[i] = a
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(file{Accounts: list, Version: Version}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func second(t time.Time) time.Time { return t.UTC().Truncate(time.Second) }
