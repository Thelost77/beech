# AGENTS.md

## Project overview

Beech is a local-first terminal mind-map editor written in Go with Bubble Tea.
It edits ordered trees stored as tab-indented Markdown list items. Existing
`.hmm` files are import-only and are never modified.

## Commands

```sh
go build -o beech .
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
```

## Architecture

| Package | Responsibility |
|---|---|
| `internal/document` | Ordered-tree model, structural edits, clone, validation |
| `internal/outline` | Parse native Markdown, import h-m-m, serialize Markdown |
| `internal/layout` | Pure left-to-right tree measurement and placement |
| `internal/highlight` | Pure one-line Markdown token classification |
| `internal/storage` | File loading, fingerprints, conflict checks, atomic saves |
| `internal/ui` | Everforest styles and grapheme-aware terminal canvas |
| `internal/app` | Bubble Tea state, input modes, history, navigation, rendering |
| `internal/buildinfo` | Installed module version reporting |

## Contracts

- Keep `document`, `outline`, and `layout` independent of Bubble Tea.
- Treat each document mutation as one undoable transaction.
- Never overwrite a file whose content differs from its loaded fingerprint.
- Write only `.md` documents. Leave imported `.hmm` sources untouched.
- Use one tab per Markdown tree level and `- ` for every node.
- Persist collapsed branches with the reserved hidden Markdown comment.
- Keep rendered frames within `terminal width - 1` and at exactly the terminal
  height.
- Handle text by grapheme width. Add Unicode tests for rendering changes.
- Route action pulses through the generic feedback manager. Do not add
  action-specific flash state.
- Resolve visual priority as selection, feedback, Markdown token, structure.
- Position children relative to their parent. A node in one branch must never
  shift an unrelated branch horizontally.
- Render inline editing through `ui.Canvas`. Do not splice styled ANSI strings
  into rendered canvas rows.
- Size an inline editor to the current wrapped value plus one cursor cell.
  Wrap at the same 42-cell limit as committed nodes and recompute its edit
  layout only when the value changes. Cursor movement and timer events must not
  affect geometry. Parent-relative layout must keep unrelated branches fixed
  while edited descendants move.
- Keep node editing and Save As in separate text input models.
- Capture values in asynchronous commands. Do not capture the application model.
- `esc` closes or cancels the active mode. It never quits.
- Lowercase `q` saves before quitting. Uppercase `Q` quits immediately and
  discards in-memory changes.
- The internal clipboard must work without an external clipboard program.
