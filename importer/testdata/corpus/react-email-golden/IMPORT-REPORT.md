# Import report: react-email.html

gsxmail import read react-email.html and produced a best-effort template. This report lists every mapping decision, every node it could not place, and what to review before you ship the result.

## Mapped nodes

| Path | Component | Confidence | Note |
|---|---|---|---|
| card/row[1] | `email.Headline` | medium | a large or bold heading led this row, matching the Headline contract |
| card/row[2-3] | `email.Panel` | high | a run of the card's own top-level two-cell rows matched the label/value Panel contract, with no wrapping sub-table |
| card/row[4] | `email.Button` | high | a background-colored button face wrapping the anchor matched the primary Button contract |
| card/row[5] | `email.Divider` | high | a content-free border-top rule matched the Divider contract |

## Unmapped nodes

Each node below survives in `template.gsx` inside an `email.Custom` block, unchanged. Review it and hand-place it onto a named component if one fits.

| Path | Reason |
|---|---|
| card/row[6] | no component fingerprint matched this row; preserved as email.Custom |

## Synthesized props fields

| Field | Type | Source |
|---|---|---|
| `Lede` | `string` | the headline's own lede paragraph |
| `OneTimeCode` | `string` | the Panel row labeled "ONE-TIME CODE" |
| `Expires` | `string` | the Panel row labeled "EXPIRES" |
| `Title` | `string` | the document's own <title> |
| `Wordmark` | `string` | the Shell header's own wordmark text |
| `Preheader` | `string` | the hidden inbox-preview div's own text |

## Theme extraction

- ColorCard: read from the card table's own background-color.
- ColorGround: read from the first wrapping table's own background-color.
- ColorAccent: read from the first button-shaped anchor's own background-color.
- ColorInk: read from the first heading-shaped element's own text color.
- ColorPanel, ColorBody, ColorMuted, ColorFaint, and both fonts are not extracted in this release; they carry DefaultTheme()'s own values — review them against the source's own body copy and muted-label colors.

## Next steps

- Review the generated template.gsx and confirm every synthesized props field carries the value you expect.
