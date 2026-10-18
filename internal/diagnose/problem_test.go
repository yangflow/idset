package diagnose

import (
	"reflect"
	"testing"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/include"
	"github.com/yangflow/idset/internal/match"
)

func TestProblemOf(t *testing.T) {
	a := identity("work", account.GitHub)
	rewrite := a
	rewrite.URLRewrites = []account.Rewrite{{From: "https://github.com/", To: "git@gh-work:", Push: true}}
	present := KeyState{HasPath: true, Exists: true, Fingerprint: fingerprint}
	locked := KeyState{HasPath: true, Exists: true, Agent: true, Passphrase: true, Fingerprint: fingerprint}
	const origin = "https://github.com/ada/tools.git"
	for _, c := range []struct {
		name    string
		a       account.Account
		check   Check
		matched bool
		origin  string
		want    Problem
	}{
		{"healthy", a, Check{OK: true}, false, origin, 0},
		{"https remote of the match", a, Check{OK: true}, true, origin, HTTPSRemote},
		{"https remote, rewritten", rewrite, Check{OK: true}, true, "HTTPS://GitHub.com/ada/tools.git", 0},
		{"ssh remote", a, Check{OK: true}, true, "git@gh-work:ada/tools.git", 0},
		{"no ssh", a, Check{Failure: NoCommand}, false, "", MissingCommand},
		{"missing key", a, Check{Failure: NoKeyFile, Key: KeyState{HasPath: true}}, false, "", MissingKey},
		{"locked", a, Check{Failure: Rejected, Key: locked}, false, "", LockedKey},
		{"locked, unknown fingerprint", a,
			Check{Failure: Rejected, Key: KeyState{HasPath: true, Exists: true, Agent: true, Passphrase: true}}, false, "",
			Denied},
		{"in agent", a, Check{Failure: Rejected, Key: KeyState{HasPath: true, Exists: true, Agent: true,
			InAgent: true, Passphrase: true, Fingerprint: fingerprint}}, false, "", Denied},
		{"no host", a, Check{Failure: NoHost, Key: present}, false, "", Unreachable},
		{"refused", a, Check{Failure: Refused, Key: present}, false, "", Unreachable},
		{"timed out", a, Check{Failure: TimedOut, Key: present}, false, "", Unreachable},
		{"host key", a, Check{Failure: HostKey, Key: present}, false, "", Unreachable},
		{"denied", a, Check{Failure: Rejected, Key: present}, false, "", Denied},
		{"denied without key path", a, Check{Failure: Rejected}, false, "", Denied},
		{"other", a, Check{Failure: Other, Detail: "kex_exchange_identification", Key: present}, false, "", Denied},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, ok := ProblemOf(c.a, c.check, c.matched, c.origin)
			if got != c.want || ok != (c.want != 0) {
				t.Errorf("got %v %v, want %v", got, ok, c.want)
			}
		})
	}
	for _, p := range []Problem{MissingCommand, MissingKey, LockedKey, Unreachable, Denied} {
		if !p.Retryable() {
			t.Errorf("%v has no check again", p)
		}
	}
	if HTTPSRemote.Retryable() {
		t.Error("an https remote has check again")
	}
}

func TestSummarize(t *testing.T) {
	work, client, home := identity("work", account.GitHub), identity("client", account.GitLab), identity("home", account.GitHub)
	work.Scope = account.Scope{Paths: []string{"~/work/"}}
	client.Scope = account.Scope{Global: true}
	home.Scope = account.Scope{Global: true}
	home.SSHAlias = ""
	list := []account.Account{work, client, home}
	got := Summarize(Input{
		List:      list,
		Checks:    []Check{{ID: "work", Failure: Rejected}, {ID: "client", OK: true}, {ID: "gone", Failure: NoHost}},
		UserHosts: []string{"GH-CLIENT", "github.com"},
		Matched:   "work",
		Findings:  []match.Finding{{Kind: match.NoEmail}},
	})
	want := []Entry{
		{Code: NET001, Severity: match.Error, ID: "work", Alias: "gh-work"},
		{Code: GIT002, Severity: match.Error, ID: "home"},
		{Code: GIT001, Severity: match.Warning, ID: "work", Regenerate: true},
		{Code: SSH001, Severity: match.Warning, ID: "client", Alias: "gh-client"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
	if Worst(got) != match.Error {
		t.Errorf("worst %v", Worst(got))
	}

	global := Summarize(Input{List: []account.Account{client}, Matched: "client",
		Findings: []match.Finding{{Kind: match.OtherKey, Value: "~/.ssh/id_other"}}})
	if want := []Entry{{Code: GIT001, Severity: match.Error, ID: "client"}}; !reflect.DeepEqual(global, want) {
		t.Errorf("global match %+v, want %+v", global, want)
	}
	if healthy := Summarize(Input{List: list, Checks: []Check{{ID: "work", OK: true}}}); len(healthy) != 1 ||
		healthy[0].Code != GIT002 {
		t.Errorf("healthy apart from two globals %+v", healthy)
	}
	if Worst(nil) != 0 {
		t.Error("an empty summary has a severity")
	}
}

func TestUserHosts(t *testing.T) {
	config := include.Begin + "\nInclude ~/.idset/ssh_config.managed\nHost gh-managed\n" + include.End + `
# Host gh-commented
Host gh-work "gh-quoted"
  HostName github.com
host=gh-equals
Host *.example.com !gh-negated gh-?
Match host gh-match
	Host	gh-tab
`
	want := []string{"gh-work", "gh-quoted", "gh-equals", "gh-tab"}
	if got := UserHosts(config); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := UserHosts(""); got != nil {
		t.Errorf("empty config %q", got)
	}
}
