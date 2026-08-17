package importer

import (
	"regexp"
	"strconv"
	"strings"
)

// detectBadge matches email.Badge: a single, short-text <span> carrying a
// border and padding (writeBadge's own shape — hardened and parity both
// use a plain bordered inline span, so one fingerprint covers both).
func detectBadge(td *node, ctx *blockCtx, path string) (mappedBlock, bool) {
	spans := elementsIgnoringText(td)
	if len(spans) != 1 || spans[0].tag != "span" {
		return mappedBlock{}, false
	}
	sp := spans[0]
	text := strings.TrimSpace(sp.innerText())
	if text == "" || len(text) > 40 || strings.Contains(text, " ") && len(strings.Fields(text)) > 4 {
		return mappedBlock{}, false
	}
	hasBorder := false
	hasPadding := false
	for _, d := range sp.style() {
		if strings.HasPrefix(d.prop, "border") {
			hasBorder = true
		}
		if d.prop == "padding" {
			hasPadding = true
		}
	}
	if !hasBorder {
		return mappedBlock{}, false
	}
	tone := badgeTone(sp)
	confidence := "medium"
	if hasPadding {
		confidence = "high"
	}
	ctx.rpt.mapped(path, "email.Badge", confidence, "a short, bordered inline span matched the Badge contract")
	gsx := `<email.Badge text=` + qexpr(text) + ` tone="` + tone + `" />` + "\n"
	return mappedBlock{kind: "Badge", gsx: gsx}, true
}

// badgeTone infers a closed-set tone (renderhtml.badgeToneColor's own
// vocabulary) from sp's literal border/color, defaulting to "neutral"
// when nothing matches — EM180 requires one of exactly these four.
func badgeTone(sp *node) string {
	color, _ := sp.styleValue("color")
	color = strings.ToUpper(strings.TrimSpace(color))
	switch color {
	case "#2F9E44":
		return "positive"
	case "#B76E00":
		return "warning"
	case "#C92A2A":
		return "critical"
	}
	if looksGreen(color) {
		return "positive"
	}
	if looksRed(color) {
		return "critical"
	}
	if looksAmber(color) {
		return "warning"
	}
	return "neutral"
}

// detectButton matches email.Button/email.CTA: a single anchor that is
// the row's entire content (task instructions' "bulletproof-button
// anchors (padded td + a, or the mso-padding-alt shapes)"), scored for
// which variant its own styling (or its wrapping td's) most resembles.
func detectButton(td *node, ctx *blockCtx, path string) (mappedBlock, bool) {
	// A button-shaped anchor legitimately sits one (or more) <table>
	// levels deep — its own face table is the button's whole contract
	// (writeCTA) — so this search crosses nested tables, unlike most of
	// this file's other findAllShallow calls.
	anchors := findAll(td, "a")
	if len(anchors) != 1 {
		return mappedBlock{}, false
	}
	a := anchors[0]
	label := strings.TrimSpace(a.innerText())
	rowText := strings.TrimSpace(td.innerText())
	if label == "" || rowText != label {
		return mappedBlock{}, false // an anchor alongside other body text is a link inside prose, not a CTA
	}

	variant := "primary"
	confidence := "low"
	note := "a lone anchor filled the row with no other content, and was assumed to be a call-to-action button"

	bg := findButtonFaceColor(td, a)
	border := findWrappingBorder(td, a)
	_, hasLineHeight := a.styleValue("line-height")
	display, _ := a.styleValue("display")

	switch {
	case bg != "":
		variant, confidence, note = "primary", "high", "a background-colored button face wrapping the anchor matched the primary Button contract"
	case border:
		variant, confidence, note = "secondary", "high", "a bordered, transparent button face wrapping the anchor matched the secondary Button contract"
	case hasLineHeight && strings.Contains(display, "inline-block"):
		variant, confidence, note = "link", "medium", "a fixed line-height, inline-block anchor matched the link Button's full-click technique"
	}

	href, _ := a.attr("href")
	href, hrefNote := sanitizeHref(href)
	if hrefNote != "" {
		ctx.rpt.NextSteps = append(ctx.rpt.NextSteps, path+": "+hrefNote)
	}
	ctx.rpt.mapped(path, "email.Button", confidence, note)
	gsx := `<email.Button variant="` + variant + `" label=` + qexpr(label) + ` href=` + qexpr(href) + ` />` + "\n"
	return mappedBlock{kind: "Button", gsx: gsx}, true
}

