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
// RealHome is Home with symbolic links resolved, when that differs, because git reports resolved paths.
type Env struct {
	Home     string
	RealHome string
	Windows  bool
}

// Current returns the environment of the running program.
func Current() (Env, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Env{}, err
	}
	env := Env{Home: home, Windows: runtime.GOOS == "windows"}
	if real, err := filepath.EvalSymlinks(home); err == nil && real != home {
		env.RealHome = real
	}
	return env, nil
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

// Tilde returns p in slash form, with the home directory, as configured or resolved, replaced by ~.
func (e Env) Tilde(p string) string {
	p = Slash(p)
	for _, home := range []string{e.home(), strings.TrimSuffix(Slash(e.RealHome), "/")} {
		switch {
		case home == "":
		case e.Equal(p, home):
			return "~"
		case e.HasPrefix(p, home+"/"):
			return "~" + p[len(home):]
		}
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
