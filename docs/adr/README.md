# Architecture Decision Records

## Index

| No. | Decision | Status |
|---|---|---|
| [0001](0001-pure-go-native-ui.md) | Pure Go native UI | accepted |
| [0002](0002-managed-files-and-one-include.md) | Write only managed files and add one Include line to the user's configuration | accepted |
| [0003](0003-drop-zone-in-tray-window.md) | The drop zone lives in the window the tray icon opens | accepted |
| [0004](0004-system-ssh-and-git.md) | Run the system's ssh and git instead of reimplementing them in process | accepted |

## Record format

Each new decision gets a new file named `NNNN-short-title.md`: a four-digit number, lowercase words joined by
hyphens. When a decision is overturned, keep the old file and mark it "superseded by NNNN".

```markdown
# NNNN Title

- Status: proposed | accepted | superseded by NNNN
- Date: YYYY-MM-DD
- Related: optional links to the docs the decision affects

## Context
## Decision
## Cost
## When to revisit
```
