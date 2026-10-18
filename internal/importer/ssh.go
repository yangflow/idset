package importer

import (
	"strings"
)

// Line is one line of an ssh_config file. Raw is the line exactly as read, with its line ending, so that joining
// the Raw of every line gives the file back byte for byte. Key is the keyword as written and Args its arguments
// with quotes removed; both are empty for blank lines and comments.
type Line struct {
	Raw  string
	Key  string
	Args []string
}

// ParseSSH cuts the text of an ssh_config file into lines. It never fails: a line it cannot read is kept as it is
// and has no keyword.
func ParseSSH(text string) []Line {
	var out []Line
	for text != "" {
		end := strings.IndexByte(text, '\n') + 1
		if end == 0 {
			end = len(text)
		}
		raw := text[:end]
		text = text[end:]
		l := Line{Raw: raw}
		body := strings.TrimSpace(raw)
		if body != "" && !strings.HasPrefix(body, "#") {
			key, rest := cutKeyword(body)
			l.Key, l.Args = key, splitArgs(rest)
		}
		out = append(out, l)
	}
	return out
}

// Text joins the lines back into the file they were read from.
func Text(lines []Line) string {
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l.Raw)
	}
	return b.String()
}

// cutKeyword splits an ssh_config line into its keyword and arguments; "=" may stand between them.
func cutKeyword(line string) (string, string) {
	i := strings.IndexAny(line, " \t=")
	if i < 0 {
		return line, ""
	}
	rest := strings.TrimLeft(line[i:], " \t")
	rest = strings.TrimPrefix(rest, "=")
	return line[:i], strings.TrimSpace(rest)
}

// splitArgs cuts the arguments of a keyword at white space; double quotes keep white space in one argument.
func splitArgs(s string) []string {
	var (
		out   []string
		cur   strings.Builder
		in    bool
		quote bool
	)
	for _, r := range s {
		switch {
		case r == '"':
			quote, in = !quote, true
		case !quote && (r == ' ' || r == '\t'):
			if in {
				out = append(out, cur.String())
				cur.Reset()
				in = false
			}
		default:
			cur.WriteRune(r)
			in = true
		}
	}
	if in {
		out = append(out, cur.String())
	}
	return out
}

// Host is one exact name of a Host line with the options of its block, keywords in lower case and only the
// first value of each, as ssh uses it.
type Host struct {
	Name    string
	Options map[string]string
}

// Config is an ssh_config file and the files its Include lines read, by the index of the Include line; each in
// the order its patterns matched.
type Config struct {
	Lines    []Line
	Included map[int][][]Line
}

// Hosts lists the exact names of the Host lines in order, each with the options of its block, and the hosts of
// an included file where its Include line is. Patterns with wildcards or negations match more than one name and
// are left out. A Match block is skipped up to the next Host or Match line, and options before the first Host
// line apply to every host and belong to none. An included file has its own blocks: the block the Include line
// is in goes on after it.
func Hosts(c Config) []Host {
	var (
		out   []Host
		names []string
		opts  map[string]string
	)
	flush := func() {
		for _, n := range names {
			out = append(out, Host{Name: n, Options: opts})
		}
		names, opts = nil, nil
	}
	var inc []Host
	for i, l := range c.Lines {
		switch key := strings.ToLower(l.Key); key {
		case "":
		case "host":
			flush()
			out = append(out, inc...)
			inc = nil
			opts = map[string]string{}
			for _, n := range l.Args {
				if n != "" && !strings.ContainsAny(n, "*?!") {
					names = append(names, n)
				}
			}
		case "match":
			flush()
			out = append(out, inc...)
			inc = nil
		case "include":
			for _, lines := range c.Included[i] {
				inc = append(inc, Hosts(Config{Lines: lines})...)
			}
		default:
			if opts != nil && len(l.Args) > 0 {
				if _, ok := opts[key]; !ok {
					opts[key] = strings.Join(l.Args, " ")
				}
			}
		}
	}
	flush()
	return append(out, inc...)
}

// Includes maps the index of each Include line outside Match blocks to its arguments.
func Includes(lines []Line) map[int][]string {
	out := map[int][]string{}
	inMatch := false
	for i, l := range lines {
		switch strings.ToLower(l.Key) {
		case "host":
			inMatch = false
		case "match":
			inMatch = true
		case "include":
			if !inMatch && len(l.Args) > 0 {
				out[i] = l.Args
			}
		}
	}
	return out
}
