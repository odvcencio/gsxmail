package doc

// Resolved is an EmailDoc after every Expr has been evaluated against one
// concrete props value. The HTML and text writers consume only Resolved
// trees; neither writer ever reads props or an Expr.
type Resolved struct {
	Shell  ResolvedShell
	Blocks []ResolvedBlock
}

// ResolvedShell is Shell with every field evaluated to a plain string.
// Preheader resolves like any other text field; Outlook carries straight
// through unevaluated (design spec section 15, WP5.2: it is already a
// plain string on Shell, never an Expr).
type ResolvedShell struct {
	Wordmark  string
	ShortCode string
	Tagline   string
	Title     string
	Lang      string
	Preheader string
	Outlook   string
}

// ResolvedBlock mirrors Block with plain-string fields. The implementations
// are the closed set of stdlib blocks WP1 ships.
type ResolvedBlock interface {
	isResolvedBlock()
}

type ResolvedSignal struct{ Text string }

func (ResolvedSignal) isResolvedBlock() {}

type ResolvedHeadline struct{ Title, Lede string }

func (ResolvedHeadline) isResolvedBlock() {}

type ResolvedPanelRow struct{ Label, Value string }

type ResolvedPanel struct{ Rows []ResolvedPanelRow }

func (ResolvedPanel) isResolvedBlock() {}

type ResolvedCTA struct{ Label, Href string }

func (ResolvedCTA) isResolvedBlock() {}

type ResolvedPickList struct {
	Title string
	Items []string
}

func (ResolvedPickList) isResolvedBlock() {}

type ResolvedFooter struct{ Signoff, Note string }

func (ResolvedFooter) isResolvedBlock() {}

type ResolvedNote struct{ Text string }

func (ResolvedNote) isResolvedBlock() {}

type ResolvedDivider struct{}

func (ResolvedDivider) isResolvedBlock() {}

// ResolvedStatRow is one resolved StatTable row: its formatted cells and
// whether it is the marked row (mirrors emailkit's per-row shape; the
// table-level 1-based MarkRow that both writers actually key off of is
// ResolvedStatTable.MarkRow, derived from every row's Mark).
type ResolvedStatRow struct {
	Cells []string
	Mark  bool
}

// ResolvedStatTable is StatTable after every Expr and FieldPath has been
// evaluated. MarkRow is the 1-based index of the first row whose Mark
// resolved true, or 0 when none did — emailkit's MarkRow shape (design
// spec section 6.5, "Mark semantics match emailkit's MarkRow"), derived
// once here so both writers key off the same field rather than re-scanning
// Rows themselves.
type ResolvedStatTable struct {
	Title   string
	Header  []string
	Rows    []ResolvedStatRow
	MarkRow int
}

func (ResolvedStatTable) isResolvedBlock() {}

// ResolvedCustom is Custom after every Expr in its subtree has been
// evaluated.
type ResolvedCustom struct {
	Root ResolvedCustomNode
}

func (ResolvedCustom) isResolvedBlock() {}

// ResolvedCustomNode mirrors CustomNode with every Expr evaluated to a
// plain string.
type ResolvedCustomNode struct {
	Tag      string
	IsText   bool
	Text     string
	Attrs    []ResolvedCustomAttr
	Children []ResolvedCustomNode
}

type ResolvedCustomAttr struct{ Name, Value string }

// ResolvedButton is Button after every Expr has been evaluated. Variant is
// always one of "primary", "secondary", "link" — resolveBlockInto defaults
// an empty Variant to "primary" the same way lower.go's own "CTA" case
// does, so a writer never has to repeat that default.
type ResolvedButton struct {
	Variant     string
	Label, Href string
	Width       string
}

func (ResolvedButton) isResolvedBlock() {}

// ResolvedColumn is Column after every Expr has been evaluated.
type ResolvedColumn struct {
	ImgSrc, ImgAlt      string
	ImgWidth, ImgHeight string
	Title, Text         string
}

// ResolvedColumns is Columns after every Column has been resolved.
type ResolvedColumns struct {
	Columns []ResolvedColumn
}

func (ResolvedColumns) isResolvedBlock() {}

// ResolvedHero is Hero after every Expr has been evaluated.
type ResolvedHero struct {
	Src, Alt      string
	Width, Height string
}

func (ResolvedHero) isResolvedBlock() {}

// ResolvedSpacer is Spacer after its Height has been evaluated.
type ResolvedSpacer struct {
	Height string
}

func (ResolvedSpacer) isResolvedBlock() {}

// ResolvedBadge is Badge after its Text has been evaluated.
type ResolvedBadge struct {
	Text, Tone string
}

func (ResolvedBadge) isResolvedBlock() {}
