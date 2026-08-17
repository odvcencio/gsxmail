// Package gsxmail compiles gosx email templates to a deterministic
// multipart pair: pixel-targeted HTML plus a matched 72-column
// plain-text twin, from one source tree. See the package README for the
// full pitch, the component reference, and every guarantee in detail;
// this file is the short overview godoc shows first.
//
// # The pipeline
//
// Load runs three stages over every *.gsx file in an fs.FS, in order,
// each one gated on the last:
//
//  1. Compile — gosx.Compile parses one *.gsx file's source into a
//     typed IR program. A file that does not parse fails Load closed
//     immediately (ErrCompile), before any other file's own compile even
//     runs.
//  2. Check — the email lint (EM001 through EM201) walks every compiled
//     component's tree: disallowed
//     elements and attributes, an expression outside the email dialect, a
//     style property no target client supports, an unknown or missing
//     email.* attribute, and more. A template's declared props struct is
//     resolved with go/types here too, so a missing field or a
//     non-scalar interpolation is a Load-time diagnostic, not a
//     render-time surprise. Every finding across every template
//     accumulates before Load decides anything; an error-severity
//     finding anywhere fails Load closed with a *LintError carrying the
//     complete list, and nothing lowers.
//  3. Lower — only once every template has cleared the lint does Lower
//     convert each one's IR into gsxmail's own EmailDoc tree: email.*
//     stdlib tags resolved, <Each>/<If> builtins inlined, a raw-element
//     Custom subtree carried through unmodified. Lower is not itself a
//     fail-closed gate (its own errors, wrapped in ErrLower, are a
//     backstop for a shape the lint's own rules do not yet police, not a
//     second lint pass).
//
// Render then evaluates one EmailDoc against one concrete props value —
// pure, no clock, no network, no unordered map iteration — and writes
// both parts from the same resolved tree, so the text part can never
// drift from the HTML part.
//
// # Two check layers, one guarantee
//
// gsxmail checks a template's props twice, on purpose: internal/typesafe
// resolves the declared props struct with go/types at Load, so most
// mistakes surface as a precise EM012/EM013 diagnostic before anything
// ever renders; internal/doc's Resolve re-checks every field by
// reflection at Render, against the actual value received, so a
// map[string]any props value (gsxmail render's own CLI path, decoded
// from JSON with no static Go type to check) or any Load/Render mismatch
// still fails closed instead of rendering a silently empty or unsafe
// value. See "Two layers, one guarantee" in the README for the worked
// example.
//
// # Two output contracts
//
// renderhtml.Write (reached through Render) emits one of two HTML
// contracts, selected by Options.Outlook or a template's own Shell
// outlook attribute: the hardened, bulletproof default (an Outlook ghost
// table, doubled DPI-fix widths, a real StatTable data-table contract,
// and every other per-component rule), or parity mode ("off"), which
// emits the original byte stream unchanged for a consumer whose own
// equivalence test pins those exact bytes. See "Output contracts" in the
// README for the full per-component table.
//
// # Package map
//
// The root gsxmail package is the whole public surface: Load, Set,
// Options, Parts, Diagnostic, Theme, and the sentinel errors below. Its
// own render path — internal/doc, internal/lower, internal/typesafe,
// internal/lint, renderhtml, rendertext — is internal, reachable only
// through Load/Render/Check; none of it is a promise this module keeps
// across a minor version. importer (m31labs.dev/gsxmail/importer) is the
// one consumer-facing package outside the root: it reverse-maps existing
// HTML onto gsxmail's own email.* components for the `gsxmail import`
// verb, and is the only place besides the CLI that imports gotreesitter.
package gsxmail
