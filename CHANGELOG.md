# Changelog

All notable changes to gsxmail are documented in this file.

## Unreleased

### Added (polish items 2, 4, 6)

- **Item 6.** New sentinel errors — `ErrCompile`, `ErrLower`,
  `ErrDuplicateTemplate`, `ErrUnknownTemplate`, `ErrPropsMismatch`,
  `ErrNilProps`, `ErrResolve` — wrap every error `Load` and `Render` can
  return, so a caller classifies a failure with `errors.Is` instead of
  matching an error message's own text. README gains a new "Error
  taxonomy" section.
- **Item 2.** Four runnable `Example` functions (`ExampleLoad`,
  `ExampleSet_Render`, `ExampleSet_Check`, `ExampleTerminalTheme`) — real
  usage, checked by `go test` on every run, and the first thing a reader
  sees on pkg.go.dev.
- **Item 4.** A new root `doc.go`: the three-stage `Load` pipeline, the
  two check layers, the two output contracts, and the package map, in
  one place — the package overview `godoc`/pkg.go.dev show first. The
  package-level doc comment moves out of `gsxmail.go` into this file.

### Added (polish items 9, 11) [bytes]

- **Item 9.** `gsxmail check` now sorts every finding by file, then line,
  then column, so findings in the same file print together, in source
  order. A new `--severity all|warn|error` flag narrows what prints
  (`--severity error` for a CI gate that only cares about hard failures);
  the exit code still reflects the full, unfiltered finding list.
- **Item 11.** The preheader pad tail is now raw UTF-8 (a literal NBSP
  and ZWNJ) instead of the `&nbsp;&zwnj;` entity spelling — 5 bytes per
  pair instead of 12, and 2 bytes instead of 6 for a lone trailing NBSP.
  Same two decoded characters every client sees; cheaper on the wire, and
  the difference compounds up to 74 pairs per preheader. Every affected
  golden (five gallery templates, `testdata/wp52/notice_preheader.html`)
  is regenerated.

### Fixed (minors: m2, m4, m7, m8, m10, m11, m12, m16, m18, m19)

- **m2.** `Theme.DarkMode`'s doc comment and README now state that
  parity mode (`Outlook: "off"`) disables the entire dark-mode style
  layer, regardless of `DarkMode` — WP1 predates every dark-mode
  strategy, and parity mode emits WP1's exact byte stream.
- **m4.** `email.Column`'s `imgSrc`/`imgAlt` now run EM111/EM112, the
  same absolute-https-src and non-empty-alt rules a raw `<img>` and
  `email.Hero` already run — only when `imgSrc` is actually set, since
  Column's image is optional.
- **m7.** `internal/structverify`'s own binary-size comment claimed
  `cmd/gsxmail` never imports gotreesitter; WP5.5's `gsxmail import` verb
  made that stale. Corrected, with the m6-measured tagged/untagged CLI
  figures in place of the old ones.
- **m8.** README's caniemail snapshot section now states the embedded
  snapshot's own capture date (2026-08-10) directly, plus a pointer to
  `Matrix.SnapshotDate()` for whichever date a specific build actually
  ships.
- **m10.** `importer/dom.go` now uses the standard library's
  `html.UnescapeString` instead of `golang.org/x/net/html`'s. The module
  dependency itself stays (a test file's own DOM-diffing use still needs
  it); `go mod tidy` confirms no other change.
- **m11.** `Load` rejects a negative `Options.MaxHTMLBytes` other than
  exactly `-1` (`EM201`) — every other negative value used to reach
  `Render`'s own budget check unvalidated, failing every call with a
  confusing "budget: -5 bytes" `EM120`.
- **m12.** `EM121`'s warning line now scales to 90% of `MaxHTMLBytes`
  when that budget is set below the fixed 90,000-byte line — otherwise a
  tighter budget made `EM121` permanently unreachable, since `EM120`
  always fired first.
- **m16.** `gsxmail import` now names every `<script>`/`<style>`/
  document-plumbing tag it drops entirely (nothing from them survives
  anywhere, unlike an unmapped node) in `IMPORT-REPORT.md`'s new
  "Dropped entirely" section. README's "never gets dropped" claim is
  corrected to state this one exception.
- **m18.** `writeHead`'s and the article wrapper's hardcoded `dir="ltr"`
  now carry a doc comment (and a new README note) stating plainly that
  gsxmail has no RTL layout support — `dir="ltr"` is honest, not a claim
  tied to `Shell.Lang`.
- **m19.** `Theme`'s own doc comment and a new README note state the
  trust boundary explicitly: every `Theme` field is written unescaped,
  the same trust level as your own Go source, never props- or
  request-driven.

### Known limitations (m3, launch-gate findings, deferred)

- An `email.*` attribute-level lint finding (EM012, EM013, EM032, EM110,
  EM190, EM191, and the rest) reports its enclosing element's own
  position (`file:line:col`), not the specific attribute's own column
  within that element's opening tag. Two attributes on the same element
  that both fail a check report the same position. Locating a diagnostic
  by scanning the named element for the named attribute is straightforward
  in practice; true per-attribute positions need `ir.Attr` to carry its
  own span, which gosx does not expose today. Deferred, not fixed in this
  pass.

