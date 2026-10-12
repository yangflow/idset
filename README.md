<h1 align="center">Idset</h1>

<p align="center">
  Use the right Git and SSH identity in every folder, from a native tray app written in pure Go.
</p>

<p align="center">
  <a href="https://github.com/yangflow/idset/actions/workflows/ci.yml"><img src="https://github.com/yangflow/idset/actions/workflows/ci.yml/badge.svg?branch=main" alt="CI"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/yangflow/idset" alt="License: MIT"></a>
</p>

<p align="center">
  <b>English</b> · <a href="README.zh-CN.md">简体中文</a>
</p>

## Why Idset

Work and personal repositories on one machine need different commit emails and different SSH keys. Getting it
right means hand-editing `~/.ssh/config` host aliases, `includeIf "gitdir:"` blocks in `~/.gitconfig`, and
remembering which alias to put in each clone URL. One mistake and a commit goes out under the wrong name, or a
push is rejected.

Idset keeps your identities in one list and writes the configuration for you. It never rewrites the files you
maintain: it generates its own files and adds a single `Include` line to `~/.ssh/config` and `~/.gitconfig`.
Private keys stay where they are; Idset only refers to them by path.

## Status

Early development, version 0.1.0, with no release yet. The behavior to build is specified in
[docs/product.md](docs/product.md) (in Chinese), in the order of the [roadmap](docs/roadmap.md).

Working now:

- **Identities in the tray.** A tray icon opens a panel listing the identities in `~/.config/idset/accounts.json`,
  most recently used first, with a filter once there are three or more.
- **Drop a folder to check it.** Drop a project folder into the panel, or choose one, to see which identity Git
  uses there.

Identities cannot be added in the app yet; that comes with the accounts window.

Planned:

- **Bind in one click.** Bind a folder to another identity from the panel, with undo.
- **Generated configuration.** SSH host aliases, `includeIf` rules and `insteadOf` URL rewrites are generated
  from the list and backed up before each change.
- **Diagnostics.** Checks each identity with `ssh -T` and says what to do when a key is rejected, locked or
  missing, or when a repository would commit as one identity and push as another.
- **Keys and import.** Generate a key, load it into ssh-agent, and import identities from an existing
  `~/.ssh/config` and `~/.gitconfig`.

Targets: macOS 12+, Windows 10/11 and Linux with GTK 3, on amd64 and arm64. Tray icons on Linux need
AppIndicator.

## Quick start

Build from source with Go 1.27.1 (see `.go-version`) and Python 3:

```bash
git clone https://github.com/yangflow/idset
cd idset
scripts/check.sh --cross   # format, vet, tests and six cross-compiled targets
go tool mygo dev           # run the desktop app
go tool mygo build         # package it into build/
```

## Security

Report vulnerabilities privately, as described in [SECURITY.md](SECURITY.md). Do not open public issues for them.

## Documentation

| Topic | Where (in Chinese) |
|---|---|
| Behavior, data model and generated files | [docs/product.md](docs/product.md) |
| Colors, type, spacing and components | [docs/design-system.md](docs/design-system.md) |
| Screens and their states | [design/README.md](design/README.md) |
| Order of work | [docs/roadmap.md](docs/roadmap.md) |
| Architecture decisions | [docs/adr/](docs/adr/README.md) |
| Rules for changing the repository | [AGENTS.md](AGENTS.md) |

## Contributing

Issues and pull requests are welcome; read [CONTRIBUTING.md](CONTRIBUTING.md) first. Every commit is signed off
with `git commit -s` for the Developer Certificate of Origin.

## Support

See [SUPPORT.md](SUPPORT.md). Questions go to issues; there are no Discussions and no paid support.

## License

[MIT](LICENSE). Licenses of dependencies are listed in [THIRD_PARTY.md](THIRD_PARTY.md).