// findButtonFaceColor looks for a background color between root and
// target, bounded to target's own immediate face table (boundedChain):
// the "background-color" property, the "background" shorthand (MJML's
// own compiled button emits both bgcolor and background:#hex — R8), or
// the legacy "bgcolor" HTML attribute, in that order, at whichever
// wrapping element carries one first. Bounding the climb matters when
// root is the whole document (extractTheme's own call, hunting for an
// accent color) rather than one row's own td: without it, a legacy
// source's outer card table (this package's own testdata/corpus/
// legacy.html sets `bgcolor="#ffffff"` on its 600px card) reads as if it
// were the button's own face color, which it plainly is not.
func findButtonFaceColor(root, target *node) string {
	for _, n := range boundedChain(root, target) {
		if v, ok := n.styleValue("background-color"); ok && strings.TrimSpace(v) != "" {
			return v
		}
		if v, ok := n.styleValue("background"); ok && looksLikeColor(v) {
			return v
		}
		// The legacy "bgcolor" HTML attribute only counts on a td/tr: a
		// <table bgcolor="..."> is the card's own page-background
		// convention (this package's own testdata/corpus/legacy.html sets
		// exactly that), not a per-button face color, even after
		// boundedChain's own table-boundary cap — when target has no
		// closer face table at all, that outer card table is still "the
		// first table in the chain."
		if n.tag != "table" {
			if v, ok := n.attr("bgcolor"); ok && strings.TrimSpace(v) != "" {
				return v
			}
		}
	}
	return ""
}

// looksLikeColor reports whether v (a CSS "background" shorthand value)
// is plausibly just a color — a hex token, or a color-looking bare word —
// rather than a shorthand that also names a gradient or image the
// dossier already bans (EM162) and this package has no business trying
// to parse.
func looksLikeColor(v string) bool {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, "#") {
		return true
	}
	return !strings.Contains(v, "(") && !strings.Contains(v, "url")
}

// findWrappingBorder reports whether any element between root and
// target, bounded to target's own immediate face table (boundedChain),
// carries a real border declaration — "border" (or a "border-*"
// longhand) present and not "none"/"0"/"0px". A td whose own "border:
// none;" reset (MJML's own compiled button carries exactly this, R8)
// must not read as "this button has a border."
func findWrappingBorder(root, target *node) bool {
	for _, n := range boundedChain(root, target) {
		for _, d := range n.style() {
			if d.prop != "border" && !strings.HasPrefix(d.prop, "border-") {
				continue
			}
			v := strings.ToLower(strings.TrimSpace(d.value))
			if v == "" || v == "none" || v == "0" || v == "0px" {
				continue
			}
			return true
		}
	}
	return false
}

// wrappingChain returns every node strictly between root and target,
// root and target themselves included, nearest to target first — the
// button-face td sits between the row's own td and the anchor itself in
// every shipped contract, and findButtonFaceColor/findWrappingBorder
// both want to check the element nearest the anchor before a looser
// ancestor.
func wrappingChain(root, target *node) []*node {
	var path []*node
	var find func(*node) bool
	find = func(n *node) bool {
		if n == target {
			path = append(path, n)
			return true
		}
		for i := range n.children {
			c := &n.children[i]
			if find(c) {
				path = append(path, n)
				return true
			}
		}
		return false
	}
	find(root)
	return path
}

// boundedChain is wrappingChain, truncated to stop right after the
// first <table> element it reaches walking outward from target — the
// button's own immediate face table, never a further-out structural
// table (a card, a section, a whole document) that happens to carry an
// unrelated color of its own. A chain with no <table> anywhere in it
// (target sits directly in a div-only structure) is returned whole.
func boundedChain(root, target *node) []*node {
	chain := wrappingChain(root, target)
	for i, n := range chain {
		if n.tag == "table" {
			return chain[:i+1]
		}
	}
	return chain
}

// findWrappingStyleValue looks up prop on every element strictly between
// root and target (root and target themselves included), returning the
// first non-empty value found nearest to target — the button-face td
// sits between the row's own td and the anchor itself in every shipped
// contract. extractTheme's own accent-color scan is the one remaining
// caller; detectButton uses the more specific findButtonFaceColor and
// findWrappingBorder above instead.
func findWrappingStyleValue(root, target *node, prop string) string {
	for _, n := range wrappingChain(root, target) {
		if v, ok := n.styleValue(prop); ok && strings.TrimSpace(v) != "" && strings.TrimSpace(v) != "0" {
			return v
		}
	}
	return ""
}

