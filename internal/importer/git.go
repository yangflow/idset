package importer

import (
	"strings"
)

// Entry is one value of `git config --list -z`, with the file it came from when --show-origin was given. Key is
// as git prints it: section and name in lower case, the subsection as written.
type Entry struct {
	Origin string
	Key    string
	Value  string
}

// ParseGit reads the output of `git config --list -z`, with origin set when --show-origin was given: every entry
// is then preceded by "file:<path>". A key without a value, such as a bare boolean, has an empty Value.
func ParseGit(out string, origin bool) []Entry {
	var (
		list []Entry
		from string
		next = !origin
	)
	for _, field := range strings.Split(out, "\x00") {
		if !next {
			from, next = strings.TrimPrefix(field, "file:"), true
			continue
		}
		if field == "" {
			continue
		}
		key, value, _ := strings.Cut(field, "\n")
		list = append(list, Entry{Origin: from, Key: key, Value: value})
		next = !origin
	}
	return list
}

// split cuts a key into section, subsection and name; the subsection may itself contain dots.
func split(key string) (section, sub, name string) {
	first, last := strings.IndexByte(key, '.'), strings.LastIndexByte(key, '.')
	if first < 0 {
		return key, "", ""
	}
	if first == last {
		return key[:first], "", key[first+1:]
	}
	return key[:first], key[first+1 : last], key[last+1:]
}

// Identity is a signature git uses, with the key its core.sshCommand names and the rewrites of the same file.
// Cond is the includeIf condition it applies under, empty for the global one; Origin is the file the includeIf
// line is in, against which a relative ./ gitdir is read.
type Identity struct {
	Name, Email, SSHCommand string
	Cond, Origin            string
}

// Rule is one url.<to>.insteadOf or pushInsteadOf value.
type Rule struct {
	From, To string
	Push     bool
}

// Gitconfig is what was read from git: the global identity, the identity of each includeIf file in order, and
// every rewrite rule of all of them.
type Gitconfig struct {
	Global   Identity
	Included []Identity
	Rules    []Rule
}

// collect reads the identity and the rules of one file's entries into id and g.
func collect(entries []Entry, id *Identity, g *Gitconfig) {
	for _, e := range entries {
		section, sub, name := split(e.Key)
		switch {
		case section == "user" && sub == "" && name == "name":
			id.Name = e.Value
		case section == "user" && sub == "" && name == "email":
			id.Email = e.Value
		case section == "core" && sub == "" && name == "sshcommand":
			id.SSHCommand = e.Value
		case section == "url" && sub != "" && (name == "insteadof" || name == "pushinsteadof"):
			g.Rules = append(g.Rules, Rule{From: e.Value, To: sub, Push: name == "pushinsteadof"})
		}
	}
}

// IncludeIf is one includeIf.<cond>.path of the global configuration, in order.
type IncludeIf struct {
	Cond, Path, Origin string
}

// IncludeIfs lists the includeIf paths of the global entries.
func IncludeIfs(entries []Entry) []IncludeIf {
	var out []IncludeIf
	for _, e := range entries {
		if section, sub, name := split(e.Key); section == "includeif" && sub != "" && name == "path" && e.Value != "" {
			out = append(out, IncludeIf{Cond: sub, Path: e.Value, Origin: e.Origin})
		}
	}
	return out
}
