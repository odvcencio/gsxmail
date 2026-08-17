package emails

// BigStatsProps fills BigStats, a synthetic template used to cross the
// EM120/EM121 size-budget thresholds: its
// row count is entirely test-controlled, so the same template proves both
// the warning line and the hard budget without two separate fixtures.
type BigStatsProps struct {
	Wordmark string
	Header   []string
	Rows     []BigRow
}

// BigRow is one BigStats row: an arbitrary-width slice of cell text.
type BigRow struct {
	Cells []string
}