// detectStatTable matches email.StatTable: a table with at least one
// <th> cell anywhere in its own rows. Rows are synthesized into an
// <Each of={props.Items}>, the same treatment every other repeated
// sibling structure gets — an Each over a synthesized props slice.
func detectStatTable(td *node, ctx *blockCtx, path string) (mappedBlock, bool) {
	tbl := findFirst(td, "table")
	if tbl == nil {
		return mappedBlock{}, false
	}
	rows := tableRows(tbl)
	if len(rows) == 0 {
		return mappedBlock{}, false
	}
	hasTH := false
	for _, r := range rows {
		if len(findAllShallow(r, "th")) > 0 {
			hasTH = true
		}
	}
	if !hasTH {
		return mappedBlock{}, false
	}

	title := kickerBefore(td, tbl)

	var header []string
	dataRows := rows
	if ths := findAllShallow(rows[0], "th"); len(ths) > 0 {
		for _, th := range ths {
			header = append(header, strings.TrimSpace(th.innerText()))
		}
		dataRows = rows[1:]
	}

	var cellRows [][]string
	for _, r := range dataRows {
		var cells []string
		for _, c := range r.elements() {
			if c.tag == "td" {
				cells = append(cells, strings.TrimSpace(c.innerText()))
			}
		}
		if len(cells) > 0 {
			cellRows = append(cellRows, cells)
		}
	}

	var b strings.Builder
	b.WriteString("<email.StatTable")
	if title != "" {
		b.WriteString(" title=" + qexpr(title))
	}
	headerAttr := ""
	if len(header) > 0 {
		field := ctx.props.sliceField("Header", header, "the table's own <th> header row")
		headerAttr = " header={props." + field + "}"
	}
	b.WriteString(headerAttr)
	if len(cellRows) == 0 {
		b.WriteString(" />\n")
	} else {
		field, elem := ctx.props.eachField("Items", "Cells", cellRows, "the table's own repeated data rows")
		b.WriteString(">\n")
		b.WriteString(indent("<Each of={props." + field + "} as=\"item\">\n"))
		b.WriteString(indent(indent("<email.StatRow cells={item.Cells} />\n")))
		b.WriteString(indent("</Each>\n"))
		b.WriteString("</email.StatTable>\n")
		_ = elem
	}
	ctx.rpt.mapped(path, "email.StatTable", "high", "a table with <th> header cells matched the StatTable data-table contract")
	return mappedBlock{kind: "StatTable", gsx: b.String()}, true
}

// tableRows returns tbl's own direct rows (tbody-flattened), not
// descending into any table nested inside a cell.
func tableRows(tbl *node) []*node {
	var out []*node
	for _, e := range tbl.elements() {
		if e.tag == "tr" {
			out = append(out, e)
		}
	}
	return out
}

// kickerBefore returns the text of a short label element that appears in
// td before tbl (StatTable and PickList's own optional mono kicker
// title), or "".
func kickerBefore(td, tbl *node) string {
	for _, c := range elementsIgnoringText(td) {
		if c == tbl {
			return ""
		}
		if c.tag == "div" || c.tag == "span" || c.tag == "p" {
			if t := strings.TrimSpace(c.innerText()); t != "" && len(t) <= 40 {
				return t
			}
		}
	}
	return ""
}

// panelPair is one label/value row candidate, before it is known whether
// the row belongs to a Panel at all.
type panelPair struct{ label, value string }

// panelRowPair reports whether row has exactly two <td> cells and no
// <th> (a <th> anywhere is StatTable's own territory, never Panel's),
// neither carrying an <img> (an image in either cell reads as a
// two-column layout row — Columns' own territory — never a text
// label/value pair), returning its label/value text when it does.
func panelRowPair(row *node) (panelPair, bool) {
	var tds []*node
	for _, c := range row.elements() {
		if c.tag == "td" {
			tds = append(tds, c)
		} else {
			return panelPair{}, false
		}
	}
	if len(tds) != 2 {
		return panelPair{}, false
	}
	for _, td := range tds {
		if findFirst(td, "img") != nil {
			return panelPair{}, false
		}
	}
	return panelPair{
		label: strings.TrimSpace(tds[0].innerText()),
		value: strings.TrimSpace(tds[1].innerText()),
	}, true
}

