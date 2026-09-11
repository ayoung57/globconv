# globconv

Converts glob pattern lists between `.gitignore` syntax and rsync filter-rule
syntax.

These two formats look almost identical (both use `*`, `**`, `?`, `[...]`,
leading/trailing `/`) but disagree on the details in ways that bite you
silently:

- Negation: gitignore prefixes a re-include pattern with `!`. rsync filter
  files prefix every line with `+ ` (include) or `- ` (exclude) instead.
- Comments: gitignore uses `#`. rsync filter files accept both `#` and `;`.
- `**`: both treat a lone `**` path segment as "any number of directories",
  but what happens when `**` is glued to other characters in the same
  segment (`a**b`) is fuzzy in both specs and not the same fuzzy.

Copy-pasting a `.gitignore` straight into an `--exclude-from`/filter file,
or the reverse, works often enough to seem safe and then quietly excludes
(or includes) the wrong files the day a pattern hits one of these edges.

`globconv` does the translation explicitly instead. By default it's strict:
any pattern it can't map onto the target format with the same meaning is a
hard error, naming the line and the reason. Pass `--lenient` to have it
make a best-effort rewrite instead, printing a warning for each one to
stderr.

## Usage

```
globconv --from gitignore --to rsync .gitignore > rsync.filter
globconv --from rsync --to gitignore rsync.filter > .gitignore

cat .gitignore | globconv --from gitignore --to rsync --lenient
```

Input is read from the files given on the command line, or from stdin if
none are given. Output goes to stdout; warnings and errors go to stderr.

### Example

`.gitignore`:

```
# build output
/dist/
*.log
!important.log
```

`globconv --from gitignore --to rsync .gitignore`:

```
# build output
- /dist/
- *.log
+ important.log
```

### Strict vs. lenient

```
$ echo 'src/a**b/*.go' | globconv --from gitignore --to rsync
globconv: line 1: segment "a**b" uses "**" outside of its own path segment
globconv: pass --lenient to convert anyway

$ echo 'src/a**b/*.go' | globconv --from gitignore --to rsync --lenient
globconv: line 1: collapsed "a**b" to "a*b" ("**" only has special meaning as a whole path segment)
- src/a*b/*.go
```

## Known limitations (first pass)

- Bracket expressions (`[abc]`, `[!abc]`) are passed through unchanged;
  the two formats are assumed compatible but this isn't verified pattern
  by pattern yet.
- Escaping inside a pattern beyond a leading `\#`/`\!` and a trailing
  `\ ` is not unescaped or re-escaped.
- rsync's per-rule modifiers (`-C`, `s`, anchoring with a second `/`, etc.)
  aren't recognized.

## License

MIT, see [LICENSE](LICENSE).
