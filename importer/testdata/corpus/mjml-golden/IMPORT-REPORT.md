# Import report: mjml.html

gsxmail import read mjml.html and produced a best-effort template. This report lists every mapping decision, every node it could not place, and what to review before you ship the result.

## Mapped nodes

| Path | Component | Confidence | Note |
|---|---|---|---|
| card/row[1] | `email.Headline` | medium | a large or bold heading led this row, matching the Headline contract |
| card/row[2] | `email.Columns` | medium | 2-4 sibling fluid-hybrid divs matched the Columns contract |
| card/row[3] | `email.Button` | high | a background-colored button face wrapping the anchor matched the primary Button contract |

## Unmapped nodes

Each node below survives in `template.gsx` inside an `email.Custom` block, unchanged. Review it and hand-place it onto a named component if one fits.

| Path | Reason |
|---|---|
| card/row[4] | no component fingerprint matched this row; preserved as email.Custom |

## Synthesized props fields

| Field | Type | Source |
|---|---|---|
| `Lede` | `string` | the headline's own lede paragraph |
| `Title` | `string` | the document's own <title> |
| `Wordmark` | `string` | the Shell header's own wordmark text |

## Theme extraction

- ColorCard: no literal background-color found on the card table; kept the default.
- ColorGround: no wrapping table with a literal background-color was found; kept the default.
- ColorAccent: read from the first button-shaped anchor's own background-color.
- ColorInk: read from the first heading-shaped element's own text color.
- ColorPanel, ColorBody, ColorMuted, ColorFaint, and both fonts are not extracted in this release; they carry DefaultTheme()'s own values — review them against the source's own body copy and muted-label colors.

## Next steps

- Review the generated template.gsx and confirm every synthesized props field carries the value you expect.
