# Theme and highlighting

Beech uses a curated Everforest Dark palette.

## Structural grammar

Color and weight have separate meanings:

- Connector hue identifies a top-level branch.
- A first-level branch label uses the same hue and is bold.
- Deeper nodes with children are neutral and bold.
- Leaves are neutral with regular weight.
- Root nodes are green and bold.
- The active path emphasizes connectors without recoloring unrelated content.

This avoids using branch color to also imply whether a node has children.

## Markdown highlighting

Highlight these one-line forms without changing source text:

- `[ ]` pending tasks in yellow
- `[x]` completed tasks in green
- `**strong**` text in bold
- inline code in orange
- links in teal and underlined
- `#tags` in purple

Malformed or unsupported inline syntax remains ordinary text.

## Visual priority

Apply styles in this order:

1. selection background
2. transient action feedback
3. Markdown token style
4. structural role
5. default foreground

Copy, create, edit, move, fold, delete, and undo feedback use distinct temporary
colors. Feedback timers are generation-safe so an older expiry cannot clear a
newer action.

## Scope

Version `v0.1.0` has no runtime theme registry, named theme presets, or dynamic
theme switching. Keep one coherent default until real use demonstrates a need
for user-configurable colors.