// buildPanelGSX renders pairs as one <email.Panel> block, synthesizing a
// props field for each row's own value (Panel has no shipped
// Each-over-rows shape, unlike StatTable, so rows stay literal per row
// rather than synthesized into a slice).
func buildPanelGSX(pairs []panelPair, props *propsBuilder) string {
	var b strings.Builder
	b.WriteString("<email.Panel>\n")
	for _, p := range pairs {
		field := props.field(panelValueName(p.label), p.value, "the Panel row labeled "+quoteForReport(p.label))
		b.WriteString(indent(`<email.PanelRow label=` + qexpr(p.label) + ` value={props.` + field + `} />` + "\n"))
	}
	b.WriteString("</email.Panel>\n")
	return b.String()
}

// detectPanel matches email.Panel: a <table> nested one level inside the
// row's own td, whose every row carries exactly two cells and no <th> —
// a uniform label/value td pattern, gsxmail's own native shape
// (writePanel). A source that
// stacks label/value pairs as the card's own top-level rows instead,
// with no wrapping sub-table, matches panelRunLength in importer.go
// instead, which this function's own row-pair logic (panelRowPair,
// buildPanelGSX) also backs.
func detectPanel(td *node, ctx *blockCtx, path string) (mappedBlock, bool) {
	tbl := findFirst(td, "table")
	if tbl == nil {
		return mappedBlock{}, false
	}
	rows := tableRows(tbl)
	if len(rows) == 0 {
		return mappedBlock{}, false
	}
	var pairs []panelPair
	for _, r := range rows {
		p, ok := panelRowPair(r)
		if !ok {
			return mappedBlock{}, false
		}
		pairs = append(pairs, p)
	}

	ctx.rpt.mapped(path, "email.Panel", "high", "a table of two-cell rows matched the label/value Panel contract")
	return mappedBlock{kind: "Panel", gsx: buildPanelGSX(pairs, ctx.props)}, true
}

// panelRunLength reports how many consecutive rows, starting at start
// (and stopping before end), are each a naked two-cell label/value row
// with no wrapping sub-table — a source that stacks Panel-shaped rows
// directly as the card's own top-level rows (react-email's own idiomatic
// per-row-a-Section shape is one example; this package's own
// react-email.html fixture reproduces it) rather than nesting them one
// level down the way gsxmail's own writePanel does. It returns 0 when
// rows[start] itself is not two-cell-shaped, so importer.go's own loop
// falls through to the ordinary per-row classifyRow path unchanged for
// every other block type.
func panelRunLength(rows []*node, start, end int) int {
	n := 0
	for i := start; i < end; i++ {
		if _, ok := panelRowPair(rows[i]); !ok {
			break
		}
		n++
	}
	return n
}

// panelValueName turns a Panel row's own label text ("SUBTOTAL", "Billed
// to") into a props field base name ("Subtotal", "BilledTo").
func panelValueName(label string) string {
	name := sanitizeIdent(label)
	if name == "" {
		return "Value"
	}
	return name
}

var numberedItemRe = regexp.MustCompile(`^\d+[.)]\s*`)

// detectPickList matches email.PickList: an <ol>/<ul> list, or a table
// whose every row's own text starts with an ordinal marker
// (writePickList's own "N. text" shape, both hardened and parity).
func detectPickList(td *node, ctx *blockCtx, path string) (mappedBlock, bool) {
	var items []string
	var title string

	if list := findAllShallow(td, "ol"); len(list) == 1 {
		items = liTexts(list[0])
	} else if list := findAllShallow(td, "ul"); len(list) == 1 {
		items = liTexts(list[0])
	} else if tbl := findFirst(td, "table"); tbl != nil {
		rows := tableRows(tbl)
		if len(rows) == 0 {
			return mappedBlock{}, false
		}
		for _, r := range rows {
			text := strings.TrimSpace(r.innerText())
			if !numberedItemRe.MatchString(text) {
				return mappedBlock{}, false
			}
			items = append(items, strings.TrimSpace(numberedItemRe.ReplaceAllString(text, "")))
		}
		title = kickerBefore(td, tbl)
	}
	if len(items) == 0 {
		return mappedBlock{}, false
	}

	var b strings.Builder
	b.WriteString("<email.PickList")
	if title != "" {
		b.WriteString(" title=" + qexpr(title))
	}
	b.WriteString(">\n")
	for _, it := range items {
		b.WriteString(indent("<email.Item>" + qexpr(it) + "</email.Item>\n"))
	}
	b.WriteString("</email.PickList>\n")
	ctx.rpt.mapped(path, "email.PickList", "high", "a numbered list matched the PickList contract")
	return mappedBlock{kind: "PickList", gsx: b.String()}, true
}

