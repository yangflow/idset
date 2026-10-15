// Package settings keeps settings.json, the interface preferences in Idset's directory. Today that is the owner
// prefix the clone helper remembers for each identity. Keys this version does not know are kept when it writes.
// A file that cannot be read is never overwritten. Methods block on disk work, so callers run them off the UI thread.
package settings

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/yangflow/idset/internal/fsutil"
)

// FileName is the name of the preferences in the Idset directory.
const FileName = "settings.json"

const prefixesKey = "clonePrefixes"

// Settings is safe for concurrent use.
type Settings struct {
	path string

	mu       sync.Mutex
	raw      map[string]json.RawMessage
	prefixes map[string]string
	readErr  error
}

// Open reads settings.json in dir. A missing file is empty settings; one that cannot be parsed is empty too, and
// Err says why.
func Open(dir string) *Settings {
	s := &Settings{path: filepath.Join(dir, FileName)}
	s.Reload()
	return s
}

// Reload reads the file again.
func (s *Settings) Reload() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.raw, s.prefixes, s.readErr = map[string]json.RawMessage{}, map[string]string{}, nil
	data, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err != nil {
		s.readErr = err
		return
	}
	if err := json.Unmarshal(data, &s.raw); err != nil {
		s.raw, s.readErr = map[string]json.RawMessage{}, err
		return
	}
	if v, ok := s.raw[prefixesKey]; ok {
		if err := json.Unmarshal(v, &s.prefixes); err != nil || s.prefixes == nil {
			s.prefixes = map[string]string{}
		}
	}
}

// Err is why the file could not be read, nil when it could.
func (s *Settings) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readErr
}

// Prefixes are the remembered owner prefixes by identity ID.
func (s *Settings) Prefixes() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]string, len(s.prefixes))
	for id, p := range s.prefixes {
		out[id] = p
	}
	return out
}

// SetPrefix remembers prefix for the identity and writes the file atomically. An empty prefix forgets it. It
// changes nothing and returns the read error when the file could not be read.
func (s *Settings) SetPrefix(id, prefix string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.readErr != nil {
		return s.readErr
	}
	if s.prefixes[id] == prefix || (prefix == "" && s.prefixes[id] == "") {
		return nil
	}
	next := make(map[string]string, len(s.prefixes)+1)
	for k, v := range s.prefixes {
		next[k] = v
	}
	if prefix == "" {
		delete(next, id)
	} else {
		next[id] = prefix
	}
	raw := make(map[string]json.RawMessage, len(s.raw)+1)
	for k, v := range s.raw {
		raw[k] = v
	}
	if len(next) == 0 {
		delete(raw, prefixesKey)
	} else {
		v, err := json.Marshal(next)
		if err != nil {
			return err
		}
		raw[prefixesKey] = v
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	if err := fsutil.WriteAtomic(s.path, append(data, '\n'), 0o600, 0o700); err != nil {
		return err
	}
	s.raw, s.prefixes = raw, next
	return nil
}
