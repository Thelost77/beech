# File safety

Beech treats the opened Markdown file as the source of truth and refuses silent
external overwrites.

## Atomic replacement

Save data to a temporary file in the target directory. Set the intended file
mode, write the complete content, flush it, close it, and then rename it over
the target.

Creating the temporary file beside the target keeps the replacement on the same
filesystem. On Unix-like systems, existing file permissions are preserved and
new files use `0644` permissions. Windows does not enforce Unix permission bits.

## External-change protection

On load and after each successful save, record a SHA-256 fingerprint of the
source content. Before replacing the file, read it again and compare its current
fingerprint with the expected one.

If another process changed the file, return a conflict and keep the in-memory
Beech document dirty. Never overwrite the external content automatically.

## Symlinks

Resolve an existing symlink when loading so an atomic save updates its target
instead of replacing the link itself.

## Quit behavior

- `q` saves dirty work before quitting.
- `ctrl+c` follows the same safe path.
- `Q` quits immediately and discards in-memory changes.
- `esc` cancels the current mode and never quits.

## h-m-m imports

Opening an existing `.hmm` file does not make it the active save target. Beech
holds the imported tree as a new dirty document and preselects the adjacent
`.md` name in Save As. The legacy source remains unchanged even when the user
edits and saves the imported map.
