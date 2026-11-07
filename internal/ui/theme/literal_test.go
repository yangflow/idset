package theme

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// literal is a number written where a token belongs: a spacing step or size, a radius or an opacity. A token sum
// such as t.Space(2*ButtonL+Space2) starts with a number too, so it is matched only when a name does not follow.
var literal = regexp.MustCompile(`\.(Space|Radius|Opacity)\(\s*[0-9.]+\s*[),]`)

// TestNoLiterals keeps the screens and the windows on the tokens: every size, radius and opacity is a name from
// tokens_gen.go, so design/tokens.json stays the one source.
func TestNoLiterals(t *testing.T) {
	var files []string
	for _, pattern := range []string{"../*/*.go", "../../../*.go"} {
		m, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, m...)
	}
	if len(files) < 20 {
		t.Fatalf("found only %d files", len(files))
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			if m := literal.FindString(line); m != "" {
				t.Errorf("%s:%d: %s: use a token from tokens_gen.go", filepath.ToSlash(f), i+1, m)
			}
		}
	}
}

func TestLiteralPattern(t *testing.T) {
	for _, s := range []string{"t.Space(3)", "t.Space(1.5)", "e.Radius(14)", "row.Opacity(0.5)"} {
		if !literal.MatchString(s) {
			t.Errorf("%s is not caught", s)
		}
	}
	for _, s := range []string{"t.Space(Space3)", "t.Space(theme.Control)", "t.Space(2*theme.ButtonL+theme.Space2)", "e.Radius(t.Radius)"} {
		if literal.MatchString(s) {
			t.Errorf("%s is caught", s)
		}
	}
}
