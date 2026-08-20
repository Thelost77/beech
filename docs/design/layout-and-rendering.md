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

## Node editing

Node titles are edited on a bottom input line, following h-m-m. While the
input is active, the map keeps the committed layout and viewport: typing,
cursor movement, and cursor blink never recompute geometry or scroll. The
input scrolls horizontally when the value exceeds the terminal width. Enter
applies the value and rebuilds the layout once; Esc cancels and restores the
previous document state.

Node editing and Save As use separate text input models.

## Rendering

All map content is written through the grapheme-aware `ui.Canvas`. Do not
splice styled ANSI fragments into rendered canvas rows.

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
