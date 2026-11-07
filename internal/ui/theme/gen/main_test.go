package main

import (
	"bytes"
	"os"
	"testing"
)

// TestGenerated fails when tokens_gen.go is not what design/tokens.json makes: run go generate in the theme package.
func TestGenerated(t *testing.T) {
	src, err := os.ReadFile("../../../../design/tokens.json")
	if err != nil {
		t.Fatal(err)
	}
	want, err := generate(src)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile("../tokens_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	// A Windows checkout may turn the file's line endings into CRLF; the generator writes LF.
	if !bytes.Equal(bytes.ReplaceAll(got, []byte("\r\n"), []byte("\n")), want) {
		t.Fatal("tokens_gen.go is stale: run go generate ./internal/ui/theme")
	}
}

func TestIdent(t *testing.T) {
	for name, want := range map[string]string{"space-0_5": "Space0_5", "badge-s": "BadgeS", "radius-pill": "RadiusPill", "accounts-min-w": "AccountsMinW"} {
		if got := ident(name); got != want {
			t.Errorf("ident(%q) = %q, want %q", name, got, want)
		}
	}
}
