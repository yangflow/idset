# Security policy

## Reporting a vulnerability

Use GitHub private vulnerability reporting: <https://github.com/yangflow/idset/security/advisories/new>. Do not
open a public issue, discussion or pull request.

Include the version or commit, the operating system, how to reproduce it and what an attacker gains. Do not paste
private keys, real email addresses or host names; redact configuration files before sharing them.

## Scope

In scope:

- Updates: bypassing the signature or checksum of an update package, or installing an update from a non-HTTPS
  source.
- Private keys: key contents being read, copied, uploaded, or appearing in logs or crash reports; deleting a key
  by following a symlink, or deleting a key another identity still uses.
- Configuration files: Idset writing outside `~/.config/idset/`, or changing anything in `~/.ssh/config` or
  `~/.gitconfig` outside its managed block.
- Injection: identity fields (label, alias, email, path) injecting extra directives into the generated SSH or Git
  configuration, or making Idset run arbitrary commands.

Out of scope:

- Vulnerabilities in GitHub, GitLab or other hosting platforms, or in the system `ssh` and `git` themselves.
- Attacks that first require local user access.
- Unsigned development builds being unsigned.

## What to expect

One maintainer handles reports in spare time. Expect an acknowledgement within about a week. Confirmed
vulnerabilities are fixed in a new release and published as a GitHub security advisory, crediting the reporter by
default. There is no bounty.

## Supported versions

Only the latest release is supported.
