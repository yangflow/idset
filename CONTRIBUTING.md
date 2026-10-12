# Contributing

Thanks for helping. Idset is mostly written by the maintainer working with AI coding agents; issues and pull
requests from anyone are welcome. Read [AGENTS.md](AGENTS.md) for the full set of repository rules.

## Before you start

- Issues and pull requests may be written in English or Simplified Chinese. 欢迎用中文或英文提 issue 和 PR。
- Open an issue before a large change or a new feature so we can agree on the direction first. Small fixes can go
  straight to a pull request.
- The scope is in [docs/product.md](docs/product.md) (in Chinese): Idset does not rewrite the body of your
  existing dotfiles, does not touch private key contents and does not sync to a cloud. Pull requests outside the
  scope may be closed.
- Do not report vulnerabilities in public issues; see [SECURITY.md](SECURITY.md).

## Development setup

Requirements: the Go version in `.go-version` and Python 3 for the helper scripts. No Node, Bun or WebView.

```bash
scripts/check.sh --cross     # gofmt, go mod tidy, version pin, vet, tests, script tests, six cross-compiled targets
go test -race ./...
go tool mygo dev             # run the app
go tool mygo build           # package it into build/
```

Both check commands must pass before merge. CI also runs `python3 scripts/scan_secrets.py` over the whole history
and `go test ./...` on Windows.

## Commits, branches and pull requests

- Commit messages and pull request titles follow [Conventional Commits](https://www.conventionalcommits.org/):
  `feat|fix|docs|refactor|perf|test|build|ci|chore|revert(scope)!: subject`.
- Name branches `<type>/<issue>-<slug>`, for example `feat/12-bind-folder`.
- Link the issue with `Closes #<n>`. The maintainer squash-merges, so the pull request title becomes the commit
  message. No DCO or sign-off is required.
- Keep a change focused. When behavior changes, update the existing tests for that area and, if users would notice
  it, add a line under `## [Unreleased]` in both `CHANGELOG.md` and `CHANGELOG.zh.md`.
- Change `docs/product.md` before changing product behavior. UI copy lives in each screen's `copy.json`, in `en`
  and `zh-Hans` only; change both together.

## UI changes

A UI change starts from the mockup in [design/](design/README.md) and
[docs/design-system.md](docs/design-system.md); the maintainer confirms the visual direction before UI code is
written. Mockup, design-system and code changes go in the same pull request. Use only MyGo native controls and
theme colors.

Put screenshots in the pull request body (light and dark, one per state), not in the repository.

## AI-assisted changes

AI tools are fine. Say in the pull request which one you used and for what. You are responsible for every line you
submit: read it, understand it and run it.

## License

Contributions are licensed under the [MIT License](LICENSE).
