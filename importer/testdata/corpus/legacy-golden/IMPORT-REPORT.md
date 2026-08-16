# Import report: legacy.html

gsxmail import read legacy.html and produced a best-effort template. This report lists every mapping decision, every node it could not place, and what to review before you ship the result.

## Mapped nodes

| Path | Component | Confidence | Note |
|---|---|---|---|
| card/row[1] | `email.Headline` | medium | a large or bold heading led this row, matching the Headline contract |
| card/row[2-3] | `email.Panel` | high | a run of the card's own top-level two-cell rows matched the label/value Panel contract, with no wrapping sub-table |
| card/row[4] | `email.Button` | low | a lone anchor filled the row with no other content, and was assumed to be a call-to-action button |
| card/row[5] | `email.Hero` | medium | a lone, sized image matched the Hero contract |
| card/row[6] | `email.Divider` | high | a content-free border-top rule matched the Divider contract |

## Unmapped nodes

Each node below survives in `template.gsx` inside an `email.Custom` block, unchanged. Review it and hand-place it onto a named component if one fits.

| Path | Reason |
|---|---|
| card/row[7] | no component fingerprint matched this row; preserved as email.Custom |

## Synthesized props fields

| Field | Type | Source |
|---|---|---|
| `Lede` | `string` | the headline's own lede paragraph |
| `Issue` | `string` | the Panel row labeled "ISSUE" |
| `Date` | `string` | the Panel row labeled "DATE" |
| `Title` | `string` | the document's own <title> |
| `Wordmark` | `string` | the Shell header's own wordmark text |

## Theme extraction

- ColorCard: no literal background-color found on the card table; kept the default.
- ColorGround: no wrapping table with a literal background-color was found; kept the default.
- ColorAccent: no button-shaped anchor with a literal background-color was found; kept the default.
- ColorInk: no heading-shaped element with a literal text color was found; kept the default.
- ColorPanel, ColorBody, ColorMuted, ColorFaint, and both fonts are not extracted in this release; they carry DefaultTheme()'s own values — review them against the source's own body copy and muted-label colors.

## Next steps

- card/row[5]: the source <img> had no width/height attributes; Hero needs both, so gsxmail import guessed 600x300 — set the real display size.
- card/row[5]: the source image "cid:logo.png" is not an absolute https URL; replaced it with a placeholder — point it at the real asset
