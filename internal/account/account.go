// Package account defines an identity, its JSON form in accounts.json and the checks it passes before saving.
package account

import (
	"crypto/rand"
	"fmt"
	"time"
)

// Provider is the hosting platform of an identity.
type Provider string

const (
	GitHub Provider = "github"
	GitLab Provider = "gitlab"
	Gitea  Provider = "gitea"
	Custom Provider = "custom"
)

// Providers lists every provider in display order.
var Providers = []Provider{GitHub, GitLab, Gitea, Custom}

type providerInfo struct {
	name, host22, host443, settings string
}

var providers = map[Provider]providerInfo{
	GitHub: {"GitHub", "github.com", "ssh.github.com", "https://github.com/settings/keys"},
	GitLab: {"GitLab", "gitlab.com", "altssh.gitlab.com", "https://gitlab.com/-/user_settings/ssh_keys"},
	Gitea:  {"Gitea", "gitea.com", "", "https://gitea.com/user/settings/keys"},
}

// Valid reports whether p is a known provider.
func (p Provider) Valid() bool { return p == Custom || providers[p].name != "" }

// DisplayName is the provider's name in the interface; Custom has a translated name instead.
func (p Provider) DisplayName() string { return providers[p].name }

// CanonicalHost is the platform's own host, the one clone URLs use; empty for Custom.
func (p Provider) CanonicalHost() string { return providers[p].host22 }

// KeySettingsURL is where the platform takes SSH public keys; empty for Custom.
func (p Provider) KeySettingsURL() string { return providers[p].settings }

// Supports443 reports whether the platform serves SSH on port 443.
func (p Provider) Supports443() bool { return providers[p].host443 != "" }

// ReservedHosts are the real host names of every platform, which an alias must not shadow.
func ReservedHosts() []string {
	var hosts []string
	for _, p := range Providers {
		info := providers[p]
		for _, h := range []string{info.host22, info.host443} {
			if h != "" {
				hosts = append(hosts, h)
			}
		}
	}
	return hosts
}

// Rewrite is one url.<to>.insteadOf rule, or pushInsteadOf when Push is set.
type Rewrite struct {
	From string `json:"from"`
	Push bool   `json:"push"`
	To   string `json:"to"`
}

// Account is one identity. Fields are in alphabetical order so that accounts.json has sorted keys.
type Account struct {
	Color          string     `json:"color"`
	CreatedAt      time.Time  `json:"createdAt"`
	GitUserEmail   string     `json:"gitUserEmail"`
	GitUserName    string     `json:"gitUserName"`
	HostName       string     `json:"hostName,omitempty"`
	ID             string     `json:"id"`
	KeyFingerprint string     `json:"keyFingerprint,omitempty"`
	KeyPath        string     `json:"keyPath"`
	Label          string     `json:"label"`
	LastUsedAt     *time.Time `json:"lastUsedAt,omitempty"`
	Notes          string     `json:"notes"`
	Provider       Provider   `json:"provider"`
	Scope          Scope      `json:"scope"`
	SSHAlias       string     `json:"sshAlias"`
	SSHPort        int        `json:"sshPort"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	URLRewrites    []Rewrite  `json:"urlRewrites"`
	Username       string     `json:"username"`
}

// DefaultPort is the port a new identity of provider p starts with.
func DefaultPort(p Provider) int {
	if p == Custom || !p.Supports443() {
		return 22
	}
	return 443
}

// New returns an identity with a fresh ID and the defaults of a new GitHub identity, created at now.
func New(now time.Time) Account {
	now = now.UTC().Truncate(time.Second)
	return Account{
		ID: NewID(), Provider: GitHub, SSHPort: DefaultPort(GitHub), Scope: Scope{Global: true},
		Color: "blue", URLRewrites: []Rewrite{}, CreatedAt: now, UpdatedAt: now,
	}
}

// RealHost is the host the SSH connection goes to: the platform's host for the port, or HostName for Custom.
func (a Account) RealHost() string {
	if a.Provider == Custom {
		return a.HostName
	}
	if a.SSHPort == 443 && a.Provider.Supports443() {
		return providers[a.Provider].host443
	}
	return providers[a.Provider].host22
}

// HostKeyAlias is the host whose key ssh checks when the connection goes to the platform's port 443 host.
func (a Account) HostKeyAlias() string {
	if a.Provider != Custom && a.SSHPort == 443 && a.Provider.Supports443() {
		return providers[a.Provider].host22
	}
	return ""
}

// NewID returns a random version 4 UUID in lower case.
func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
