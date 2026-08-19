# Markdown format

Beech stores maps as a strict Markdown outline. Each node is one unordered list
item, and each tree level adds one tab.

```md
<!-- beech:outline v1 -->

- Project
	- Research <!-- beech:collapsed -->
		- Read existing tools
	- Implementation
```

## Decision

Use Markdown as the only native writable format.

Use one tab per level and `- ` for every node. Do not write space-indented tree
levels.

Store collapsed state with the exact trailing marker:

```md
<!-- beech:collapsed -->
```

The format marker and collapse markers are HTML comments. Markdown renderers
hide them while preserving the nested list.

## Supported scope

Beech accepts:

- UTF-8 text
- zero or one `<!-- beech:outline v1 -->` marker before the outline
- blank lines
- tab-indented unordered list items
- one logical line per node
- inline task markers, emphasis, code, links, and tags as node text
- exact Beech collapse markers

Beech rejects unsupported Markdown with a line-specific error. It does not
silently remove paragraphs, headings, code blocks, tables, multiline list
items, ordered lists, or space-indented tree levels.

The serializer preserves LF or CRLF and whether the source ended with a final
newline. It writes the format marker and canonical tab-indented list body.

## Collapse state

Collapse is view state stored in the document. Toggling a branch therefore
marks the document modified. Saving adds or removes the marker on the branch's
list item. Reopening the file restores the fold.

Selection, viewport position, search state, and transient action feedback are
not persisted.

## h-m-m import

Existing `.hmm` files are import-only. Beech reads their indentation tree into
memory, leaves the source file untouched, and proposes an adjacent `.md` path.
Beech does not create, update, or export `.hmm` files.

Imported files may use tabs or consistent spaces because this tolerance belongs
to the one-way importer, not the native Markdown contract.
