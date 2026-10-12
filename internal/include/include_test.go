package include

import (
	"errors"
	"testing"
)

var (
	ssh = SSHBlock("~/.config/idset/ssh_config.managed")
	git = GitBlock("~/.config/idset/gitconfig.managed")
)

func TestBlocks(t *testing.T) {
	if ssh != "# --- idset managed ---\nInclude \"~/.config/idset/ssh_config.managed\"\n# --- idset managed end ---\n" {
		t.Errorf("ssh block %q", ssh)
	}
	if git != "# --- idset managed ---\n[include]\n\tpath = ~/.config/idset/gitconfig.managed\n# --- idset managed end ---\n" {
		t.Errorf("git block %q", git)
	}
}

func TestEnsureTop(t *testing.T) {
	for in, want := range map[string]string{
		"":                 ssh,
		"Host a\n":         ssh + "\nHost a\n",
		"\nHost a\n":       ssh + "\nHost a\n",
		ssh + "\nHost a\n": ssh + "\nHost a\n",
		"# --- idset managed ---\nInclude old\n# --- idset managed end ---\n\nHost a\n": ssh + "\nHost a\n",
	} {
		got, _, err := EnsureTop(in, ssh)
		if err != nil || got != want {
			t.Errorf("EnsureTop(%q) = %q, %v", in, got, err)
		}
	}
	if _, changed, _ := EnsureTop(ssh+"\nHost a\n", ssh); changed {
		t.Error("a current block must not be rewritten")
	}
}

func TestEnsureBottom(t *testing.T) {
	for in, want := range map[string]string{
		"":                     git,
		"[user]\n\tname = A\n": "[user]\n\tname = A\n\n" + git,
		"[user]\n\tname = A":   "[user]\n\tname = A\n\n" + git,
		"[core]\n\n" + git:     "[core]\n\n" + git,
	} {
		got, _, err := EnsureBottom(in, git)
		if err != nil || got != want {
			t.Errorf("EnsureBottom(%q) = %q, %v", in, got, err)
		}
	}
}

func TestUnterminatedBlockIsLeftAlone(t *testing.T) {
	in := "# --- idset managed ---\nInclude x\nHost a\n"
	for _, f := range []func(string) (string, bool, error){
		func(c string) (string, bool, error) { return EnsureTop(c, ssh) },
		func(c string) (string, bool, error) { return EnsureBottom(c, git) },
		Strip,
	} {
		out, changed, err := f(in)
		if !errors.Is(err, ErrUnterminated) || changed || out != in {
			t.Errorf("got %q, %v, %v", out, changed, err)
		}
	}
}

func TestStripRestoresTheOriginal(t *testing.T) {
	for _, orig := range []string{"", "Host a\n\tUser git\n", "[user]\n\tname = A\n", "# only a comment\n"} {
		top, _, _ := EnsureTop(orig, ssh)
		if got, _, _ := Strip(top); got != orig {
			t.Errorf("top: %q -> %q", orig, got)
		}
		bottom, _, _ := EnsureBottom(orig, git)
		if got, _, _ := Strip(bottom); got != orig {
			t.Errorf("bottom: %q -> %q", orig, got)
		}
	}
	if out, changed, _ := Strip("Host a\n"); changed || out != "Host a\n" {
		t.Error("a file without a block must not change")
	}
}
