# Idset

English | [中文](README.zh-CN.md)

Use the right Git and SSH identity in every folder, from a native tray app written in pure Go.

[![CI](https://github.com/yangflow/idset/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/yangflow/idset/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/github/license/yangflow/idset)](LICENSE)

Work and personal repositories on one machine need different commit emails and different SSH keys. Getting it
right means hand-editing `~/.ssh/config` host aliases, `includeIf "gitdir:"` blocks in `~/.gitconfig`, and
remembering which alias to put in each clone URL. One mistake and a commit goes out under the wrong name, or a
push is rejected.

Idset keeps your identities in one list and writes the configuration for you. It never rewrites the files you
maintain: it generates its own files and adds a single `Include` line to `~/.ssh/config` and `~/.gitconfig`.
Private keys stay where they are; Idset only refers to them by path.

## Features

Early development, version 0.1.0, with no release yet. The behavior to build is specified in
[docs/product.md](docs/product.md) (in Chinese), issue by issue, in the order of the
[milestones](https://github.com/yangflow/idset/milestones).

Working now:

- **Identities in the tray.** A tray icon opens a panel listing the identities in `~/.config/idset/accounts.json`,
  most recently used first, with a filter once there are three or more.
- **Drop a folder to check it.** Drop a project folder into the panel, or choose one, to see which identity Git
  uses there.
- **Check the repository against the identity.** After a match, the card says when the repository's own
  `user.email` is missing, belongs to another identity or to none, or when its `core.sshCommand` picks a different
  key, so commits and pushes would not be as that identity.
- **Bind from the result card.** Bind a folder to an identity, unbind it, move it to another one, or settle several
  identities covering the same folder, all from the card for a dropped folder. Binding, rebinding and "Use only"
  can be undone for five seconds.
- **Follow a renamed or moved folder.** When nothing matches a dropped folder but it looks like a folder an identity
  still points to, renamed or moved, the panel offers to change that old path to the new one, or to keep it.
- **Manage identities in the app.** The accounts window creates, edits and deletes identities: signature, platform,
  SSH key and alias, the folders each one applies to, and URL rewrites. Deleting can remove the private key too, but
  only when you ask for it.
- **Generated configuration.** SSH host aliases, `includeIf` rules and `insteadOf` URL rewrites are generated
  from the list after every change, and the list is backed up before a new identity is added. Your own
  `~/.ssh/config` and `~/.gitconfig` get one marked `Include` block and nothing else.
- **Use the SSH alias for clones and remotes.** When a repository's origin is an HTTPS address, the card copies the
  command that switches it to the identity's SSH alias. A clone helper turns `owner/repo`, an address or a
  `git clone` command into the clone command for that identity.
- **Settings, backups and restore.** The settings window sets the language and opening at login, lists the backups
  to restore or delete (also when `accounts.json` cannot be read), and removes the `Include` blocks while keeping
  your accounts.
- **Diagnostics.** Checks each identity with `ssh -T` in the background and shows the result as a dot on its row.
  When a key is rejected, locked or missing, a host cannot be reached or `ssh` is not installed, the row says what to
  do next; a summary under the list and a badge on the tray icon show the worst problem.
- **Keys.** The settings window generates an SSH key, copies its public key, loads it into ssh-agent and gives it to
  an identity, or starts a new identity with it. A missing key's bar opens it filled in for that identity.
- **Import.** The settings window reads your existing `~/.ssh/config` and Git configuration, lists the identities
  found there, and adds the ones you check. Your files are only read.

Targets: macOS 12+, Windows 10/11 and Linux with GTK 3, on amd64 and arm64. Tray icons on Linux need
AppIndicator.

Releases are built and published from a version tag ([docs/release.md](docs/release.md)), and installed copies
update themselves from them. The macOS app is signed and notarized. The Windows installers are not signed yet, so
SmartScreen warns on first launch. The Debian package needs `libayatana-appindicator3-1`.

## Quick start

There is no release yet. Build from source with the Go version in `.go-version` and Python 3:

```bash
git clone https://github.com/yangflow/idset
cd idset
go tool mygo dev           # run the desktop app
go tool mygo build         # package it into build/
```

## Usage

1. Start Idset; it lives in the tray (menu bar on macOS, notification area on Windows).
2. Click the tray icon to open the panel, then choose "Add Identity" to open the accounts window and create one. Click
   an identity in the panel to edit or delete it. Identities are stored in `~/.config/idset/accounts.json` (on
   Windows `%USERPROFILE%\.config\idset\accounts.json`); the format is in [docs/product.md](docs/product.md).
3. Drop a project folder into the panel, or choose one, to see which identity Git uses there, and bind it from the card if it should use another.

## Development

```bash
scripts/check.sh --cross          # gofmt, go mod tidy, vet, tests, script tests, six cross-compiled targets
go test -race ./...
python3 scripts/scan_secrets.py   # secret scan of the whole history
```

| Topic | Where |
|---|---|
| Behavior, data model and generated files (in Chinese) | [docs/product.md](docs/product.md) |
| Design language: layout, color, type, components, accessibility | [docs/design.md](docs/design.md) |
| Screen mockups | [design/](design/README.md) |
| Order of work | [Milestones](https://github.com/yangflow/idset/milestones) and [issues](https://github.com/yangflow/idset/issues) |
| Architecture decisions | [docs/adr/](docs/adr/README.md) |

Repository rules for people and agents are in [AGENTS.md](AGENTS.md). Report vulnerabilities privately as
described in [SECURITY.md](SECURITY.md).

## Contributing

Issues and pull requests are welcome, in English or Simplified Chinese. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE). Licenses of dependencies are listed in [THIRD_PARTY.md](THIRD_PARTY.md).
