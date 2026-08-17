// Package receipt is the gallery's complete worked example: Shell,
// Badge, Headline, StatTable (+Each), Panel (totals), Button, Footer.
//
// Deviation from an earlier worked example (documented, since its
// byte-pinned goldens (TestRecapGolden) must not move): that earlier
// example writes <email.StatRow cells={[item.Name, item.Qty,
// item.Amount]} /> and header={["ITEM", "QTY", "AMOUNT"]} — inline
// slice literals. The shipped
// StatTable/StatRow API only accepts a bare props- or binding-rooted
// field path for header/cells (lower.parseFieldPathAttr; the same shape
// testdata/emails/recap.gsx's own header={props.HaulHeader} already
// uses), not a computed literal. ReceiptItem therefore carries a Cells
// []string field (mirroring recap.go's own HaulRow.Cells), and
// ReceiptProps carries a Header []string field, instead of the dossier's
// three separate Name/Qty/Amount fields and inline header array — the
// same rendered content and shape, fitted to the real API.
package receipt

// ReceiptItem is one purchased line item: its StatTable row cells, in
// column order (name, quantity, amount). All fields are pre-formatted
// strings; the caller owns currency formatting.
type ReceiptItem struct {
	Cells []string // ["Standard seat", "1", "$80.00"]
}

// ReceiptProps fills ReceiptEmail.
type ReceiptProps struct {
	Product    string   // "ACME"
	OrderID    string   // "1042"
	IssuedOn   string   // "August 14, 2026"
	BilledTo   string   // "ada@example.com"
	Header     []string // ["ITEM", "QTY", "AMOUNT"]
	Items      []ReceiptItem
	Subtotal   string // "$80.00"
	Tax        string // "$6.00"
	Total      string // "$86.00"
	ReceiptURL string // https enforced by EM110/EM111
}
