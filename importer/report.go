package importer

import (
	"fmt"
	"sort"
	"strings"
)

// mappedLine is one report line for a node the mapper placed onto a
// component, at some confidence.
type mappedLine struct {
	path       string // a short structural path, e.g. "card/row[3]"
	component  string // "email.Headline", "email.Custom", ...
	confidence string // "high" | "medium" | "low"
	note       string // one ASD-STE100 sentence explaining the call
}

// unmappedLine is one report line for a node Import could not place at
// all: it survives in the generated .gsx as an email.Custom block.
type unmappedLine struct {
	path   string
	reason string
}

// Report is Import's honest accounting of what it did: every mapped
// node's confidence and rationale, every unmapped node's path and
// reason, every synthesized props field's
// source, and a next-steps list telling the template's new owner what to
// hand-tune. The CLI prints it as a stdout summary and writes it in full
// as IMPORT-REPORT.md (WriteMarkdown).
type Report struct {
	SourceFile string
	Mapped     []mappedLine
	Unmapped   []unmappedLine
	// Dropped counts every dropEntirely tag actually encountered while
	// sanitizing an unmapped subtree into
	// email.Custom: <script>, <style>, and the handful of document/head
	// tags that carry no email content at all. Unlike Unmapped, a dropped
	// tag's own content never survives anywhere in the generated
	// template — this is the one case "a node it cannot confidently
	// place never gets dropped" does not cover, and the report says so.
	Dropped    map[string]int
	Fields     []propsFieldReportLine
	ThemeNotes []string
	NextSteps  []string
}

// propsFieldReportLine is one synthesized props field's report entry.
type propsFieldReportLine struct {
	Name   string
	GoType string
	Source string
}

func (r *Report) mapped(path, component, confidence, note string) {
	r.Mapped = append(r.Mapped, mappedLine{path: path, component: component, confidence: confidence, note: note})
}

func (r *Report) unmapped(path, reason string) {
	r.Unmapped = append(r.Unmapped, unmappedLine{path: path, reason: reason})
}

// dropped records one dropEntirely tag occurrence.
func (r *Report) dropped(tag string) {
	if r.Dropped == nil {
		r.Dropped = make(map[string]int)
	}
	r.Dropped[tag]++
}

// Summary renders a short, stdout-friendly digest: counts and every
// unmapped path (the CLI's own printed summary; task instructions,
// "report printed to stdout as a summary").
func (r *Report) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "gsxmail import: %s\n", r.SourceFile)
	fmt.Fprintf(&b, "  mapped: %d node(s), unmapped: %d node(s), synthesized props: %d field(s)\n",
		len(r.Mapped), len(r.Unmapped), len(r.Fields))
	if len(r.Unmapped) > 0 {
		b.WriteString("  unmapped:\n")
		for _, u := range r.Unmapped {
			fmt.Fprintf(&b, "    %s: %s\n", u.path, u.reason)
		}
	}
	return b.String()
}

// WriteMarkdown renders the full IMPORT-REPORT.md: the heuristic mapping
// table, the unmapped-node list, the synthesized props table with source
// snippets, theme-extraction notes, and a next-steps section (task
// instructions, item 3).
func (r *Report) WriteMarkdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Import report: %s\n\n", r.SourceFile)
	fmt.Fprintf(&b, "gsxmail import read %s and produced a best-effort template. ", r.SourceFile)
	b.WriteString("This report lists every mapping decision, every node it could not place, and what to review before you ship the result.\n\n")

	b.WriteString("## Mapped nodes\n\n")
	if len(r.Mapped) == 0 {
		b.WriteString("No node mapped to a named component.\n\n")
	} else {
		b.WriteString("| Path | Component | Confidence | Note |\n|---|---|---|---|\n")
		for _, m := range r.Mapped {
			fmt.Fprintf(&b, "| %s | `%s` | %s | %s |\n", m.path, m.component, m.confidence, m.note)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Unmapped nodes\n\n")
	if len(r.Unmapped) == 0 {
		b.WriteString("Every node mapped to a named component; nothing fell back to `email.Custom`.\n\n")
	} else {
		b.WriteString("Each node below survives in `template.gsx` inside an `email.Custom` block, unchanged. Review it and hand-place it onto a named component if one fits.\n\n")
		b.WriteString("| Path | Reason |\n|---|---|\n")
		for _, u := range r.Unmapped {
			fmt.Fprintf(&b, "| %s | %s |\n", u.path, u.reason)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Synthesized props fields\n\n")
	if len(r.Fields) == 0 {
		b.WriteString("Import synthesized no props fields; every value stayed literal in `template.gsx`.\n\n")
	} else {
		b.WriteString("| Field | Type | Source |\n|---|---|---|\n")
		for _, f := range r.Fields {
			fmt.Fprintf(&b, "| `%s` | `%s` | %s |\n", f.Name, f.GoType, f.Source)
		}
		b.WriteString("\n")
	}

	if len(r.Dropped) > 0 {
		b.WriteString("## Dropped entirely\n\n")
		b.WriteString("Unlike an unmapped node (above), the tags below carry no email content at all — a `<script>`, a `<style>` block, or document/head plumbing — so nothing from them survives anywhere in `template.gsx`, not even inside `email.Custom`.\n\n")
		b.WriteString("| Tag | Occurrences |\n|---|---|\n")
		tags := make([]string, 0, len(r.Dropped))
		for tag := range r.Dropped {
			tags = append(tags, tag)
		}
		sort.Strings(tags)
		for _, tag := range tags {
			fmt.Fprintf(&b, "| `<%s>` | %d |\n", tag, r.Dropped[tag])
		}
		b.WriteString("\n")
	}

	if len(r.ThemeNotes) > 0 {
		b.WriteString("## Theme extraction\n\n")
		for _, n := range r.ThemeNotes {
			fmt.Fprintf(&b, "- %s\n", n)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Next steps\n\n")
	steps := r.NextSteps
	if len(steps) == 0 {
		steps = []string{"Review the generated template.gsx and confirm every synthesized props field carries the value you expect."}
	}
	for _, s := range steps {
		fmt.Fprintf(&b, "- %s\n", s)
	}
	return b.String()
}
