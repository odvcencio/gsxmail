package doc

// Resolved is an EmailDoc after every Expr has been evaluated against one
// concrete props value. The HTML and text writers consume only Resolved
// trees; neither writer ever reads props or an Expr.
type Resolved struct {
	Shell  ResolvedShell
	Blocks []ResolvedBlock
}

// ResolvedShell is Shell with every field evaluated to a plain string.
type ResolvedShell struct {
	Wordmark  string
	ShortCode string
	Tagline   string
	Title     string
	Lang      string
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
