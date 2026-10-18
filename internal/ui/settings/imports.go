package settings

import (
	"fmt"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/importer"
	"github.com/yangflow/idset/internal/keygen"
)

// imports is the state of the Import pane: scanning, the candidates with their check boxes, importing, then the
// identities imported.
type imports struct {
	scanning bool
	scanned  bool
	scanErr  string

	cands   []importer.Candidate
	checked []bool

	busy bool
	// sent are the candidates being imported, by index, in order.
	sent  []int
	err   string
	added []account.Account
}

// scan reads the configuration again, unless a scan or an import is running.
func (s *Screen) scan() {
	if s.imports.scanning || s.imports.busy {
		return
	}
	s.imports = imports{scanning: true}
	s.act.Scan()
}

// ScanDone reports Scan: the candidates, those Idset does not have yet checked.
func (s *Screen) ScanDone(cands []importer.Candidate, err error) {
	s.imports.scanning, s.imports.scanned = false, true
	switch {
	case err == nil:
	case keygen.NotFound(err):
		s.imports.scanErr = s.text.ScanNoGit
		return
	default:
		s.imports.scanErr = fmt.Sprintf(s.text.ScanFail, cause(err))
		return
	}
	s.imports.cands = cands
	s.imports.checked = make([]bool, len(cands))
	for i, c := range cands {
		s.imports.checked[i] = !c.Exists
	}
}

// checkedCount is how many candidates are checked.
func (s *Screen) checkedCount() int {
	n := 0
	for _, on := range s.imports.checked {
		if on {
			n++
		}
	}
	return n
}

// limited reports whether a candidate that is checked comes in as limited, for the note that explains it.
func (s *Screen) limited() bool {
	for i, c := range s.imports.cands {
		if c.Limited && s.imports.checked[i] {
			return true
		}
	}
	return false
}

// importChecked adds the checked candidates in their order.
func (s *Screen) importChecked() {
	if s.imports.busy || s.imports.scanning || s.readOnly || s.checkedCount() == 0 {
		return
	}
	var list []account.Account
	s.imports.sent = nil
	for i, c := range s.imports.cands {
		if s.imports.checked[i] {
			list = append(list, c.Account)
			s.imports.sent = append(s.imports.sent, i)
		}
	}
	s.imports.busy, s.imports.err = true, ""
	s.act.Import(list)
}

// ImportDone reports Import: the identities added, as stored, in the order they were sent, and the error that
// stopped the rest. After an error the candidates stay, those added marked as existing and unchecked, so that
// importing again adds the rest. When every one was added and only regenerating failed, the list shows with the
// error.
func (s *Screen) ImportDone(added []account.Account, err error) {
	s.imports.busy = false
	if err == nil {
		s.imports.added = added
		s.announce = s.doneTitle(len(added))
		return
	}
	n := min(len(added), len(s.imports.sent))
	if n == len(s.imports.sent) {
		s.imports.added, s.imports.err = added, fmt.Sprintf(s.text.RegenFail, cause(err))
		return
	}
	for _, i := range s.imports.sent[:n] {
		s.imports.cands[i].Exists, s.imports.checked[i] = true, false
	}
	failed := s.imports.cands[s.imports.sent[n]].Account.Label
	if n == 0 {
		s.imports.err = fmt.Sprintf(s.text.ImportFailFirst, failed, cause(err))
		return
	}
	s.imports.err = fmt.Sprintf(s.text.ImportFail, failed, cause(err), n)
}

// openImported opens the accounts window on the first identity imported.
func (s *Screen) openImported() {
	if len(s.imports.added) > 0 {
		s.act.OpenAccounts(s.imports.added[0].ID)
	}
}

func (s *Screen) foundTitle(n int) string {
	if n == 1 {
		return fmt.Sprintf(s.text.FoundOne, n)
	}
	return fmt.Sprintf(s.text.Found, n)
}

func (s *Screen) importLabel(n int) string {
	if n == 1 {
		return fmt.Sprintf(s.text.ImportOne, n)
	}
	return fmt.Sprintf(s.text.ImportN, n)
}

func (s *Screen) doneTitle(n int) string {
	if n == 1 {
		return fmt.Sprintf(s.text.DoneOne, n)
	}
	return fmt.Sprintf(s.text.Done, n)
}
