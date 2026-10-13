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
[docs/product.md](docs/product.md) (in Chinese), in the order of the [roadmap](docs/roadmap.md).

Working now:

- **Identities in the tray.** A tray icon opens a panel listing the identities in `~/.config/idset/accounts.json`,
  most recently used first, with a filter once there are three or more.
- **Drop a folder to check it.** Drop a project folder into the panel, or choose one, to see which identity Git
  uses there.

Planned:

- **Bind in one click.** Bind a folder to another identity from the panel, with undo.
- **Generated configuration.** SSH host aliases, `includeIf` rules and `insteadOf` URL rewrites are generated
  from the list and backed up before each change. The write path is in place: every saved change regenerates the
  files and the `Include` blocks, but no screen edits identities yet, so nothing is written in practice.
- **Diagnostics.** Checks each identity with `ssh -T` and says what to do when a key is rejected, locked or
  missing, or when a repository would commit as one identity and push as another.
- **Keys and import.** Generate a key, load it into ssh-agent, and import identities from an existing
  `~/.ssh/config` and `~/.gitconfig`.

Targets: macOS 12+, Windows 10/11 and Linux with GTK 3, on amd64 and arm64. Tray icons on Linux need
AppIndicator.

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
2. Identities cannot be added in the app yet; that comes with the accounts window. Until then they are read from
   `~/.config/idset/accounts.json` (on Windows `%USERPROFILE%\.config\idset\accounts.json`); the format is in
   [docs/product.md](docs/product.md).
3. Click the tray icon to open the panel, then drop a project folder into it, or choose one, to see which identity
   Git uses there.

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
| Order of work (in Chinese) | [docs/roadmap.md](docs/roadmap.md) |
| Architecture decisions | [docs/adr/](docs/adr/README.md) |

Repository rules for people and agents are in [AGENTS.md](AGENTS.md). Report vulnerabilities privately as
described in [SECURITY.md](SECURITY.md).

## Contributing

Issues and pull requests are welcome, in English or Simplified Chinese. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE). Licenses of dependencies are listed in [THIRD_PARTY.md](THIRD_PARTY.md).
