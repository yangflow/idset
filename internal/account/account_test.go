package account

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)

func work() Account {
	used := time.Date(2026, 8, 20, 9, 30, 0, 0, time.UTC)
	return Account{
		ID: "22222222-2222-2222-2222-222222222222", Label: "work", Provider: GitHub, SSHPort: 443,
		SSHAlias: "github-work", Username: "bob", KeyPath: "~/.ssh/id_work", KeyFingerprint: "SHA256:AbCdEf",
		GitUserName: "Bob", GitUserEmail: "bob@example.com",
		Scope:       Scope{Paths: []string{"~/work/", "~/src/new-app/"}},
		URLRewrites: []Rewrite{{From: "https://github.com/example-org/", To: "git@github-work:example-org/"}},
		Color:       "#3A7BD5", CreatedAt: t0, UpdatedAt: t0.Add(19*24*time.Hour - time.Hour), LastUsedAt: &used,
	}
}

func TestEncodeSortsKeysAndOmitsEmptyOptionals(t *testing.T) {
	a := work()
	a.KeyFingerprint, a.LastUsedAt = "", nil
	a.CreatedAt = a.CreatedAt.Add(400 * time.Millisecond).In(time.FixedZone("x", 3600))
	data, err := Encode([]Account{a})
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "accounts": [
    {
      "color": "#3A7BD5",
      "createdAt": "2026-08-01T10:00:00Z",
      "gitUserEmail": "bob@example.com",
      "gitUserName": "Bob",
      "id": "22222222-2222-2222-2222-222222222222",
      "keyPath": "~/.ssh/id_work",
      "label": "work",
      "notes": "",
      "provider": "github",
      "scope": {
        "gitdir": {
          "paths": [
            "~/work/",
            "~/src/new-app/"
          ]
        }
      },
      "sshAlias": "github-work",
      "sshPort": 443,
      "updatedAt": "2026-08-20T09:00:00Z",
      "urlRewrites": [
        {
          "from": "https://github.com/example-org/",
          "push": false,
          "to": "git@github-work:example-org/"
        }
      ],
      "username": "bob"
    }
  ],
  "version": 1
}
`
	if string(data) != want {
		t.Fatalf("encoded:\n%s", data)
	}
}

func TestAccountKeysAreSorted(t *testing.T) {
	typ := reflect.TypeFor[Account]()
	var keys []string
	for i := range typ.NumField() {
		keys = append(keys, strings.Split(typ.Field(i).Tag.Get("json"), ",")[0])
	}
	if !slices.IsSorted(keys) {
		t.Fatalf("fields are not in key order: %v", keys)
	}
}

func TestRoundTrip(t *testing.T) {
	global := New(t0)
	global.Label = "personal"
	empty := work()
	empty.ID = NewID()
	empty.Scope = Scope{Paths: []string{}}
	in := []Account{work(), global, empty}
	data, err := Encode(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("round trip changed the list:\n%+v\n%+v", in, out)
	}
	if !strings.Contains(string(data), `"global": {}`) || !strings.Contains(string(data), `"paths": []`) {
		t.Fatalf("scope encoding:\n%s", data)
	}
}

func TestDecodeDefaults(t *testing.T) {
	list, err := Decode([]byte(`{"version":1,"accounts":[
		{"id":"11111111-1111-1111-1111-111111111111","scope":{"global":{}}},
		{"id":"33333333-3333-3333-3333-333333333333","provider":"custom","hostName":"git.example.com","scope":{"gitdir":{"paths":["~/corp/"]}}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if list[0].Provider != GitHub || list[0].SSHPort != 443 || list[0].URLRewrites == nil {
		t.Errorf("defaults: %+v", list[0])
	}
	if list[1].SSHPort != 22 {
		t.Errorf("custom port default: %d", list[1].SSHPort)
	}
}

func TestDecodeRejectsDamage(t *testing.T) {
	id := `"id":"11111111-1111-1111-1111-111111111111"`
	for name, body := range map[string]string{
		"not json":       `{"version":1,`,
		"no version":     `{"accounts":[]}`,
		"no list":        `{"version":1}`,
		"no id":          `{"version":1,"accounts":[{"scope":{"global":{}}}]}`,
		"no scope":       `{"version":1,"accounts":[{` + id + `}]}`,
		"both scopes":    `{"version":1,"accounts":[{` + id + `,"scope":{"global":{},"gitdir":{"paths":[]}}}]}`,
		"unknown vendor": `{"version":1,"accounts":[{` + id + `,"provider":"bitbucket","scope":{"global":{}}}]}`,
	} {
		if _, err := Decode([]byte(body)); err == nil {
			t.Errorf("%s: decoded", name)
		}
	}
	if _, err := Decode([]byte(`{"version":2,"accounts":[]}`)); !errors.Is(err, ErrNewerVersion) {
		t.Errorf("newer version: %v", err)
	}
}

func TestHosts(t *testing.T) {
	for _, c := range []struct {
		p          Provider
		port       int
		host, keys string
	}{
		{GitHub, 443, "ssh.github.com", "github.com"},
		{GitHub, 22, "github.com", ""},
		{GitLab, 443, "altssh.gitlab.com", "gitlab.com"},
		{GitLab, 22, "gitlab.com", ""},
		{Gitea, 22, "gitea.com", ""},
	} {
		a := Account{Provider: c.p, SSHPort: c.port}
		if a.RealHost() != c.host || a.HostKeyAlias() != c.keys {
			t.Errorf("%s:%d = %s, %s", c.p, c.port, a.RealHost(), a.HostKeyAlias())
		}
	}
	custom := Account{Provider: Custom, HostName: "git.example.com", SSHPort: 443}
	if custom.RealHost() != "git.example.com" || custom.HostKeyAlias() != "" {
		t.Errorf("custom: %s, %s", custom.RealHost(), custom.HostKeyAlias())
	}
	if DefaultPort(Gitea) != 22 || DefaultPort(GitHub) != 443 || DefaultPort(Custom) != 22 {
		t.Error("default ports")
	}
}

func TestNewID(t *testing.T) {
	a, b := NewID(), NewID()
	if !validID(a) || a == b || a[14] != '4' {
		t.Fatalf("ids %s %s", a, b)
	}
}

func codes(p Problems) []string {
	var out []string
	for _, q := range p {
		out = append(out, q.Field+":"+string(q.Code))
	}
	return out
}

func TestValidate(t *testing.T) {
	ok := work()
	if p := Validate([]Account{ok}); len(p) != 0 {
		t.Fatalf("valid account: %v", p)
	}
	mod := func(f func(*Account)) Account { a := work(); f(&a); return a }
	for name, c := range map[string]struct {
		a    Account
		want string
	}{
		"alias chars":    {mod(func(a *Account) { a.SSHAlias = "work alias" }), "sshAlias:bad-alias"},
		"alias reserved": {mod(func(a *Account) { a.SSHAlias = "GitHub.com" }), "sshAlias:reserved-alias"},
		"gitea 443":      {mod(func(a *Account) { a.Provider = Gitea }), "sshPort:bad-port"},
		"github 2222":    {mod(func(a *Account) { a.SSHPort = 2222 }), "sshPort:bad-port"},
		"custom no host": {mod(func(a *Account) { a.Provider = Custom }), "hostName:missing-host"},
		"custom bad":     {mod(func(a *Account) { a.Provider, a.HostName = Custom, "git example" }), "hostName:bad-host"},
		"newline":        {mod(func(a *Account) { a.GitUserEmail = "a@example.com\n[core]" }), "gitUserEmail:new-line"},
		"quote in key":   {mod(func(a *Account) { a.KeyPath = `~/.ssh/"x` }), "keyPath:quote"},
		"newline path":   {mod(func(a *Account) { a.Scope.Paths = []string{"~/a\n"} }), "scope:new-line"},
		"rewrite":        {mod(func(a *Account) { a.URLRewrites[0].To = "x\ny" }), "urlRewrites:new-line"},
		"color":          {mod(func(a *Account) { a.Color = "teal" }), "color:bad-color"},
		"id":             {mod(func(a *Account) { a.ID = "x" }), "id:bad-id"},
	} {
		if got := codes(Validate([]Account{c.a})); !slices.Equal(got, []string{c.want}) {
			t.Errorf("%s: %v, want %s", name, got, c.want)
		}
	}
	if p := Validate([]Account{mod(func(a *Account) { a.Notes = "line\nline" })}); len(p) != 0 {
		t.Errorf("notes may span lines: %v", p)
	}
	custom := mod(func(a *Account) { a.Provider, a.HostName, a.SSHPort = Custom, "git.example.com", 2222 })
	if p := Validate([]Account{custom}); len(p) != 0 {
		t.Errorf("custom port: %v", p)
	}
}

func TestValidateAcrossTheList(t *testing.T) {
	a, b := work(), work()
	b.ID = NewID()
	b.SSHAlias = "GITHUB-WORK"
	p := Validate([]Account{a, b})
	if len(p) != 1 || p[0].ID != b.ID || p[0].Code != DuplicateAlias || p[0].Other != "work" {
		t.Errorf("duplicate alias: %+v", p)
	}
	b.SSHAlias = "corp"
	b.ID = a.ID
	if got := codes(Validate([]Account{a, b})); !slices.Equal(got, []string{"id:duplicate-id"}) {
		t.Errorf("duplicate id: %v", got)
	}
	g1, g2 := New(t0), New(t0)
	g1.Label = "personal"
	p = Validate([]Account{g1, a, g2})
	if len(p) != 1 || p[0].ID != g2.ID || p[0].Code != SecondGlobal || p[0].Other != "personal" {
		t.Errorf("second global: %+v", p)
	}
	corp := work()
	corp.ID, corp.SSHAlias, corp.Provider, corp.HostName, corp.SSHPort = NewID(), "corp", Custom, "git.example.com", 22
	a.SSHAlias = "git.example.com"
	if got := codes(Validate([]Account{a, corp})); !slices.Equal(got, []string{"sshAlias:reserved-alias"}) {
		t.Errorf("alias shadowing a custom host: %v", got)
	}
}
