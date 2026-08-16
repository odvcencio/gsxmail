package importer

import (
	"regexp"
	"strconv"
	"strings"
)

// mappedBlock is one recovered card row: the stdlib component kind it
// maps to (matching doc.Block's own type names, so a caller comparing
// self-round-trip structure can diff against the source Set's own
// component list) and the .gsx source fragment gen.go's writeTemplate
// splices into the Shell body, already indented and newline-terminated.
type mappedBlock struct {
	kind string
	gsx  string
}

// blockCtx carries the shared, per-Import state every classifier needs:
// the props/theme builder, the report, and a running index for path
// labels ("card/row[3]").
type blockCtx struct {
	props *propsBuilder
	rpt   *Report
}

// classifyRow maps one top-level card row to a Block, trying gsxmail's
// own hardened-output fingerprints first (native, high confidence — this
// is what makes the self-round-trip proof bar achievable: a template
// gsxmail itself rendered matches one of these exactly) and falling back
// to looser, generic shape checks for foreign HTML (pixel dossier
// section 7.2(1)'s own numbered heuristics). path is this row's report
// label. It never returns ok==false silently: the caller wraps an
// unrecognized row as email.Custom and this func's own rpt.unmapped call
// records why.
func classifyRow(row *node, ctx *blockCtx, path string) mappedBlock {
	td := soleContentCell(row)
	if td == nil {
		td = row
	}

	type detector func(*node, *blockCtx, string) (mappedBlock, bool)
	detectors := []detector{
		detectSpacer,
		detectDivider,
		detectHero,
		detectBadge,
		detectSignal,
		detectButton,
		detectStatTable,
		detectPanel,
		detectPickList,
		detectColumns,
		detectNote,
		detectHeadline,
	}
	for _, d := range detectors {
		if b, ok := d(td, ctx, path); ok {
			return b
		}
	}

	ctx.rpt.unmapped(path, "no component fingerprint matched this row; preserved as email.Custom")
	return mappedBlock{kind: "Custom", gsx: writeCustomFallback(row)}
}

// soleContentCell returns row's one <td>/<th> child when it has exactly
// one, or nil (the caller falls back to treating row itself as the
// content root — the header and multi-cell rows classifyRow's own
// callers already special-case before reaching here).
func soleContentCell(row *node) *node {
	var cell *node
	n := 0
	for _, c := range row.elements() {
		if c.tag == "td" || c.tag == "th" {
			n++
			cell = c
		}
	}
	if n == 1 {
		return cell
	}
	return nil
}

// unwrapTrivial descends through a chain of single-row, single-cell
// wrapper tables — MJML and react-email's compiled output both nest one
// or more purely structural tables around a section's real content
// (pixel dossier section 7.2(1), heuristic 1's own "ghost tables ...
// map to nothing", generalized to any structural single-cell wrapper,
// not just an "[if mso]" one) — stopping at the first level that is not
// a trivial wrapper.
func unwrapTrivial(td *node) *node {
	for {
		kids := elementsIgnoringText(td)
		if len(kids) != 1 || kids[0].tag != "table" {
			return td
		}
		rows := kids[0].elements()
		var trs []*node
		for _, r := range rows {
			if r.tag == "tr" {
				trs = append(trs, r)
			}
		}
		if len(trs) != 1 {
			return td
		}
		inner := soleContentCell(trs[0])
		if inner == nil {
			return td
		}
		td = inner
	}
}

func elementsIgnoringText(n *node) []*node {
	var out []*node
	for _, c := range n.elements() {
		out = append(out, c)
	}
	return out
}

var bulletRe = regexp.MustCompile(`^\x{25CF}|^\x{2022}|^\*`)

// detectSignal matches email.Signal's shape (writeSignal): a bullet
// glyph ("●") followed by short uppercase text, alone in the row.
func detectSignal(td *node, ctx *blockCtx, path string) (mappedBlock, bool) {
	text := td.innerText()
	if text == "" || !bulletRe.MatchString(text) {
		return mappedBlock{}, false
	}
	label := strings.TrimSpace(bulletRe.ReplaceAllString(text, ""))
	if label == "" || len(label) > 60 {
		return mappedBlock{}, false
	}
	ctx.rpt.mapped(path, "email.Signal", "medium", "a bullet-prefixed short line matched the Signal glyph pattern")
	return mappedBlock{kind: "Signal", gsx: "<email.Signal text=" + qexpr(label) + " />\n"}, true
}

