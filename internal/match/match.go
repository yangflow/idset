// Package match finds the identity git uses in a folder: the last matching gitdir block in write order, or the
// global identity when no block matches.
package match

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/gitdir"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/sysexec"
)

// Kind is the outcome of resolving a folder.
type Kind int

const (
	Matched Kind = iota
	NoMatch
	NotRepo
	NotFolder
)

// Cover is an identity whose gitdir path covers a folder, with the path that does.
type Cover struct{ ID, Path string }

// Result describes a dropped folder. Root is the work tree root and Origin the remote URL when there is one;
// ID is the identity git uses when Kind is Matched. Covering lists the identities whose gitdir paths cover Root
// in the order git reads them, the last being ID, and Exact says whether ID has a path that is Root itself, so
// that the folder can be unbound from it. Stale is set only for NoMatch: a gitdir path that is gone and that the
// folder probably replaced. All three are computed here because they read the disk.
type Result struct {
	Kind     Kind
	Dir      string
	Root     string
	Origin   string
	ID       string
	Covering []Cover
	Exact    bool
	Stale    *Stale
}

// Git runs git with args in dir and returns its trimmed standard output and exit code.
type Git func(ctx context.Context, dir string, args ...string) (string, int, error)

// SystemGit runs the git on PATH, each call limited to timeout.
func SystemGit(timeout time.Duration) Git {
	return func(ctx context.Context, dir string, args ...string) (string, int, error) {
		res, err := sysexec.Run(ctx, timeout, nil, "git", append([]string{"-C", dir}, args...)...)
		return strings.TrimSpace(res.Stdout), res.ExitCode, err
	}
}

// Resolve looks at dir and decides which identity applies. The error is for git itself failing to run.
func Resolve(ctx context.Context, git Git, dir string, list []account.Account, env paths.Env) (Result, error) {
	r := Result{Dir: paths.Slash(dir)}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return r, err
		}
		r.Kind = NotFolder
		return r, nil
	}
	root, code, err := git(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return r, err
	}
	if code != 0 || root == "" {
		r.Kind = NotRepo
		return r, nil
	}
	r.Root = paths.Slash(root)
	if origin, code, err := git(ctx, dir, "config", "--get", "remote.origin.url"); err == nil && code == 0 {
		r.Origin = origin
	}
	if r.Origin == "" {
		// Without an origin remote, ls-remote --get-url echoes the name back.
		if url, code, err := git(ctx, dir, "ls-remote", "--get-url", "origin"); err == nil && code == 0 && url != "origin" {
			r.Origin = url
		}
	}
	r.Kind, r.ID = Identity(list, r.Root, env)
	r.Covering = Covering(list, r.Root, env)
	if r.Kind == NoMatch {
		r.Stale = FindStale(list, r.Root, env, Gone, Canon)
	}
	for _, a := range list {
		if a.ID == r.ID && !a.Scope.Global {
			r.Exact = slices.ContainsFunc(a.Scope.Paths, func(p string) bool { return gitdir.Exact(r.Root, p, env) })
		}
	}
	return r, nil
}

// Identity is the pure part of Resolve: the identity git uses in the work tree at root.
func Identity(list []account.Account, root string, env paths.Env) (Kind, string) {
	if m := gitdir.Matching(gitdir.Blocks(list, env), root, env); len(m) > 0 {
		return Matched, list[m[len(m)-1].Index].ID
	}
	// With several global identities (only from a hand-edited file) the generated [user] is the last one's.
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].Scope.Global {
			return Matched, list[i].ID
		}
	}
	return NoMatch, ""
}

// Covering lists the identities whose gitdir paths cover the work tree at root, in the order git reads them; the
// last one is what git uses. An identity with several paths covering the folder appears once, with the one that
// is read last. A global identity covers nothing by path and is never listed.
func Covering(list []account.Account, root string, env paths.Env) []Cover {
	matching := gitdir.Matching(gitdir.Blocks(list, env), root, env)
	var out []Cover
	for i := len(matching) - 1; i >= 0; i-- {
		id := list[matching[i].Index].ID
		if !slices.ContainsFunc(out, func(c Cover) bool { return c.ID == id }) {
			out = append(out, Cover{ID: id, Path: matching[i].Path})
		}
	}
	slices.Reverse(out)
	return out
}
