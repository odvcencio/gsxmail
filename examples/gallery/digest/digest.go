// Package digest is the gallery's weekly-digest template (pixel dossier
// section 8.1): Shell, Hero, Columns, StatTable, Divider, PickList —
// fluid-hybrid columns and a retina hero, its fixture highlights.
package digest

// DigestStat is one StatTable row's cells, in column order.
type DigestStat struct {
	Cells []string
}

// DigestProps fills DigestEmail.
type DigestProps struct {
	Product string
	Period  string // "Week of August 10, 2026"

	HeroSrc    string
	HeroAlt    string
	HeroWidth  string
	HeroHeight string

	Col1ImgSrc    string
	Col1ImgAlt    string
	Col1ImgWidth  string
	Col1ImgHeight string
	Col1Title     string
	Col1Text      string

	Col2ImgSrc    string
	Col2ImgAlt    string
	Col2ImgWidth  string
	Col2ImgHeight string
	Col2Title     string
	Col2Text      string

	StatHeader []string
	Stats      []DigestStat
}
