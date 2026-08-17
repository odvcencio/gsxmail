package emails

// NoticeProps fills Notice, a golden fixture: small on purpose, so the
// preheader-on and dark-mode golden
// variants stay easy to review. Preheader is empty for the dark-mode
// goldens and set for the preheader-on golden, isolating each shape's own
// diff against the other.
type NoticeProps struct {
	Wordmark   string
	ShortCode  string
	Tagline    string
	Title      string
	Preheader  string
	HeadTitle  string
	HeadLede   string
	CTALabel   string
	CTAHref    string
	FooterSign string
	FooterNote string
}
