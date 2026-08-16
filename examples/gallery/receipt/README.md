# Receipt

An order receipt: a `PAID` badge, a line-item table, and a totals panel.
This template is the pixel dossier's complete worked example (section
8.3), reproduced from its own `.gsx` source and fixture.

## Components

`Shell`, `Badge` (tone `"positive"`), `Headline`, `StatTable` (built from
`<Each>` over line items), `Panel` (totals), `Button` (variant
`"primary"` — `email.CTA`'s own alias), `Footer`.

## A documented deviation from the dossier

The dossier's own `.gsx` source writes
`<email.StatRow cells={[item.Name, item.Qty, item.Amount]} />` and
`header={["ITEM", "QTY", "AMOUNT"]}` — inline slice literals. The shipped
`StatTable`/`StatRow` API only accepts a bare props- or binding-rooted
field path for `header`/`cells` (the same shape
`../../../testdata/emails/recap.gsx`'s own
`header={props.HaulHeader}` already uses), not a computed literal, and
`StatTable` itself is out of this work package's scope — its byte-pinned
goldens (`TestRecapGolden`) must not move. `ReceiptItem` here carries a
`Cells []string` field instead of separate `Name`/`Qty`/`Amount` fields,
and `ReceiptProps` carries a `Header []string` field, fitted to the real
API. The rendered content and shape match the dossier's own example; a
few pixel values (StatTable cell padding, header background) do not,
for the same reason — see the CHANGELOG's WP5.3 entry.

## Files

- `receipt.gsx` — the template.
- `receipt.go` — `ReceiptProps`/`ReceiptItem`.
- `receipt.props.json` — fixture props.
- `receipt.html` / `receipt.txt` — the golden render,
  `gsxmail.DefaultTheme()`.

## Render it

```go
set, err := gsxmail.Load(os.DirFS("receipt"), gsxmail.Options{})
parts, err := set.Render("ReceiptEmail", ReceiptProps{
    Product:  "ACME",
    OrderID:  "1042",
    IssuedOn: "August 14, 2026",
    BilledTo: "ada@example.com",
    Header:   []string{"ITEM", "QTY", "AMOUNT"},
    Items: []ReceiptItem{
        {Cells: []string{"Standard seat", "1", "$80.00"}},
        {Cells: []string{"Priority support", "1", "$0.00"}},
    },
    Subtotal:   "$80.00",
    Tax:        "$6.00",
    Total:      "$86.00",
    ReceiptURL: "https://acme.example/r/1042",
})
```
