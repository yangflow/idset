package accounts

import (
	"errors"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/keyfile"
	"github.com/yangflow/idset/internal/store"
)

// askDelete opens the confirmation for the selected identity and answers the key inspection.
func (r *rig) askDelete(t *testing.T, reason keyfile.Reason, other string) {
	t.Helper()
	r.click(t, r.s.text.Delete)
	r.tt.Frame()
	r.s.SetKeyReason(r.s.Selected(), reason, other)
	r.tt.Frame()
}

// clickDialogDelete clicks the Delete button of the dialog, which sits right of Cancel; the footer has a Delete too.
func (r *rig) clickDialogDelete(t *testing.T) {
	t.Helper()
	box, ok := r.tt.Find("Cancel")
	if !ok {
		t.Fatal("no dialog")
	}
	r.tt.ClickAt(box.X+box.W+50, box.Y+box.H/2)
}

func TestDeleteConfirmationExplainsWhatStays(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: workID})
	r.click(t, "Delete")
	if !slices.Equal(r.rec.inspected, []string{workID}) {
		t.Fatalf("the key must be inspected: %v", r.rec.inspected)
	}
	r.want(t, "Delete identity work?", "Private key", "~/.ssh/id_ed25519_work", "Folders", "~/work/", "~/src/new/", "~/clients/acme/",
		"Project folders stay as they are.", "Also delete the private key", "Cancel")
	r.wantNot(t, "Only the private key is deleted")
	// Until the key is inspected the box and Delete wait.
	r.click(t, "Also delete the private key")
	if r.s.confirm.alsoKey {
		t.Error("the box is not available while the key is inspected")
	}
	r.s.SetKeyReason(workID, keyfile.None, "")
	r.tt.Frame()
	r.wantNot(t, "No key path", "uses the same key")
	r.click(t, "Also delete the private key")
	r.tt.Frame()
	r.want(t, "Deletes the private key; the .pub file stays.")
	if !r.s.confirm.alsoKey {
		t.Fatal("the box is checked")
	}
}

func TestDeleteWithAndWithoutTheKey(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: workID})
	r.askDelete(t, keyfile.None, "")
	r.click(t, "Cancel")
	r.tt.Frame()
	r.wantNot(t, "Delete identity work?")
	if len(r.rec.deletes) != 0 || r.s.confirm != nil {
		t.Fatal("cancel deletes nothing")
	}
	// Escape cancels too.
	r.askDelete(t, keyfile.None, "")
	r.tt.Key(0, ui.KeyEscape)
	r.tt.Frame()
	r.wantNot(t, "Delete identity work?")
	if len(r.rec.deletes) != 0 {
		t.Fatal("escape deletes nothing")
	}
	// Delete without the key: the box is unchecked by default.
	r.askDelete(t, keyfile.None, "")
	r.clickDialogDelete(t)
	if !slices.Equal(r.rec.deletes, []deleteCall{{workID, false}}) {
		t.Fatalf("deletes: %v", r.rec.deletes)
	}
}

func TestDeleteButtonsInTheDialog(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: workID})
	r.askDelete(t, keyfile.None, "")
	if r.s.confirm.alsoKey {
		t.Fatal("the box is unchecked by default")
	}
	r.s.confirmDelete()
	if !slices.Equal(r.rec.deletes, []deleteCall{{workID, false}}) {
		t.Fatalf("delete without the key: %v", r.rec.deletes)
	}
	r.tt.Frame()
	// While deleting, the dialog stays, even for Escape.
	r.tt.Key(0, ui.KeyEscape)
	r.tt.Frame()
	r.want(t, "Delete identity work?")
	r.s.busy = false
	r.s.confirm.alsoKey = true
	r.s.confirmDelete()
	if got := r.rec.deletes[len(r.rec.deletes)-1]; got != (deleteCall{workID, true}) {
		t.Fatalf("delete with the key: %+v", got)
	}
}

