package gsxmail_test

import (
	"fmt"
	"html"
	"strings"
	"testing"

	xhtml "golang.org/x/net/html"
)

// adminInviteHTMLTemplate is copied verbatim from gridiron-2000's
// internal/league/admin.go (inviteEmailHTMLTemplate, admin.go:179-275): the
// production invite email T3 render-equivalence targets. It stays a test
// fixture only; the spec (section 6.4) states the gsxmail writer's
// declared byte deltas from this template are rendering-neutral (entity
// spelling, and two added color-scheme meta tags).
const adminInviteHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>GRIDIRON 2000</title>
</head>
<body style="margin:0; padding:0; background-color:#070A16;">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="width:100%%; background-color:#070A16; margin:0; padding:0;">
<tr>
<td align="center" style="padding:32px 16px;">
<table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px; max-width:600px; background-color:#0B1024; border:1px solid #26305A; border-radius:4px;">
<tr>
<td style="padding:24px 32px 20px 32px; border-bottom:1px solid #26305A;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0">
<tr>
<td width="44" valign="middle" style="padding-right:12px;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="border:1px solid #38E8FF; border-radius:2px;"><tr><td style="width:40px; height:40px; text-align:center; vertical-align:middle; color:#38E8FF; font-family:'SFMono-Regular',Consolas,Menlo,monospace; font-size:13px; font-weight:700;">G2K</td></tr></table>
</td>
<td valign="middle">
<div style="color:#F7F4EA; font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif; font-size:19px; font-weight:800; letter-spacing:-0.02em; text-transform:uppercase;">GRIDIRON 2000</div>
<div style="color:#8995B8; font-family:'SFMono-Regular',Consolas,Menlo,monospace; font-size:11px; letter-spacing:0.08em; text-transform:uppercase; margin-top:3px;">DYNASTY FANTASY LEAGUE</div>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td style="padding:20px 32px 0 32px;">
<div style="color:#38E8FF; font-family:'SFMono-Regular',Consolas,Menlo,monospace; font-size:12px; letter-spacing:0.08em; text-transform:uppercase;">&#9679; DRAFT EVENT // %s</div>
</td>
</tr>
<tr>
<td style="padding:14px 32px 0 32px;">
<div style="color:#F7F4EA; font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif; font-size:32px; font-weight:800; letter-spacing:-0.03em; text-transform:uppercase; line-height:1.08;">YOU'RE IN.</div>
<div style="color:#C2CAE1; font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif; font-size:15px; line-height:1.6; margin-top:12px;">A seat is holding for you in an eight-manager dynasty league. Aqua vs Orange. Rosters carry over. Receipts are forever.</div>
</td>
</tr>
<tr>
<td style="padding:24px 32px 0 32px;">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="width:100%%; border:1px solid #26305A; border-radius:4px; background-color:#131B3B;">
<tr>
<td style="padding:16px 20px; border-bottom:1px solid #26305A;">
<span style="display:inline-block; width:88px; color:#8995B8; font-family:'SFMono-Regular',Consolas,Menlo,monospace; font-size:11px; letter-spacing:0.06em; text-transform:uppercase; vertical-align:top;">DRAFT</span>
<span style="color:#F7F4EA; font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif; font-size:14px;">%s &middot; %s</span>
</td>
</tr>
<tr>
<td style="padding:16px 20px; border-bottom:1px solid #26305A;">
<span style="display:inline-block; width:88px; color:#8995B8; font-family:'SFMono-Regular',Consolas,Menlo,monospace; font-size:11px; letter-spacing:0.06em; text-transform:uppercase; vertical-align:top;">VENUE</span>
<span style="color:#F7F4EA; font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif; font-size:14px;">During the Dolphins preseason game &mdash; bring both screens.</span>
</td>
</tr>
<tr>
<td style="padding:16px 20px;">
<span style="display:inline-block; width:88px; color:#8995B8; font-family:'SFMono-Regular',Consolas,Menlo,monospace; font-size:11px; letter-spacing:0.06em; text-transform:uppercase; vertical-align:top;">YOUR KEY</span>
<span style="color:#F7F4EA; font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif; font-size:14px;">Sign in with Google as %s</span>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td align="center" style="padding:28px 32px 0 32px;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0">
<tr>
<td style="border-radius:2px; background-color:#38E8FF;">
<a href="%s" style="display:inline-block; padding:14px 30px; color:#070A16; font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif; font-size:14px; font-weight:800; letter-spacing:0.04em; text-transform:uppercase; text-decoration:none; border-radius:2px;">CLAIM YOUR SEAT &rarr;</a>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td style="padding:28px 32px 0 32px;">
<div style="color:#8995B8; font-family:'SFMono-Regular',Consolas,Menlo,monospace; font-size:11px; letter-spacing:0.08em; text-transform:uppercase; margin-bottom:12px;">NEXT STEPS //</div>
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0">
<tr><td style="padding:5px 0; color:#C2CAE1; font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif; font-size:14px;"><span style="color:#38E8FF; font-family:'SFMono-Regular',Consolas,Menlo,monospace; font-weight:700;">1.</span>&nbsp; Claim your seat</td></tr>
<tr><td style="padding:5px 0; color:#C2CAE1; font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif; font-size:14px;"><span style="color:#38E8FF; font-family:'SFMono-Regular',Consolas,Menlo,monospace; font-weight:700;">2.</span>&nbsp; Rename your team</td></tr>
<tr><td style="padding:5px 0; color:#C2CAE1; font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif; font-size:14px;"><span style="color:#38E8FF; font-family:'SFMono-Regular',Consolas,Menlo,monospace; font-weight:700;">3.</span>&nbsp; Build your Big Board</td></tr>
<tr><td style="padding:5px 0; color:#C2CAE1; font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif; font-size:14px;"><span style="color:#38E8FF; font-family:'SFMono-Regular',Consolas,Menlo,monospace; font-weight:700;">4.</span>&nbsp; Read the Rules page</td></tr>
</table>
</td>
</tr>
<tr>
<td style="padding:28px 32px 32px 32px;">
<div style="border-top:1px solid #26305A; padding-top:20px; color:#C2CAE1; font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif; font-size:14px;">&mdash; The Commissioner</div>
<div style="color:#5C6690; font-family:'SFMono-Regular',Consolas,Menlo,monospace; font-size:10px; letter-spacing:0.04em; text-transform:uppercase; margin-top:10px;">GRIDIRON 2000 &middot; Eight seats. One trophy. Permanent group-chat evidence.</div>
</td>
</tr>
</table>
</td>
</tr>
</table>
</body>
</html>
`

// adminInviteHTML reproduces admin.go's inviteEmailHTML: it escapes every
// attacker-influenced field with html.EscapeString and fills the template
// above, exactly as the production handler does.
func adminInviteHTML(shortDate, longDate, draftTime, leagueURL, email string) string {
	return fmt.Sprintf(adminInviteHTMLTemplate,
		html.EscapeString(shortDate),
		html.EscapeString(longDate),
		html.EscapeString(draftTime),
		html.EscapeString(email),
		html.EscapeString(leagueURL),
	)
}

// TestInvitePartyWithAdmin is T3: it renders gsxmail's InviteEmail and
// admin.go's hand-written template from the same fixture props, parses
// both as DOM trees, and compares them structurally. Entity spelling
// ("&middot;" vs "·") is excluded by design: an HTML parser decodes both
// to the same rune before this test ever compares them. The two
// color-scheme meta tags are gsxmail's one documented, spec-sanctioned
// addition (section 6.4) and are stripped from gsxmail's tree before the
// comparison.
func TestInvitePartyWithAdmin(t *testing.T) {
	fixture := inviteFixtureProps()

	adminHTML := adminInviteHTML(fixture.ShortDate, fixture.LongDate, fixture.DraftTime, fixture.LeagueURL, fixture.Email)

	set := loadInviteSet(t)
	parts, err := set.Render("InviteEmail", fixture)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	adminDoc, err := xhtml.Parse(strings.NewReader(adminHTML))
	if err != nil {
		t.Fatalf("parsing admin.go's HTML: %v", err)
	}
	gsxDoc, err := xhtml.Parse(strings.NewReader(parts.HTML))
	if err != nil {
		t.Fatalf("parsing gsxmail's HTML: %v", err)
	}

	stripColorSchemeMetas(gsxDoc)

	var diffs []string
	diffNodes(adminDoc, gsxDoc, &diffs)
	if len(diffs) > 0 {
		t.Errorf("gsxmail's InviteEmail is not DOM-equivalent to admin.go's invite template:\n%s", strings.Join(diffs, "\n"))
	}
}

// stripColorSchemeMetas removes the two <meta name="color-scheme"...> and
// <meta name="supported-color-schemes"...> elements gsxmail adds (spec
// section 6.4's one documented addition over the hand template).
func stripColorSchemeMetas(n *xhtml.Node) {
	var toRemove []*xhtml.Node
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode && n.Data == "meta" {
			for _, a := range n.Attr {
				if a.Key == "name" && (a.Val == "color-scheme" || a.Val == "supported-color-schemes") {
					toRemove = append(toRemove, n)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	for _, node := range toRemove {
		node.Parent.RemoveChild(node)
	}
}

// diffNodes walks a and b in lockstep and appends a description of every
// structural difference to diffs: mismatched element tags, mismatched
// attribute sets (by name and decoded value), and mismatched normalized
// text content. Whitespace-only text nodes are skipped on both sides, and
// every other text node's content is collapsed to single spaces and
// trimmed before comparing, matching the "compare tree, text, and style
// declarations" charge from spec section 11, T3.
func diffNodes(a, b *xhtml.Node, diffs *[]string) {
	ac, bc := significantChildren(a), significantChildren(b)
	if len(ac) != len(bc) {
		*diffs = append(*diffs, fmt.Sprintf("child count mismatch under <%s>: admin has %d, gsxmail has %d", a.Data, len(ac), len(bc)))
		n := len(ac)
		if len(bc) < n {
			n = len(bc)
		}
		ac, bc = ac[:n], bc[:n]
	}
	for i := range ac {
		an, bn := ac[i], bc[i]
		if an.Type != bn.Type {
			*diffs = append(*diffs, fmt.Sprintf("node type mismatch: admin %v, gsxmail %v", an.Type, bn.Type))
			continue
		}
		switch an.Type {
		case xhtml.ElementNode:
			if an.Data != bn.Data {
				*diffs = append(*diffs, fmt.Sprintf("tag mismatch: admin <%s>, gsxmail <%s>", an.Data, bn.Data))
			}
			diffAttrs(an, bn, diffs)
			diffNodes(an, bn, diffs)
		case xhtml.TextNode:
			at, bt := normalizeText(an.Data), normalizeText(bn.Data)
			if at != bt {
				*diffs = append(*diffs, fmt.Sprintf("text mismatch: admin %q, gsxmail %q", at, bt))
			}
		}
	}
}

func diffAttrs(a, b *xhtml.Node, diffs *[]string) {
	am := attrMap(a)
	bm := attrMap(b)
	for k, av := range am {
		bv, ok := bm[k]
		if !ok {
			*diffs = append(*diffs, fmt.Sprintf("<%s>: admin has attribute %s=%q, gsxmail is missing it", a.Data, k, av))
			continue
		}
		if av != bv {
			*diffs = append(*diffs, fmt.Sprintf("<%s>: attribute %s mismatch: admin %q, gsxmail %q", a.Data, k, av, bv))
		}
	}
	for k, bv := range bm {
		if _, ok := am[k]; !ok {
			*diffs = append(*diffs, fmt.Sprintf("<%s>: gsxmail has attribute %s=%q, admin does not", a.Data, k, bv))
		}
	}
}

func attrMap(n *xhtml.Node) map[string]string {
	m := make(map[string]string, len(n.Attr))
	for _, a := range n.Attr {
		m[a.Key] = a.Val
	}
	return m
}

// significantChildren returns n's element and non-blank text children,
// dropping the whitespace-only text nodes an HTML parser inserts between
// tags (formatting artifacts, not content).
func significantChildren(n *xhtml.Node) []*xhtml.Node {
	var out []*xhtml.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case xhtml.ElementNode:
			out = append(out, c)
		case xhtml.TextNode:
			if strings.TrimSpace(c.Data) != "" {
				out = append(out, c)
			}
		}
	}
	return out
}

func normalizeText(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
