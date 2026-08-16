# The gsxmail template gallery

Five complete templates (pixel dossier section 8.1), each with typed
props, a `.gsx` source, a fixture props JSON file, a byte-exact golden
HTML/text pair, and its own README. Every template renders through the
same `Load`/`Render` pipeline as any consumer's own code — nothing here
is special-cased.

| Template | Components exercised | Theme | Fixture highlight |
|---|---|---|---|
| [`welcome/`](welcome) | Shell, Headline, PickList, Button, Footer | Paper (default) | Preheader, hardened head |
| [`magiclink/`](magiclink) | Shell, Headline, Panel, Note, Button | Paper (default) | OTP code as text, never an image |
| [`receipt/`](receipt) | Shell, Badge, Headline, StatTable (+Each), Panel, Button, Footer | Paper (default) | The pixel dossier's complete worked example |
| [`digest/`](digest) | Shell, Hero, Columns, StatTable, Divider, PickList | Ledger (adaptive) | Fluid-hybrid columns, retina hero, adaptive dark mode |
| [`alert/`](alert) | Shell, Signal, Badge, Note, Button | Terminal (locked dark) | Severity as text and structure, dark-native theme |

Each directory is a self-contained `fs.FS` root: `gsxmail.Load` reads
only its own `.gsx` and `.go` files, so any one template loads and
renders independently. `examples/gallery/gallery_test.go` loads all five,
renders each against its own fixture, pins the golden pair, and runs the
structural verification pass over the whole gallery in both output
contracts (hardened and parity).

## A rendered snippet

`receipt/receipt.gsx` renders its `Badge` and `Button` like this
(hardened mode, `Paper` theme):

```html
<span style="display:inline-block; padding:2px 8px; border:1px solid #2F9E44; border-radius:2px; color:#2F9E44; font-family:'SFMono-Regular',Consolas,Menlo,monospace; font-size:10px; letter-spacing:0.06em; text-transform:uppercase;">PAID</span>
...
<a href="https://acme.example/r/1042" style="display:inline-block; padding:14px 30px; mso-padding-alt:0; color:#F4F4F6; font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif; font-size:14px; font-weight:800; letter-spacing:0.04em; text-transform:uppercase; text-decoration:none; border-radius:2px;">VIEW RECEIPT →</a>
```

and its text twin:

```
[PAID]
...
  -> VIEW RECEIPT: https://acme.example/r/1042
```

See `receipt/receipt.html` and `receipt/receipt.txt` for the complete
render, and `receipt/README.md` for the one documented deviation from
the dossier's own worked example (a `StatTable` API mismatch — `StatTable`
itself is out of this work package's scope).

## Named themes

`digest/` and `alert/` each render under one of the two WP5.3 named
themes (`gsxmail.LedgerTheme()` and `gsxmail.TerminalTheme()`); `welcome/`,
`magiclink/`, and `receipt/` use `gsxmail.DefaultTheme()` ("Paper"). See
the root README's "Named themes" section for both palettes.
