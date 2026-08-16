// Package doc defines EmailDoc: the typed block tree gsxmail lowers every
// template to. lower.Lower produces an EmailDoc from a compiled gosx
// component. EmailDoc.Resolve evaluates every expression hole against a
// concrete props value and returns a Resolved tree that the HTML and text
// writers consume. Splitting EmailDoc (expressions, built once per Set) from
// Resolved (plain strings, built once per Render call) keeps Render pure and
// avoids re-parsing template source on every call.
package doc

// EmailDoc is one compiled template: a Shell plus its ordered Blocks. It
// holds no props value, so one EmailDoc renders any number of times with
// different props.
type EmailDoc struct {
	Shell  Shell
	Blocks []Block
}

// Shell carries the frame fields every email shares: header wordmark and
// badge, document title, and language.
type Shell struct {
	Wordmark  Expr
	ShortCode Expr
	Tagline   Expr
	Title     Expr
	Lang      Expr
}

// Block is one content primitive inside a Shell. The set of implementations
// is closed to this package's own types (email.Signal, email.Headline, and
// so on map one-to-one to a Block type), matching the stdlib component list
// WP1 ships (spec section 15, WP1).
type Block interface {
	isBlock()
}

// Signal is the "● LABEL" line under the header (email.Signal).
type Signal struct {
	Text Expr
}

func (Signal) isBlock() {}

// Headline is the big uppercase title plus a lede sentence (email.Headline).
type Headline struct {
	Title Expr
	Lede  Expr
}

func (Headline) isBlock() {}

// PanelRow is one labeled fact inside a Panel (email.PanelRow).
type PanelRow struct {
	Label Expr
	Value Expr
}

// Panel is the bordered info card that holds PanelRows (email.Panel).
type Panel struct {
	Rows []PanelRow
}

func (Panel) isBlock() {}

// CTA is the single call-to-action button (email.CTA).
type CTA struct {
	Label Expr
	Href  Expr
}

func (CTA) isBlock() {}

// Item is one line of a PickList (email.Item). Its content is the element's
// inline text/expression children, concatenated in source order.
type Item struct {
	Text Expr
}

// PickList is the numbered next-steps list (email.PickList).
type PickList struct {
	Title Expr
	Items []Item
}

func (PickList) isBlock() {}

// Footer is the closing signoff plus a small print note (email.Footer).
type Footer struct {
	Signoff Expr
	Note    Expr
}

func (Footer) isBlock() {}