### Fixed (M6, M10, M4: Outlook option, child-kind checks, preheader truncation)

- **M6.** `Load` now rejects `Options.Outlook` outside `""`,
  `"ghost-tables"`, or `"off"` (`EM195`) — a case typo ("OFF") or a
  stray value ("gost-tables", "true") previously fell through
  `WriteOptions.hardened`'s own `!= "off"` check and silently defaulted
  to hardened mode.
- **M10.** Member-name and child-kind checks that used to surface only as
  a plain Go error from `Lower` (no source position) now run in `lint`
  first, with a real `file:line:col`: `EM196` (an unrecognized email.*
  member, such as `<email.Bogus>` or a typo like `<email.Colum>`),
  `EM197` (`<email.Panel>` children must be `<email.PanelRow>`), `EM198`
  (`<email.PickList>` children must be `<email.Item>`), and `EM199`
  (`<email.StatTable>` children must be `<email.StatRow>` or `<Each>`,
  and an `<Each>` there must wrap exactly one `<email.StatRow>`).
  `lower.Lower`'s own errors stay as a backstop, unchanged. This work
  also caught and fixed a real B4 gap: `lower.Schema` was missing an
  entry for `"Panel"` itself, which made every `<email.Panel>` fail
  `EM196` as an unrecognized member the moment this check went live.
- **M4.** A dynamic (`{props.X}`) preheader over 150 runes now truncates
  to exactly 150 at render time and reports a warn-severity `EM200` in
  `Parts.Diagnostics`, instead of rendering past the "always pads to
  exactly 150" contract unchecked — `EM171` only ever caught a *static*
  preheader literal at `Load` time, since a dynamic one's length is not
  known until a real props value resolves it.

### Fixed (M1, M2, M3: link buttons, badge contrast, href diagnostics) [bytes]

- **M1.** `email.Button` variant="link"'s two Outlook-only balancing runs
  now each carry half of `(width - estimatedTextWidth)`, floored at 0
  (reusing the existing per-rune width estimate), instead of the full
  button width each. The old shape pushed the label to the right edge of
  its own click box in Outlook; the label now centers there the way
  `text-align:center` already centers it everywhere else.
  `renderhtml.linkButtonSpacerWidth`'s own test pins the formula;
  `testdata/newblocks*.html` are regenerated.
- **M2.** Badge's `positive`/`warning`/`critical` tones now carry two
  hex values each — one verified >=4.5:1 against a light card
  (`#FFFFFF`), one against a dark card (`#101611`, Terminal's own
  `ColorCard`) — selected by `renderhtml.cardIsDark(theme.ColorCard)`. A
  single flat hex cannot clear 4.5:1 against both: the WCAG 2 formula has
  no solution that does, confirmed by direct computation before picking
  replacements. `lint.CheckTheme`'s EM141 now also checks badge tones
  against `ColorCard`, under every theme, not just `locked`/`adaptive`.
  Affected goldens (`examples/gallery/alert`, `examples/gallery/receipt`,
  `testdata/newblocks*.html`) are regenerated.
- **M3.** A CTA/Button href that fails the scheme allowlist (EM110) now
  appends a warn-severity `Diagnostic` to `Parts.Diagnostics` instead of
  dropping the link silently — visible, never fatal: a mid-send loop
  must not die for one bad optional link. `renderhtml.Write`/
  `WriteWithOptions` now return `(string, []RenderFinding)`, a breaking
  signature change (pre-tag). README's "Two layers, one guarantee"
  section states the corrected behavior.

### Fixed (m9: caniemail's actual data license — MIT, not CC BY 4.0)

