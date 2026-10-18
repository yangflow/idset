package importer

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/sysexec"
)

var env = paths.Env{Home: "/home/ada"}

func TestParseSSHRoundTrips(t *testing.T) {
	for _, text := range []string{
		"",
		"Host gh-work\n\tHostName github.com\n",
		"# comment\r\nHost a b\r\n  IdentityFile=\"~/.ssh/my key\"\r\n\r\n",
		"Host x\n    Port = 2222",
		"\n\n  \t\nInclude conf.d/*\n# --- idset managed ---\n",
	} {
		if got := Text(ParseSSH(text)); got != text {
			t.Errorf("Text(ParseSSH(%q)) = %q", text, got)
		}
	}
	lines := ParseSSH("  IdentityFile=\"~/.ssh/my key\"\r\nHost\ta  b\n# Port 1\nport= 22\n")
	want := []Line{
		{Raw: "  IdentityFile=\"~/.ssh/my key\"\r\n", Key: "IdentityFile", Args: []string{"~/.ssh/my key"}},
		{Raw: "Host\ta  b\n", Key: "Host", Args: []string{"a", "b"}},
		{Raw: "# Port 1\n"},
		{Raw: "port= 22\n", Key: "port", Args: []string{"22"}},
	}
	if !reflect.DeepEqual(lines, want) {
		t.Errorf("ParseSSH = %#v", lines)
	}
}

