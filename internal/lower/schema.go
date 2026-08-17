package lower

// AttrSchema describes one email.* component's attribute surface: every
// attribute name Lower's own switch (lowerBlock, lowerShell, lowerColumn,
// lowerStatRow) reads, and which of those names it treats as required for
// the component to mean anything. Required omits a few names package lint
// already gates with a more specific, pre-existing rule (email.Hero's
// width/height/src/alt through EM178, email.Spacer's height through
// EM179) — see Schema's own doc comment.
type AttrSchema struct {
	// Attrs is every attribute name this component accepts. An attribute
	// not in this list is unknown (package lint's EM190).
	Attrs []string
	// Required is the subset of Attrs that must be present for the
	// component to render anything meaningful (package lint's EM191).
	Required []string
}

// Schema is the closed table of every email.* component's own attribute
// surface, keyed by the component's local name (the "email." namespace
// prefix already stripped, splitTag's own convention) — "Panel", "Item",
// and "Columns" carry no attribute-name entries of their own, since
// Panel and Columns accept only their own child component (PanelRow,
// Column) and Item's content is inline text/expression children
// (inlineContent), not attributes.
//
// Schema is this package's single source of truth for attribute names:
// hand-derived from this same file's lowerBlock/lowerShell/lowerColumn/
// lowerStatRow switch, so lint's EM190 (unknown attribute), EM191
// (required attribute absent), and EM196 (unknown email.* member) never
// drift from what Lower itself accepts. Required deliberately omits
// email.Hero's src/alt/width/height and email.Spacer's height: those
// already have their own pre-existing, more specific EM178/EM179
// checks, and duplicating them under EM191 would report the same
// mistake twice under two different codes. The required set otherwise
// covers: Shell.wordmark/title, Signal.text, Headline.title,
// PanelRow.label/value, CTA.label/href, Button.label/href, Note.text,
// Badge.text, Footer.signoff. email.Item's required "text" is content,
// not an attribute, so package lint checks it separately (checkItem's
// own EM191 case), rather than through this table.
var Schema = map[string]AttrSchema{
	"Shell":     {Attrs: []string{"wordmark", "shortCode", "tagline", "title", "lang", "preheader", "outlook"}, Required: []string{"wordmark", "title"}},
	"Signal":    {Attrs: []string{"text"}, Required: []string{"text"}},
	"Headline":  {Attrs: []string{"title", "lede"}, Required: []string{"title"}},
	"Panel":     {Attrs: []string{}},
	"PanelRow":  {Attrs: []string{"label", "value"}, Required: []string{"label", "value"}},
	"CTA":       {Attrs: []string{"label", "href"}, Required: []string{"label", "href"}},
	"Button":    {Attrs: []string{"variant", "label", "href", "width"}, Required: []string{"label", "href"}},
	"Column":    {Attrs: []string{"imgSrc", "imgAlt", "imgWidth", "imgHeight", "title", "text"}},
	"Hero":      {Attrs: []string{"src", "alt", "width", "height"}},
	"Spacer":    {Attrs: []string{"height"}},
	"Badge":     {Attrs: []string{"text", "tone"}, Required: []string{"text"}},
	"PickList":  {Attrs: []string{"title"}},
	"Item":      {Attrs: []string{}},
	"Footer":    {Attrs: []string{"signoff", "note"}, Required: []string{"signoff"}},
	"Note":      {Attrs: []string{"text"}, Required: []string{"text"}},
	"Divider":   {Attrs: []string{}},
	"StatTable": {Attrs: []string{"title", "header"}},
	"StatRow":   {Attrs: []string{"cells", "mark"}},
	"Columns":   {Attrs: []string{}},
}
