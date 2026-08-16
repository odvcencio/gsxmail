# Changelog

All notable changes to gsxmail are documented in this file.

## Unreleased

### Added

- `typesafe/`: resolves each template's declared props struct with
  `go/types`. It reads the sibling `*.go` files in the template's own
  directory, then type-checks every expression against the resolved
  fields.
- The email lint catalog, EM001 through EM112 (design spec section 8):
  - disallowed-element rules: EM001-EM006
  - expression-dialect rules: EM010-EM015
  - component rules: EM020, EM030-EM033
  - caniemail client-support rules: EM101-EM102
  - style-syntax and class rules: EM103-EM104
  - href and img rules: EM110-EM112
- `Load` now runs the full lint catalog before it lowers anything. It
  fails closed on any error-severity finding, and returns the complete
  diagnostic list as a `*LintError`.
- `Set.Check` returns every finding `Load` collected for a successfully
  loaded `Set`, including warnings.
- An embedded, dated snapshot of the
  [caniemail](https://www.caniemail.com) client-support dataset
  (`lint/snapshot.json`). It covers CSS-property features only, for one
  default client set:
  - Gmail: web, iOS, Android
  - Apple Mail: macOS, iOS
  - Outlook: Windows desktop, web
  - Yahoo: web
- The `gsxmail check` CLI verb. It runs `Load` and prints every finding,
  in text or JSON, and exits 1 on any error-severity finding.
- The `gsxmail matrix refresh` CLI verb. It downloads the caniemail
  dataset, rewrites the embedded snapshot, and prints the per-client
  support diffs for review. It is the only gsxmail command that touches
  the network.

### Changed

- Props type checking now happens in two layers, not one. `typesafe/`
  resolves props types and checks expressions at `Load` time.
  `doc.Resolve` and `renderhtml`'s href-scheme check still re-verify the
  same guarantees at `Render` time, against the actual props value.
  Neither layer replaces the other; both must pass.

### Known limitations

- `gsxmail dev` is not yet implemented.
- The Gmail clip size budget (`Options.MaxHTMLBytes`) is accepted but not
  enforced yet.
- `<If>` and `<Each>` pass the lint catalog (EM030-EM033) when used
  correctly. `lower.Lower` cannot render either one yet; both are a
  later work package.
- `gsxmail check` always runs with an empty `Options.Helpers`. It cannot
  validate EM014/EM015 helper bindings meaningfully. Use `Set.Check()`
  in your own test instead.
