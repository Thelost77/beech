# Layout and rendering

Beech renders an ordered tree as a compact left-to-right mind map.

## Horizontal layout

Place each child relative to its parent:

```text
child.x = parent.x + parent.width + level_gap
```

Do not align every node at one depth to a global column. A long node may move
its own descendants, but it must never move an unrelated branch.

## Vertical layout

Measure visible subtree heights from the leaves upward. Stack sibling subtrees
with a fixed gap, then center each parent between the first and last visible
child. When the complete map is shorter than the content area, center it
vertically in the terminal.

Collapsed descendants do not participate in visible layout. The collapsed node
shows a `▸ N` marker where `N` is its hidden descendant count.

## Inline editing

Inline editing uses the same wrapping function and 42-cell text limit as the
committed layout.

The editor starts at the current wrapped value plus one cursor cell. Inserting
or deleting text recomputes the transient edit layout. Crossing the 42-cell
boundary adds or removes a row immediately. Only the edited node's descendants
may move horizontally; vertical changes follow normal subtree measurement.

Cursor movement and cursor blink messages must not recompute geometry. Enter
commits the displayed preview. Esc restores the original document and layout.

Node editing and Save As use separate text input models.

## Rendering

All map content, including inline input and its cursor, is written through the
grapheme-aware `ui.Canvas`. Do not splice styled ANSI fragments into rendered
canvas rows.

Every frame has exactly the terminal height, and every row stays below the full
terminal width to avoid bottom-right automatic wrapping.

Canvas operations must:

- preserve grapheme clusters
- account for wide terminal cells
- clear every cell occupied by an overwritten grapheme
- merge overlapping connector directions
- keep styling independent from geometry

## Color grammar

- Roots are green and bold.
- First-level branch labels and their connectors use one branch color.
- Deeper parent nodes are neutral and bold.
- Leaves are neutral with regular weight.
- Selection uses a high-contrast background.
- Markdown tokens provide syntax color.
- Action feedback temporarily overrides structural color.
