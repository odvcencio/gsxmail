package importer

import "strings"

// extractHeaderRow matches the Shell header row's own native shape
// (renderhtml.writeBodyOpen/writeBodyOpenParity): a single td holding one
// nested table whose one row holds two cells — a short-code box, then a
// wordmark/tagline pair. It returns ok==false for any other shape, and
// the caller keeps the row in the normal per-row classify loop instead
// of consuming it.
func extractHeaderRow(row *node) (wordmark, tagline, shortCode string, ok bool) {
	content := soleContentCell(row)
	if content == nil {
		return "", "", "", false
	}
	tbl := findFirst(content, "table")
	if tbl == nil {
		return "", "", "", false
	}
	rows := tableRows(tbl)
	if len(rows) != 1 {
		return "", "", "", false
	}
	var cells []*node
	for _, c := range rows[0].elements() {
		if c.tag == "td" {
			cells = append(cells, c)
		}
	}
	if len(cells) != 2 {
		return "", "", "", false
	}
	shortCode = strings.TrimSpace(cells[0].innerText())
	var texts []string
	for _, c := range elementsIgnoringText(cells[1]) {
		if t := strings.TrimSpace(c.innerText()); t != "" {
			texts = append(texts, t)
		}
	}
	if len(texts) == 0 {
		return "", "", "", false
	}
	wordmark = texts[0]
	if len(texts) > 1 {
		tagline = texts[1]
	}
	return wordmark, tagline, shortCode, true
}

// extractFooterRow matches Footer's own native shape (renderhtml.
// writeFooter): a single td holding exactly two divs, the first carrying
// a border-top rule (the signoff line's own visual divider).
func extractFooterRow(row *node) (signoff, note string, ok bool) {
	content := soleContentCell(row)
	if content == nil {
		return "", "", false
	}
	divs := elementsIgnoringText(content)
	if len(divs) != 2 || divs[0].tag != "div" || divs[1].tag != "div" {
		return "", "", false
	}
	if v, has := divs[0].styleValue("border-top"); !has || strings.TrimSpace(v) == "" {
		return "", "", false
	}
	signoff = strings.TrimSpace(divs[0].innerText())
	note = strings.TrimSpace(divs[1].innerText())
	if signoff == "" {
		return "", "", false
	}
	return signoff, note, true
}

// deriveWordmark falls back to a Shell wordmark when extractHeaderRow
// finds no native header shape: title's first word, or "Imported" when
// title itself is empty.
func deriveWordmark(title string) string {
	fields := strings.Fields(title)
	if len(fields) == 0 {
		return "Imported"
	}
	return fields[0]
}

// deriveShortCode builds a 2-3 letter short code from wordmark, matching
// the shipped gallery's own convention (receipt.gsx's "ACM" for "ACME").
func deriveShortCode(wordmark string) string {
	upper := strings.ToUpper(wordmark)
	var letters []rune
	for _, r := range upper {
		if r >= 'A' && r <= 'Z' {
			letters = append(letters, r)
		}
		if len(letters) == 3 {
			break
		}
	}
	if len(letters) == 0 {
		return "IMP"
	}
	return string(letters)
}

// extractTheme recovers a best-effort themeTokens from card's own literal
// colors (task instructions, design constraint 2: "theme-token extraction
// from the dominant colors into a generated Theme literal"). It is a
// deliberately narrow extraction — card background, card border, the
// page ground behind the card, the accent color a button/anchor uses,
// and the ink color a heading uses — rather than a full palette
// clustering pass; notes records exactly what it did and did not
// recover, for the report's "Theme extraction" section.
func extractTheme(body, card *node) (themeTokens, []string) {
	t := defaultThemeTokens()
	var notes []string

	if v, ok := card.styleValue("background-color"); ok && isHex(v) {
		t.ColorCard = normalizeHex(v)
		notes = append(notes, "ColorCard: read from the card table's own background-color.")
	} else {
		notes = append(notes, "ColorCard: no literal background-color found on the card table; kept the default.")
	}

	if v, ok := card.styleValue("border"); ok {
		if hex := lastHexIn(v); hex != "" {
			t.ColorBorder = hex
			notes = append(notes, "ColorBorder: read from the card table's own border shorthand.")
		}
	}

	groundFound := false
	for _, tbl := range findAll(body, "table") {
		if tbl == card {
			continue
		}
		if v, ok := tbl.styleValue("background-color"); ok && isHex(v) {
			t.ColorGround = normalizeHex(v)
			groundFound = true
			break
		}
	}
	if groundFound {
		notes = append(notes, "ColorGround: read from the first wrapping table's own background-color.")
	} else {
		notes = append(notes, "ColorGround: no wrapping table with a literal background-color was found; kept the default.")
	}

	accentFound := false
	for _, a := range findAll(body, "a") {
		if v := findButtonFaceColor(body, a); v != "" && isHex(v) {
			t.ColorAccent = normalizeHex(v)
			accentFound = true
			break
		}
	}
	if accentFound {
		notes = append(notes, "ColorAccent: read from the first button-shaped anchor's own background-color.")
	} else {
		notes = append(notes, "ColorAccent: no button-shaped anchor with a literal background-color was found; kept the default.")
	}

	inkFound := false
	var walk func(*node) *node
	walk = func(n *node) *node {
		if looksLikeHeading(n) {
			return n
		}
		for i := range n.children {
			if f := walk(&n.children[i]); f != nil {
				return f
			}
		}
		return nil
	}
	if h := walk(body); h != nil {
		if v, ok := h.styleValue("color"); ok && isHex(v) {
			t.ColorInk = normalizeHex(v)
			inkFound = true
		}
	}
	if inkFound {
		notes = append(notes, "ColorInk: read from the first heading-shaped element's own text color.")
	} else {
		notes = append(notes, "ColorInk: no heading-shaped element with a literal text color was found; kept the default.")
	}

	notes = append(notes, "ColorPanel, ColorBody, ColorMuted, ColorFaint, and both fonts are not extracted in this release; they carry DefaultTheme()'s own values — review them against the source's own body copy and muted-label colors.")

	return t, notes
}

func isHex(s string) bool {
	return lastHexIn(s) != ""
}

// lastHexIn returns the last "#RRGGBB"-shaped token in s (a border
// shorthand like "1px solid #E2E2E8" carries its color last), or "".
func lastHexIn(s string) string {
	fields := strings.Fields(s)
	for i := len(fields) - 1; i >= 0; i-- {
		if _, _, _, ok := parseHex(fields[i]); ok {
			return normalizeHex(fields[i])
		}
	}
	return ""
}

func normalizeHex(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "#") {
		return s
	}
	return "#" + strings.ToUpper(strings.TrimPrefix(s, "#"))
}
