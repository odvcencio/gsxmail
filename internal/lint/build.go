package lint

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// caniemailLicense and caniemailAttribution record the caniemail
// dataset's own terms (m9, launch-gate findings), verified by fetching
// https://github.com/HTeuMeuLeu/caniemail's own README directly: it
// states "MIT Licence" (with a link to its own MIT LICENSE file), not
// the CC BY 4.0 this project once assumed. `gsxmail matrix refresh`
// writes both into every regenerated snapshot.json.
const caniemailLicense = "MIT"

const caniemailAttribution = "Client-support data from caniemail (https://www.caniemail.com), maintained at https://github.com/HTeuMeuLeu/caniemail, Copyright (c) 2019 Rémi Parmentier, MIT License."

// clientSpec names one client/platform pair in the caniemail dataset's own
// vocabulary (family + platform slugs) alongside the label and ID the
// trimmed Snapshot exposes it under.
type clientSpec struct {
	id, label, family, platform string
}

// defaultClients is the client/platform set BuildSnapshot resolves support
// data for: the design spec's proposed default (section 16, open question
// 3 — Gmail web/iOS/Android, Apple Mail macOS/iOS, Outlook Windows
// desktop/web, Yahoo web), which the framework owner has not yet ratified
// beyond this proposal. Widening or narrowing this set is a `gsxmail
// matrix refresh` code change, reviewed like any other.
var defaultClients = []clientSpec{
	{"gmail-web", "Gmail (web)", "gmail", "desktop-webmail"},
	{"gmail-ios", "Gmail (iOS)", "gmail", "ios"},
	{"gmail-android", "Gmail (Android)", "gmail", "android"},
	{"apple-mail-macos", "Apple Mail (macOS)", "apple-mail", "macos"},
	{"apple-mail-ios", "Apple Mail (iOS)", "apple-mail", "ios"},
	{"outlook-windows", "Outlook (Windows desktop)", "outlook", "windows"},
	{"outlook-web", "Outlook (web)", "outlook", "outlook-com"},
	{"yahoo-web", "Yahoo (web)", "yahoo", "desktop-webmail"},
}

// excludedSlugPrefixes drops caniemail features that are not style
// properties at all: at-rules, pseudo-classes/elements, selectors, unit
// tests, and value functions. EM101/EM102 lint style properties, not these.
var excludedSlugPrefixes = []string{
	"css-at-", "css-pseudo-", "css-selector-", "css-unit-", "css-function-",
}

var excludedSlugs = map[string]bool{
	"css-comments": true, "css-important": true, "css-variables": true,
	"css-nesting": true, "css-rgb": true, "css-rgba": true,
	"css-modern-color": true, "css-linear-gradient": true,
	"css-radial-gradient": true, "css-conic-gradient": true,
	"css-sytem-ui": true, "css-column-layout-properties": true,
}

// propertyAliases names the extra literal CSS property names one caniemail
// feature's support data also answers for, beyond the property its slug
// names directly (for example, the single "border" feature covers
// border-top, border-bottom, and the other physical-side longhands).
var propertyAliases = map[string][]string{
	"css-border":                   {"border-top", "border-bottom", "border-left", "border-right", "border-width", "border-style", "border-color"},
	"css-margin":                   {"margin-top", "margin-bottom", "margin-left", "margin-right"},
	"css-padding":                  {"padding-top", "padding-bottom", "padding-left", "padding-right"},
	"css-left-right-top-bottom":    {"left", "right", "top", "bottom"},
	"css-margin-inline-start-end":  {"margin-inline-start", "margin-inline-end"},
	"css-padding-inline-start-end": {"padding-inline-start", "padding-inline-end"},
	"css-margin-block-start-end":   {"margin-block-start", "margin-block-end"},
	"css-padding-block-start-end":  {"padding-block-start", "padding-block-end"},
	"css-block-inline-size":        {"block-size", "inline-size"},
}

type rawDataset struct {
	LastUpdateDate string       `json:"last_update_date"`
	Data           []rawFeature `json:"data"`
}

