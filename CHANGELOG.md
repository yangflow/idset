# Changelog

All notable user-facing changes are recorded here, following [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

### Added

- A tray icon opens a panel listing your identities, most recently used first, with a filter for three or more.
- Drop a folder into the panel, or choose one, to see which identity Git uses there.
- Screen readers announce the result for a dropped folder and a copied path, and can navigate the panel by its
  headings. Spacing and text sizes in the panel follow the theme and the desktop's text size.
- Every change to the identity list now regenerates the managed SSH and Git files and keeps the `Include` blocks in
  `~/.ssh/config` and `~/.gitconfig` current. Recording a use does not regenerate.
- The accounts window creates, edits and deletes identities. Click an identity in the panel to open it, or "Add
  Identity" for a new one. Saving (Ctrl or Command+S) checks the identity first,
  writes it in the background and regenerates the configuration, and says "Written" or why it could not. A new
  identity starts with the name and email from `git config --global`.
- Deleting an identity asks first and lists what stays. "Also delete the private key" is off by default and cannot be
  checked when the identity has no key path, another identity uses the key, or the path is a symbolic link, is
  missing or is not a regular file. The identity goes first; only the key file is removed, never its `.pub` file,
  and a key that cannot be removed is reported without bringing the identity back.
- The result card for a dropped folder binds, unbinds and rebinds it. Without a match, tap an identity to bind the
  folder to it, or "New Identity and Bind Here" to create one for this folder. With a match, "Unbind" (only when the
  identity lists this very folder) and "Rebind to" (a global identity first asks, since it then applies to this
  folder only). A bind that is already covered changes nothing and says which path covers it. When several identities
  cover a folder, the card lists them in the order Git reads them, and offers "Use only <identity> here" and removing
  a path from another one. Binding, rebinding and "Use only" show a five second "Undo" bar; unbinding has none. Each
  of these regenerates the configuration once.
- On macOS the Dock icon shows while the accounts window is open; closing the window does not quit Idset.
