package doc

import (
	"net/url"
	"strings"
)

// SafeURL reports whether raw parses as a URL whose scheme is on the
// allowlist EM110 states: https, http, or mailto. It is the single,
// centralized scheme policy for every dynamic URL in a resolved tree —
// CTA and Button hrefs, and every <a href> anywhere in a Custom subtree —
// and both writers (renderhtml, rendertext) enforce it at emission time,
// so an unsafe value never reaches either output part even when a caller
// drives a writer directly, skipping the email lint. Ported from
// gridiron's internal/emailkit hasSafeURLScheme (a 2026 security review
// item there), widened from http/https to also allow mailto per EM110.
// Before this function existed, each writer carried its own unexported
// copy of the same switch; two copies can drift apart and silently leave
// one part enforcing less than the other.
func SafeURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "mailto":
		return true
	default:
		return false
	}
}

// SafeImgSrc reports whether raw is an image src on the allowlist EM111
// states: an absolute https URL. Mail clients cannot resolve relative
// paths, and a non-https origin would both leak the recipient's request
// over plaintext and let a dynamic props value point the rendered <img>
// anywhere at send time. One rule covers every image site in a resolved
// tree — Hero, Column, and every <img src> anywhere in a Custom subtree —
// enforced by both writers at emission time for the same reason SafeURL
// is: Load-time lint (EM111) cannot see a value that only exists once
// props resolve.
func SafeImgSrc(raw string) bool {
	return strings.HasPrefix(strings.TrimSpace(raw), "https://")
}

// URLSite is one unsafe dynamic value AuditURLs found in a resolved tree.
type URLSite struct {
	Component string // "email.CTA" | "email.Button" | "email.Hero" | "email.Column" | "email.Custom"
	Field     string // "href" or "img src"
	Value     string
}

// AuditURLs walks a fully resolved tree and returns, in deterministic
// block order (Custom subtrees in depth-first source order), one URLSite
// for every href failing SafeURL and every img src failing SafeImgSrc —
// the complete set of values the two writers will reject: CTA and Button
// hrefs, Hero images, Column images (a Column with no imgSrc carries no
// URL to judge, so none is reported), and every <a href>/<img src>
// however deeply nested inside a Custom block. It is the coverage oracle
// for the writers' own emission-time checks: a test can assert that every
// site AuditURLs reports disappears from both rendered parts while its
// alt text survives, and that a value AuditURLs does not report renders
// untouched. It deliberately reports rather than rewrites: the fallback
// shape (label-only button face, alt-text stand-in) is a presentation
// decision each writer makes for its own medium.
func AuditURLs(resolved *Resolved) []URLSite {
	var out []URLSite
	add := func(component, field, value string) {
		out = append(out, URLSite{Component: component, Field: field, Value: value})
	}
	for _, b := range resolved.Blocks {
		switch v := b.(type) {
		case ResolvedCTA:
			if !SafeURL(v.Href) {
				add("email.CTA", "href", v.Href)
			}
		case ResolvedButton:
			if !SafeURL(v.Href) {
				add("email.Button", "href", v.Href)
			}
		case ResolvedHero:
			if !SafeImgSrc(v.Src) {
				add("email.Hero", "img src", v.Src)
			}
		case ResolvedColumns:
			for _, c := range v.Columns {
				if c.ImgSrc != "" && !SafeImgSrc(c.ImgSrc) {
					add("email.Column", "img src", c.ImgSrc)
				}
			}
		case ResolvedCustom:
			auditCustomNode(v.Root, add)
		}
	}
	return out
}

func auditCustomNode(n ResolvedCustomNode, add func(component, field, value string)) {
	if n.IsText {
		return
	}
	switch n.Tag {
	case "a":
		for _, a := range n.Attrs {
			if a.Name == "href" && !SafeURL(a.Value) {
				add("email.Custom", "href", a.Value)
			}
		}
	case "img":
		for _, a := range n.Attrs {
			if a.Name == "src" && !SafeImgSrc(a.Value) {
				add("email.Custom", "img src", a.Value)
			}
		}
	}
	for _, c := range n.Children {
		auditCustomNode(c, add)
	}
}
