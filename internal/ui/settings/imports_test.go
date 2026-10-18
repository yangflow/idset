package settings

import (
	"errors"
	"fmt"
	"os/exec"
	"testing"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/importer"
	"github.com/yangflow/idset/internal/store"
)

// sampleCandidates are the candidates of the mockup: three new, one without an email left unchecked by the
// test that needs it, and one Idset already has.
func sampleCandidates() []importer.Candidate {
	cand := func(label, alias, email string) importer.Candidate {
		return importer.Candidate{Account: account.Account{Label: label, SSHAlias: alias, GitUserEmail: email,
			KeyPath: "~/.ssh/id_" + label, Provider: account.GitHub, Scope: account.Scope{Global: true}}}
	}
	exists := cand("gh-idset", "gh-idset", "bob@example.org")
	exists.Exists = true
	return []importer.Candidate{
		cand("gh-work", "gh-work", "bob@example.com"),
		cand("github.com", "", "bob@example.net"),
		cand("client", "client", "bob@client.example"),
		cand("corp-git", "corp-git", ""),
		exists,
	}
}

// importOpen opens the Import pane and reports the candidates of the scan it starts.
func importOpen(t *testing.T, lang string, cands []importer.Candidate) *rig {
	t.Helper()
	r := open(t, lang, Request{Pane: Import})
	if r.rec.scans != 1 {
		t.Fatalf("scans %d on entering", r.rec.scans)
	}
	r.want(t, r.s.text.Scanning)
	r.s.ScanDone(cands, nil)
	r.frame()
	return r
}

// stored is what the store makes of an imported candidate.
func stored(a account.Account, color string) account.Account {
	a.ID, a.Color = a.Label+"-id", color
	return a
}

func TestImportScansOnEveryEntry(t *testing.T) {
	r := importOpen(t, "en", nil)
	r.want(t, "No identities found in your SSH and Git configuration")
	r.click(t, "Scan Again")
	if r.rec.scans != 2 {
		t.Fatalf("scans %d after Scan Again", r.rec.scans)
	}
	r.s.ScanDone(nil, nil)
	r.click(t, "General")
	r.click(t, "Import")
	if r.rec.scans != 3 {
		t.Fatalf("scans %d after entering again", r.rec.scans)
	}
}

func TestImportChecksTheNewOnes(t *testing.T) {
	r := importOpen(t, "en", sampleCandidates())
	r.want(t, "5 identities found", "gh-work · bob@example.com", "No alias · bob@example.net", "corp-git", "Already added",
		"These files are only read", "Import 4 Identities")
	r.wantNot(t, "Imported as scoped")
	r.click(t, "corp-git")
	r.want(t, "Import 3 Identities")
	r.click(t, "Import 3 Identities")
	if len(r.rec.imports) != 1 {
		t.Fatalf("imports %d", len(r.rec.imports))
	}
	var labels []string
	for _, a := range r.rec.imports[0] {
		labels = append(labels, a.Label)
	}
	if fmt.Sprint(labels) != "[gh-work github.com client]" {
		t.Fatalf("imported %v", labels)
	}
	r.want(t, "Importing…")
	r.click(t, "Import 3 Identities")
	r.click(t, "Scan Again")
	if len(r.rec.imports) != 1 || r.rec.scans != 1 {
		t.Fatalf("imports %d, scans %d while importing", len(r.rec.imports), r.rec.scans)
	}

	var added []account.Account
	for i, a := range r.rec.imports[0] {
		added = append(added, stored(a, account.ColorPresets[i]))
	}
	r.s.ImportDone(added, nil)
	r.frame()
	r.want(t, "3 identities imported", "bob@client.example", "accounts.json was backed up")
	r.click(t, "Open Identities")
	if fmt.Sprint(r.rec.opened) != "[gh-work-id]" {
		t.Fatalf("opened %v", r.rec.opened)
	}
}

func TestImportNothingChecked(t *testing.T) {
	cands := sampleCandidates()[4:]
	r := importOpen(t, "en", cands)
	r.want(t, "1 identity found", "Import 0 Identities")
	r.click(t, "Import 0 Identities")
	if len(r.rec.imports) != 0 {
		t.Fatal("imported with nothing checked")
	}
	r.click(t, "gh-idset")
	r.want(t, "Import 1 Identity")
}

func TestImportMarksTheScoped(t *testing.T) {
	cands := sampleCandidates()
	cands[1].Limited, cands[2].Limited = true, true
	r := importOpen(t, "en", cands)
	r.want(t, "Imported as scoped", "There is already a global identity")
	r.click(t, "github.com")
	r.click(t, "client")
	r.wantNot(t, "Imported as scoped", "There is already a global identity")
}

func TestImportStopsAtTheFirstFailure(t *testing.T) {
	r := importOpen(t, "en", sampleCandidates())
	r.click(t, "Import 4 Identities")
	sent := r.rec.imports[0]
	r.s.ImportDone([]account.Account{stored(sent[0], "blue"), stored(sent[1], "green")}, errors.New("accounts.json is a symbolic link"))
	r.frame()
	r.want(t, "Could not import client: accounts.json is a symbolic link. The 2 before it were imported", "Import 2 Identities")
	r.wantNot(t, "These files are only read")
	r.click(t, "Import 2 Identities")
	var labels []string
	for _, a := range r.rec.imports[1] {
		labels = append(labels, a.Label)
	}
	if fmt.Sprint(labels) != "[client corp-git]" {
		t.Fatalf("imported again %v", labels)
	}
	r.s.ImportDone(nil, errors.New("disk full"))
	r.frame()
	r.want(t, "Could not import client: disk full. Nothing was imported.")
}

func TestImportSavedButNotRegenerated(t *testing.T) {
	r := importOpen(t, "en", sampleCandidates()[:1])
	r.click(t, "Import 1 Identity")
	a := stored(r.rec.imports[0][0], "blue")
	r.s.ImportDone([]account.Account{a}, &store.RegenerateError{Err: errors.New("permission denied")})
	r.frame()
	r.want(t, "1 identity imported", "The identities were imported, but the configuration could not be written: permission denied",
		"Open Identities")
	r.wantNot(t, "accounts.json was backed up")
}

func TestImportScanFailures(t *testing.T) {
	r := importOpen(t, "en", nil)
	r.s.scan()
	r.s.ScanDone(nil, fmt.Errorf("git: %w", exec.ErrNotFound))
	r.frame()
	r.want(t, "git command not found")
	r.s.scan()
	r.s.ScanDone(nil, errors.New("reading ~/.ssh/config: permission denied"))
	r.frame()
	r.want(t, "Could not read your SSH and Git configuration: ")
	r.click(t, "Scan Again")
	if r.rec.scans != 4 {
		t.Fatalf("scans %d", r.rec.scans)
	}
}

func TestImportReadOnly(t *testing.T) {
	r := open(t, "en", Request{})
	r.s.SetReadOnly(true)
	r.s.Open(Request{Pane: Import})
	r.frame()
	r.s.ScanDone(sampleCandidates(), nil)
	r.frame()
	r.want(t, "accounts.json cannot be read, so nothing can be imported")
	r.click(t, "Import 4 Identities")
	if len(r.rec.imports) != 0 {
		t.Fatal("imported while read-only")
	}
}
