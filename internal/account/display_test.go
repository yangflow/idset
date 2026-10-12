package account

import (
	"slices"
	"testing"
	"time"
)

func labels(list []Account) []string {
	var out []string
	for _, a := range list {
		out = append(out, a.Label)
	}
	return out
}

func TestSorted(t *testing.T) {
	at := func(h int) *time.Time { v := t0.Add(time.Duration(h) * time.Hour); return &v }
	list := []Account{
		{ID: "1", Label: "zeta"},
		{ID: "2", Label: "", LastUsedAt: at(1)},
		{ID: "3", Label: "Alpha", CreatedAt: t0},
		{ID: "4", Label: "old", LastUsedAt: at(1)},
		{ID: "5", Label: "new", LastUsedAt: at(5)},
		{ID: "6", Label: " "},
		{ID: "7", Label: "alpha", CreatedAt: t0.Add(-time.Hour)},
	}
	got := labels(Sorted(list))
	want := []string{"new", "old", "", "alpha", "Alpha", "zeta", " "}
	if !slices.Equal(got, want) {
		t.Fatalf("%q, want %q", got, want)
	}
	if list[0].Label != "zeta" {
		t.Fatal("Sorted must not change its argument")
	}
}

func TestFilter(t *testing.T) {
	list := []Account{
		{Label: "Café", Provider: GitHub},
		{Label: "work", SSHAlias: "gitlab-work", Provider: GitLab},
		{Label: "home", GitUserEmail: "me@example.com", Provider: Gitea},
		{Label: "corp", Provider: Custom},
	}
	for q, want := range map[string][]string{
		"":         {"Café", "work", "home", "corp"},
		"cafe":     {"Café"},
		"CAFÉ":     {"Café"},
		"LAB-W":    {"work"},
		"example":  {"home"},
		"gitea":    {"home"},
		"  corp  ": {"corp"},
		"nothing":  nil,
	} {
		if got := labels(Filter(list, q, nil)); !slices.Equal(got, want) {
			t.Errorf("%q: %q, want %q", q, got, want)
		}
	}
	if got := labels(Filter(list, "o", []Provider{GitLab, Custom})); !slices.Equal(got, []string{"work", "corp"}) {
		t.Errorf("providers: %q", got)
	}
	if got := ProvidersInUse(list[1:3]); !slices.Equal(got, []Provider{GitLab, Gitea}) {
		t.Errorf("in use: %v", got)
	}
}

func TestRGB(t *testing.T) {
	for color, want := range map[string][3]uint8{
		"green": {0x2E, 0x9E, 0x5B}, "#102030": {0x10, 0x20, 0x30}, "teal": {0x3A, 0x7B, 0xD5},
	} {
		r, g, b := Account{Color: color}.RGB()
		if [3]uint8{r, g, b} != want {
			t.Errorf("%s: %v", color, [3]uint8{r, g, b})
		}
	}
}
