// Package paths converts between the stored form of a path (forward slashes, ~ for the home directory) and the
// absolute form, for the platform the program runs on or one given in tests.
package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Env is the home directory and whether paths compare as on Windows: case-insensitively, with drive letters.
type Env struct {
	Home    string
	Windows bool
}

// Current returns the environment of the running program.
func Current() (Env, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Env{}, err
	}
	return Env{Home: home, Windows: runtime.GOOS == "windows"}, nil
}

// Slash returns p with forward slashes, which git, ssh and Go accept on every platform.
func Slash(p string) string { return strings.ReplaceAll(p, `\`, "/") }

// Equal compares two paths in slash form, ignoring case on Windows.
func (e Env) Equal(a, b string) bool {
	if e.Windows {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// HasPrefix reports whether p starts with prefix, ignoring case on Windows.
func (e Env) HasPrefix(p, prefix string) bool {
	return len(p) >= len(prefix) && e.Equal(p[:len(prefix)], prefix)
}

func (e Env) home() string { return strings.TrimSuffix(Slash(e.Home), "/") }

// Tilde returns p in slash form, with the home directory replaced by ~.
func (e Env) Tilde(p string) string {
	p = Slash(p)
	home := e.home()
	switch {
	case home == "":
		return p
	case e.Equal(p, home):
		return "~"
	case e.HasPrefix(p, home+"/"):
		return "~" + p[len(home):]
	}
	return p
}

// Expand returns p in slash form with a leading ~ replaced by the home directory.
func (e Env) Expand(p string) string {
	p = Slash(p)
	switch {
	case p == "~":
		return e.home()
	case strings.HasPrefix(p, "~/"):
		return e.home() + p[1:]
	}
	return p
}

// Native returns p expanded and with the platform's separators, for the file system.
func (e Env) Native(p string) string { return filepath.FromSlash(e.Expand(p)) }
