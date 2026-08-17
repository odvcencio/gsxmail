package emails

// AllBlocksProps fills AllBlocks, a fixture that renders every block
// with sentinel prop values, so a test can assert every sentinel
// appears in both parts (emailkit rule 4, inherited structurally). Every
// field carries its own distinct sentinel string, so a field-swap bug
// between two blocks is as visible as a missing one.
type AllBlocksProps struct {
	Wordmark   string
	ShortCode  string
	Tagline    string
	Title      string
	Preheader  string // rendered into the hidden inbox-preview div
	ShoutName  string // rendered through the "shout" helper into the Signal
	HeadTitle  string
	HeadLede   string
	PanelLabel string
	PanelValue string
	CTALabel   string
	CTAHref    string
	PickTitle  string
	PickItem   string
	StatTitle  string
	StatHeader []string
	StatRows   []AllBlocksRow
	ShowNote   bool
	NoteText   string
	LinkLabel  string
	LinkHref   string
	ImgAlt     string
	ImgSrc     string
	TableCell  string
	FooterSign string
	FooterNote string
}

// AllBlocksRow is one AllBlocks StatTable row.
type AllBlocksRow struct {
	Cells []string
	Mark  bool
}
