package apply

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/gitdir"
	"github.com/yangflow/idset/internal/include"
	"github.com/yangflow/idset/internal/paths"
	"github.com/yangflow/idset/internal/project"
)

var now = time.Date(2026, 8, 22, 8, 0, 0, 0, time.UTC)

// home returns a temporary home directory with symbolic links resolved, as git sees it.
func home(t *testing.T) paths.Env {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return paths.Env{Home: dir, Windows: runtime.GOOS == "windows"}
}

func identity(label, email string, scope ...string) account.Account {
	a := account.New(now)
	a.Label, a.SSHAlias, a.KeyPath, a.GitUserName, a.GitUserEmail = label, "github-"+label, "~/.ssh/id_"+label, label, email
	if len(scope) > 0 {
		a.Scope = account.Scope{Paths: scope}
	}
	return a
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestWriteCreatesFilesAndIncludes(t *testing.T) {
	env := home(t)
	target := DefaultTarget(env)
	userSSH := "Host personal-box\n\tHostName 192.0.2.1\n"
	userGit := "[user]\r\n\tname = Someone\r\n"
	os.MkdirAll(filepath.Dir(target.SSHConfig), 0o700)
	os.WriteFile(target.SSHConfig, []byte(userSSH), 0o644)
	os.WriteFile(target.GitConfig, []byte(userGit), 0o644)

	list := []account.Account{identity("personal", "p@example.com"), identity("work", "w@example.com", "~/work/")}
	out := project.Project(list, env, now)
	if err := Write(out, target); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(target.Dir, project.SSHFile)); got != out.SSH {
		t.Error("ssh managed file")
	}
	if got := read(t, filepath.Join(target.Dir, project.SubFile(list[1].ID))); got != out.Subs[project.SubFile(list[1].ID)] {
		t.Error("sub file")
	}
	ssh := read(t, target.SSHConfig)
	if ssh != include.SSHBlock("~/.config/idset/ssh_config.managed")+"\n"+userSSH {
		t.Errorf("ssh config:\n%s", ssh)
	}
	git := read(t, target.GitConfig)
	if !strings.HasPrefix(git, userGit+"\r\n# --- idset managed ---\r\n[include]\r\n") {
		t.Errorf("git config keeps CRLF:\n%q", git)
	}
	if runtime.GOOS != "windows" {
		if info, _ := os.Stat(target.SSHConfig); info.Mode().Perm() != 0o644 {
			t.Errorf("existing file mode changed to %v", info.Mode())
		}
		if info, _ := os.Stat(filepath.Join(target.Dir, project.GitFile)); info.Mode().Perm() != 0o600 {
			t.Errorf("managed file mode %v", info.Mode())
		}
	}

	stat, _ := os.Stat(target.SSHConfig)
	list[1].Scope.Paths = nil
	if err := Write(project.Project(list, env, now.Add(time.Hour)), target); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target.Dir, project.SubFile(list[1].ID))); !os.IsNotExist(err) {
		t.Error("a sub file that is no longer generated must be deleted")
	}
	if again, _ := os.Stat(target.SSHConfig); !again.ModTime().Equal(stat.ModTime()) || read(t, target.SSHConfig) != ssh {
		t.Error("a current block must not rewrite the user's file")
	}

	if err := RemoveIncludes(target); err != nil {
		t.Fatal(err)
	}
	if read(t, target.SSHConfig) != userSSH || read(t, target.GitConfig) != userGit {
		t.Errorf("remove left:\n%q\n%q", read(t, target.SSHConfig), read(t, target.GitConfig))
	}
}

func TestWriteCreatesMissingUserFiles(t *testing.T) {
	env := home(t)
	target := DefaultTarget(env)
	if err := Write(project.Project(nil, env, now), target); err != nil {
		t.Fatal(err)
	}
	if read(t, target.SSHConfig) != include.SSHBlock("~/.config/idset/ssh_config.managed") {
		t.Error("new ssh config")
	}
	if runtime.GOOS != "windows" {
		for _, p := range []string{target.SSHConfig, filepath.Dir(target.SSHConfig)} {
			info, _ := os.Stat(p)
			if want := map[bool]os.FileMode{false: 0o600, true: 0o700}[info.IsDir()]; info.Mode().Perm() != want {
				t.Errorf("%s mode %v", p, info.Mode())
			}
		}
	}
}

func TestWriteFollowsLinkedDotfiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic links need privileges on Windows")
	}
	env := home(t)
	target := DefaultTarget(env)
	real := filepath.Join(env.Home, "dotfiles", "gitconfig")
	os.MkdirAll(filepath.Dir(real), 0o700)
	os.WriteFile(real, []byte("[core]\n"), 0o644)
	if err := os.Symlink(real, target.GitConfig); err != nil {
		t.Fatal(err)
	}
	if err := Write(project.Project(nil, env, now), target); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Lstat(target.GitConfig); info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("the link was replaced")
	}
	if !strings.Contains(read(t, real), include.Begin) {
		t.Fatal("the linked file did not get the block")
	}
}

// TestGitAgreesWithMatching writes the configuration into a temporary home and asks the real git which email
// it uses in each repository; it must be the identity the matching code picks, whatever the storage order.
func TestGitAgreesWithMatching(t *testing.T) {
	gitBin, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	env := home(t)
	list := []account.Account{
		identity("client", "client@example.com", "~/work/client/"),
		identity("personal", "personal@example.com"),
		identity("work", "work@example.com", "~/work/"),
		identity("side", "side@example.com", "~/src/side/", "~/work/client/side/"),
	}
	if err := Write(project.Project(list, env, now), DefaultTarget(env)); err != nil {
		t.Fatal(err)
	}
	gitEnv := append(os.Environ(),
		"HOME="+env.Home, "USERPROFILE="+env.Home, "XDG_CONFIG_HOME="+filepath.Join(env.Home, ".config"),
		"GIT_CONFIG_GLOBAL="+filepath.Join(env.Home, ".gitconfig"), "GIT_CONFIG_NOSYSTEM=1")
	run := func(dir string, args ...string) string {
		cmd := exec.Command(gitBin, append([]string{"-C", dir}, args...)...)
		cmd.Env = gitEnv
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("git %v in %s: %v", args, dir, err)
		}
		return strings.TrimSpace(string(out))
	}
	blocks := gitdir.Blocks(list, env)
	for _, rel := range []string{"work/api", "work/client/app", "work/client/side", "src/side", "other/x", "work"} {
		dir := filepath.Join(env.Home, filepath.FromSlash(rel))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		run(dir, "init", "-q")
		want := "personal@example.com"
		if m := gitdir.Matching(blocks, dir, env); len(m) > 0 {
			want = list[m[len(m)-1].Index].GitUserEmail
		}
		if got := run(dir, "config", "user.email"); got != want {
			t.Errorf("%s: git uses %s, matching says %s", rel, got, want)
		}
	}
}
