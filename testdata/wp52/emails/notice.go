package emails

// NoticeProps fills Notice, the WP5.2 golden fixture (design spec section
// 15, WP5.2): small on purpose, so the preheader-on and dark-mode golden
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
