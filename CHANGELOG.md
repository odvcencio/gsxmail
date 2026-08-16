# Changelog

All notable changes to gsxmail are documented in this file.

## Unreleased

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
