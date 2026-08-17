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
// badge, document title, and language, plus two per-Shell options:
//
//   - Preheader is the hidden inbox-preview text, authored on the Shell
//     tag itself (like MJML's mj-preview) rather than passed at Render
//     time — the emailkit precedent kept subjects out of templates, but a
//     preheader is presentation, not delivery metadata, so it stays
//     template-owned and can read props like any other Shell field.
//   - Outlook is a plain string, not an Expr: the ghost-table output
//     contract is a structural, compile-time choice a template author
//     makes once, never a value computed from props. "" defers to the
//     Set-level Options.Outlook default; "ghost-tables" or "off" override
//     it for this one template. lower.lowerShell reads it as a static
//     attribute value.
type Shell struct {
	Wordmark  Expr
	ShortCode Expr
	Tagline   Expr
	Title     Expr
	Lang      Expr
	Preheader Expr
	Outlook   string
}

// Block is one content primitive inside a Shell. The set of implementations
// is closed to this package's own types (email.Signal, email.Headline, and
// so on map one-to-one to a Block type), matching the stdlib component
// list gsxmail ships.
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

// Note is one body paragraph (email.Note).
type Note struct {
	Text Expr
}

func (Note) isBlock() {}

// Divider is a border-top rule with no fields of its own (email.Divider).
type Divider struct{}

func (Divider) isBlock() {}

// StatRow is one row of a StatTable: its cells (a props- or
// binding-rooted slice of strings, from a StatRow's own cells attribute)
// and whether it is the marked row (mark; see StatTable.Rows and
// ResolvedStatTable.MarkRow for how per-row marks become emailkit's
// single 1-based MarkRow).
type StatRow struct {
	Cells FieldPath
	Mark  Expr
}

// EachRows is <Each>'s shape inside a StatTable body: it binds As to each
// element of the slice at Of and resolves Row once per element, in order.
type EachRows struct {
	Of  FieldPath
	As  string
	Row StatRow
}

// RowSpec is one entry in a StatTable's Rows list, in source order:
// exactly one of Literal or Each is set. A Literal entry is one authored
// <email.StatRow> (its cells and mark always read directly from props,
// since no loop binding is active outside <Each>); an Each entry is an
// <Each> wrapping one <email.StatRow> template, expanded once per element
// of the bound slice at Resolve time (empty slice renders nothing).
type RowSpec struct {
	Literal *StatRow
	Each    *EachRows
}

// StatTable is a bordered data table: an optional mono kicker title, an
// optional header row (a slice-valued props or binding path — no header
// when Header is the zero FieldPath), and its rows, built from a mix of
// literal <email.StatRow> children and <Each>-driven ones (email.StatTable).
type StatTable struct {
	Title  Expr
	Header FieldPath
	Rows   []RowSpec
}

func (StatTable) isBlock() {}

// Each iterates Of (a props- or binding-rooted slice) and resolves Body
// once per element, with As bound to the current element
// (<Each of={...} as="...">). It expands to zero
// ResolvedBlocks when Of resolves to an empty slice; a nested control
// construct inside Body (another Each, an If) resolves normally against
// the extended binding scope.
type Each struct {
	Of   FieldPath
	As   string
	Body []Block
}

func (Each) isBlock() {}

// If resolves Body when Cond evaluates true, and to nothing otherwise
// (<If cond={...}>). EM031 (lint) already
// guarantees every child is an element, never bare text, so Body's
// resolution never needs to invent placement for raw text.
type If struct {
	Cond Expr
	Body []Block
}

func (If) isBlock() {}

// Custom is a raw-element subtree: the escape hatch for markup the
// email.* stdlib does not express. Every literal style property inside
// it is linted like any other raw element (EM101-EM104); gsxmail
// carries the subtree through unmodified at render time and derives its
// text form structurally.
type Custom struct {
	Root CustomNode
}

func (Custom) isBlock() {}

