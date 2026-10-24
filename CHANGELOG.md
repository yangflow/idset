# Changelog

release-please writes each release's section from the titles of the pull requests merged since the last one
(docs/release.md); do not edit it by hand.

## [0.2.1](https://github.com/yangflow/idset/compare/v0.2.0...v0.2.1) (2026-10-24)


### Fixed

* keep background loads and dialogs within MyGo's threading rules ([#67](https://github.com/yangflow/idset/issues/67)) ([729e762](https://github.com/yangflow/idset/commit/729e7629f7dbe73641cb40cea474fa03e0227ef9))

## [0.2.0] - 2026-10-19

### Changed

- The accent is now commit mint instead of orange, in tune with the app icon, and meets WCAG AA in light and dark.
- Windows and dialogs follow the macOS 26 glass look: 16 DIP corners, a light rim on the material, capsule buttons and
  a capsule filter with a focus ring.
- In the panel, the drop zone steps aside while a result shows, so the identity list stays in view, and the whole
  panel takes the next folder; the result card leads with the identity, puts the open and copy actions in its menu
  with "Choose Another Folder…", and opens its clone helper from a button.

### Fixed

- The accounts window opens in the language chosen in Settings, not always in the system's.
- The badge in the accounts window header has the same rounded corners as the one on the result card.

## [0.1.0] - 2026-10-19

### Added

- A tray icon opens a panel listing your identities, most recently used first, with a filter for three or more.
- Drop a folder into the panel, or choose one, to see which identity Git uses there.
- Screen readers announce the result for a dropped folder and a copied path, and can navigate the panel by its
  headings. Spacing and text sizes in the panel follow the theme and the desktop's text size.
- Every change to the identity list now regenerates the managed SSH and Git files and keeps the `Include` blocks in
  `~/.ssh/config` and `~/.gitconfig` current. Recording a use does not regenerate.
- The accounts window creates, edits and deletes identities. Click an identity in the panel to open it, or "Add
  Identity" for a new one. Saving (Ctrl or Command+S) checks the identity first,
  writes it in the background and regenerates the configuration, and says "Saved" or why it could not. A new
  identity starts with the name and email from `git config --global`.
- Deleting an identity asks first and lists what stays. "Also delete the private key" is off by default and cannot be
  checked when the identity has no key path, another identity uses the key, or the path is a symbolic link, is
  missing or is not a regular file. The identity goes first; only the key file is removed, never its `.pub` file,
  and a key that cannot be removed is reported without bringing the identity back.
- The result card for a dropped folder binds, unbinds and rebinds it. Without a match, tap an identity to bind the
  folder to it, or "New Identity for This Folder" to create one for this folder. With a match, "Unbind" (only when the
  identity lists this very folder) and "Rebind to" (a global identity first asks, since it then applies to this
  folder only). A bind that is already covered changes nothing and says which path covers it. When several identities
  cover a folder, the card lists them in the order Git reads them, and offers "Use Only <identity> Here" and removing
  a path from another one. Binding, rebinding and "Use Only" show a five second "Undo" bar; unbinding has none. Each
  of these regenerates the configuration once.
- When a dropped folder has no match but looks like an identity's folder that was renamed (same parent folder) or
  moved (same last folder name), and the old path is gone from the disk, a card above the result offers "Change to
  This Path", which replaces the old path in place and regenerates the configuration once, or "Keep Old Path".
- Long paths in the bind list, in the identity list and in the stale-path card are shortened in the middle only: the
  start ("~/" or a drive) and the last folder stay.
- After a match, the card reads the repository's effective `user.email` and `core.sshCommand` and lists what
  disagrees with the identity, one row each, email first: no `user.email` (warning), an email that belongs to another
  identity (error), an email no identity has (warning), or a `-i` key other than the identity's (error). The card's
  outline takes the worst severity and the reason is in the row's tooltip. Nothing is checked for an identity
  without an email or a key path.
- When the repository's origin is an HTTPS address, the matched card has a quiet "Remote is HTTPS" row with a button
  that copies `git remote set-url origin git@<alias>:<path>.git`. A clone helper at the bottom of the card, and under
  an identity row opened with its arrow (one at a time), takes `owner/repo`, an address or a whole `git clone`
  command and copies `git clone git@<alias>:<path>.git`; Enter copies too, and a copy counts as a use. The helper
  starts with the origin's `owner/repo`, else the owner prefix last copied for that identity, which is kept in
  `settings.json`. A prefix alone (`acme/`) cannot be copied and shows a hint. Identities without an SSH alias have
  neither.
