package panel

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/bind"
	"github.com/yangflow/idset/internal/match"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/store"
)

const (
	// undoFor is how long a bind can be undone, counted from an absolute deadline: the panel closing and opening
	// again does not add to it.
	undoFor = 5 * time.Second
	// undoSteps is how many steps the countdown line on the undo bar shortens in.
	undoSteps = 5
	undoneFor = 1200 * time.Millisecond
	// updatedFor is how long "Updated" shows after a stale path was changed.
	updatedFor = 1200 * time.Millisecond
)

// undoBar is what Undo puts back, and until when.
type undoBar struct {
	before []account.Account
	who    string
	until  time.Time
}

// binding is the state of the result card's bind actions. It is cleared with the result.
type binding struct {
	// busy is set while the owner works on a request.
	busy bool
	// menuOpen is the rebind menu; confirmTo is the global identity whose rebind waits for confirmation.
	menuOpen  bool
	confirmTo string
	// note says that a bind changed nothing; failure says why a request failed.
	note, failure string
	undo          *undoBar
	undoneUntil   time.Time
	updatedUntil  time.Time
	// staleKept hides the stale path prompt of the result on the card.
	staleKept bool
	// say is a sentence for screen readers, said once by the next frame.
	say string
}

// run hands a request for the folder on the card to the owner.
func (s *Screen) run(op bind.Op) {
	if s.busy || s.result == nil || s.readErr != nil {
		return
	}
	op.Dir, op.Root = s.lastDir, s.result.Root
	s.binding = binding{busy: true, staleKept: s.staleKept}
	s.act.Run(op)
}

// bindTo binds the folder to an identity, from the list of a card without a match.
func (s *Screen) bindTo(id string) { s.run(bind.Op{Kind: bind.KindBind, To: id}) }

// rebindTo moves the folder to another identity; a global one first asks for confirmation.
func (s *Screen) rebindTo(id string) {
	s.menuOpen = false
	if a, ok := s.find(id); ok && a.Scope.Global {
		s.confirmTo = id
		return
	}
	s.run(bind.Op{Kind: bind.KindRebind, From: s.result.ID, To: id})
}

func (s *Screen) confirmRebind() {
	if s.confirmTo != "" {
		s.run(bind.Op{Kind: bind.KindRebind, From: s.result.ID, To: s.confirmTo})
	}
}

func (s *Screen) unbind() { s.run(bind.Op{Kind: bind.KindUnbind, From: s.result.ID}) }

func (s *Screen) useOnly() { s.run(bind.Op{Kind: bind.KindUseOnly, To: s.result.ID}) }

// stale is the identity whose gone path the folder on the card probably replaced, while the prompt shows.
func (s *Screen) stale() (account.Account, bool) {
	if s.staleKept || s.result == nil || s.result.Kind != match.NoMatch || s.result.Stale == nil {
		return account.Account{}, false
	}
	return s.find(s.result.Stale.ID)
}

// fixStale changes the stale path to the folder on the card.
func (s *Screen) fixStale() {
	if _, ok := s.stale(); ok {
		s.run(bind.Op{Kind: bind.KindFixPath, From: s.result.Stale.ID, Path: s.result.Stale.Path})
	}
}

// keepStale hides the prompt for this result and changes nothing else.
func (s *Screen) keepStale() {
	s.staleKept, s.failure = true, ""
}

func (s *Screen) removePath(id, path string) {
	s.run(bind.Op{Kind: bind.KindRemovePath, From: id, Path: path})
}

// Undo puts the identities of the last bind back, while the undo bar is still on.
func (s *Screen) Undo() {
	if u := s.undo; u != nil && s.undoLeft() > 0 {
		s.run(bind.Op{Kind: bind.KindUndo, Before: u.before})
	}
}

// undoLeft is how many steps of the countdown are left, 0 when there is nothing to undo or the time is up.
func (s *Screen) undoLeft() int {
	if s.undo == nil {
		return 0
	}
	left := s.undo.until.Sub(s.now())
	if left <= 0 {
		return 0
	}
	return min(undoSteps, int((left+undoFor/undoSteps-1)/(undoFor/undoSteps)))
}

// Recheck looks at the folder again when the card shows it, for an identity saved to apply there.
func (s *Screen) Recheck(folder string) {
	if s.result != nil && s.lastDir != "" && s.env.Equal(paths.Slash(s.result.Root), paths.Slash(folder)) {
		s.Drop([]string{s.lastDir})
	}
}

// OpDone shows the outcome of a request. A request for a folder that is no longer on the card only ends the wait.
func (s *Screen) OpDone(o bind.Outcome) {
	s.busy = false
	if o.Op.Dir != s.lastDir || s.result == nil {
		return
	}
	var regenerate *store.RegenerateError
	switch {
	case errors.As(o.Err, &regenerate):
		s.failure = fmt.Sprintf(s.text.BindRegenFailed, cause(regenerate.Err))
	case o.Err != nil:
		s.failure = fmt.Sprintf(s.text.BindFailed, cause(o.Err))
		s.say = s.failure
		return
	}
	s.menuOpen, s.confirmTo, s.note, s.undo = false, "", "", nil
	if o.ResultErr != nil {
		s.result, s.resultErr = nil, o.ResultErr
		return
	}
	s.result = &o.Result
	switch {
	case o.CoveredBy != "":
		if a, ok := s.find(o.CoveredID); ok {
			s.note = fmt.Sprintf(s.text.Covered, s.label(a), o.CoveredBy)
			s.say = s.note
		}
	case o.Op.Kind == bind.KindUndo:
		s.undoneUntil = s.now().Add(undoneFor)
		s.say = s.text.Undone
	case o.Op.Kind == bind.KindFixPath && o.Result.Kind == match.Matched:
		s.updatedUntil = s.now().Add(updatedFor)
		s.say = s.text.Updated
	case o.Undo != nil:
		if a, ok := s.find(o.Op.To); ok {
			s.undo = &undoBar{before: o.Undo, who: s.label(a), until: s.now().Add(undoFor)}
			s.say = fmt.Sprintf(s.text.BoundTo, s.label(a))
		}
	default:
		s.say = s.resultMessage()
	}
	if s.failure != "" {
		s.say = s.failure
	}
}

// cause is the innermost error: the message without the paths the layers above it add.
func cause(err error) string {
	for {
		inner := errors.Unwrap(err)
		if inner == nil {
			return err.Error()
		}
		err = inner
	}
}

// others are the identities the folder can be rebound to: every identity but the one in use here.
func (s *Screen) others(r *match.Result) []account.Account {
	return slices.DeleteFunc(slices.Clone(s.accounts), func(a account.Account) bool { return a.ID == r.ID })
}
