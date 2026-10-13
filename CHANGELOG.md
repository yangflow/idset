# Changelog

All notable user-facing changes are recorded here, following [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

### Added

- A tray icon opens a panel listing your identities, most recently used first, with a filter for three or more.
- Drop a folder into the panel, or choose one, to see which identity Git uses there.
- Screen readers announce the result for a dropped folder and a copied path, and can navigate the panel by its
  headings. Spacing and text sizes in the panel follow the theme and the desktop's text size.
- Every change to the identity list now regenerates the managed SSH and Git files and keeps the `Include` blocks in
  `~/.ssh/config` and `~/.gitconfig` current. Recording a use does not regenerate. No screen edits identities yet,
  so this takes effect with the screens that follow.
