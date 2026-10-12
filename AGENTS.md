# AGENTS.md

## What it is

Idset is a native tray app, written in pure Go with MyGo, that keeps a list of Git and SSH identities and generates
the configuration so every folder uses the right one. Targets: macOS 12+, Windows 10/11 and Linux with GTK 3
(AppIndicator for the tray), on amd64 and arm64. MIT licensed. `CLAUDE.md` is the single line `@AGENTS.md`; put
nothing else in it and keep no other agent file.

Reference documents (in Chinese):

- `docs/product.md`: behavior, data model, format of the generated files, flows, scope.
- `docs/design-system.md`: colors, type, spacing, components.
- `design/`: a mockup per screen with all its states.
- `docs/adr/`: architecture decisions.

When a document and the code disagree, decide which is wrong first: if the document records a changed decision,
change the code; if the document is still valid and the code quietly falls short, change the code; change the
document only when following it would lead someone to do the wrong thing.

## Layout

```
main.go                  lifecycle, single instance, tray, panel window position and visibility,
                         background loading and matching, native updater plugin
internal/ui/<screen>/    one package per screen: state, copy.json catalog, headless interaction tests;
                         the name matches design/screens/<screen>.html
internal/ui/panel/       the panel
internal/ui/locale/      system language to UI language (Simplified Chinese, English), shared by all catalogs
internal/paths/          stored path form (forward slashes, `~`) to absolute and back; case-insensitive on Windows
internal/fsutil/         atomic writes, refusing symlinks, following symlinks to the real file
internal/account/        identity fields, accounts.json encoding, validation before save; fields are sorted by
                         JSON key, enforced by a test
internal/store/          reading and writing accounts.json, read-only state, backup and restore
internal/gitdir/         normalizing `gitdir` paths, includeIf write order and matching
internal/project/        managed file contents generated from the identity list; pure, no disk access
internal/include/        inserting, replacing and removing the Include marker block; pure
internal/apply/          the only place that writes config files: managed files and the marker block in user files
internal/sysexec/        running system commands: timeout, no stdin, no shell (ADR 0004)
internal/match/          which identity a dropped folder uses; the `git` calls are injectable, tests use a fake
internal/trayicon/       the tray icon, drawn in code, no image files
resources/               shipped licenses and product assets
scripts/                 checks, version bump, secret scan, packaging
mygo.json                name, identifier, version, packaging
docs/, design/           product spec, design system, ADRs, mockups
```

## Commands

```sh
scripts/check.sh --cross     # gofmt, go mod tidy, version pin, vet, tests, script tests, six cross-compiled targets
go test -race ./...
python3 scripts/scan_secrets.py
go tool mygo dev
go tool mygo build
python3 scripts/bump.py <version>
```

`scripts/check.sh --cross` must pass before merge.

## Conventions

- Commit messages and PR titles use Conventional Commits: `feat|fix|docs|refactor|perf|test|build|ci|chore|revert(scope)!: subject`.
  Branches are `<type>/<issue>-<slug>`, for example `feat/12-bind-folder`. Open an issue first; the PR body says
  `Closes #<n>`; squash merge.
- Commit author email is the GitHub noreply address `269347247+yangflow@users.noreply.github.com`: set it with
  `git config user.email` after cloning; do not use a personal address or the machine's global config.
  `GIT_AUTHOR_EMAIL` and `GIT_COMMITTER_EMAIL` override git config, so check `git var GIT_AUTHOR_IDENT` before
  committing.
- Product behavior changes start in `docs/product.md`. A UI change starts from the mockup in design/; the
  maintainer confirms the visual direction before UI code is written. Mockup, `docs/design-system.md` and code go
  in the same PR. Screenshots go in the PR body, never in the repository.
- User-visible changes go under `[Unreleased]` in `CHANGELOG.md`. The version lives only in `mygo.json`; change it
  with `python3 scripts/bump.py <version>`, which also dates the changelog.
- `README.md` (English) and `README.zh-CN.md` (Chinese) say the same thing; change both together. Feature status
  lists only merged work; planned work is marked as planned.
- Only MyGo native views and the native updater plugin. No WebView, HTML frontend, production cgo dependency or
  frontend toolchain.
- No disk, network or subprocess work in the draw path. Do slow work in the background and change state back on
  the UI thread.
- Calls to `ssh`, `git` and other system commands always have a timeout and pass arguments as a list, never
  through a shell.
- Path handling covers all three platforms: `~` expansion, Windows drive letters and backslashes, the trailing
  slash of `gitdir:`.
- UI copy lives in each screen package's `copy.json`, in `en` and `zh-Hans` only; change both together. Add no
  other languages.
- When behavior changes, update the existing tests for that area. Passing headless tests is not acceptance on a
  real machine.
- Do not split files just because they are long. Leave no temporary scripts, debug output or commented-out code.
- The MyGo library and CLI are one module; upgrade them together in `go.mod`. When dependencies change, update
  `THIRD_PARTY.md` and `resources/ThirdPartyLicenses/`.

## Red lines

- Never read, copy or log private key contents. Refer to private keys by path only.
- Write only to Idset's own directory. In the user's `~/.ssh/config` and `~/.gitconfig`, touch only the marker
  block (ADR 0002).
- Credentials (passphrases, tokens, signing keys) never appear in code, PRs, issues, logs, screenshots or chat.
  `scripts/scan_secrets.py` scans the whole history in CI.
- Releases, signing and anything sent to an external service need the maintainer's confirmation.
- Review the staged file list before committing; no blind `git add -A` of unknown files.
- Personal email addresses, local user names and local paths stay out of commit messages, files, test data and
  logs. Test data uses `example.com`.
