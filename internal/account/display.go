package account

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Sorted returns the list in display order: most recently used first and never used last, then by label
// ignoring case with blank labels last, then by creation time and ID.
func Sorted(list []Account) []Account {
	out := slices.Clone(list)
	slices.SortStableFunc(out, func(a, b Account) int {
		switch {
		case a.LastUsedAt != nil && b.LastUsedAt == nil:
			return -1
		case a.LastUsedAt == nil && b.LastUsedAt != nil:
			return 1
		case a.LastUsedAt != nil:
			if c := b.LastUsedAt.Compare(*a.LastUsedAt); c != 0 {
				return c
			}
		}
		la, lb := strings.TrimSpace(a.Label), strings.TrimSpace(b.Label)
		if (la == "") != (lb == "") {
			if la == "" {
				return 1
			}
			return -1
		}
		return cmp.Or(
			strings.Compare(strings.ToLower(la), strings.ToLower(lb)),
			a.CreatedAt.Compare(b.CreatedAt),
			strings.Compare(a.ID, b.ID),
		)
	})
	return out
}

// ProvidersInUse returns the providers the list uses, in display order.
func ProvidersInUse(list []Account) []Provider {
	var out []Provider
	for _, p := range Providers {
		if slices.ContainsFunc(list, func(a Account) bool { return a.Provider == p }) {
			out = append(out, p)
		}
	}
	return out
}

// Filter keeps the identities whose label, alias, email, name, username or provider contains query, ignoring
// case and diacritics, and whose provider is selected when any is. The order is kept.
func Filter(list []Account, query string, selected []Provider) []Account {
	q := fold(strings.TrimSpace(query))
	var out []Account
	for _, a := range list {
		if len(selected) > 0 && !slices.Contains(selected, a.Provider) {
			continue
		}
		if q == "" || slices.ContainsFunc([]string{a.Label, a.SSHAlias, a.GitUserEmail, a.GitUserName, a.Username, string(a.Provider)},
			func(f string) bool { return f != "" && strings.Contains(fold(f), q) }) {
			out = append(out, a)
		}
	}
	return out
}

func fold(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, s)
	if err != nil {
		out = s
	}
	return strings.ToLower(out)
}

var presetRGB = map[string][3]uint8{
	"blue": {0x3A, 0x7B, 0xD5}, "green": {0x2E, 0x9E, 0x5B}, "orange": {0xE0, 0x7A, 0x1F},
	"red": {0xD6, 0x45, 0x45}, "purple": {0x8E, 0x44, 0xAD}, "yellow": {0xD4, 0xA0, 0x17},
}

// RGB is the identity's color: a preset's fixed value or the #RRGGBB itself; anything else shows as blue.
func (a Account) RGB() (r, g, b uint8) {
	if c, ok := presetRGB[a.Color]; ok {
		return c[0], c[1], c[2]
	}
	if colorPattern.MatchString(a.Color) {
		v, _ := strconv.ParseUint(a.Color[1:], 16, 32)
		return uint8(v >> 16), uint8(v >> 8), uint8(v)
	}
	c := presetRGB["blue"]
	return c[0], c[1], c[2]
}
