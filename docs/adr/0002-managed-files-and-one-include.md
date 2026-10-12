# 0002 Write only managed files and add one Include line to the user's configuration

- Status: accepted
- Date: 2026-10-12
- Related: [docs/product.md](../product.md), "Generated configuration" (生成配置)

## Context

The user's `~/.ssh/config` and `~/.gitconfig` are often written by hand, with comments and configuration from other
tools, and some live in a dotfiles repository. Rewriting these two files directly would, on any mistake, break the
user's whole SSH and Git setup, and would conflict with dotfiles managers.

## Decision

- Idset owns only its own directory (the path on each platform is in product.md). All identity configuration is
  generated into managed files in that directory, regenerated as a whole from `accounts.json` every time.
- In the user's `~/.ssh/config` and `~/.gitconfig`, Idset inserts only one marked block, which holds a single
  `Include` line (for Git, `[include] path = ...`). Insertion is idempotent: inserting a second time adds nothing.
- Removal deletes only the marked block and leaves the rest of the file alone; it does not delete `accounts.json` or
  private keys.
- Private keys are referred to by path only. Idset does not read, copy or upload their contents.

## Cost

In `~/.ssh/config` the `Include` must come first to apply to every `Host`, so the marked block is always inserted at
the top of the file. A `Host` or `includeIf` of the same name written by the user can conflict with the managed
files; diagnostics point this out, it cannot be fixed automatically.

## When to revisit

`ssh` or `git` changes the meaning of `Include`, or `ssh` on some platform does not support `Include`.
