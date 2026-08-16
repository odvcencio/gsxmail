# Changelog

All notable changes to gsxmail are documented in this file.

## Unreleased

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
