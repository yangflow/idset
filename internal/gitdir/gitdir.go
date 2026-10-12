// Package gitdir normalizes includeIf gitdir paths, orders them the way they must be written so that the most
// specific path wins, and finds which identity git applies in a repository.
package gitdir

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/paths"
)

// Normalize returns the stored form of a path: trimmed, slashes, ~ for the home directory and a trailing slash.
// An empty or blank path stays empty.
func Normalize(raw string, env paths.Env) string {
	p := strings.TrimSpace(raw)
	if p == "" {
		return ""
	}
	p = env.Tilde(env.Expand(p))
	if !strings.HasSuffix(p, "/") {
		p += "/"
	}
	return p
}

// Block is one includeIf the generated gitconfig contains: a stored path of the identity at Index.
type Block struct {
	Index int
	Path  string
}

// Blocks returns every includeIf in the order it is written. git applies every matching includeIf in file order
// and the last value wins, so the least specific paths come first: fewer segments, then shorter, then storage
// order, then the path. Blank paths and duplicates within one identity are left out.
func Blocks(list []account.Account, env paths.Env) []Block {
	var out []Block
	for i, a := range list {
		if a.Scope.Global {
			continue
		}
		seen := map[string]bool{}
		for _, raw := range a.Scope.Paths {
			p := Normalize(raw, env)
			key := p
			if env.Windows {
				key = strings.ToLower(p)
			}
			if p == "" || seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, Block{Index: i, Path: p})
		}
	}
	slices.SortStableFunc(out, func(a, b Block) int {
		ea, eb := env.Expand(a.Path), env.Expand(b.Path)
		return cmp.Or(
			cmp.Compare(segments(ea), segments(eb)),
			cmp.Compare(len(ea), len(eb)),
			cmp.Compare(a.Index, b.Index),
			strings.Compare(a.Path, b.Path),
		)
	})
	return out
}

func segments(p string) int {
	n := 0
	for _, s := range strings.Split(p, "/") {
		if s != "" {
			n++
		}
	}
	return n
}

// Keyword is the includeIf condition for the platform: gitdir/i on Windows, whose file system ignores case.
func Keyword(env paths.Env) string {
	if env.Windows {
		return "gitdir/i:"
	}
	return "gitdir:"
}

// resolve expands p and follows symbolic links where the path exists, so that /var and /private/var on macOS
// compare equal. The result is in slash form without a trailing slash.
func resolve(p string, env paths.Env) string {
	abs := env.Expand(p)
	if real, err := filepath.EvalSymlinks(filepath.FromSlash(abs)); err == nil {
		abs = paths.Slash(real)
	}
	if len(abs) > 1 {
		abs = strings.TrimSuffix(abs, "/")
	}
	return abs
}

// Matches reports whether git applies a gitdir path in the repository whose work tree root is repoRoot.
func Matches(repoRoot, stored string, env paths.Env) bool {
	stored = Normalize(stored, env)
	if stored == "" {
		return false
	}
	pattern := resolve(stored, env)
	if !strings.HasSuffix(pattern, "/") {
		pattern += "/"
	}
	return env.HasPrefix(resolve(repoRoot, env)+"/.git", pattern)
}

// Matching returns the blocks that apply in repoRoot, in write order; the last one is what git uses.
func Matching(blocks []Block, repoRoot string, env paths.Env) []Block {
	var out []Block
	for _, b := range blocks {
		if Matches(repoRoot, b.Path, env) {
			out = append(out, b)
		}
	}
	return out
}