func TestDeleteCheckboxIsDisabledForEveryReason(t *testing.T) {
	cases := []struct {
		name   string
		id     string
		reason keyfile.Reason
		other  string
		text   string
	}{
		{"no key path", miscID, keyfile.NoPath, "", "This identity has no key path."},
		{"shared", workID, keyfile.Shared, "personal", "Another identity (personal) uses the same key."},
		{"symlink", workID, keyfile.Symlink, "", "The key path is a symbolic link."},
		{"missing", workID, keyfile.Missing, "", "The key file does not exist."},
		{"not a regular file", workID, keyfile.NotFile, "", "The key path is not a regular file."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := open(t, "en", sample(), Request{ID: c.id})
			r.askDelete(t, c.reason, c.other)
			r.want(t, "Also delete the private key", c.text)
			r.wantNot(t, "Only the private key is deleted")
			r.click(t, "Also delete the private key")
			if r.s.confirm.alsoKey {
				t.Fatal("the box cannot be checked")
			}
			// Even a box that was checked earlier cannot carry the key into the deletion.
			r.s.confirm.alsoKey = true
			r.s.confirmDelete()
			if len(r.rec.deletes) != 1 || r.rec.deletes[0].withKey {
				t.Fatalf("deletes: %v", r.rec.deletes)
			}
		})
	}
	// A key that became unusable after it was checked unchecks the box.
	r := open(t, "en", sample(), Request{ID: workID})
	r.askDelete(t, keyfile.None, "")
	r.click(t, "Also delete the private key")
	r.s.SetKeyReason(workID, keyfile.Missing, "")
	if r.s.confirm.alsoKey {
		t.Error("a later reason unchecks the box")
	}
	r.s.SetKeyReason("other", keyfile.Shared, "")
}

func TestNoKeyShowsNone(t *testing.T) {
	r := open(t, "en", sample(), Request{ID: miscID})
	r.askDelete(t, keyfile.NoPath, "")
	r.want(t, "Delete identity (unnamed)?", "none", "~/misc/")
}

func TestDeleteDone(t *testing.T) {
	remaining := func() []account.Account { return sample()[1:] }
	r := open(t, "en", sample(), Request{ID: workID})
	r.askDelete(t, keyfile.None, "")
	r.s.confirmDelete()
	r.s.DeleteDone(workID, remaining(), DeleteResult{})
	r.tt.Frame()
	r.wantNot(t, "Delete identity work?", "could not")
	if r.s.Selected() != personalID || r.s.confirm != nil || r.s.busy {
		t.Fatalf("selected %q", r.s.Selected())
	}
	if _, ok := r.s.forms[workID]; ok {
		t.Error("the form of a deleted identity goes")
	}

	// The configuration could not be rewritten: the identity is gone all the same.
	r = open(t, "en", sample(), Request{ID: workID})
	r.askDelete(t, keyfile.None, "")
	r.s.confirmDelete()
	r.s.DeleteDone(workID, remaining(), DeleteResult{Err: &store.RegenerateError{Err: errors.New("permission denied")}})
	r.tt.Frame()
	r.want(t, "Saved, but the generated configuration could not be rewritten: permission denied.")

	// The key could not be removed: reported, without undoing the deletion.
	r = open(t, "en", sample(), Request{ID: workID})
	r.askDelete(t, keyfile.None, "")
	r.s.confirmDelete()
	r.s.DeleteDone(workID, remaining(), DeleteResult{KeyErr: errors.New("permission denied")})
	r.tt.Frame()
	r.want(t, "The identity is deleted, but its private key is not: permission denied.")
	if r.s.stored(workID) != nil {
		t.Error("the deletion stands")
	}

	// The identity could not be deleted: the dialog stays and says why.
	r = open(t, "en", sample(), Request{ID: workID})
	r.askDelete(t, keyfile.None, "")
	r.s.confirmDelete()
	r.s.DeleteDone(workID, sample(), DeleteResult{Err: &store.ReadOnlyError{Err: errors.New("unexpected end of JSON input")}})
	r.tt.Frame()
	r.want(t, "Delete identity work?", "Could not delete the identity: unexpected end of JSON input")
	if r.s.busy || r.s.stored(workID) == nil {
		t.Error("nothing was deleted")
	}

	// The last identity: the empty state returns.
	r = open(t, "en", sample()[:1], Request{ID: workID})
	r.askDelete(t, keyfile.None, "")
	r.s.confirmDelete()
	r.s.DeleteDone(workID, []account.Account{}, DeleteResult{})
	r.tt.Frame()
	r.want(t, "No identities yet")
}
