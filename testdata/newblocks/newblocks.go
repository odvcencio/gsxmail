// Package newblocks is a block-corpus fixture: one template exercising
// every new component — email.Button (all three variants),
// email.Columns/Column, email.Hero, email.Spacer, and email.Badge
// (every tone) — so their sentinels, goldens, and the structural
// verification pass all run against one source tree, the same shape
// testdata/allblocks already established for the earlier stdlib.
package newblocks

// NewBlocksProps fills NewBlocks. Every field is a unique sentinel string
// (or the exact pixel value a golden pins), mirroring
// testdata/allblocks/allblocks.go's own fixture-naming convention.
type NewBlocksProps struct {
	Wordmark  string
	ShortCode string
	Tagline   string
	Title     string

	BadgeNeutralText  string
	BadgePositiveText string
	BadgeWarningText  string
	BadgeCriticalText string

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
	Col2Title     string
	Col2Text      string

	SpacerHeight string

	PrimaryLabel   string
	PrimaryHref    string
	SecondaryLabel string
	SecondaryHref  string
	LinkLabel      string
	LinkHref       string
	LinkWidth      string
}
