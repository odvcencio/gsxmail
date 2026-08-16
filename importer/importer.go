package importer

import (
	"fmt"
	"strings"
)

// Options configures Import.
type Options struct {
	// PackageName names the generated Go package (props.go's own package
	// clause and template.gsx's own package clause must match — gosx
	// compiles one directory as one package). Defaults to "emails".
	PackageName string
	// TemplateName names the generated Go function/component
	// ("WelcomeEmail"). Defaults to a name derived from the source
	// file's own <title>, or "ImportedEmail" when no title is found.
	TemplateName string
}

// Result is Import's complete output: the generated .gsx source, the
// generated props.go source, the sample props JSON fixture, and the
// honest Report (task instructions, item 3 and item 5). The CLI's own
// runImport is a thin wrapper that writes these four values to disk as
// template.gsx, props.go, props.sample.json, and IMPORT-REPORT.md.
type Result struct {
	// TemplateName is the generated component's own name ("WelcomeEmail"):
	// Options.TemplateName when set, else derived from the source
	// document's own <title>.
	TemplateName    string
	TemplateGSX     string
	PropsGo         string
	SamplePropsJSON string
	Report          *Report

	// blocks is kept for tests that assert the recovered block-type
	// sequence (the self-round-trip and idempotence proof bar; task
	// instructions, item 4). It is not one of the four generated files.
	blocks []mappedBlock
}

// BlockKinds returns the recovered block sequence's own component names,
// in source order — "Headline", "Panel", "StatTable", "Custom", and so
// on — the shape a structure-fidelity assertion compares (task
// instructions, item 4(a): "assert the importer recovers the component
// structure ... assert block types in order").
func (r *Result) BlockKinds() []string {
	out := make([]string, len(r.blocks))
	for i, b := range r.blocks {
		out[i] = b.kind
	}
	return out
}

// Import parses html (an existing email template's rendered HTML) and
// reverse-maps it onto gsxmail's shipped email.* components (design spec
// section 15, WP5.5; pixel dossier section 7.2(1)). It never returns an
// error for malformed or unrecognized markup — gotreesitter's error-
// tolerant HTML grammar guarantees a walkable tree (pixel dossier section
// 7.1), and every node Import cannot place becomes an email.Custom
// fallback plus a Report line instead of a failure. The one error case is
// a byte stream gotreesitter itself cannot parse into any tree at all.
func Import(html []byte, sourceName string, opts Options) (*Result, error) {
	root, err := parseHTML(html)
	if err != nil {
		return nil, err
	}

	pkg := opts.PackageName
	if pkg == "" {
		pkg = "emails"
	}

	body := findBody(root)
	title := findTitle(root)
	lang := findLang(root)

	templateName := opts.TemplateName
	if templateName == "" {
		templateName = deriveTemplateName(title)
	}

	rpt := &Report{SourceFile: sourceName}
	props := newPropsBuilder()
	ctx := &blockCtx{props: props, rpt: rpt}

	preheaderText := findPreheader(body)
	if len(preheaderText) > 150 {
		rpt.NextSteps = append(rpt.NextSteps,
			"the source preheader text was longer than 150 characters and was truncated; EM171 caps a literal preheader at 150.")
		preheaderText = truncateRunes(preheaderText, 150)
	}

	card := findCard(body)

	var blocks []mappedBlock
	var wordmark, tagline, shortCode, signoff, footerNote string
	cardWidth := 600
	theme := defaultThemeTokens()
	themeNotes := []string{"No card table was found, so no theme extraction ran; the generated Theme is DefaultTheme()'s own values."}

	if card == nil {
		rpt.unmapped("document", "no plausible 320-820px card table was found; the whole body was preserved as email.Custom")
		blocks = append(blocks, mappedBlock{kind: "Custom", gsx: writeWholeBodyCustom(body)})
	} else {
		if w, ok := tableWidth(card); ok {
			cardWidth = w
		}
		theme, themeNotes = extractTheme(body, card)

		rows := tableRows(card)
		if len(rows) == 0 {
			rpt.unmapped("card", "the card table has no rows")
		}

		startIdx, endIdx := 0, len(rows)
		headerMapped := false
		if len(rows) > 0 {
			if w, tg, sc, ok := extractHeaderRow(rows[0]); ok {
				wordmark, tagline, shortCode = w, tg, sc
				startIdx = 1
				headerMapped = true
			}
		}
		footerMapped := false
		if endIdx > startIdx {
			if s, n, ok := extractFooterRow(rows[endIdx-1]); ok {
				signoff, footerNote = s, n
				endIdx--
				footerMapped = true
			}
		}
		// Report lines are appended in row order (task instructions, item
		// 3: the report is a first-class output, and a reader scans it
		// top to bottom against the source document) even though the
		// header and footer rows are recovered before the main loop runs.
		if headerMapped {
			rpt.mapped("card/row[1]", "email.Shell (header)", "high",
				"the first row's own short-code + wordmark/tagline shape matched the Shell header contract")
		}
		for i := startIdx; i < endIdx; i++ {
			path := fmt.Sprintf("card/row[%d]", i+1)
			blocks = append(blocks, classifyRow(rows[i], ctx, path))
		}
		if footerMapped {
			rpt.mapped(fmt.Sprintf("card/row[%d]", len(rows)), "email.Footer", "high",
				"the last row's own two-div, border-top shape matched the Footer contract")
		}
	}
	rpt.ThemeNotes = themeNotes

	if wordmark == "" {
		wordmark = deriveWordmark(title)
	}
	if shortCode == "" {
		shortCode = deriveShortCode(wordmark)
	}
	if signoff != "" {
		footerField := props.field("Signoff", signoff, "the Footer's own signoff line")
		gsx := "<email.Footer signoff={props." + footerField + "}"
		if footerNote != "" {
			noteField := props.field("FooterNote", footerNote, "the Footer's own small-print note")
			gsx += " note={props." + noteField + "}"
		} else {
			gsx += ` note=""`
		}
		gsx += " />\n"
		blocks = append(blocks, mappedBlock{kind: "Footer", gsx: gsx})
	}

	titleField := props.field("Title", orDefault(title, wordmark), "the document's own <title>")
	wordmarkField := props.field("Wordmark", wordmark, "the Shell header's own wordmark text")
	taglineField := ""
	if tagline != "" {
		taglineField = props.field("Tagline", tagline, "the Shell header's own tagline text")
	}
	preheaderField := ""
	if preheaderText != "" {
		preheaderField = props.field("Preheader", preheaderText, "the hidden inbox-preview div's own text")
	}

	shell := shellInfo{
		title: title, lang: lang, wordmark: wordmark, tagline: tagline,
		shortCode: shortCode, preheader: preheaderText, cardWidth: cardWidth, theme: theme,
	}

	return assemble(pkg, templateName, sourceName, shell, blocks, props, rpt,
		titleField, wordmarkField, taglineField, preheaderField)
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func deriveTemplateName(title string) string {
	name := sanitizeIdent(title)
	if name == "" {
		return "ImportedEmail"
	}
	if !strings.HasSuffix(name, "Email") {
		name += "Email"
	}
	return name
}