- Verified directly against caniemail's own maintained repository
  (https://github.com/HTeuMeuLeu/caniemail): its README states "MIT
  Licence," not the CC BY 4.0 this project had assumed. `Snapshot` gains
  `License` and `Attribution` fields; `gsxmail matrix refresh` writes
  both into every regenerated `snapshot.json`; the embedded
  `snapshot.json` is patched with both, unchanged otherwise.
- README gains a new "Prior art and attribution" section (also polish
  item 8): MJML, react-email, goodemailcode, Maizzle, and Litmus's own
  guides, alongside caniemail's license and attribution line. No NOTICE
  file: MIT requires only that the license and copyright notice travel
  with the software itself, and gsxmail redistributes a trimmed subset
  of caniemail's data, not its code — README's own attribution and
  `snapshot.json`'s new fields already carry the required notice.

### Added (M9: CI, version, gosx skew guard)

- `.github/workflows/ci.yml`: build, vet, gofmt, and `go test -race`,
  once against the `m31labs.dev/gosx` version go.mod pins and once
  against the latest tagged gosx release (the "pinned"/"latest" matrix),
  plus a size-gate job that fails if the recommended-tags CLI build
  exceeds 30 MB.
- `version.go`: `Version` const (gsxmail's own release version).
- `skew.go`: `Load` now runs a gosx version-skew check (design spec
  section 13.2, following `cmd/gosx`'s own `checkVersionSkew`): a
  warn-severity `EM194` finding when the `m31labs.dev/gosx` release
  actually linked into the build differs from the one gsxmail's own test
  suite and goldens are verified against. Unlike `cmd/gosx`, this never
  fails `Load` closed — gsxmail is a library many processes import, and
  the lint/lower pass already ran against whatever gosx build resolved.

### Changed (m6: invert the CLI's default build tags)

- `gsxmail import` never asks gotreesitter for any grammar but HTML, so
  README's "The CLI" section now leads with
  `go install -tags 'grammar_subset grammar_subset_html' ...` as the
  recommended command: roughly 24.4 MB, instead of the untagged
  default's roughly 43.0 MB (gotreesitter's own default embeds all
  ~540 grammars it ships). The untagged build still works; it is no
  longer the documented default. `binsize_test.go`'s
  `TestCLIBinarySizeUnderBudget` builds the tagged CLI and asserts it
  stays under 30 MB.
- Go's own toolchain has no mechanism for a library to force default
  build tags onto a downstream `go install` invocation that passes none
  — this change makes the small build the *documented and CI-gated*
  default, which is the lever gsxmail actually has; a user who runs bare
  `go install` with no flags at all still gets gotreesitter's own
  untagged default.

### Fixed (M7: the README quickstart snippet did not compile)

- The "60-second quick start" snippet used a bare `WelcomeProps{...}`
  with no import for the `emails` package step 2 says to copy in. It now
  imports `example.com/myapp/emails` and writes `emails.WelcomeProps{...}`,
  and also passes `Options.Dir` (the B3 fix's own recommended practice).
  `readme_quickstart_test.go`'s `TestReadmeQuickstartCompiles` extracts
  the snippet verbatim from between two new HTML-comment markers and
  compiles it, against a real copy of `examples/quickstart/emails`, in a
  temporary module on every test run — the snippet and the test can no
  longer drift apart silently.
- `examples/quickstart` itself gains the same `Options.Dir` value, a
  preheader on its one template (m13), and now prints every `Check`
  finding before rendering (m13) — the shipped example now models the
  practice the README and this section otherwise only describe.

### Changed (M8: narrow the public surface before the tag)

- `doc/`, `lower/`, and `typesafe/` move to `internal/doc/`,
  `internal/lower/`, and `internal/typesafe/`. `lint/` (the check-time
  rule catalog) moves to `internal/lint/`. None of these were part of the
  documented public API; `gsxmail.Diagnostic` (a type alias for
  `lint.Diagnostic`) stays at the root, unchanged for every caller.
  `renderhtml`, `rendertext`, and `importer` stay public: `importer` is a
  consumer surface (`gsxmail import`'s own package), and `renderhtml`/
  `rendertext` were not in scope for this move.
- Every import path across the module is updated accordingly; no public
  API changed.

### Fixed (launch-gate B2: complete the adaptive dark-mode class-hook sweep) [bytes]

- `DarkMode: "adaptive"` now swaps every one of `Theme.Dark`'s nine
  tokens at every site the writer emits that token's matching inline
  color, not just the Shell wordmark/tagline and Headline title WP5.2
  shipped. Five new class hooks join the existing `gsx-ink`/`gsx-muted`:
  `gsx-copy` (body copy), `gsx-panel` (panel backgrounds), `gsx-border`
  (structural borders), `gsx-accent`/`gsx-accent-bg`/`gsx-accent-border`
  (accent text, backgrounds, and borders), `gsx-faint` (footer fine
  print), and `gsx-ground` (a primary/link button's own inverse face
  text). `gsx-body` (the page background) now swaps to `Dark.ColorGround`
  instead of sharing `gsx-card`'s `Dark.ColorCard` rule (m1, folded in
  here).
- Before this fix, the digest gallery's body copy rendered at 1.73:1
  contrast against a forced-dark card (WCAG AA requires 4.5:1) with no
  adaptive hook at all — `EM141`'s own body-on-card contrast check was
  validating a token pair the writer never actually emitted. It is
  10.85:1 now; `examples/gallery/dark_contrast_test.go`'s
  `TestDigestDarkContrastMeetsWCAGAA` computes and pins the ratio
  programmatically against the rendered golden, not just the theme
  value in isolation.
- Regenerated goldens: `examples/gallery/digest/digest.html` (the one
  shipped adaptive-mode gallery example) and
  `testdata/wp52/notice_dark_adaptive.html`. No other golden changed —
  every other fixture uses a `"none"` or `"locked"` theme, neither of
  which this sweep touches.

### Fixed (launch-gate B3: props resolution false EM012s, importer output outside its module)

- A props type whose own package fails to resolve (most often an
  unresolvable import) now reports EM192 once, with the real cause,
  instead of silently falling through to a misleading EM012 "no such
  field" for every `props.field` read in the template.
- Added `Options.Dir`: the real on-disk directory `fsys` is rooted at.
  Setting it (as `gsxmail check` and `gsxmail render` now always do) makes
  props type resolution work regardless of the calling process's current
  working directory, instead of only when that directory happened to be
  inside the owning module. See the README's new "Props type resolution"
  section for the residual case this does not cover.
- `gsxmail import`'s generated `props.go` no longer imports
  `m31labs.dev/gsxmail`. The extracted `ImportedTheme()` moves to a new,
  optional `theme.go` companion file — the only generated file that still
  imports gsxmail, and safe to delete. `gsxmail import` now writes five
  files, not four. `importer/testdata/corpus/*/props.go` and the new
  `theme.go` goldens are regenerated to match.
- Before this fix, `gsxmail check` on the importer's own generated output
  failed outside its own module. `importer/outsidemodule_test.go`'s
  `TestCheckSucceedsOutsideModule` reproduces that case in a real, separate
  consumer module on disk and pins that it now passes.

### Added (launch-gate B4: unknown/required attribute checks)

- `lower.Schema` is the new single source of truth for every email.*
  component's attribute surface, hand-derived from package lower's own
  lowering switch. Package lint now checks every email.* node against it:
  EM190 flags an attribute name the component does not accept; EM191
  flags a required attribute that is absent (`Shell.wordmark`/`title`,
  `Signal.text`, `Headline.title`, `PanelRow.label`/`value`,
  `CTA.label`/`href`, `Button.label`/`href`, `Note.text`, `Badge.text`,
  `Item` non-blank content, `Footer.signoff`).
- Before this fix, `heading=` for `title=`, a missing `href`, and
  case-mismatched attribute names (`Label`/`HREF` for `label`/`href`) all
  passed `gsxmail check` silently and rendered an empty or dead element.
  `testdata/lint/newcomer` reproduces the examiner's own probe template
  and pins that every one of those mistakes now reports.

### Fixed (launch-gate B1: numeric attribute injection)

- `email.Hero`'s width/height, `email.Spacer`'s height, `email.Column`'s
  imgWidth/imgHeight, and `email.Button`'s link-variant width now reject a
  props-driven value at render time unless it is empty or a positive
  decimal integer (`doc.Resolve`'s new `resolvePositiveInt`). Every write
  site for these five fields in `renderhtml` also passes through
  `escapeAttr`, as defense in depth. A static (non-`{expression}`) value
  that fails the same rule now fails `gsxmail check` too, as diagnostic
  code EM181.
- Before this fix, a caller-controlled value at any of these five sites
  rendered unescaped and unvalidated into an HTML attribute or inline
  style, so a value such as `20" onmouseover="x" data-evil="` broke out of
  the attribute it was written into.

### Added (`gsxmail import`; WP5.5)

- `importer/`: a new top-level package that parses an existing email HTML
  file — MJML compiled output, react-email output, or hand-written table
  soup — with `github.com/odvcencio/gotreesitter`'s error-tolerant HTML
  grammar, and reverse-maps it onto gsxmail's shipped `email.*`
  components (pixel dossier section 7.2(1)). It is gsxmail's second
  deliberate gotreesitter import, after `internal/structverify`; the core
  render path (`renderhtml`, `doc`, `lower`, `gsxmail.go`) still never
  imports it — see `importer/doc.go` and
  `structural_isolation_test.go`'s new `TestImporterIsolatedFromRenderPath`.
- The mapper recognizes gsxmail's own hardened output shapes at high
  confidence (the self-round-trip proof: importing a template gsxmail
  itself rendered recovers the exact original block sequence) and a set
  of looser, generic fingerprints for foreign HTML: a ghost-table/card
  shell (with a fallback for MJML's own `<div style="max-width">` +
  `width:100%` table shape, and for a source that compiles each block to
  its own separate 600px table rather than one shared card), bulletproof
  button anchors (primary/secondary/link variants), a data table with
  `<th>` cells, label/value row pairs (nested in their own sub-table, or
  stacked directly as the card's own top-level rows), fluid-hybrid
  columns, a spacer/divider td, a bordered badge span, a hidden preheader
  div, a sized image, and a numbered list. Every node it cannot place
  lands inside an `email.Custom` fallback block, sanitized (tag
  remapping, `class`/event-attribute stripping, href-scheme and image-src
  validation, and CSS-matrix-safe style filtering) so the generated
  template still loads cleanly.
- Props synthesis: variable text (Panel row values, a Headline's lede, a
  Shell's wordmark/tagline/title/preheader, a Button's confirmed source
  text) becomes a named, deduplicated props field; repeated StatTable
  rows synthesize into an `<Each>` over a generated row-slice type,
  matching the shipped gallery's own `Cells []string` convention. A
  best-effort `Theme` literal is extracted from the card's own literal
  colors (`ImportedTheme()` in the generated `props.go`).
- `Import` returns a `Report`: every mapped node's path, component, and
  confidence; every unmapped node's path and reason; every synthesized
  props field's source snippet; theme-extraction notes; and a next-steps
  list. `gsxmail import` prints its own summary to stdout and writes the
  full report as `IMPORT-REPORT.md`.
- CLI: `gsxmail import <in.html> --out <dir> [--name Welcome] [--package
  emails]` writes `template.gsx`, `props.go`, `props.sample.json`, and
  `IMPORT-REPORT.md`. Building the CLI with `-tags 'grammar_subset
  grammar_subset_html'` keeps the gotreesitter delta to roughly 819 KiB
  over the pre-WP5.5 binary (25,513,420 vs 24,674,716 bytes), well under
  the pixel dossier's ~5 MB gate; the untagged default build embeds every
  grammar and costs roughly 19.4 MiB more.
- `structural_isolation_test.go`'s own `corePackages` no longer lists
  `cmd/gsxmail`: the pixel dossier's isolation rule extends, rather than
  tightens, for WP5.5 — "the importer package and CLI may [import
  gotreesitter]." The new `TestImporterIsolatedFromRenderPath` proves the
  CLI reaches gotreesitter through exactly the one expected path.
- `importer/testdata/corpus/`: three checked-in foreign-HTML fixtures —
  `mjml.html` (MJML's real compiled section/button/column shapes, citing
  R6-R9), `react-email.html` (react-email's Preview suppression styles
  and per-Section layout, citing R3-R5), and `legacy.html` (hand-written
  table soup with genuinely unclosed tags) — each with a pinned
  `.gsx`/`props.go`/`props.sample.json`/`IMPORT-REPORT.md` golden.

### Documentation (WP5.3)

- README: a new "New components (WP5.3)" section documents `Button`
  (variants and the CTA-alias guarantee), `Columns`/`Column`, `Hero`,
  `Spacer`, and `Badge`, alongside the existing WP5.1 component list. A
  new "Named themes" section documents `TerminalTheme()` and
  `LedgerTheme()`'s full palettes. A new "The template gallery" section
  is the gallery's showcase: the five-template table and one rendered
  `Badge`/`Button` snippet from `receipt/`, both linking to
  `examples/gallery/README.md`. The "Status" line and "The library API"
  section both gain the new theme constructors.

### Added (template gallery; WP5.3)

- `examples/gallery/`: five complete templates (pixel dossier section
  8.1), each a self-contained `fs.FS` root with typed props, a `.gsx`
  source, a fixture props JSON file, a byte-exact golden HTML/text pair,
  and its own README: `welcome/` (Shell, Headline, PickList, Button,
  Footer), `magiclink/` (Shell, Headline, Panel, Note, Button), `receipt/`
  (Shell, Badge, Headline, StatTable, Panel, Button, Footer — the
  dossier's own complete worked example, section 8.3), `digest/` (Shell,
  Hero, Columns, StatTable, Divider, PickList, rendered under
  `LedgerTheme()`), and `alert/` (Shell, Signal, Badge, Note, Button,
  rendered under `TerminalTheme()`).
- `examples/gallery/gallery_test.go` loads every template independently,
  renders each against its own fixture, pins the golden pair, and runs
  the structural verification pass over the whole gallery in both output
  contracts (hardened and parity) — the corpus doubling as the
  structural-verification test suite the task set out for it. Every
  gallery HTML stays under the EM121 90,000-byte warning line (the
  largest, `digest.html`, is 10,402 bytes).
- Documented deviation: `receipt/`'s dossier-cited `.gsx` source writes
  inline slice literals for `StatTable`'s `header`/`cells`
  (`cells={[item.Name, item.Qty, item.Amount]}`), which the shipped
  `StatTable`/`StatRow` API does not accept — it only takes a bare props-
  or binding-rooted field path, and `StatTable` is out of WP5.3 scope
  (its own byte-pinned goldens must not move). `receipt/`'s `ReceiptItem`
  carries a `Cells []string` field instead of separate fields, and
  `ReceiptProps` carries an explicit `Header []string` field; the
  rendered content and shape still match the dossier's worked example
  (see `receipt/README.md`).

### Added (named themes; WP5.3)

- `gsxmail.TerminalTheme()`/`renderhtml.TerminalTheme()`: a dark,
  mono-forward named theme, green-on-near-black, `DarkMode: "locked"`
  (pixel dossier section 8.2). It is deliberately not the private
  gridiron aqua/navy palette, which stays unshipped (dossier section
  8.2's own note; spec section 16.5's default). Its token values are the
  same ones WP5.2's own private `darkLockedTheme` test fixture already
  proved against EM140-144; they now ship as a named theme.
- `gsxmail.LedgerTheme()`/`renderhtml.LedgerTheme()`: a warm, print-like
  light named theme, `DarkMode: "adaptive"` with a genuine `Dark`
  palette. Terminal is dark-native by construction and needs no separate
  swapped-in presentation; Ledger is light-native, so it carries a real
  companion dark palette instead — together the two themes demonstrate
  both of gsxmail's non-trivial dark-mode strategies with real, shipped
  themes.
- Both themes pass EM140-144 (`TestCheckThemeNamedGalleryThemesPass`,
  called through their real constructors, not a copied token table).

### Added (new components; WP5.3)

- `email.Button` (variants `"primary"`, `"secondary"`, `"link"`; pixel
  dossier section 4.4). `"primary"` (the default) is byte-identical to
  `email.CTA` in both output contracts: `email.CTA` is documented as
  `Button`'s `variant="primary"` alias, and `renderhtml.writeButton`
  routes that variant straight through the existing, untouched `writeCTA`
  function, so no CTA byte ever moves. `"secondary"` adds a
  transparent-face, accent-bordered variant; `"link"` adds
  goodemailcode's full-click glyph-spacing technique (an MSO-only hidden
  run stretched with a negative `mso-font-width`, so the whole box —
  not just the text — is clickable in Outlook), with an optional `width`
  attribute and a documented, approximate default width when unset. A VML
  roundrect button variant is explicitly out of scope (pixel dossier
  section 4.4's own rejection: MJML itself does not ship one, and dark-mode
  transforms recolor VML fills unpredictably).
- `email.Columns`/`email.Column` (pixel dossier section 4.9): a
  two-to-four-wide fluid-hybrid row. Each column is an `inline-block`
  `max-width` div that stacks under a 480px viewport with no `<style>`
  dependency, wrapped in an `"[if mso | IE]"` ghost table for Outlook,
  which never applies `inline-block` at all. `email.Column` is a leaf
  component (an optional image, title, and text) rather than a nested
  block container — a column is too narrow for a full-card block's own
  padding and font sizes.
- `email.Hero` (pixel dossier section 4.10): a full-width retina `<img>`.
  `src` is served at 2x pixels; `width`/`height` are the display-size HTML
  attributes the retina rule requires, plus the `max-width`/`height:auto`
  Outlook workaround. `srcset` is rejected (24.39% caniemail support).
  `alt` is mandatory; lint's `checkHero` reuses EM111/EM112 verbatim
  (extended from the existing raw-`<img>` rules to this new component)
  rather than inventing parallel codes.
- `email.Spacer` (pixel dossier section 4.8): a fixed-height gap row
  (`font-size:0; line-height:0; mso-line-height-rule:exactly`). Its text
  twin is one blank line: like `email.Divider`, it is skipped in
  `rendertext.Write`'s own loop, so the ambient single blank-line
  separator between its neighbors is its entire derivation — text has no
  concept of a variable-height gap, so every Spacer height folds to the
  same one blank line.
- `email.Badge` (pixel dossier section 4.11): a small bordered status
  label. `tone` selects a fixed, theme-independent color (`"positive"`
  green, `"warning"` amber, `"critical"` red) or the active theme's own
  muted token (`"neutral"`, the default). Its text twin is its label in
  brackets (`[PAID]`), regardless of tone.
- New lint rules (design spec section 15, WP5.3; pixel dossier section
  10, continued): EM175 (`Button`'s `variant` attribute is not a static
  `"primary"`/`"secondary"`/`"link"`), EM176 (`Columns`' children are not
  all `Column`, or there are not 2-4 of them), EM177 (a `Column` outside
  `Columns`), EM178 (`Hero` missing `width` or `height`), EM179 (`Spacer`
  missing `height`), EM180 (`Badge`'s `tone` attribute is not one of the
  four allowed values).
- New goldens under `testdata/`, rendered from a new
  `testdata/newblocks/NewBlocks` fixture exercising every new component
  and Button variant: `newblocks.html`/`newblocks.txt` (hardened) and
  `newblocks_parity.html` (parity/`Outlook:"off"`). No existing golden's
  bytes changed; `TestInvitePartyWithAdmin` and `TestRecapGolden` are
  unmoved.

### Added (dark mode, preheader, Shell options; WP5.2)

- `Theme.DarkMode` (`"none"` default, `"locked"`, `"adaptive"`) and
  `Theme.Dark` (a new `DarkPalette` type) select a dark-mode strategy
  (design spec section 15, WP5.2; pixel dossier section 5). `"locked"`
  adds a `:root{color-scheme:...}` rule to the `<style>` block.
  `"adaptive"` adds an `@media (prefers-color-scheme:dark)` layer that
  swaps `Theme.Dark`'s tokens into the Shell wordmark/tagline and
  Headline title (marked with new `gsx-ink`/`gsx-muted` classes), plus
  best-effort `[data-ogsc]`/`[data-ogsb]` Outlook-app hooks. No strategy
  claims control of Gmail's own forced dark transform — see the README's
  "Dark mode" section for the honest per-client reach each one has.
  `"none"` (the default) changes nothing: every existing hardened golden
  stays byte-identical.
- `<email.Shell preheader={...}>`: the hidden inbox-preview div, written
  as the first child of `<body>` in both output contracts, with
  react-email's shipped suppression styles and an `&nbsp;`/`&zwnj;` pad
  tail that brings the decoded text to exactly 150 characters (pixel
  dossier section 6.1). It is authored on the Shell, like MJML's
  `mj-preview`, and can read `props`. Writes nothing when unset, so an
  existing template's bytes do not change until its author sets one.
- `<email.Shell outlook="off"|"ghost-tables">`: a per-Shell override of
  `Options.Outlook`'s Set-wide default (design spec section 15, WP5.2's
  Shell options surface). An unset Shell attribute keeps using
  `Options.Outlook`, so every WP5.1 consumer's own Set-level setting
  keeps working with no change. `outlook` must be a static string
  literal, never a `{props.X}` expression.
- New lint rules (design spec section 15, WP5.2; pixel dossier section
  10): EM140 (`DarkMode: "adaptive"` with a nil `Theme.Dark`), EM141
  (a dark palette's ink-on-card or body-on-card pair under 4.5:1 WCAG AA
  contrast), EM142 (a pure black or pure white color in a dark palette),
  EM143 (a `Custom` block color with no dark-palette counterpart, under
  `DarkMode: "adaptive"` only), EM144 (`ColorScheme` conflicts with
  `DarkMode`), EM170 (a Shell with no preheader attribute), EM171 (a
  static preheader text over 150 characters), EM172 (an `outlook`
  attribute that is not a static `""`, `"ghost-tables"`, or `"off"`).
  EM140-144 run once per `Load` call, against the Set's own `Theme`;
  EM170-172 run per Shell, like every other check-time rule.
- `internal/structverify` gains EM173 (a preheader div missing part of
  its suppression-style stack) and EM174 (an adaptive dark-mode style
  layer missing an Outlook-app hook or carrying unbalanced braces), both
  folded into `Verify` so every existing caller picks them up for free.
  Fixing this also fixed a pre-existing bug in `isElementNamed`: the
  gotreesitter HTML grammar tags `<style>`/`<script>` as distinct
  `style_element`/`script_element` node types, not the generic `element`
  type every other tag uses, so a name lookup for `"style"` never
  matched before this change.
- New goldens under `testdata/wp52/`, rendered from a new
  `testdata/wp52/emails/Notice` fixture: `notice_preheader.html`
  (preheader-on), `notice_dark_locked.html` and
  `notice_dark_adaptive.html` (hardened+dark, both strategies). No
  existing golden's bytes changed.

### Added (structural verification, WP5.1)

- `internal/structverify`: a test-layer-only package that re-parses
  gsxmail's own rendered HTML with gotreesitter's HTML grammar and proves
  the WP5.1 output contract holds mechanically — zero parse-error nodes,
  balanced `<!--[if mso]>` conditional comments, layout-table nesting
  under a 12-level cap, and (for hardened-mode output) that a
  `role="presentation"` table always carries `border="0" cellpadding="0"
  cellspacing="0"` and never has a `<th>` descendant. This is the pixel
  dossier's "structural verification pass" (section 7.2). Its findings use
  the dossier's reserved EM131-EM134 codes.
- New root-level tests: `TestStructuralPassOnGoldens`,
  `TestStructuralPassOnBlockCorpus`, and
  `TestStructuralPassOnInviteAndRecapRenders` run the pass over every
  golden and the full `testdata/allblocks` block corpus, in both output
  contracts. `TestGotreesitterIsolatedFromCorePath` is a module-graph
  proof: no gsxmail render-path or CLI package directly imports
  gotreesitter or `internal/structverify` outside of `_test.go` files.
  `github.com/odvcencio/gotreesitter` moves from an indirect to a direct
  `go.mod` dependency as a result.

### Changed

- `[bytes]` `testdata/recap.html` is regenerated for the WP5.1 hardened
  output contract: `DraftRecap` renders through `Shell`, `Headline`,
  `StatTable`, `Note`, and `CTA`, all now hardened by default. The golden
  grows from 17,176 to 18,730 bytes (+1,554). `testdata/recap.txt` is
  byte-identical: the text writer is untouched by WP5.1.

### Added

- WP5.1 bulletproof output contracts (design spec section 15; pixel
  dossier section 4): `renderhtml.Write` now emits hardened, bulletproof
  markup by default for `Shell`, `Headline`, `Panel`, `CTA`, `StatTable`,
  `Note`, and `Divider`. The hardened contract adds an Outlook ghost table
  around the 600px card, `xmlns:v`/`xmlns:o` namespaces, the
  `o:PixelsPerInch` DPI fix, MJML's reset `<style>` block, a
  `role="article"` accessible wrapper, doubled width attributes/CSS on
  sized elements, td-pair `Panel` rows (Outlook has no
  `display:inline-block`), an `<h1>` `Headline` title, `mso-padding-alt`
  on the `CTA` button, a real `StatTable` data-table contract (no
  `role="presentation"`, `<th scope="col">` headers), a border-left `Note`
  aside, and a spacer-technique `Divider`
  (`font-size:0;line-height:0;mso-line-height-rule:exactly`). `Signal`,
  `PickList`, and `Footer` are unchanged: their WP1 markup already met the
  contract.
- New `renderhtml.WriteOptions` and `renderhtml.WriteWithOptions`
  (additive; `Write` keeps its signature) and a new `Options.Outlook`
  field on `gsxmail.Options`: `""`/`"ghost-tables"` (the default) selects
  the hardened contract above; `"off"` selects parity mode, the exact WP1
  byte stream, unchanged. A consumer with its own byte- or DOM-equivalence
  test against WP1 output — gsxmail's own gridiron invite fixture is the
  example — sets `Outlook: "off"` to keep pinning the old bytes.
- `<Each of={...} as="name">` and `<If cond={...}>` now render, not just
  lint-pass: `lower.Lower` expands `<Each>` into zero or more sibling
  blocks per element of the bound slice (empty slice renders nothing), and
  `<If>` includes its body only when `cond` resolves true. Both builtins
  work at the top level and inside a `StatTable` body.
- The `<Each>` loop variable participates in expression resolution the
  same way `props` does: `row.Field` reads a field off the bound element,
  type-checked at `Load` time (`typesafe.Binding`, one level of nesting
  into a `[]struct` props field) and re-proved at `Render` time by
  reflection.
- `email.StatTable` and `email.StatRow`: a bordered data table with an
  optional title and header row, built from a mix of literal `StatRow`
  children and `<Each>`-driven ones. A `StatRow`'s `mark` (bool, defaults
  false) selects the accent-colored, `* `-prefixed row — one row, or none,
  1-based — the same semantics `internal/emailkit`'s `MarkRow` established.
- `email.Note` (one body paragraph) and `email.Divider` (a border-top
  rule with no text of its own).
- The `Custom` pass-through: a raw `<table>/<td>/<div>/<span>/<a>/<img>/...`
  subtree renders through unmodified HTML-side, with its text form derived
  structurally (block elements start a paragraph; `<a>` renders "label
  (url)"; `<img>` renders `[alt]`; a `<table>` renders as guttered
  columns; any element's own `text` attribute overrides its derived
  content). Every literal style property inside it still runs through the
  EM101-EM104 lint, exactly like any other raw element.
- `Render` now invokes every registered `Options.Helpers` call
  (`ExprCall`) by reflection, converting each argument to the helper's
  declared parameter type and formatting its return value back in. `Load`
  already proved registration and arity (EM014/EM015); `Render` is what
  actually calls the function now.
- The Gmail-clip size budget (EM120/EM121) is enforced on every `Render`
  call's HTML part: over `Options.MaxHTMLBytes` (default 100,000 bytes)
  returns a zero `Parts` and a `*SizeBudgetError`; over the fixed
  90,000-byte warning line, but still within budget, returns the rendered
  `Parts` with one EM121 entry in the new `Parts.Diagnostics` field.
  `Options.MaxHTMLBytes: -1` disables both checks.
- `gsxmail render` now prints any `Parts.Diagnostics` finding (today, only
  EM121) to stderr after a successful render.

### Changed

- `Parts` gained a `Diagnostics []Diagnostic` field, populated only by the
  EM121 size warning today. Existing callers that read only `HTML`/`Text`
  are unaffected.
- `typesafe.CheckExpr`, `CheckInterpolation`, and the lint walker's loop-
  binding scope now carry a `typesafe.Binding` (a `Kind` plus, for a
  slice-of-struct element, that struct's own resolved fields) instead of a
  bare `Kind`, so `<Each>` bodies can type-check `row.Field` reads.
- `email.StatTable`'s `header` and `email.StatRow`'s `cells` attributes
  are now lint-checked as slice-valued paths (`typesafe.CheckSlicePath`,
  reusing EM032's rule and message shape) instead of the scalar-only
  interpolation check every other `email.*` attribute still uses.

### Known limitations

- `gsxmail dev` is not yet implemented.
- `gsxmail check` always runs with an empty `Options.Helpers`. It cannot
  validate EM014/EM015 helper bindings meaningfully, and it cannot run
  the EM120/EM121 size budget check (that check is `Render`-time only, and
  the standalone CLI has no fixture props to render with yet). Use
  `Set.Check()` and `Set.Render()` in your own test instead.
- A `Custom` subtree may only contain raw elements and text/expression
  content — no nested `email.*` component or another `<Each>`/`<If>`. Its
  derived text quality is not pixel-perfect for a deeply nested,
  non-tabular layout (design spec section 16, open question 4);
  `text="..."` on any element overrides its own derived content when the
  default reads badly.
- The WP5.2 `"adaptive"` dark-mode `gsx-ink`/`gsx-muted` class hooks apply
  only at the Shell wordmark/tagline and Headline title — the highest-
  visibility ink/muted text, and the exact elements the pixel dossier's
  own section 5.2 worked example shows. A template-wide sweep over every
  ink- or muted-colored element (Panel labels, StatTable headers,
  PickList titles, Footer copy) is a natural follow-on, not required for
  the strategy to work end to end; those elements keep their light-theme
  inline color under a forced dark transform until a later work package
  widens the sweep.
- `email.Button` (react-email's fuller button API and its `mso-padding-
  alt`/`mso-font-width` variants), `email.Columns`, and the named
  `Terminal`/`Ledger` gallery themes the pixel dossier's section 8
  describes stay out of scope for WP5.2; they are WP5.3 work. WP5.2's own
  `lint/theme_test.go` and `wp52_test.go` build a private, unshipped
  dark-theme fixture with the dossier's own numbers to prove EM140-144
  against a genuine dark palette, without shipping `Terminal` early.