// CustomNode is one node of a Custom subtree: a raw element, or a literal
// text run (a plain text node and a {expression} hole both lower to
// IsText true — Text's Expr already covers both shapes).
type CustomNode struct {
	// Tag is the element name ("div", "a", "img", ...); empty for a text
	// node.
	Tag string
	// IsText marks a text node; Text holds its (possibly expression-
	// bearing) content. Element-only fields (Attrs, Children) are unused
	// on a text node, other than Children, which stays empty.
	IsText bool
	Text   Expr

	Attrs    []CustomAttr
	Children []CustomNode
}

// CustomAttr is one attribute on a Custom element. A "text" attribute is
// not special-cased here: it is a plain attribute lowered like any other;
// the text writer's derivation (package rendertext) is what gives it its
// override meaning.
type CustomAttr struct {
	Name  string
	Value Expr
}

// Button is a call-to-action link rendered as a button-shaped element in
// one of three variants: "primary"
// (the default; a solid accent face, MJML's mso-padding-alt pattern —
// byte-identical to email.CTA's own long-shipped hardened and parity
// output, since lower.lowerBlock's "CTA" case and renderhtml's writeButton
// both route the primary variant through the same, untouched writeCTA
// function), "secondary" (a transparent face with a 1px accent border),
// and "link" (goodemailcode's full-click glyph-spacing technique, R11).
// email.CTA is documented as Button's variant="primary" alias; it is not
// deprecated, and this release keeps it as its own doc.CTA type rather
// than rewriting it in terms of Button, precisely so its bytes can never
// move by construction.
type Button struct {
	Variant string // "primary" (default), "secondary", or "link"
	Label   Expr
	Href    Expr
	// Width is a pixel width the "link" variant's click-area technique
	// uses; every other variant ignores it. Empty selects
	// renderhtml.linkButtonDefaultWidth's own computed default.
	Width Expr
}

func (Button) isBlock() {}

// Column is one email.Columns child: an optional image (imgSrc/imgAlt/
// imgWidth/imgHeight, all four together or none), an optional title, and
// an optional body text (email.Column). Column is deliberately a leaf
// component, not a nested Block container: a fluid-hybrid column is
// roughly 40% of the card's own width, so reusing a top-level block's
// full-card padding and font sizes inside one would misrender; a richer
// per-column layout is a later release's scope.
type Column struct {
	ImgSrc, ImgAlt      Expr
	ImgWidth, ImgHeight Expr
	Title               Expr
	Text                Expr
}

// Columns is a two-to-four-wide fluid-hybrid row (email.Columns/
// email.Column): inline-block max-width divs stack
// under a 480px viewport without depending on a surviving <style> block,
// wrapped in an "[if mso | IE]" ghost table for Outlook, which never
// applies inline-block at all (caniemail css-display).
type Columns struct {
	Columns []Column
}

func (Columns) isBlock() {}

// Hero is a full-width retina image: Src is
// served at 2x pixels; Width and Height are the display-size HTML
// attributes the retina rule requires (both attributes, not srcset,
// which caniemail's own 24.39% support figure rules out). Alt is mandatory
// (lint's checkHero reuses EM111/EM112, the existing raw-<img> rules) and
// is the text twin's sole derived content.
type Hero struct {
	Src, Alt      Expr
	Width, Height Expr
}

func (Hero) isBlock() {}

// Spacer is a fixed-height vertical gap: a spacer-table technique using
// a td at Height pixels, with font-size:0,
// line-height:0, and mso-line-height-rule:exactly pinning an exact-height
// box across clients that a bare margin or empty div does not (caniemail
// css-margin's own partial-support note).
type Spacer struct {
	Height Expr
}

func (Spacer) isBlock() {}

// Badge is a small bordered status label: Tone selects a fixed,
// theme-independent semantic color ("positive"
// green, "warning" amber, "critical" red) or the active theme's own muted
// token ("neutral", the default — the one tone with no status meaning of
// its own to protect from a template's brand palette).
type Badge struct {
	Text Expr
	Tone string // "neutral" (default), "positive", "warning", "critical"
}

func (Badge) isBlock() {}
