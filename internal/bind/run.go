package bind

import (
	"context"
	"errors"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/match"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/store"
)

// Kind is what a request does.
type Kind int

const (
	// KindBind adds the folder to To.
	KindBind Kind = iota
	// KindUnbind removes the folder from From.
	KindUnbind
	// KindRebind moves the folder from From to To.
	KindRebind
	// KindUseOnly adds the folder's own path to To, the identity git uses there.
	KindUseOnly
	// KindRemovePath removes Path from From.
	KindRemovePath
	// KindUndo puts Before back.
	KindUndo
)

// Op is a request. Dir is the folder as it was dropped and Root its work tree root; both are looked at again
// afterwards.
type Op struct {
	Kind     Kind
	Dir      string
	Root     string
	From, To string
	Path     string
	// Before is what Undo restores.
	Before []account.Account
}

// Outcome is what happened. A *store.RegenerateError in Err means the change stands and only the configuration
// was not rewritten; any other Err means nothing was written and the folder was not looked at again.
type Outcome struct {
	Op  Op
	Err error
	// CoveredID and CoveredBy are set when nothing changed because that identity's path covers the folder.
	CoveredID, CoveredBy string
	// Undo is the previous state of the identities the change touched, to give to an Op of KindUndo; nil when there is
	// nothing to undo.
	Undo []account.Account
	// Result and ResultErr are the folder looked at again, as match.Resolve answers.
	Result    match.Result
	ResultErr error
}

// Written reports whether the change reached accounts.json.
func (o Outcome) Written() bool {
	var regenerate *store.RegenerateError
	return o.Err == nil || errors.As(o.Err, &regenerate)
}

// Runner does the requests against the store. Run blocks on disk and git, so callers run it off the UI thread.
type Runner struct {
	Store *store.Store
	Env   paths.Env
	Git   match.Git
	Now   func() time.Time
}

// Run plans the request on the stored list, writes it with one store write, and looks at the folder again.
func (r Runner) Run(ctx context.Context, op Op) Outcome {
	out := Outcome{Op: op}
	if err := r.Store.ReadOnly(); err != nil {
		out.Err = err
		return out
	}
	list := r.Store.Accounts()
	now := r.Now()
	var plan Plan
	var err error
	switch op.Kind {
	case KindBind:
		plan, err = Bind(list, op.To, op.Root, r.Env, now)
	case KindUnbind:
		plan, err = Unbind(list, op.From, op.Root, r.Env, now)
	case KindRebind:
		plan, err = Rebind(list, op.From, op.To, op.Root, r.Env, now)
	case KindUseOnly:
		plan, err = UseOnly(list, op.To, op.Root, r.Env, now)
	case KindRemovePath:
		plan, err = RemovePath(list, op.From, op.Path, r.Env, now)
	case KindUndo:
		err = r.Store.RestoreMany(op.Before)
	}
	if op.Kind != KindUndo && err == nil && !plan.Empty() {
		err = r.Store.UpdateMany(plan.Changes)
	}
	out.Err = err
	if !out.Written() {
		return out
	}
	out.CoveredID, out.CoveredBy = plan.CoveredID, plan.CoveredBy
	if op.Kind == KindBind || op.Kind == KindRebind || op.Kind == KindUseOnly {
		out.Undo = plan.Before
	}
	out.Result, out.ResultErr = match.Resolve(ctx, r.Git, op.Dir, r.Store.Accounts(), r.Env)
	return out
}