- On macOS the Dock icon shows while the accounts window is open; closing the window does not quit Idset.
- The settings window opens from the gear in the panel title row or "Settings…" in the tray menu. General sets the
  language (follow the system, English or Simplified Chinese; it applies to every open window at once) and opens
  Idset at login. If macOS needs approval for the login item, a note says where to give it.
- Backups lists every backup, newest first, with its identities. Restore asks first and lists what comes back;
  Delete removes a backup. When `accounts.json` cannot be read, the panel's "Restore from Backup" opens the window
  on this list.
- Configuration shows whether `~/.ssh/config` and `~/.gitconfig` hold the Include block and removes both while
  keeping `accounts.json`.
- Each identity with an SSH alias is checked with `ssh -T` in the background when the panel opens; a passing check
  stands until the identity changes, a failed one is tried again after ten minutes, and "Check Again" in the footer
  checks them all at once. A dot at the end of each row shows the result; its tooltip says as whom it connected or
  why it failed.
- A failed check shows a bar under the identity with the reason and the next step: a missing key file; a key with a
  passphrase, which "Unlock with Keychain" loads into the agent on macOS (elsewhere, a copyable `ssh-add` command);
  a key the server rejects, with "Copy Public Key" and a link to the platform's SSH key settings; or a host that
  cannot be reached. When the matched repository's origin is HTTPS, "Add Rewrite" adds the identity's default URL
  rewrites. When `ssh` is not installed, one notice says how to get it.
- A summary under the list counts the errors and warnings (an identity that failed its check, an alias your own
  `~/.ssh/config` also defines, a dropped folder that disagrees with its identity, several global identities) and says how to look into each; it copies the
  `ssh -vT` command or regenerates the configuration. The tray icon gets a warning or error badge for the worst one.
- The Keys pane of the settings window generates an Ed25519 or RSA 4096 key in `~/.ssh/` with `ssh-keygen`, without
  a passphrase and never over an existing file. The result shows the public key to copy, the files and fingerprint,
  a link to the platform's SSH key settings, "Load into ssh-agent", and "Use for identity", which writes the key path
  and fingerprint to an identity or starts a new identity with them in the accounts window. "Generate Key" on a
  missing key's bar opens the pane filled in for that identity, and the new key goes to it at once.
- The Import pane of the settings window reads `~/.ssh/config` and the global Git configuration each time it opens and
  lists the identities found there: an SSH host with a key, joined with the Git name, email and folders that use it.
  Those Idset does not have yet are checked; only one global identity comes in as global, the others as scoped.
  "Import N Identities" adds them one by one, backing up first; the files are only read. The empty panel and the
  empty accounts window have "Import Existing".
- The about window shows the version, copyright and license, opens the GitHub repository and checks for updates. A
  copy that cannot update itself says so and points to GitHub. It opens from "About" in the tray menu, the About button
  in the panel footer and "About Idset" in the macOS app menu.
- Releases are built for macOS (signed and notarized), Windows and Linux, and installed copies update themselves
  from GitHub releases. On Linux, the Debian package needs `libayatana-appindicator3-1` for the tray.
- Idset has its own icon, a Git key on a graphite tile, in the Dock, the taskbar, the installers and the about
  window.

### Changed

- The panel is more compact. The title row and the footer are 36 DIPs high, and the footer has icon buttons for Add
  Identity and Quit, with tooltips. On result cards, open, copy, a "⋯" menu (rebind, and unbind when the identity lists
  this very folder) and clear are icon buttons next to the folder. The overlap card has a pin ("Use Only") and minus
  ("Remove") button on each row. Shorter labels replace long ones, and tooltips carry the full text: the one-line drop
  zone, "Covered by …", the stale-path prompt's "Update" and "Ignore", and the read-only and `git` errors.
- The tray icon is the product icon's key, solid and as tall as the menu bar's other icons, instead of an
  identity card.
- A new look: the panel and the windows are translucent glass over the desktop (the system material on macOS and
  Windows 11, a solid fill elsewhere or with reduced transparency), with Git orange as the accent, in light and dark.
  Dialogs open in a window of their own on the same material, menus are the system's, and text fields are lightly
  tinted instead of white. Text uses five sizes, and actions a common symbol names clearly are icon buttons
  with tooltips. Labels and notes are shorter.
