package match

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/paths"
)

func person(id, email, key string) account.Account {
	return account.Account{ID: id, GitUserEmail: email, KeyPath: key, Scope: account.Scope{Global: true}}
}

func gitConfig(email string, emailCode int, command string, commandCode int) Git {
	return func(_ context.Context, _ string, args ...string) (string, int, error) {
		switch args[len(args)-1] {
		case "user.email":
			return email, emailCode, nil
		case "core.sshCommand":
			return command, commandCode, nil
		}
		return "", 1, nil
	}
}

func TestCheckFindings(t *testing.T) {
	env := paths.Env{Home: "/home/ada"}
	work := person("work", "bob@example.com", "~/.ssh/id_work")
	client := person("client", "carol@example.com", "~/.ssh/id_client")
	list := []account.Account{work, client}
	for _, c := range []struct {
		name  string
		who   account.Account
		email string
		eCode int
		cmd   string
		cCode int
		want  []Finding
	}{
		{"all agree", work, "bob@example.com", 0, "ssh -i ~/.ssh/id_work", 0, nil},
		{"nothing set", work, "bob@example.com", 0, "", 1, nil},
		{"no email", work, "", 1, "", 1, []Finding{{Kind: NoEmail}}},
		{"empty email", work, "  ", 0, "", 1, []Finding{{Kind: NoEmail}}},
		{"another identity's email", work, "carol@example.com", 0, "", 1, []Finding{{Kind: OtherEmail, Value: "carol@example.com", Owner: "client"}}},
		{"a stranger's email", work, "someone@example.org", 0, "", 1, []Finding{{Kind: UnknownEmail, Value: "someone@example.org"}}},
		{"case and white space", work, "  Bob@Example.COM \n", 0, "", 1, nil},
		{"another key", work, "bob@example.com", 0, "ssh -i ~/.ssh/id_other", 0, []Finding{{Kind: OtherKey, Value: "~/.ssh/id_other"}}},
		{"email first, then key", work, "carol@example.com", 0, "ssh -i ~/.ssh/id_client -F /dev/null", 0,
			[]Finding{{Kind: OtherEmail, Value: "carol@example.com", Owner: "client"}, {Kind: OtherKey, Value: "~/.ssh/id_client"}}},
		{"home spelled out", work, "bob@example.com", 0, "ssh -i /home/ada/.ssh/id_work", 0, nil},
		{"no key option", work, "bob@example.com", 0, "ssh -F /dev/null", 0, nil},
		{"identity without email", person("w", "", "~/.ssh/id_work"), "anyone@example.org", 0, "", 1, nil},
		{"identity without key", person("w", "bob@example.com", ""), "bob@example.com", 0, "ssh -i ~/.ssh/id_other", 0, nil},
		{"git cannot say the email", work, "", 128, "", 1, nil},
		{"git cannot say the command", work, "bob@example.com", 0, "ssh -i ~/.ssh/id_other", 128, nil},
	} {
		got := Check(context.Background(), gitConfig(c.email, c.eCode, c.cmd, c.cCode), "/home/ada/work/api", c.who, list, env)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestCheckSurvivesGitFailing(t *testing.T) {
	failing := func(context.Context, string, ...string) (string, int, error) { return "", -1, errors.New("timed out") }
	work := person("work", "bob@example.com", "~/.ssh/id_work")
	if got := Check(context.Background(), failing, "/home/ada/work", work, []account.Account{work}, paths.Env{Home: "/home/ada"}); got != nil {
		t.Errorf("a failing git gives no findings: %+v", got)
	}
}

func TestEmailIsAskedOfTheRepository(t *testing.T) {
	var asked [][]string
	git := func(_ context.Context, dir string, args ...string) (string, int, error) {
		asked = append(asked, append([]string{dir}, args...))
		return "", 1, nil
	}
	work := person("work", "bob@example.com", "~/.ssh/id_work")
	Check(context.Background(), git, "/r", work, nil, paths.Env{})
	want := [][]string{{"/r", "config", "--get", "user.email"}, {"/r", "config", "--get", "core.sshCommand"}}
	if !reflect.DeepEqual(asked, want) {
		t.Errorf("git calls %v, want %v", asked, want)
	}
}

func TestSeverity(t *testing.T) {
	for kind, want := range map[FindingKind]Severity{NoEmail: Warning, UnknownEmail: Warning, OtherEmail: Error, OtherKey: Error} {
		if got := (Finding{Kind: kind}).Severity(); got != want {
			t.Errorf("kind %d: %d, want %d", kind, got, want)
		}
	}
	if Worst(nil) != 0 || Worst([]Finding{{Kind: NoEmail}}) != Warning || Worst([]Finding{{Kind: NoEmail}, {Kind: OtherKey}}) != Error {
		t.Error("the worst severity wins")
	}
}

func TestIdentityFiles(t *testing.T) {
	for _, c := range []struct {
		cmd     string
		windows bool
		want    []string
	}{
		{"ssh -i ~/.ssh/id_work", false, []string{"~/.ssh/id_work"}},
		{"ssh -i~/.ssh/id_work", false, []string{"~/.ssh/id_work"}},
		{`ssh -i "/home/ada/my keys/id_work" -o IdentitiesOnly=yes`, false, []string{"/home/ada/my keys/id_work"}},
		{`ssh -i '/home/ada/my keys/id_work'`, false, []string{"/home/ada/my keys/id_work"}},
		{`ssh -i /home/ada/my\ keys/id_work`, false, []string{"/home/ada/my keys/id_work"}},
		{`ssh -i "/home/ada/a\"b/id"`, false, []string{`/home/ada/a"b/id`}},
		{`ssh -i '/home/ada/a\b/id'`, false, []string{`/home/ada/a\b/id`}},
		{"ssh -o IdentityFile=~/.ssh/id_work", false, []string{"~/.ssh/id_work"}},
		{"ssh -oIdentityFile=~/.ssh/id_work", false, []string{"~/.ssh/id_work"}},
		{`ssh -o "IdentityFile ~/.ssh/id_work"`, false, []string{"~/.ssh/id_work"}},
		{`ssh -o 'identityfile="~/.ssh/id work"'`, false, []string{"~/.ssh/id work"}},
		{"ssh -i ~/.ssh/a -i ~/.ssh/b", false, []string{"~/.ssh/a", "~/.ssh/b"}},
		{"ssh -o StrictHostKeyChecking=no", false, nil},
		{"ssh -i", false, nil},
		{`ssh -i "C:\Users\ada\.ssh\id_work"`, true, []string{`C:\Users\ada\.ssh\id_work`}},
		{`ssh -i C:\Users\ada\.ssh\id_work`, true, []string{`C:\Users\ada\.ssh\id_work`}},
		{`ssh -i C:/Users/ada/.ssh/id_work`, true, []string{`C:/Users/ada/.ssh/id_work`}},
		{`ssh -i "C:\Program Files\keys\id" -v`, true, []string{`C:\Program Files\keys\id`}},
	} {
		if got := IdentityFiles(c.cmd, c.windows); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %q, want %q", c.cmd, got, c.want)
		}
	}
}

func TestKeyPathsCompareAsText(t *testing.T) {
	posix := paths.Env{Home: "/home/ada"}
	win := paths.Env{Home: `C:\Users\ada`, Windows: true}
	for _, c := range []struct {
		env  paths.Env
		a, b string
		same bool
	}{
		{posix, "~/.ssh/id_work", "/home/ada/.ssh/id_work", true},
		{posix, "~/.ssh/./id_work", "~/.ssh/id_work", true},
		{posix, "~/.ssh//id_work", "~/.ssh/id_work", true},
		{posix, "~/.ssh/ID_WORK", "~/.ssh/id_work", false},
		{posix, "~/.ssh/id_work", "~/.ssh/id_work.pub", false},
		{win, `C:\Users\ada\.ssh\id_work`, "~/.ssh/id_work", true},
		{win, "c:/users/ADA/.ssh/ID_WORK", `C:\Users\ada\.ssh\id_work`, true},
		{win, "D:/keys/id", "C:/keys/id", false},
	} {
		if got := samePath(c.a, c.b, c.env); got != c.same {
			t.Errorf("%s vs %s: %v, want %v", c.a, c.b, got, c.same)
		}
	}
}

func TestWindowsKeyInTheCommand(t *testing.T) {
	env := paths.Env{Home: `C:\Users\ada`, Windows: true}
	work := person("work", "bob@example.com", "~/.ssh/id_work")
	git := gitConfig("bob@example.com", 0, `ssh -i "C:\Users\ada\.ssh\id_other"`, 0)
	got := Check(context.Background(), git, "C:/work/api", work, []account.Account{work}, env)
	if want := []Finding{{Kind: OtherKey, Value: `C:\Users\ada\.ssh\id_other`}}; !reflect.DeepEqual(got, want) {
		t.Errorf("%+v, want %+v", got, want)
	}
	git = gitConfig("bob@example.com", 0, `ssh -i C:\Users\ada\.ssh\id_work`, 0)
	if got := Check(context.Background(), git, "C:/work/api", work, []account.Account{work}, env); got != nil {
		t.Errorf("the same key: %+v", got)
	}
}