func liTexts(list *node) []string {
	var out []string
	for _, li := range findAllShallow(list, "li") {
		if t := strings.TrimSpace(li.innerText()); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// detectColumns matches email.Columns: 2-4 sibling divs each shaped like
// a fluid-hybrid column (display:inline-block or display:table-cell, or
// an explicit width/max-width). EM176 caps the count at 2-4, so a 1- or
// 5-plus-column row is rejected before it can trip that check at Load
// time.
func detectColumns(td *node, ctx *blockCtx, path string) (mappedBlock, bool) {
	kids := elementsIgnoringText(td)
	var cols []*node
	for _, k := range kids {
		if k.tag != "div" && k.tag != "td" {
			continue
		}
		display, hasDisplay := k.styleValue("display")
		_, hasWidth := k.styleValue("width")
		_, hasMaxWidth := k.styleValue("max-width")
		widthAttr, hasWidthAttr := k.attr("width")
		looksColumnar := (hasDisplay && (strings.Contains(display, "inline-block") || strings.Contains(display, "table-cell"))) ||
			hasMaxWidth || (hasWidth && k.tag == "div") || (hasWidthAttr && widthAttr != "")
		if !looksColumnar {
			return mappedBlock{}, false
		}
		cols = append(cols, k)
	}
	if len(cols) < 2 || len(cols) > 4 {
		return mappedBlock{}, false
	}

	var b strings.Builder
	b.WriteString("<email.Columns>\n")
	for i, col := range cols {
		imgSrc, imgAlt, imgW, imgH, title, text := extractColumn(col)
		b.WriteString(indent("<email.Column"))
		if imgSrc != "" {
			imgSrc, note := sanitizeImgSrc(imgSrc)
			if note != "" {
				ctx.rpt.NextSteps = append(ctx.rpt.NextSteps, path+": "+note)
			}
			b.WriteString(" imgSrc=" + qexpr(imgSrc) + " imgAlt=" + qexpr(orDefault(imgAlt, "Imported image")) +
				" imgWidth=" + qexpr(orDefault(imgW, "268")) + " imgHeight=" + qexpr(orDefault(imgH, "160")))
		}
		if title != "" {
			b.WriteString(" title=" + qexpr(title))
		}
		if text != "" {
			b.WriteString(" text=" + qexpr(text))
		}
		b.WriteString(" />\n")
		_ = i
	}
	b.WriteString("</email.Columns>\n")
	ctx.rpt.mapped(path, "email.Columns", "medium", "2-4 sibling fluid-hybrid divs matched the Columns contract")
	return mappedBlock{kind: "Columns", gsx: b.String()}, true
}

func orDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

func extractColumn(col *node) (imgSrc, imgAlt, imgW, imgH, title, text string) {
	if img := findFirst(col, "img"); img != nil {
		imgSrc, _ = img.attr("src")
		imgAlt, _ = img.attr("alt")
		imgW, _ = img.attr("width")
		imgH, _ = img.attr("height")
	}
	var texts []string
	for _, c := range elementsIgnoringText(col) {
		if c.tag == "img" {
			continue
		}
		if t := strings.TrimSpace(c.innerText()); t != "" {
			texts = append(texts, t)
		}
	}
	switch len(texts) {
	case 0:
	case 1:
		text = texts[0]
	default:
		title = texts[0]
		text = strings.Join(texts[1:], " ")
	}
	return
}

// detectNote matches email.Note: a border-left-accented block with plain
// text and no nested table or list.
func detectNote(td *node, ctx *blockCtx, path string) (mappedBlock, bool) {
	target := unwrapTrivial(td)
	borderLeft, ok := target.styleValue("border-left")
	if !ok || strings.TrimSpace(borderLeft) == "" || strings.TrimSpace(borderLeft) == "0" {
		if divs := elementsIgnoringText(target); len(divs) == 1 && divs[0].tag == "div" {
			if v, ok2 := divs[0].styleValue("border-left"); ok2 && strings.TrimSpace(v) != "" {
				target = divs[0]
				ok = true
			}
		}
	} else {
		ok = true
	}
	if !ok {
		return mappedBlock{}, false
	}
	if findFirst(target, "table") != nil || findFirst(target, "ol") != nil || findFirst(target, "ul") != nil {
		return mappedBlock{}, false
	}
	text := strings.TrimSpace(target.innerText())
	if text == "" {
		return mappedBlock{}, false
	}
	ctx.rpt.mapped(path, "email.Note", "high", "a border-left accented block matched the Note aside contract")
	return mappedBlock{kind: "Note", gsx: "<email.Note text=" + qexpr(text) + " />\n"}, true
}

// bigTextRe matches a plausible headline font-size (22px or larger).
var bigTextRe = regexp.MustCompile(`^(\d+)`)

// detectHeadline matches email.Headline: a large or bold heading element,
// optionally followed by one lede text block — generalized from "big
// bold headline text" to any table-free row whose first element reads
// as a heading.
func detectHeadline(td *node, ctx *blockCtx, path string) (mappedBlock, bool) {
	if findFirst(td, "table") != nil {
		return mappedBlock{}, false
	}
	kids := elementsIgnoringText(td)
	if len(kids) == 0 {
		return mappedBlock{}, false
	}
	first := kids[0]
	if !looksLikeHeading(first) {
		return mappedBlock{}, false
	}
	title := strings.TrimSpace(first.innerText())
	if title == "" {
		return mappedBlock{}, false
	}
	lede := ""
	if len(kids) > 1 {
		lede = strings.TrimSpace(kids[1].innerText())
	}
	field := ""
	if lede != "" {
		field = ctx.props.field("Lede", lede, "the headline's own lede paragraph")
	}
	ctx.rpt.mapped(path, "email.Headline", "medium", "a large or bold heading led this row, matching the Headline contract")
	gsx := "<email.Headline title=" + qexpr(title)
	if field != "" {
		gsx += " lede={props." + field + "}"
	} else {
		gsx += ` lede=""`
	}
	gsx += " />\n"
	return mappedBlock{kind: "Headline", gsx: gsx}, true
}

// looksLikeHeading reports whether n reads as a headline title: a real
// heading tag, a styled div/p carrying bold or large text, or (legacy
// table-soup HTML predates CSS-driven styling) a bare <b>/<strong>, or a
// <font size="N"> at N >= 4 — the pre-CSS way authors marked a line as
// "big" (HTML 4's own font size scale tops out at 7; 4 and up reads
// larger than body copy in every client still rendering the tag at all).
func looksLikeHeading(n *node) bool {
	switch n.tag {
	case "h1", "h2", "h3":
		return true
	case "b", "strong":
		return true
	case "font":
		if size, ok := n.attr("size"); ok {
			if v, err := strconv.Atoi(strings.TrimSpace(size)); err == nil && v >= 4 {
				return true
			}
		}
	}
	if n.tag != "div" && n.tag != "p" {
		return false
	}
	fw, hasFW := n.styleValue("font-weight")
	if hasFW {
		if w, err := strconv.Atoi(strings.TrimSpace(fw)); err == nil && w >= 700 {
			return true
		}
		if strings.EqualFold(strings.TrimSpace(fw), "bold") {
			return true
		}
	}
	fs, hasFS := n.styleValue("font-size")
	if hasFS {
		m := bigTextRe.FindString(strings.TrimSpace(fs))
		if m != "" {
			if size, err := strconv.Atoi(m); err == nil && size >= 22 {
				return true
			}
		}
	}
	return false
}

func looksGreen(hex string) bool { return hexBias(hex, 1) }
func looksRed(hex string) bool   { return hexBias(hex, 0) }
func looksAmber(hex string) bool { return hexBiasAmber(hex) }

// hexBias reports whether hex's channel-index-th 2-hex-digit component
// (0=red, 1=green, 2=blue) is the clear maximum of the three — a coarse,
// dependency-free "is this basically red/green" classifier good enough
// for Badge's own closed tone vocabulary; it never claims real color
// science.
func hexBias(hex string, channel int) bool {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return false
	}
	vals := []int{r, g, b}
	target := vals[channel]
	for i, v := range vals {
		if i != channel && v >= target {
			return false
		}
	}
	return true
}

func hexBiasAmber(hex string) bool {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return false
	}
	return r > 120 && g > 60 && b < 80 && r >= g
}

func parseHex(hex string) (r, g, b int, ok bool) {
	hex = strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	rv, err1 := strconv.ParseInt(hex[0:2], 16, 32)
	gv, err2 := strconv.ParseInt(hex[2:4], 16, 32)
	bv, err3 := strconv.ParseInt(hex[4:6], 16, 32)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, false
	}
	return int(rv), int(gv), int(bv), true
}

func quoteForReport(s string) string {
	return "\"" + s + "\""
}
