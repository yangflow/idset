package about

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/ui/locale"
)

// The window's content is 320 x 292 DIPs, under the system title bar.
const width, height = 320, 292

func open(t *testing.T, lang string, u Updates) (*ui.Tester, *int) {
	t.Helper()
	checks := 0
	s := New(Text(lang), "0.1.0", Actions{CheckForUpdates: func() { checks++ }})
	s.SetUpdates(u)
	tt := ui.NewTester(s.View, width, height)
	tt.Frame()
	return tt, &checks
}

func TestCopy(t *testing.T) {
	for _, lang := range locale.Languages {
		v := reflect.ValueOf(translations[lang])
		for i := range v.NumField() {
			if v.Field(i).String() == "" {
				t.Errorf("%s: %s is empty", lang, v.Type().Field(i).Name)
			}
		}
	}
}

func TestShowsTheVersionAndLicense(t *testing.T) {
	for lang, want := range map[string][]string{
		"en":      {"Idset", "Version 0.1.0", "© 2026 yangflow", "Released under the MIT License", "GitHub", "Check for Updates…"},
		"zh-Hans": {"Idset", "版本 0.1.0", "© 2026 yangflow", "以 MIT 许可发布", "GitHub", "检查更新…"},
	} {
		tt, _ := open(t, lang, Available)
		for _, text := range want {
			if !tt.HasText(text) {
				t.Errorf("%s: no %q in %q", lang, text, tt.Texts())
			}
		}
	}
}

func TestGitHubOpensTheRepository(t *testing.T) {
	tt, _ := open(t, "en", Unavailable)
	if err := tt.Click("GitHub"); err != nil {
		t.Fatal(err)
	}
	if got := tt.OpenedURLs(); !slices.Equal(got, []string{Repository}) {
		t.Errorf("opened %q", got)
	}
}

func TestCheckForUpdates(t *testing.T) {
	for _, c := range []struct {
		updates Updates
		checks  int
		note    bool
	}{{Unknown, 0, false}, {Available, 1, false}, {Unavailable, 0, true}} {
		tt, checks := open(t, "en", c.updates)
		_ = tt.Click("Check for Updates…")
		if *checks != c.checks {
			t.Errorf("%d: %d checks, want %d", c.updates, *checks, c.checks)
		}
		if got := tt.HasText("This copy cannot update itself"); got != c.note {
			t.Errorf("%d: note %v, want %v", c.updates, got, c.note)
		}
	}
}

// Everything fits the window, in both languages, also with the note.
func TestFits(t *testing.T) {
	for _, lang := range locale.Languages {
		tt, _ := open(t, lang, Unavailable)
		text := Text(lang)
		check, ok := tt.Find(text.Check)
		note, ok2 := tt.Find(text.Cannot)
		version, ok3 := tt.Find(fmt.Sprintf(text.Version, "0.1.0"))
		if !ok || !ok2 || !ok3 {
			t.Fatalf("%s: %q", lang, tt.Texts())
		}
		if note.Y+note.H > height || note.X < 0 || note.X+note.W > width || check.Y <= version.Y {
			t.Errorf("%s: check %v, note %v in %d x %d", lang, check, note, width, height)
		}
	}
}
