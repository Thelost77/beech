# beech

A keyboard-first terminal mind map for structured Markdown notes.

Beech edits ordered trees as tab-indented Markdown lists and renders them as a
compact left-to-right mind map. Files remain readable in ordinary Markdown
viewers and version control.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Features

- Open or create Markdown outlines
- Create and edit nodes on a bottom input line, h-m-m style
- Cut, copy, paste, reorder, promote, and demote complete subtrees
- Navigate spatially with Vim-style keys or arrows
- Collapse branches and persist their state in the Markdown file
- Undo and redo document changes
- Colored action feedback and explicit no-op messages
- Compact parent-relative layout with unrelated-branch isolation
- Everforest branch colors and Markdown syntax highlighting
- Unicode and grapheme-aware terminal rendering
- Atomic saves with external-change protection
- One-way import of legacy h-m-m files
- No external runtime dependencies

## Requirements

- Go 1.25 or newer for `go install` or source builds
- A Unicode-capable terminal

## Installation

```sh
go install github.com/Thelost77/beech@latest
```

Ensure the Go binary directory, normally `~/go/bin`, is in `PATH`.

To build the current checkout:

```sh
go build -o beech .
```

## Usage

```sh
beech notes.md      # open or create a Markdown outline
beech notes         # create notes.md
beech legacy.hmm    # import an existing h-m-m file
beech                # start an untitled map
beech --version
```

An imported `.hmm` file remains unchanged. Beech treats the result as a new,
unsaved document and proposes an adjacent `.md` filename.

## Common keybindings

Keys are context-specific. Press `?` in Beech for the complete active help.

| Key | Action |
|-----|--------|
| `h/j/k/l`, arrows | Navigate parent/down/up/child |
| `enter`, `o` | Create a sibling |
| `tab`, `O` | Create a child |
| `e`, `i` | Edit the selected node |
| `J`, `K` | Move the node down or up |
| `[`, `]` | Promote or demote the node |
| `d` | Cut a subtree |
| `y` | Copy a subtree |
| `p`, `P` | Paste as child or sibling |
| `space` | Collapse or expand |
| `u`, `ctrl+r` | Undo or redo |
| `c` | Center the selected node |
| `g` | Select the first root |
| `ctrl+arrows` | Pan the viewport |
| `s` | Save |
| `q`, `ctrl+c` | Save and quit |
| `Q` | Discard changes and quit |
| `?` | Toggle help |

Node editing opens a full-width input line at the bottom of the screen,
following h-m-m. The map stays frozen while you type and updates once when
you press Enter. Esc cancels and restores the original state.

## Markdown format

Beech writes a strict, one-line-per-node Markdown subset. Every node is an
unordered list item, and each tree level uses one tab:

```md
<!-- beech:outline v1 -->

- Project
	- Research <!-- beech:collapsed -->
		- Read existing tools
		- Define interactions
	- Implementation
		- Tree operations
		- Terminal renderer
```

The format and collapse comments are valid Markdown and hidden when rendered on
GitHub. Beech preserves newline style and the final newline. Unsupported
Markdown is rejected instead of being silently rewritten.

See [Markdown format](docs/design/markdown-format.md) for the complete contract.

## Highlighting and layout

First-level branch labels and connectors use a stable Everforest palette. Roots
are green, deeper parent nodes are neutral and bold, and leaves are neutral with
regular weight. Beech also highlights task markers, strong text, inline code,
links, and tags without changing source text.

The layout is parent-relative: a long node may move its descendants but never an
unrelated branch. See [Layout and rendering](docs/design/layout-and-rendering.md)
and [Theme and highlighting](docs/design/theme-and-highlighting.md).

## Development

```sh
go mod verify
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
go build ./...
```

Main packages:

```text
main.go                command-line entry point
internal/app/          Bubble Tea state, actions, navigation, and rendering
internal/buildinfo/    installed module version reporting
internal/document/     ordered-tree model and structural edits
internal/highlight/    one-line Markdown token classification
internal/layout/       tree measurement, placement, and connectors
internal/outline/      native Markdown parser and h-m-m importer
internal/storage/      fingerprints, conflict checks, and atomic saves
internal/ui/           Everforest styles and grapheme-aware canvas
```

## Releases

Beech uses SemVer tags with a leading `v`. Release notes live in
`docs/releases/`. Commit and push all release changes to `main`, then run:

```sh
./scripts/release.sh v0.1.0
```

The script verifies the clean repository, tests, race detector, vet, build,
tag, and GitHub Release. It can safely retry release creation after a partial
failure.

## Status

Active development. Version `v0.1.0` is the first public release. The core
Markdown editor, tree operations, persisted folds, layout, and safe save path
are functional.

## Acknowledgements

Inspired by [h-m-m](https://github.com/nadrad/h-m-m), a keyboard-centric
terminal mind mapper.

Beech is an independent implementation. It can import legacy `.hmm` files but
does not modify or write that format.

## License

[MIT](LICENSE)
