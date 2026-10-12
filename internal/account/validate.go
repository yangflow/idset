package account

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Code names one reason an identity cannot be saved. The interface maps it to a translated message.
type Code string

const (
	BadID          Code = "bad-id"
	DuplicateID    Code = "duplicate-id"
	BadProvider    Code = "bad-provider"
	BadAlias       Code = "bad-alias"
	DuplicateAlias Code = "duplicate-alias"
	ReservedAlias  Code = "reserved-alias"
	MissingHost    Code = "missing-host"
	BadHost        Code = "bad-host"
	BadPort        Code = "bad-port"
	NewLine        Code = "new-line"
	Quote          Code = "quote"
	SecondGlobal   Code = "second-global"
	BadColor       Code = "bad-color"
)

// Problem is one failed check. Field is the JSON name of the field; Other is the label of the identity it
// conflicts with, when there is one.
type Problem struct {
	ID    string
	Field string
	Code  Code
	Other string
}

// Problems is every failed check of a list; it is an error when not empty.
type Problems []Problem

func (p Problems) Error() string {
	parts := make([]string, len(p))
	for i, q := range p {
		parts[i] = fmt.Sprintf("%s %s: %s", q.ID, q.Field, q.Code)
	}
	return "invalid accounts: " + strings.Join(parts, "; ")
}

var (
	idPattern    = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	aliasPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	hostPattern  = regexp.MustCompile(`^[A-Za-z0-9._:-]+$`)
	colorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
)

// ColorPresets are the named colors in the order new identities take them.
var ColorPresets = []string{"blue", "green", "orange", "red", "purple", "yellow"}

func validID(id string) bool { return idPattern.MatchString(id) }

// Validate checks a whole list, since aliases and the single global identity depend on the others.
func Validate(list []Account) Problems {
	var out Problems
	add := func(a Account, field string, code Code, other string) {
		out = append(out, Problem{ID: a.ID, Field: field, Code: code, Other: other})
	}
	ids := map[string]bool{}
	aliases := map[string]Account{}
	var global *Account
	reserved := map[string]bool{}
	for _, h := range ReservedHosts() {
		reserved[h] = true
	}
	for _, a := range list {
		if a.Provider == Custom && a.HostName != "" {
			reserved[strings.ToLower(a.HostName)] = true
		}
	}
	for i, a := range list {
		if !validID(a.ID) {
			add(a, "id", BadID, "")
		} else if ids[strings.ToLower(a.ID)] {
			add(a, "id", DuplicateID, "")
		}
		ids[strings.ToLower(a.ID)] = true
		if !a.Provider.Valid() {
			add(a, "provider", BadProvider, "")
		}
		if alias := a.SSHAlias; alias != "" {
			key := strings.ToLower(alias)
			switch {
			case !aliasPattern.MatchString(alias):
				add(a, "sshAlias", BadAlias, "")
			case reserved[key]:
				add(a, "sshAlias", ReservedAlias, "")
			case aliases[key].ID != "":
				add(a, "sshAlias", DuplicateAlias, aliases[key].Label)
			default:
				aliases[key] = a
			}
		}
		if a.Provider == Custom {
			switch {
			case a.HostName == "":
				add(a, "hostName", MissingHost, "")
			case !hostPattern.MatchString(a.HostName):
				add(a, "hostName", BadHost, "")
			}
			if a.SSHPort < 1 || a.SSHPort > 65535 {
				add(a, "sshPort", BadPort, "")
			}
		} else if a.SSHPort != 22 && !(a.SSHPort == 443 && a.Provider.Supports443()) {
			add(a, "sshPort", BadPort, "")
		}
		for _, f := range [][2]string{
			{"label", a.Label}, {"username", a.Username}, {"keyPath", a.KeyPath}, {"keyFingerprint", a.KeyFingerprint},
			{"gitUserName", a.GitUserName}, {"gitUserEmail", a.GitUserEmail}, {"hostName", a.HostName},
		} {
			if strings.ContainsAny(f[1], "\r\n") {
				add(a, f[0], NewLine, "")
			}
		}
		if strings.Contains(a.KeyPath, `"`) {
			add(a, "keyPath", Quote, "")
		}
		for _, p := range a.Scope.Paths {
			if strings.ContainsAny(p, "\r\n") {
				add(a, "scope", NewLine, "")
			} else if strings.Contains(p, `"`) {
				add(a, "scope", Quote, "")
			}
		}
		for _, r := range a.URLRewrites {
			if strings.ContainsAny(r.From+r.To, "\r\n") {
				add(a, "urlRewrites", NewLine, "")
			}
		}
		if a.Scope.Global {
			if global != nil {
				add(a, "scope", SecondGlobal, global.Label)
			} else {
				global = &list[i]
			}
		}
		if !colorPattern.MatchString(a.Color) && !slices.Contains(ColorPresets, a.Color) {
			add(a, "color", BadColor, "")
		}
	}
	return out
}