func TestHosts(t *testing.T) {
	main := ParseSSH(`IdentityFile ~/.ssh/default
Host gh-a gh-b *.corp !bad
	IdentityFile ~/.ssh/a
	IdentityFile ~/.ssh/second
	HOSTNAME github.com
	Include inc
	Port 443
Match host x
	IdentityFile ~/.ssh/matched
Host gh-c
	User git
`)
	inc := ParseSSH("Port 1\nHost gh-inc\n\tIdentityFile ~/.ssh/inc\n")
	got := Hosts(Config{Lines: main, Included: map[int][][]Line{5: {inc}}})
	a := map[string]string{"identityfile": "~/.ssh/a", "hostname": "github.com", "port": "443"}
	want := []Host{
		{Name: "gh-a", Options: a}, {Name: "gh-b", Options: a},
		{Name: "gh-inc", Options: map[string]string{"identityfile": "~/.ssh/inc"}},
		{Name: "gh-c", Options: map[string]string{"user": "git"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Hosts = %v", got)
	}
}

func TestIncludesOutsideMatch(t *testing.T) {
	got := Includes(ParseSSH("Include a b\nHost x\n\tInclude c\nMatch all\n\tInclude d\nHost y\nInclude e\n"))
	want := map[int][]string{0: {"a", "b"}, 2: {"c"}, 6: {"e"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Includes = %v", got)
	}
}

func TestParseGit(t *testing.T) {
	out := "file:/home/ada/.gitconfig\x00user.name\nAda Lovelace\x00file:/home/ada/.gitconfig\x00core.bare\x00" +
		"file:/home/ada/.gitconfig\x00url.git@gh-work:org.x.insteadof\nhttps://github.com/org/\x00"
	want := []Entry{
		{Origin: "/home/ada/.gitconfig", Key: "user.name", Value: "Ada Lovelace"},
		{Origin: "/home/ada/.gitconfig", Key: "core.bare"},
		{Origin: "/home/ada/.gitconfig", Key: "url.git@gh-work:org.x.insteadof", Value: "https://github.com/org/"},
	}
	if got := ParseGit(out, true); !reflect.DeepEqual(got, want) {
		t.Errorf("ParseGit with origin = %v", got)
	}
	if got := ParseGit("user.email\na@example.com\x00a.b\nmulti\nline\x00", false); !reflect.DeepEqual(got, []Entry{
		{Key: "user.email", Value: "a@example.com"}, {Key: "a.b", Value: "multi\nline"},
	}) {
		t.Errorf("ParseGit = %v", got)
	}
	if s, sub, n := split("url.git@gh-work:org.x.insteadof"); s != "url" || sub != "git@gh-work:org.x" || n != "insteadof" {
		t.Errorf("split = %q %q %q", s, sub, n)
	}
}

// fake is a file system and git: files by slash path, and the output of git by its arguments joined with spaces,
// with slash paths.
type fake struct {
	files map[string]string
	git   map[string]string
	fail  error
	runs  []string
}

func (f *fake) source() Source {
	return Source{
		Env: env, Git: "git",
		ReadFile: func(name string) ([]byte, error) {
			if s, ok := f.files[filepath.ToSlash(name)]; ok {
				return []byte(s), nil
			}
			return nil, fs.ErrNotExist
		},
		Glob: func(pattern string) ([]string, error) {
			var out []string
			for name := range f.files {
				if ok, _ := path.Match(filepath.ToSlash(pattern), name); ok {
					out = append(out, filepath.FromSlash(name))
				}
			}
			return out, nil
		},
		Run: func(_ context.Context, _ []string, name string, args ...string) (sysexec.Result, error) {
			cmd := name + " " + filepath.ToSlash(strings.Join(args, " "))
			f.runs = append(f.runs, cmd)
			if f.fail != nil {
				return sysexec.Result{}, f.fail
			}
			if out, ok := f.git[cmd]; ok {
				return sysexec.Result{Stdout: out}, nil
			}
			return sysexec.Result{ExitCode: 128}, nil
		},
	}
}

const globalList = "git config --global --list -z --show-origin"

func origin(file string, kv ...string) string {
	var b strings.Builder
	for i := 0; i < len(kv); i += 2 {
		b.WriteString("file:" + file + "\x00" + kv[i] + "\n" + kv[i+1] + "\x00")
	}
	return b.String()
}

func entries(kv ...string) string {
	var b strings.Builder
	for i := 0; i < len(kv); i += 2 {
		b.WriteString(kv[i] + "\n" + kv[i+1] + "\x00")
	}
	return b.String()
}

func scan(t *testing.T, f *fake, list []account.Account) []Candidate {
	t.Helper()
	got, err := Scan(context.Background(), f.source(), list)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestScan(t *testing.T) {
	f := &fake{
		files: map[string]string{
			"/home/ada/.ssh/config": `# --- idset managed ---
Include "~/.config/idset/ssh_config.managed"
# --- idset managed ---

Host github.com
	IdentityFile ~/.ssh/id_personal

Host gh-personal
	HostName github.com
	IdentityFile /home/ada/.ssh/id_personal

Include conf.d/*.conf

Match host *.example.com
	Include ignored

Host gh-work
	HostName ssh.github.com
	IdentityFile ~/.ssh/id_work
`,
			"/home/ada/.config/idset/ssh_config.managed": "Host gh-idset\n\tHostName github.com\n\tIdentityFile \"~/.ssh/id_idset\"\n",
			"/home/ada/.ssh/conf.d/b.conf":               "Host lab\n\tHostName gitlab.com\n\tPort 22\n\tIdentityFile %d/.ssh/id_lab\nInclude deeper\n",
			"/home/ada/.ssh/conf.d/a.conf":               "Host corp\n\tHostName git.example.com\n\tPort 2222\n\tUser git\n\tIdentityFile /keys/corp\n",
			"/home/ada/.ssh/conf.d/c.txt":                "Host skipped\n\tIdentityFile ~/.ssh/skipped\n",
			"/home/ada/.ssh/deeper":                      "Host deeper\n\tIdentityFile ~/.ssh/deeper\n",
			"/home/ada/.ssh/ignored":                     "Host ignored\n\tIdentityFile ~/.ssh/ignored\n",
		},
		git: map[string]string{
			globalList: origin("/home/ada/.gitconfig",
				"user.name", "Ada Lovelace", "user.email", "ada@example.com",
				"url.git@gh-personal:.insteadof", "https://github.com/",
				"includeif.gitdir:~/work/.path", ".gitconfig-work",
				"includeif.gitdir:~/oss/**.path", "~/.gitconfig-work",
				"url.git@gh-work:org/.pushinsteadof", "git@github.com:org/",
				"url.git@other:.insteadof", "https://example.com/"),
			"git config --file /home/ada/.gitconfig-work --list -z": entries(
				"user.name", "Ada at Work", "user.email", "ada@work.example.com",
				"core.sshcommand", `ssh -i "~/.ssh/id_work" -o IdentitiesOnly=yes`),
		},
	}
	list := []account.Account{{SSHAlias: "gh-idset", Provider: account.GitHub, KeyPath: "~/.ssh/id_idset"}}
	got := scan(t, f, list)
	want := []Candidate{
		{Account: account.Account{Label: "gh-idset", Provider: account.GitHub, SSHPort: 22, SSHAlias: "gh-idset",
			KeyPath: "~/.ssh/id_idset", Scope: account.Scope{Paths: []string{}}, URLRewrites: []account.Rewrite{}},
			Exists: true},
		{Account: account.Account{Label: "github.com", Provider: account.GitHub, SSHPort: 22, SSHAlias: "gh-personal",
			KeyPath: "~/.ssh/id_personal", GitUserName: "Ada Lovelace", GitUserEmail: "ada@example.com",
			Scope: account.Scope{Global: true}, URLRewrites: []account.Rewrite{{From: "https://github.com/", To: "git@gh-personal:"}}}},
		{Account: account.Account{Label: "corp", Provider: account.Custom, HostName: "git.example.com", SSHPort: 2222,
			SSHAlias: "corp", KeyPath: "/keys/corp", Scope: account.Scope{Paths: []string{}}, URLRewrites: []account.Rewrite{}}},
		{Account: account.Account{Label: "lab", Provider: account.GitLab, SSHPort: 22, SSHAlias: "lab",
			KeyPath: "~/.ssh/id_lab", Scope: account.Scope{Paths: []string{}}, URLRewrites: []account.Rewrite{}}},
		{Account: account.Account{Label: "gh-work", Provider: account.GitHub, SSHPort: 443, SSHAlias: "gh-work",
			KeyPath: "~/.ssh/id_work", GitUserName: "Ada at Work", GitUserEmail: "ada@work.example.com",
			Scope:       account.Scope{Paths: []string{"~/work/", "~/oss/"}},
			URLRewrites: []account.Rewrite{{From: "git@github.com:org/", To: "git@gh-work:org/", Push: true}}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Scan =\n%+v\nwant\n%+v", got, want)
	}
	if want := []string{globalList, "git config --file /home/ada/.gitconfig-work --list -z",
		"git config --file /home/ada/.gitconfig-work --list -z"}; !reflect.DeepEqual(f.runs, want) {
		t.Errorf("runs = %q", f.runs)
	}
}

func TestPlatformAndPort(t *testing.T) {
	for _, c := range []struct {
		host     string
		port     int
		provider account.Provider
		want     int
	}{
		{"github.com", 0, account.GitHub, 22},
		{"GitHub.com", 22, account.GitHub, 22},
		{"github.com", 443, account.GitHub, 443},
		{"ssh.github.com", 0, account.GitHub, 443},
		{"ssh.github.com", 443, account.GitHub, 443},
		{"altssh.gitlab.com", 0, account.GitLab, 443},
		{"gitlab.com", 443, account.GitLab, 443},
		{"gitea.com", 22, account.Gitea, 22},
		{"gitea.com", 443, account.Custom, 443},
		{"github.com", 2222, account.Custom, 2222},
		{"git.example.com", 0, account.Custom, 22},
		{"git.example.com", 443, account.Custom, 443},
	} {
		if p, port := platform(c.host, c.port); p != c.provider || port != c.want {
			t.Errorf("platform(%q, %d) = %s %d, want %s %d", c.host, c.port, p, port, c.provider, c.want)
		}
	}
}

func hostsFile(hosts ...string) *fake {
	return &fake{files: map[string]string{"/home/ada/.ssh/config": strings.Join(hosts, "")}, git: map[string]string{}}
}

func aliases(cs []Candidate) []string {
	var out []string
	for _, c := range cs {
		out = append(out, c.Account.SSHAlias)
	}
	return out
}

func TestOneGlobalIdentity(t *testing.T) {
	f := hostsFile(
		"Host github.com\n\tIdentityFile ~/.ssh/id_a\n",
		"Host gitlab.com\n\tIdentityFile ~/.ssh/id_b\n",
		"Host gh-c\n\tHostName github.com\n\tIdentityFile ~/.ssh/id_c\n",
	)
	f.git[globalList] = origin("/home/ada/.gitconfig", "user.email", "ada@example.com", "core.sshcommand", "ssh -i ~/.ssh/id_c")
	got := scan(t, f, nil)
	var scopes []bool
	var marks []bool
	for _, c := range got {
		scopes, marks = append(scopes, c.Account.Scope.Global), append(marks, c.Limited)
	}
	if !slices.Equal(scopes, []bool{true, false, false}) || !slices.Equal(marks, []bool{false, true, true}) {
		t.Errorf("global %v, limited %v", scopes, marks)
	}
	for _, c := range got[1:] {
		if c.Account.Scope.Paths == nil || len(c.Account.Scope.Paths) != 0 {
			t.Errorf("%s: scope %+v, want limited without paths", c.Account.Label, c.Account.Scope)
		}
	}

	got = scan(t, f, []account.Account{{SSHAlias: "gh-c", Scope: account.Scope{Global: true}}})
	if !got[0].Limited || !got[1].Limited || !got[2].Exists || got[2].Limited {
		t.Errorf("with a global identity: %+v", got)
	}
}

func TestMergeSameKey(t *testing.T) {
	f := hostsFile(
		"Host github.com\n\tIdentityFile ~/.ssh/id_a\n",
		"Host gh-a gh-b\n\tHostName github.com\n\tPort 443\n\tIdentityFile /home/ada/.ssh/id_a\n",
		"Host lab-a\n\tHostName gitlab.com\n\tIdentityFile ~/.ssh/id_a\n",
		"Host gh-x\n\tHostName github.com\n\tIdentityFile ~/.ssh/id_x\n",
		"Host gh-y\n\tHostName github.com\n\tIdentityFile ~/.ssh/id_x\n",
	)
	f.git[globalList] = origin("/home/ada/.gitconfig",
		"user.name", "Ada", "includeif.gitdir:~/x/.path", "~/.gx", "includeif.gitdir:~/y/.path", "~/.gy",
		"url.git@gh-b:.insteadof", "https://github.com/")
	f.git["git config --file /home/ada/.gx --list -z"] = entries("user.email", "x@example.com", "core.sshcommand", "ssh -i ~/.ssh/id_x")
	f.git["git config --file /home/ada/.gy --list -z"] = entries("user.email", "y@example.com", "core.sshcommand", "ssh -i ~/.ssh/id_x")
	got := scan(t, f, nil)
	if want := []string{"gh-a", "lab-a", "gh-x"}; !slices.Equal(aliases(got), want) {
		t.Fatalf("aliases %q, want %q", aliases(got), want)
	}
	a := got[0].Account
	if a.Label != "github.com" || a.SSHPort != 22 || !a.Scope.Global || a.GitUserName != "Ada" ||
		!reflect.DeepEqual(a.URLRewrites, []account.Rewrite{{From: "https://github.com/", To: "git@gh-b:"}}) {
		t.Errorf("merged = %+v", a)
	}
	if x := got[2].Account; x.GitUserEmail != "x@example.com" || !reflect.DeepEqual(x.Scope.Paths, []string{"~/x/"}) {
		t.Errorf("first signature wins: %+v", x)
	}
}

func TestSignatureFromIncludeIf(t *testing.T) {
	f := hostsFile(
		"Host gh-a\n\tHostName github.com\n\tIdentityFile ~/.ssh/id_a\n",
		"Host gh-b\n\tHostName github.com\n\tIdentityFile ~/.ssh/id_b\n",
		"Host gh-c\n\tHostName github.com\n\tIdentityFile ~/.ssh/id_c\n",
	)
	f.git[globalList] = origin("/home/ada/dotfiles/gitconfig",
		"includeif.gitdir/i:./src/.path", "a.inc",
		"includeif.onbranch:main.path", "/etc/b.inc",
		"includeif.gitdir:~/gone/.path", "~/missing")
	f.git["git config --file /home/ada/dotfiles/a.inc --list -z"] = entries("user.email", "a@example.com", "core.sshcommand", `ssh -o "IdentityFile=/home/ada/.ssh/id_a"`)
	f.git["git config --file /etc/b.inc --list -z"] = entries("user.email", "b@example.com", "core.sshcommand", "ssh -i~/.ssh/id_b")
	got := scan(t, f, nil)
	if a := got[0].Account; a.GitUserEmail != "a@example.com" || !reflect.DeepEqual(a.Scope.Paths, []string{"~/dotfiles/src/"}) {
		t.Errorf("gh-a = %+v", a)
	}
	if b := got[1].Account; b.GitUserEmail != "b@example.com" || b.Scope.Global || len(b.Scope.Paths) != 0 {
		t.Errorf("gh-b = %+v", b)
	}
	if c := got[2].Account; c.GitUserEmail != "" || c.Scope.Global || got[2].Limited {
		t.Errorf("gh-c = %+v", got[2])
	}
}

func TestLeftOut(t *testing.T) {
	f := hostsFile(
		"Host nokey\n\tHostName github.com\n",
		"Host none\n\tHostName github.com\n\tIdentityFile none\n",
		"Host server\n\tHostName 192.0.2.1\n\tUser ada\n\tIdentityFile ~/.ssh/id_server\n",
		"Host corp\n\tHostName git.example.com\n\tIdentityFile ~/.ssh/id_corp\n",
		"Host git.example.org\n\tUser git\n\tIdentityFile ~/.ssh/id_org\n",
		"Host corp\n\tHostName github.com\n\tIdentityFile ~/.ssh/id_dup\n",
	)
	f.git[globalList] = origin("/home/ada/.gitconfig", "url.ssh://git@corp/.insteadof", "https://git.example.com/")
	got := scan(t, f, nil)
	if want := []string{"corp", ""}; !slices.Equal(aliases(got), want) {
		t.Fatalf("aliases %q, want %q", aliases(got), want)
	}
	if org := got[1].Account; org.Label != "git.example.org" || org.HostName != "git.example.org" {
		t.Errorf("a host named after itself has no alias: %+v", org)
	}
	got = scan(t, f, []account.Account{{Provider: account.Custom, KeyPath: "/home/ada/.ssh/id_org"}})
	if !got[1].Exists {
		t.Errorf("a candidate without alias exists by its key: %+v", got[1])
	}
}

func TestNothingToRead(t *testing.T) {
	f := &fake{files: map[string]string{}, git: map[string]string{}}
	if got := scan(t, f, nil); len(got) != 0 {
		t.Errorf("Scan = %v", got)
	}
	f.fail = errors.New("git not found")
	if _, err := Scan(context.Background(), f.source(), nil); !errors.Is(err, f.fail) {
		t.Errorf("err = %v", err)
	}
	src := f.source()
	src.ReadFile = func(string) ([]byte, error) { return nil, fs.ErrPermission }
	if _, err := Scan(context.Background(), src, nil); !errors.Is(err, fs.ErrPermission) {
		t.Errorf("err = %v", err)
	}
}

func TestWindowsPaths(t *testing.T) {
	win := paths.Env{Home: `C:\Users\Ada`, Windows: true}
	if got := keyPath(`C:\Users\ada\.ssh\id_work`, win); got != "~/.ssh/id_work" {
		t.Errorf("keyPath = %q", got)
	}
	if got := includePath("conf.d/*", win); got != "C:/Users/Ada/.ssh/conf.d/*" {
		t.Errorf("includePath = %q", got)
	}
	if got := includePath(`D:\ssh\extra`, win); got != "D:/ssh/extra" {
		t.Errorf("includePath = %q", got)
	}
	g := Gitconfig{Included: []Identity{{Email: "a@example.com", Cond: "gitdir/i:C:/Users/Ada/Work/",
		SSHCommand: `ssh -i "C:\Users\Ada\.ssh\id_work"`}}}
	got := Recognize([]Host{{Name: "gh-work", Options: map[string]string{"hostname": "github.com", "identityfile": `~\.ssh\id_work`}}},
		g, nil, win)
	if a := got[0].Account; a.GitUserEmail != "a@example.com" || !reflect.DeepEqual(a.Scope.Paths, []string{"~/Work/"}) {
		t.Errorf("Recognize = %+v", a)
	}
}