// detectSpacer matches email.Spacer's exact-height, empty-content
// contract (writeSpacer): a cell whose own height (attribute or style)
// is set and which carries no meaningful text.
func detectSpacer(td *node, ctx *blockCtx, path string) (mappedBlock, bool) {
	if td.tag != "td" {
		return mappedBlock{}, false
	}
	if strings.TrimSpace(td.innerText()) != "" {
		return mappedBlock{}, false
	}
	height := ""
	if v, ok := td.attr("height"); ok {
		height = v
	} else if v, ok := td.styleValue("height"); ok {
		height = v
	}
	n, ok := parsePixels(height)
	if !ok || n <= 0 {
		return mappedBlock{}, false
	}
	fs, hasFS := td.styleValue("font-size")
	hasBorder := false
	for _, d := range td.style() {
		if strings.HasPrefix(d.prop, "border") {
			hasBorder = true
		}
	}
	if hasBorder {
		return mappedBlock{}, false // a border-carrying empty td is a Divider, not a Spacer
	}
	confidence := "medium"
	if hasFS && strings.TrimSpace(fs) == "0" {
		confidence = "high"
	}
	ctx.rpt.mapped(path, "email.Spacer", confidence, "an empty, fixed-height cell matched the spacer-table technique")
	return mappedBlock{kind: "Spacer", gsx: "<email.Spacer height=" + qexpr(strconv.Itoa(n)) + " />\n"}, true
}

// detectDivider matches email.Divider: a border-top rule with no other
// content, either directly on td (parity mode's own bare div, or a
// hand-written <hr>) or on the sole row of a nested single-cell table
// (hardened mode's own spacer-table technique, writeDivider).
func detectDivider(td *node, ctx *blockCtx, path string) (mappedBlock, bool) {
	target := unwrapTrivial(td)
	hasBorderTop := false
	if v, ok := target.styleValue("border-top"); ok && strings.TrimSpace(v) != "" && strings.TrimSpace(v) != "0" {
		hasBorderTop = true
	}
	for _, hr := range findAllShallow(td, "hr") {
		_ = hr
		hasBorderTop = true
	}
	if !hasBorderTop {
		if divs := findAllShallow(td, "div"); len(divs) == 1 {
			if v, ok := divs[0].styleValue("border-top"); ok && strings.TrimSpace(v) != "" {
				hasBorderTop = true
			}
		}
	}
	if !hasBorderTop {
		return mappedBlock{}, false
	}
	if strings.TrimSpace(target.innerText()) != "" {
		return mappedBlock{}, false
	}
	ctx.rpt.mapped(path, "email.Divider", "high", "a content-free border-top rule matched the Divider contract")
	return mappedBlock{kind: "Divider", gsx: "<email.Divider />\n"}, true
}

// findAllShallow finds descendants tagged name without crossing into a
// nested <table> (mirrors structverify's own elementHasDescendant
// reasoning: a check on this row must not see a header cell that belongs
// to a deeper, unrelated nested table).
func findAllShallow(n *node, name string) []*node {
	var out []*node
	var walk func(*node)
	walk = func(x *node) {
		for i := range x.children {
			c := &x.children[i]
			if c.tag == name {
				out = append(out, c)
			}
			if c.tag == "table" {
				continue
			}
			walk(c)
		}
	}
	walk(n)
	return out
}

// detectHero matches email.Hero: a row whose only meaningful content is
// one <img> with width and height set.
func detectHero(td *node, ctx *blockCtx, path string) (mappedBlock, bool) {
	imgs := findAllShallow(td, "img")
	if len(imgs) != 1 {
		return mappedBlock{}, false
	}
	img := imgs[0]
	// Reject when the row carries other meaningful text alongside the
	// image (a Column's own image+title+text shape, for instance) —
	// Hero is a full-width image block with no siblings.
	other := td.innerText()
	if strings.TrimSpace(other) != "" {
		return mappedBlock{}, false
	}
	src, _ := img.attr("src")
	alt, hasAlt := img.attr("alt")
	width, hasW := img.attr("width")
	height, hasH := img.attr("height")
	if !hasW {
		if v, ok := img.styleValue("width"); ok {
			width = strings.TrimSuffix(v, "px")
			hasW = true
		}
	}
	if !hasH {
		if v, ok := img.styleValue("height"); ok {
			height = strings.TrimSuffix(v, "px")
			hasH = true
		}
	}
	if src == "" {
		return mappedBlock{}, false
	}
	src, srcNote := sanitizeImgSrc(src)
	if !hasAlt || strings.TrimSpace(alt) == "" {
		alt = "Imported image"
		ctx.rpt.NextSteps = append(ctx.rpt.NextSteps, path+": the source <img> had no alt text; review the synthesized \"Imported image\" placeholder.")
	}
	if !hasW || !hasH {
		width, height = "600", "300"
		ctx.rpt.NextSteps = append(ctx.rpt.NextSteps, path+": the source <img> had no width/height attributes; Hero needs both, so gsxmail import guessed 600x300 — set the real display size.")
	}
	if srcNote != "" {
		ctx.rpt.NextSteps = append(ctx.rpt.NextSteps, path+": "+srcNote)
	}
	confidence := "medium"
	if hasW && hasH {
		confidence = "high"
	}
	ctx.rpt.mapped(path, "email.Hero", confidence, "a lone, sized image matched the Hero contract")
	gsx := "<email.Hero src=" + qexpr(src) + " alt=" + qexpr(alt) +
		" width=" + qexpr(width) + " height=" + qexpr(height) + " />\n"
	return mappedBlock{kind: "Hero", gsx: gsx}, true
}
