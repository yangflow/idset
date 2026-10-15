package remote

import (
	"testing"

	"github.com/yangflow/idset/internal/account"
)

func id(p account.Provider, alias string, rewrites ...account.Rewrite) account.Account {
	return account.Account{Provider: p, SSHAlias: alias, SSHPort: 22, URLRewrites: rewrites, HostName: "git.example.com"}
}

func TestRemoteCommand(t *testing.T) {
	gh := id(account.GitHub, "github-work")
	for _, c := range []struct {
		name   string
		a      account.Account
		origin string
		want   string
	}{
		{"https", gh, "https://github.com/org/repo.git", "git remote set-url origin git@github-work:org/repo.git"},
		{"no suffix", gh, "https://github.com/org/repo", "git remote set-url origin git@github-work:org/repo.git"},
		{"trailing slash", gh, "https://github.com/org/repo/", "git remote set-url origin git@github-work:org/repo.git"},
		{"http", gh, "http://github.com/org/repo.git", "git remote set-url origin git@github-work:org/repo.git"},
		{"upper case scheme", gh, "HTTPS://github.com/org/repo.git", "git remote set-url origin git@github-work:org/repo.git"},
		{"user info", gh, "https://bob@github.com/org/repo.git", "git remote set-url origin git@github-work:org/repo.git"},
		{"subgroups", id(account.GitLab, "gitlab-work"), "https://gitlab.com/group/sub/project.git", "git remote set-url origin git@gitlab-work:group/sub/project.git"},
		{"other host is left alone", gh, "https://example.org/org/repo.git", ""},
		{"custom host", id(account.Custom, "corp-git"), "https://git.example.com/team/app", "git remote set-url origin git@corp-git:team/app.git"},
		{"own rule wins over the default", id(account.GitHub, "github-work", account.Rewrite{From: "https://github.com/org/", To: "git@github-org:org/"}),
			"https://github.com/org/repo.git", "git remote set-url origin git@github-org:org/repo.git"},
		{"ssh origin", gh, "git@github.com:org/repo.git", ""},
		{"no alias", id(account.GitHub, ""), "https://github.com/org/repo.git", ""},
		{"no path", gh, "https://github.com/", ""},
		{"one segment", gh, "https://github.com/org", ""},
		{"empty", gh, "", ""},
	} {
		got, ok := RemoteCommand(c.a, c.origin)
		if got != c.want || ok != (c.want != "") {
			t.Errorf("%s: %q, %v; want %q", c.name, got, ok, c.want)
		}
	}
}

func TestCloneCommand(t *testing.T) {
	gh := id(account.GitHub, "github-work")
	for _, c := range []struct {
		name  string
		a     account.Account
		input string
		want  string
	}{
		{"shorthand", gh, "org/repo", "git clone git@github-work:org/repo.git"},
		{"shorthand with subgroups", id(account.GitLab, "gitlab-work"), "group/sub/project", "git clone git@gitlab-work:group/sub/project.git"},
		{"shorthand with git", gh, "org/repo.git", "git clone git@github-work:org/repo.git"},
		{"spaces around", gh, "  org/repo \n", "git clone git@github-work:org/repo.git"},
		{"https", gh, "https://github.com/org/repo.git", "git clone git@github-work:org/repo.git"},
		{"whole command", gh, "git clone git@github.com:org/repo.git", "git clone git@github-work:org/repo.git"},
		{"command with a flag", gh, "git clone --depth 1 https://github.com/org/repo", "git clone git@github-work:org/repo.git"},
		{"scp address", id(account.GitLab, "gitlab-work"), "git@gitlab.com:group/sub/project.git", "git clone git@gitlab-work:group/sub/project.git"},
		{"scp address of a custom host", id(account.Custom, "corp-git"), "git@github.com:org/repo.git", "git clone git@corp-git:org/repo.git"},
		{"ssh url", gh, "ssh://git@github.com/org/repo.git", "git clone git@github-work:org/repo.git"},
		{"already the alias", gh, "git@github-work:org/repo.git", "git clone git@github-work:org/repo.git"},
		{"own rule", id(account.GitHub, "github-work", account.Rewrite{From: "https://github.com/org/", To: "git@github-org:org/"}),
			"https://github.com/org/repo", "git clone git@github-org:org/repo.git"},
		{"prefix only", gh, "acme/", ""},
		{"bare name", gh, "repo", ""},
		{"empty", gh, "", ""},
		{"blank", gh, "   ", ""},
		{"no alias", id(account.GitHub, ""), "org/repo", ""},
		{"git clone alone", gh, "git clone", ""},
		{"words", gh, "not a repo", ""},
	} {
		got, ok := CloneCommand(c.a, c.input)
		if got != c.want || ok != (c.want != "") {
			t.Errorf("%s: %q, %v; want %q", c.name, got, ok, c.want)
		}
	}
}

func TestRepoAndPrefix(t *testing.T) {
	for _, c := range []struct{ in, repo, prefix string }{
		{"https://github.com/acme/api.git", "acme/api", "acme/"},
		{"git@github.com:group/sub/project.git", "group/sub/project", "group/sub/"},
		{"git clone git@github-work:acme/api.git", "acme/api", "acme/"},
		{"acme/api", "acme/api", "acme/"},
		{"acme/", "", ""},
		{"api", "", ""},
		{"", "", ""},
	} {
		if got := Repo(c.in); got != c.repo {
			t.Errorf("Repo(%q) = %q, want %q", c.in, got, c.repo)
		}
		if got := Prefix(c.in); got != c.prefix {
			t.Errorf("Prefix(%q) = %q, want %q", c.in, got, c.prefix)
		}
	}
}
