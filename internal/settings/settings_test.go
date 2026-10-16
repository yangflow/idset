package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrefixesAreRemembered(t *testing.T) {
	dir := t.TempDir()
	s := Open(dir)
	if len(s.Prefixes()) != 0 || s.Err() != nil {
		t.Fatalf("a missing file is empty: %v %v", s.Prefixes(), s.Err())
	}
	if err := s.SetPrefix("work", "acme/"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetPrefix("home", "me/"); err != nil {
		t.Fatal(err)
	}
	again := Open(dir)
	if p := again.Prefixes(); p["work"] != "acme/" || p["home"] != "me/" || len(p) != 2 {
		t.Fatalf("prefixes %v", p)
	}
	if err := again.SetPrefix("work", ""); err != nil {
		t.Fatal(err)
	}
	if p := Open(dir).Prefixes(); len(p) != 1 || p["home"] != "me/" {
		t.Fatalf("forgotten: %v", p)
	}
	info, err := os.Stat(filepath.Join(dir, FileName))
	if err != nil || info.Mode().Perm() != 0o600 && filepath.Separator == '/' {
		t.Fatalf("mode %v %v", info, err)
	}
}

func TestUnknownKeysSurviveAndABrokenFileIsKept(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(`{"theme":"dark"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s := Open(dir)
	if err := s.SetPrefix("work", "acme/"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if want := "{\n  \"clonePrefixes\": {\n    \"work\": \"acme/\"\n  },\n  \"theme\": \"dark\"\n}\n"; string(data) != want {
		t.Fatalf("file %q", data)
	}
	if err := os.WriteFile(path, []byte(`{broken`), 0o600); err != nil {
		t.Fatal(err)
	}
	s = Open(dir)
	if s.Err() == nil || s.SetPrefix("work", "x/") == nil {
		t.Fatal("a file that cannot be read is not overwritten")
	}
	if data, _ := os.ReadFile(path); string(data) != "{broken" {
		t.Fatalf("file changed: %q", data)
	}
}

func TestLanguageIsStoredAndKeepsOtherKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(`{"theme":"dark"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s := Open(dir)
	if s.Language() != "" {
		t.Fatalf("language %q", s.Language())
	}
	if err := s.SetPrefix("work", "acme/"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetLanguage("zh-Hans"); err != nil {
		t.Fatal(err)
	}
	again := Open(dir)
	if again.Language() != "zh-Hans" || again.Prefixes()["work"] != "acme/" {
		t.Fatalf("language %q prefixes %v", again.Language(), again.Prefixes())
	}
	data, _ := os.ReadFile(path)
	if want := "{\n  \"clonePrefixes\": {\n    \"work\": \"acme/\"\n  },\n  \"language\": \"zh-Hans\",\n  \"theme\": \"dark\"\n}\n"; string(data) != want {
		t.Fatalf("file %q", data)
	}
	if err := again.SetLanguage(""); err != nil {
		t.Fatal(err)
	}
	if got := Open(dir); got.Language() != "" || got.Prefixes()["work"] != "acme/" {
		t.Fatalf("following the system again: %q %v", got.Language(), got.Prefixes())
	}
	data, _ = os.ReadFile(path)
	if want := "{\n  \"clonePrefixes\": {\n    \"work\": \"acme/\"\n  },\n  \"theme\": \"dark\"\n}\n"; string(data) != want {
		t.Fatalf("file %q", data)
	}
}

func TestLanguageNotWrittenWhenUnreadable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(`{broken`), 0o600); err != nil {
		t.Fatal(err)
	}
	s := Open(dir)
	if s.SetLanguage("en") == nil {
		t.Fatal("a file that cannot be read is not overwritten")
	}
	if data, _ := os.ReadFile(path); string(data) != "{broken" {
		t.Fatalf("file changed: %q", data)
	}
}

func TestLanguageOfWrongTypeReadsAsFollowSystem(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(`{"language":3}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := Open(dir).Language(); got != "" {
		t.Fatalf("language %q", got)
	}
}
