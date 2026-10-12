# 0004 Run the system's ssh and git instead of reimplementing them in process

- Status: accepted
- Date: 2026-10-12
- Related: [docs/product.md](../product.md), "Connectivity check" (连通性检查) and "Keys" (密钥)

## Context

Checking connectivity, generating keys, loading them into ssh-agent, and reading a repository's `user.email` and
remote URL can all be done in process with Go libraries (`golang.org/x/crypto/ssh`, go-git), or by running the
system's `ssh`, `ssh-keygen`, `ssh-add` and `git`.

## Decision

Run the system commands. A connectivity check has to take the same path as the user's `git push` in a terminal: the
same `~/.ssh/config`, the same agent, the same `ProxyCommand` and `Include`. An in-process implementation can hardly
match it exactly, and its results would differ from what really happens. All three platforms ship these commands:
macOS and most Linux distributions install them by default, Windows 10/11 ships the OpenSSH client, and Git for
Windows brings its own.

- Every command has a timeout, and arguments are passed as a list, never assembled into a shell string.
- When a command is missing, diagnostics say which one and how to install it; everything else keeps working.

## Cost

Idset depends on the versions of the commands on the user's machine and has to handle their output across versions.
On Windows there may be both the system OpenSSH and the ssh that comes with Git, and Idset has to choose (follow
`git config core.sshCommand`, otherwise the first on PATH).

## When to revisit

The system commands are commonly missing on some platform, or their output changes so often that it is hard to keep
up.
