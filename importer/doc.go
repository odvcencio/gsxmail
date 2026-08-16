// Package importer implements `gsxmail import` (design spec section 15,
// WP5.5; pixel dossier section 7.2(1)): it parses an existing email HTML
// file — MJML compiled output, react-email output, or hand-written table
// soup — and reverse-maps it onto gsxmail's shipped email.* components,
// emitting a best-effort .gsx template, a typed props struct, a sample
// props JSON fixture, and an honest report of everything it could not
// place.
//
// # Isolation
//
// This package is gsxmail's second deliberate gotreesitter import (the
// first is internal/structverify): it parses with gotreesitter's HTML
// grammar (github.com/odvcencio/gotreesitter), whose error-tolerant
// grammar is the enabling property — real email HTML is full of unclosed
// tags and conditional comments, and the grammar still produces a
// walkable tree instead of a hard failure (pixel dossier section 7.1).
// gsxmail's own render path (renderhtml, doc, lower, gsxmail.go) never
// imports gotreesitter or this package; only the CLI does. See
// isolation_test.go for the build-graph proof, mirroring
// structural_isolation_test.go's own guard at the module root.
//
// # What Import produces
//
// Import never fails closed on a node it cannot place: every
// unrecognized subtree becomes an email.Custom escape-hatch block (the
// subtree is preserved, sanitized just enough to stay lint-clean — see
// Result and the CLI's IMPORT-REPORT.md) and a line in the returned
// Report. The generated .gsx is meant to load and render immediately
// (gsxmail.Load, then Set.Render with the emitted sample props): that
// "it just works, then you tune it" promise is WP5.5's whole point, not
// a nice-to-have — see the README's "Import from existing HTML" section.
package importer
