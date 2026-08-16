# Digest

A weekly digest: a retina hero image, a two-column fluid-hybrid
highlights row, a stats table, and a plain next-steps list.

## Components

`Shell`, `Hero` (the retina image), `Columns`/`Column` (fluid-hybrid, two
columns), `StatTable` (built from `<Each>`), `Divider`, `PickList`.

## Theme

`gsxmail.LedgerTheme()` — warm, print-like light, `DarkMode: "adaptive"`.
Digest is the gallery's adaptive-mode demonstration: its golden carries
the `@media (prefers-color-scheme:dark)` style layer, the
`gsx-ink`/`gsx-muted` class hooks (including on each Column's own title —
Columns nests ordinary card content, so the WP5.2 dark-mode hooks reach
it for free), and the `[data-ogsc]`/`[data-ogsb]` Outlook-app hooks.

## Files

- `digest.gsx` — the template.
- `digest.go` — `DigestProps`/`DigestStat`.
- `digest.props.json` — fixture props.
- `digest.html` / `digest.txt` — the golden render, `gsxmail.LedgerTheme()`.

## Render it

```go
set, err := gsxmail.Load(os.DirFS("digest"), gsxmail.Options{Theme: gsxmail.LedgerTheme()})
parts, err := set.Render("DigestEmail", props)
```