type rawFeature struct {
	Slug     string                                  `json:"slug"`
	Category string                                  `json:"category"`
	Stats    map[string]map[string]map[string]string `json:"stats"`
}

// BuildSnapshot trims raw — the exact bytes of
// https://www.caniemail.com/api/data.json — to the CSS-property,
// default-client-set shape EM101/EM102 need, and returns it both parsed
// and marshaled ready to write to snapshot.json. It performs no I/O:
// `gsxmail matrix refresh` is the only caller that fetches raw over the
// network, so BuildSnapshot itself stays safe to call from a test.
func BuildSnapshot(raw []byte) (*Snapshot, []byte, error) {
	var rd rawDataset
	if err := json.Unmarshal(raw, &rd); err != nil {
		return nil, nil, fmt.Errorf("gsxmail: parsing caniemail dataset: %w", err)
	}
	date := rd.LastUpdateDate
	if len(date) >= 10 {
		date = date[:10]
	}
	if date == "" {
		return nil, nil, fmt.Errorf("gsxmail: caniemail dataset has no last_update_date")
	}

	props := make(map[string]PropertySupport)
	for _, feat := range rd.Data {
		if feat.Category != "css" {
			continue
		}
		if excludedSlugs[feat.Slug] || hasAnyPrefix(feat.Slug, excludedSlugPrefixes) {
			continue
		}
		support := make(map[string]string, len(defaultClients))
		for _, cl := range defaultClients {
			vers := feat.Stats[cl.family][cl.platform]
			if len(vers) == 0 {
				support[cl.id] = "u"
				continue
			}
			support[cl.id] = latestCode(vers)
		}
		names := map[string]bool{strings.TrimPrefix(feat.Slug, "css-"): true}
		for _, alias := range propertyAliases[feat.Slug] {
			names[alias] = true
		}
		for name := range names {
			props[name] = PropertySupport{Slug: feat.Slug, Support: support}
		}
	}
	if len(props) == 0 {
		return nil, nil, fmt.Errorf("gsxmail: trimmed caniemail snapshot has no properties; the dataset shape may have changed")
	}

	clients := make([]Client, len(defaultClients))
	for i, cl := range defaultClients {
		clients[i] = Client{ID: cl.id, Label: cl.label}
	}

	snap := &Snapshot{
		Date:        date,
		Source:      "https://www.caniemail.com/api/data.json",
		License:     caniemailLicense,
		Attribution: caniemailAttribution,
		Clients:     clients,
		Properties:  props,
	}
	out, err := json.MarshalIndent(snap, "", " ")
	if err != nil {
		return nil, nil, fmt.Errorf("gsxmail: encoding trimmed snapshot: %w", err)
	}
	out = append(out, '\n')
	return snap, out, nil
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

var versionSplitRe = regexp.MustCompile(`[.\-]`)

// latestCode picks the highest version key in vers (caniemail orders a
// feature's tested versions oldest to newest, but does not guarantee sort
// order in the JSON map) and returns its support code with any footnote
// marker ("y #1") stripped.
func latestCode(vers map[string]string) string {
	keys := make([]string, 0, len(vers))
	for k := range vers {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return versionLess(keys[i], keys[j]) })
	raw := vers[keys[len(keys)-1]]
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return "u"
	}
	return fields[0]
}

// versionLess compares two caniemail version strings ("2022-02", "16.80",
// "2019") by splitting on '.' and '-' and comparing each part numerically
// when both sides parse as a number, falling back to a string comparison
// otherwise; a numeric part sorts before a non-numeric one at the same
// position.
func versionLess(a, b string) bool {
	pa := versionSplitRe.Split(a, -1)
	pb := versionSplitRe.Split(b, -1)
	for i := 0; i < len(pa) && i < len(pb); i++ {
		na, errA := strconv.Atoi(pa[i])
		nb, errB := strconv.Atoi(pb[i])
		aNum, bNum := errA == nil, errB == nil
		switch {
		case aNum && bNum:
			if na != nb {
				return na < nb
			}
		case aNum != bNum:
			return aNum
		default:
			if pa[i] != pb[i] {
				return pa[i] < pb[i]
			}
		}
	}
	return len(pa) < len(pb)
}
