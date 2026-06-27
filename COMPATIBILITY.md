# Ls Command Compatibility

## Summary

Partially compatible with GNU `ls`: the default one-name-per-line listing matches `ls -1` exactly, but `-a`, `-R`, and `-l` diverge by design (see below).

## Key Behaviors

```bash
# Default: one entry per line, sorted, hidden (dot) entries excluded.
# Byte-identical to GNU `ls -1`.
$ ls /dir
alpha.txt
bravo.txt
sized.txt
sub

# With no operand, lists the current directory.
$ ls
alpha.txt
bravo.txt
sized.txt
sub

# -a: also list dot entries (but no synthetic "." / "..").
$ ls -a /dir
.hidden
alpha.txt
bravo.txt
sized.txt
sub

# -R: recurse, emitting every entry as a flat path relative to the listing root.
$ ls -R /dir
alpha.txt
bravo.txt
sized.txt
sub
sub/b.txt

# -l: long format, "<perm> <size> <name>" only.
$ ls -l /only
-rw-r--r-- 5 sized.txt
```

## Intentional Divergences

- `-a` lists dot-prefixed entries but omits the synthetic `.` and `..` entries that GNU `ls -a` prints; `afero.ReadDir` does not surface them.
- `-R` emits a single flat list of paths relative to the listing root, sorted lexically. It does not print the per-directory `dir:` headers or the blank-line grouping that GNU `ls -R` produces.
- `-l` long format emits only `<perm> <size> <name>` — no owner, group, link count, or modification time, and no leading `total` line. `afero.Fs` does not surface owner/group or a uniform mtime across its backends (the in-memory and OS filesystems disagree), so ls emits only the portable, deterministic subset: file mode and byte size.
- Default output has no column or color formatting, so it is equivalent to GNU `ls -1` rather than the multi-column, colorized default. Entries are emitted in the lexical order `afero.ReadDir`/`afero.Walk` guarantee (sorted by name); ls does not re-sort.
