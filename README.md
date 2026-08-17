# gsxmail

Write email templates as gosx components. Get pixel-targeted HTML plus a
matched plain-text part from one source. Every template is validated
against a real client-support matrix before it can render.

gsxmail is a Go library (`m31labs.dev/gsxmail`) and a CLI (`gsxmail`). It
compiles `.gsx` templates through the public [gosx](https://github.com/odvcencio/gosx)
compiler, lowers them through its own email pipeline, and writes both parts
from one tree — so the text part can never drift from the HTML part.

## Guarantees

- **Fail-closed.** `Load` rejects a template that mail clients would break.
- **Deterministic.** The same props always render the same bytes.
- **Paired by construction.** The text part cannot drift from the HTML
  part; both come from the same tree.

## What gsxmail refuses to do

- No sending. gsxmail renders; your own mailer (SMTP, Resend, a queue)
  delivers.
- No JavaScript. Mail clients do not run it, so gsxmail never emits it.
- No forms. Mail clients cannot submit them reliably; use a CTA link.
- No list management. Recipient lists, preferences, and unsubscribe
  tokens stay with your application.
- No Word-engine emulation in preview. The dev preview approximates
  webmail; it does not emulate Outlook's Word rendering engine.

## Status: WP5.5, `gsxmail import`, new components, the gallery, and named themes

The stdlib now covers `Shell`, `Signal`, `Headline`, `Panel`/`PanelRow`,
`CTA`, `Button`, `Columns`/`Column`, `Hero`, `Spacer`, `Badge`,
`PickList`/`Item`, `Footer`, `Note`, `Divider`, and `StatTable`/`StatRow`.
Templates can loop over a slice with `<Each>` and branch on a bool with
`<If>`; a raw-element `Custom` subtree stays available as an escape hatch
for markup the stdlib does not express. It ships the `gsxmail render`,
`gsxmail check`, and `gsxmail import` CLI verbs, `gsxmail matrix
refresh`, and the full `Load`/`Render`/`Check` library API. `Render`'s
HTML part is hardened, bulletproof markup by default, mechanically
proven by a structural verification pass — see "Output contracts" below.
A `Theme` can declare a dark-mode strategy, and a Shell can set a
preheader and its own output-contract override — see "Dark mode",
"Preheader", and "Shell options" below. Two named themes,
`TerminalTheme()` and `LedgerTheme()`, ship alongside the neutral
`DefaultTheme()` — see "Named themes" below. The
[`examples/gallery`](examples/gallery) directory holds five complete,
golden-tested templates — see "The template gallery" below.
`gsxmail import` reverse-maps an email you already send onto these same
components — see "Import from existing HTML" below.

`Load` runs the full email lint catalog (EM001 through EM112) before it
lowers anything. A missing props field, an expression outside the email
dialect, a disallowed HTML element, or an unsupported style property all
fail `Load` closed. Each failure carries an exact diagnostic, not a
runtime surprise. `Render` adds its own render-time check: the rendered
HTML part must fit the Gmail-clip size budget (see "Size budget" below).

`gsxmail dev` (the live preview server) lands in a later release. Until
then, validate a template with `gsxmail check` and by rendering it.

## Dynamic data: `Each`, `If`, and `StatTable`

`<Each of={props.Field} as="name">` iterates a slice props path (or,
inside another `<Each>`, a slice field of the current loop binding) and
binds `name` to the current element for its body. An empty slice renders
nothing. `<If cond={props.Field}>` renders its children only when the
bool expression is true; a bare-text child is a check-time error (EM031),
so the text twin always has an element to place.

```gsx
func DraftRecap(props RecapProps) Node {
    return <email.Shell wordmark={props.League} shortCode={props.Code}
        tagline={props.Tagline} title={props.League} lang="en">
        <email.StatTable title="YOUR HAUL //" header={props.HaulHeader}>
            <Each of={props.Haul} as="row">
                <email.StatRow cells={row.Cells} mark={row.IsKeystone} />
            </Each>
        </email.StatTable>
        <If cond={props.HasAutoPicks}>
            <email.Note text={props.AutoPickNote} />
        </If>
        <email.CTA label="SEE THE FULL BOARD →" href={props.BoardURL} />
    </email.Shell>
}
```

`row.Cells` and `row.IsKeystone` read fields off the loop-bound element
itself, not off `props` — the same expression grammar as a `props.Field`
read, resolved against whichever struct `<Each>` bound. `StatTable`'s
`header` attribute, and `StatRow`'s `cells` attribute, are both a bare
slice-valued path for the same reason a computed slice is not: `<Each
of={...}>`, `header={...}`, and `cells={...}` all require one, never a
concatenation or a helper call.

A `StatRow`'s `mark` attribute (a bool expression, defaulting to false
when omitted) selects the one row that renders with the accent color in
HTML and a leading `* ` in text — the same 1-based "one marked row, or
none" semantics `internal/emailkit`'s `MarkRow` established, derived here
from whichever row's `mark` resolves true first.

A registered helper (`Options.Helpers`) can appear in any expression hole,
including a `StatRow`'s `cells`/`mark` or an `<If>`'s `cond`: `Load`
checks its registration and arity (EM014/EM015); `Render` invokes it by
reflection against the same map.

## Output contracts

`Render`'s HTML part is hardened, bulletproof markup by default: an
Outlook ghost table around the 600px card, `xmlns:v`/`xmlns:o` and the
`o:PixelsPerInch` DPI fix, MJML's reset `<style>` block, a
`role="presentation"` invariant on every layout table, and doubled width
attributes/CSS on sized elements. Per component:

- **Shell**: the ghost table, DPI namespaces, and reset styles above,
  plus a `role="article"` accessible wrapper around the card.
- **Headline**: the title renders as a semantic `<h1>` (margins zeroed),
  not a plain `<div>`.
- **Panel**: each row is a two-cell table row, not two `<span>`s sharing
  one cell — Outlook Windows has no `display:inline-block`.
- **CTA**: the button face carries `mso-padding-alt`, so Outlook draws
  the visual box the padding gives every other client.
- **StatTable**: a real data table (no `role="presentation"`, `<th
  scope="col">` headers) — it holds facts, not layout.
- **Note**: a border-left accent bar and a tinted background, marking the
  aside structurally, never by color alone.
- **Divider**: the spacer technique
  (`font-size:0;line-height:0;mso-line-height-rule:exactly`), which pins
  an exact-height rule across clients that a bare `border-top` div does
  not.

`Signal`, `PickList`, and `Footer` needed no change: their WP1 markup
already met the contract.

### New components (WP5.3)

- **`Button`** (`variant="primary"|"secondary"|"link"`, default
  `"primary"`). `email.CTA` is `Button`'s `variant="primary"` alias: both
  render byte-identically, in both output contracts, because
  `renderhtml.writeButton` routes the primary variant through the same
  `writeCTA` function `email.CTA` always used. `"secondary"` swaps the
  solid accent face for a transparent one with a 1px accent border.
  `"link"` uses goodemailcode's full-click glyph-spacing technique (an
  MSO-only hidden run stretched with a negative `mso-font-width`, so
  Outlook's whole box — not just the text — is clickable), with an
  optional `width` attribute; unset, `Write` computes an approximate
  width from the label's own length. A VML roundrect button is not
  planned: MJML itself does not ship one, and dark-mode transforms
  recolor VML fills unpredictably (pixel dossier section 4.4).
- **`Columns`/`Column`** (fluid-hybrid, two to four columns). Each
  `Column` is an `inline-block`, `max-width` div that stacks under a
  480px viewport with no `<style>` dependency, wrapped in an
  `"[if mso | IE]"` ghost table for Outlook, which never applies
  `inline-block` at all. `Column` is a leaf component — an optional
  image (`imgSrc`/`imgAlt`/`imgWidth`/`imgHeight`), an optional `title`,
  an optional `text` — not a nested block container.
- **`Hero`**. A full-width retina `<img>`: `src` at 2x pixels,
  `width`/`height` at display size (both required), `alt` mandatory.
  `srcset` is not supported (24.39% caniemail support).
- **`Spacer`** (`height` in pixels, required). An exact-height gap row
  (`font-size:0;line-height:0;mso-line-height-rule:exactly`).
- **`Badge`** (`text`, optional `tone="neutral"|"positive"|"warning"|"critical"`,
  default `"neutral"`). A small bordered status label: `"positive"`,
  `"warning"`, and `"critical"` are fixed, theme-independent colors (green,
  amber, red); `"neutral"` tracks the active theme's own muted token.

Every new component renders one contract regardless of
`Options.Outlook`: none of them carry a WP1 byte stream to protect, so
there is nothing for parity mode to preserve. `Button`'s `"primary"`
variant is the one exception, by construction — see above.

Set `Options.Outlook: "off"` to render the exact WP1 byte stream instead
— parity mode, for a consumer with its own byte- or DOM-equivalence test
pinned to the old bytes. gsxmail's own gridiron invite fixture, and its
DOM-parity test against the hand-written production template, is the
worked example: it loads with `Outlook: "off"` for exactly this reason.

### The structural verification pass

gsxmail re-parses its own rendered HTML with a pure-Go tree-sitter HTML
grammar ([gotreesitter](https://github.com/odvcencio/gotreesitter)) and
mechanically proves the contract holds: zero parse-error nodes (malformed
HTML still parses to a walkable tree under tree-sitter's error recovery,
so a clean parse is a real guarantee), balanced `<!--[if mso]>`
conditional comments, layout-table nesting under a 12-level cap, and (in
hardened mode) that a `role="presentation"` table never has a `<th>`
descendant. The pass runs over every golden and the full stdlib block
corpus in the test suite.

This is a test-layer dependency only. `internal/structverify` imports
gotreesitter; no render-path package (`renderhtml`, `doc`, `lower`,
`gsxmail` itself) and no CLI package does — a module-graph test at the
repo root, `TestGotreesitterIsolatedFromCorePath`, proves it on every
run. gotreesitter's default build embeds its full ~206-grammar registry,
so linking it into a binary is not free: measure before you import it
anywhere outside a test. WP5.2 extends the pass with two more checks: a
configured preheader div must carry its full suppression-style stack
(EM173), and an "adaptive" dark-mode style layer must carry both its
Outlook-app hooks with balanced braces (EM174).

## Dark mode

Set `Theme.DarkMode` to one of three strategies (pixel dossier section 5).
Each strategy states its own honest reach: **no strategy controls Gmail's
forced dark transform**. Every strategy is a mitigation, never a claim of
control, and the README says so on purpose — state the mitigation, not
pixel parity, in your own product copy too.

- **`"none"`** (the default). `Render` adds no dark-mode markup at all. A
  `Theme` that sets `ColorScheme` still emits its own meta pair, exactly
  as WP5.1 shipped.
- **`"locked"`**. The `Theme` itself is dark-native — gridiron's own
  palette is this case. `Render` adds a `:root{color-scheme:dark}` rule to
  the `<style>` block and a `dark`/`dark` meta pair. Apple Mail 16+ honors
  the root rule; Gmail's forced transform may still lighten the theme, so
  keep every color a midtone, never pure black or pure white.
- **`"adaptive"`**. The `Theme` carries both a light presentation (its own
  fields) and a dark one (`Theme.Dark`, a `DarkPalette`). `Render` emits a
  `light dark` meta pair and an `@media (prefers-color-scheme:dark)` layer
  that swaps every one of `Theme.Dark`'s nine tokens into its own class
  hook, at every site the writer emits that token's matching inline
  color: the page background, the card, borders, panel backgrounds, ink
  and body-copy text, muted labels, accent text/backgrounds/borders, and
  footer fine print — plus best-effort `[data-ogsc]`/`[data-ogsb]` hooks
  for Outlook's own app-level inversion. Apple Mail, iOS Mail, and
  Outlook.com switch cleanly; Gmail ignores the media query and applies
  its own forced transform regardless.

`Load` checks a Set's `Theme` before it renders anything:

- **EM140** (error): `"adaptive"` requires `Theme.Dark`.
- **EM141** (error): the active dark palette's ink-on-card and
  body-on-card pairs must clear 4.5:1 contrast (WCAG AA body text).
- **EM142** (warn): no color in the active dark palette may be pure black
  or pure white — a forced transform maps extremes the hardest.
- **EM143** (warn): a raw `Custom` element's literal color that matches
  neither palette's tokens, under `DarkMode: "adaptive"` only — it has
  nowhere to go when the style layer swaps in.
- **EM144** (error): an explicit `ColorScheme` must agree with what
  `DarkMode` implies.

## Preheader

Set `preheader={...}` on `<email.Shell>` to control the text a mail client
shows next to the subject line as the inbox preview. Like MJML's
`mj-preview`, a preheader belongs to the template, not the caller: it is
authored on the Shell and can read `props` the same way any other Shell
field does.

`Render` writes it as a hidden `<div>`, first inside `<body>`, with
react-email's own shipped suppression styles (`display:none;
overflow:hidden; line-height:1px; opacity:0; max-height:0; max-width:0`)
and an alternating `&nbsp;`/`&zwnj;` pad tail that brings the decoded text
to exactly 150 characters — long enough that no supported client falls
back to pulling in body copy. A Shell with no `preheader` attribute at all
triggers **EM170** (warn) at `Load`; a static preheader text over 150
characters triggers **EM171** (error).

## Shell options

`Options.Outlook` (WP5.1) still selects a Set's default output contract.
WP5.2 adds a per-Shell override: set `outlook="off"` or
`outlook="ghost-tables"` directly on one template's `<email.Shell>` to
pick that template's own contract, regardless of the Set's own default.
A Shell that leaves `outlook` unset keeps using `Options.Outlook` —
every WP5.1 consumer keeps working with no change. `outlook` must be a
static string literal, never a `{props.X}` expression: the output
contract is a structural, compile-time choice, not a per-render one.
**EM172** (error) rejects anything else.

```gsx
<email.Shell
    wordmark={props.Product}
    title={props.Product + " receipt"}
    lang="en"
    preheader={"Receipt for order " + props.OrderID}
    outlook="off">
    ...
</email.Shell>
```

## Named themes

`DefaultTheme()` ("Paper") stays the neutral light default. Two more
named themes ship alongside it (pixel dossier section 8.2, WP5.3):

- **`TerminalTheme()`** — dark, mono-forward, green-on-near-black,
  `DarkMode: "locked"`. Ground `#0C100D`, card `#101611`, panel
  `#16201A`, border `#23402F`, accent `#33E68C`, ink `#E8F5EC`, muted
  `#7FA28D`. It is not gridiron's own aqua/navy palette, which stays
  unshipped.
- **`LedgerTheme()`** — warm, print-like light, `DarkMode: "adaptive"`
  with its own `Dark` palette. Ground `#FBF7EF`, card `#FFFFFF`, border
  `#E7DECB`, accent `#B4451F`, ink `#26201A`, muted `#8A7E6C`.

Terminal is dark-native by construction, so it needs no separate
swapped-in presentation; Ledger is light-native, so it carries a real
companion `Dark` palette instead. Together the two themes demonstrate
both of gsxmail's non-trivial dark-mode strategies with real, shipped
themes — see "Dark mode" above for what each strategy actually reaches.
Both themes pass EM140-144.

## The template gallery

[`examples/gallery`](examples/gallery) holds five complete templates,
each with typed props, a `.gsx` source, a fixture props JSON file, a
byte-exact golden HTML/text pair, and its own README:

| Template | Components | Theme |
|---|---|---|
| [`welcome`](examples/gallery/welcome) | Shell, Headline, PickList, Button, Footer | Paper |
| [`magiclink`](examples/gallery/magiclink) | Shell, Headline, Panel, Note, Button | Paper |
| [`receipt`](examples/gallery/receipt) | Shell, Badge, Headline, StatTable, Panel, Button, Footer | Paper |
| [`digest`](examples/gallery/digest) | Shell, Hero, Columns, StatTable, Divider, PickList | Ledger |
| [`alert`](examples/gallery/alert) | Shell, Signal, Badge, Note, Button | Terminal |

`receipt` is the pixel dossier's own complete worked example (section
8.3); `digest` and `alert` render under the two named themes above, so
the gallery shows off both an adaptive dark-mode style layer and a
dark-native one. `receipt/receipt.gsx` renders its `Badge` and `Button`
like this (hardened mode, Paper theme):

```html
<span style="display:inline-block; padding:2px 8px; border:1px solid #2F9E44; border-radius:2px; color:#2F9E44; font-family:'SFMono-Regular',Consolas,Menlo,monospace; font-size:10px; letter-spacing:0.06em; text-transform:uppercase;">PAID</span>
```

and its text twin: `[PAID]`. See `examples/gallery/README.md` for the
full table and a longer snippet.

## Import from existing HTML

Every other template compiler starts from a blank file. `gsxmail import`
starts from the email you already send:

```sh
gsxmail import newsletter.html --out emails/ --name Newsletter
```

It reads an existing email's rendered HTML — MJML's compiled output,
react-email's rendered output, or a hand-written table-soup mail — and
reverse-maps it onto the `email.*` components above. It writes five
files into `--out`:

| File | Contents |
|---|---|
| `template.gsx` | The best-effort `.gsx` source: every row it recognized becomes a named component; every row it does not recognize survives inside a raw `email.Custom` block instead of being dropped. |
| `props.go` | The declared props struct, one field per piece of text the mapper judged likely to vary (a Panel value, a headline's lede, the Shell's own wordmark and preheader). Imports nothing beyond the standard library, so it stands alone. |
| `theme.go` | `ImportedTheme()`, reproducing the source's own dominant colors as a `gsxmail.Theme`. This is the one generated file that imports `m31labs.dev/gsxmail`; delete it (and the `gsxmail.Options{Theme: ImportedTheme()}` reference at your call site) if you do not want it, or are generating `props.go` outside a gsxmail module entirely. |
| `props.sample.json` | The literal values the mapper harvested from the source HTML, so `template.gsx` renders correctly the moment you load it — no placeholder data to invent first. |
| `IMPORT-REPORT.md` | The honest accounting: every mapping decision and its confidence, every unmapped node and why, every synthesized props field's own source snippet, what the theme extraction did and did not recover, and a next-steps list. |

**Parsing never fails closed.** `gsxmail import` parses with
[gotreesitter](https://github.com/odvcencio/gotreesitter)'s HTML
grammar, whose error tolerance is the whole point: an unclosed `<td>`, a
stray `<b>` with no matching close, or a malformed comment still
produces a walkable tree, never a hard parse failure. A node the mapper
cannot confidently place never gets dropped — it lands inside
`email.Custom`, sanitized just enough to stay lint-clean (a
non-allowlisted tag is remapped or unwrapped, `class` and event
attributes are stripped, an unsafe `href` or a non-`https` image `src`
is swapped for a placeholder and flagged in the report), and gets a line
in `IMPORT-REPORT.md` explaining why.

**What it recognizes**, matching the output contracts above:

| Source shape | Maps to |
|---|---|
| A ghost-table/max-width card, one 600px table per compiled section, or a fluid `<div style="max-width">` wrapping one | `email.Shell` (+ a `Theme` literal extracted from its own dominant colors) |
| A hidden, first-child-of-`<body>` div with `display:none`/`overflow:hidden` | The Shell's own `preheader` |
| A large or bold heading, optionally followed by one paragraph | `email.Headline` |
| A padded `<td>` + `<a>` (MJML's `mso-padding-alt` shape), a bordered anchor, or a lone styled link | `email.Button` (`primary`/`secondary`/`link`, by which signal matched) |
| A table with `<th>` cells | `email.StatTable`, with its data rows synthesized into an `<Each>` |
| A run of two-cell rows — nested in their own sub-table, or stacked directly as the card's own rows | `email.Panel` |
| 2-4 sibling `inline-block`/`table-cell` divs | `email.Columns`/`email.Column` |
| A lone, sized `<img>` | `email.Hero` |
| An empty, fixed-height cell | `email.Spacer` |
| A content-free border-top rule, or `<hr>` | `email.Divider` |
| A border-left-accented block of plain text | `email.Note` |
| A short, bordered inline span | `email.Badge` (tone inferred from its color) |
| An `<ol>`/`<ul>`, or rows starting with "1.", "2." ... | `email.PickList` |
| Everything else | `email.Custom`, reported |

**The product promise is that it just works, then you tune it.** Every
imported template loads through `Load` and renders through `Render`
immediately, using the harvested sample props — that guarantee is
proven in CI against three checked-in foreign fixtures
(`importer/testdata/corpus/`: an MJML-compiled shape, a react-email
shape, and a deliberately crufty legacy table-soup mail with unclosed
tags) and against gsxmail's own five gallery templates rendered back to
HTML and re-imported, which recover their exact original component
sequence. `IMPORT-REPORT.md` is not an apology; it is the map of what to
review before you ship the result.

Build the CLI with `-tags 'grammar_subset grammar_subset_html'` to keep
gotreesitter's own footprint small: the tagged build adds roughly 819 KiB
over a build with no import verb at all (25,513,420 vs 24,674,716
bytes); the default, untagged build embeds every grammar gotreesitter
ships and costs roughly 19.4 MiB more. `gsxmail import`, and the CLI it
ships in, are the only places in this repository that import
gotreesitter outside a test file — `renderhtml`, `doc`, `lower`, and
`gsxmail.go` (the render path `Load`/`Render` execute) never do, proven
by `structural_isolation_test.go`.

## Size budget

Gmail clips an HTML email near 102,400 bytes and hides everything after
the cut, including an unsubscribe footer. `Render` checks the rendered
HTML part on every call:

- Over `Options.MaxHTMLBytes` (0 selects the default 100,000 bytes) fails
  closed: `Render` returns a zero `Parts` and a `*gsxmail.SizeBudgetError`
  carrying EM120's message.
- Over the fixed 90,000-byte warning line, but still within budget,
  succeeds: the returned `Parts.Diagnostics` carries one EM121 warning.
- `Options.MaxHTMLBytes: -1` disables both checks.

A template with an unbounded list — a `StatTable` fed from a large
`<Each>` — is the shape most likely to cross either line; size it against
a realistic worst-case fixture, not just your happy-path preview data.

## 60-second quick start

1. Create a module and add gsxmail:

   ```sh
   go mod init example.com/myapp
   go get m31labs.dev/gsxmail
   ```

2. Copy the [`examples/quickstart`](examples/quickstart) directory's
   `emails/` folder into your project.
3. Render it:

   ```go
   package main

   import (
       "os"

       "m31labs.dev/gsxmail"
   )

   func main() {
       set, err := gsxmail.Load(os.DirFS("emails"), gsxmail.Options{})
       if err != nil {
           panic(err)
       }
       parts, err := set.Render("WelcomeEmail", WelcomeProps{
           Name:     "Ada",
           Product:  "Acme",
           LoginURL: "https://acme.example/login",
       })
       if err != nil {
           panic(err)
       }
       os.WriteFile("welcome.html", []byte(parts.HTML), 0o644)
       os.WriteFile("welcome.txt", []byte(parts.Text), 0o644)
   }
   ```

See [`examples/quickstart`](examples/quickstart) for the full runnable
version, including the template and its props.

## The CLI

```sh
go install m31labs.dev/gsxmail/cmd/gsxmail@latest

gsxmail render WelcomeEmail \
  --dir emails \
  --props emails/welcome.props.json \
  --out .
```

This writes `WelcomeEmail.html` and `WelcomeEmail.txt`. Pass `--html -` or
`--text -` to stream one part to stdout instead.

### `gsxmail check`

```sh
gsxmail check --dir emails
gsxmail check --dir emails --format json
```

`check` runs `Load` and prints every finding from the email lint (design
spec section 8). Each line shows the file, line, column, EM code, and
exact message. It exits 1 if any finding is error-severity, and 0
otherwise. `--format json` prints the same findings as a JSON array for
CI annotations.

`check` never calls the network. It also cannot see your registered
helpers: it always runs with an empty `Options.Helpers`. It cannot tell a
genuinely missing helper from one your own program registers correctly.
Treat an EM014 or EM015 finding from `check` as informational. Validate
helper bindings with `Set.Check()` in your own test instead, where
`Options.Helpers` holds your real functions.

### `gsxmail import`

```sh
gsxmail import newsletter.html --out emails/ --name Newsletter
```

Reverse-maps an existing email's rendered HTML onto `email.*`
components: writes `template.gsx`, `props.go`, `theme.go`,
`props.sample.json`, and `IMPORT-REPORT.md` into `--out`, and prints the
report's own summary.
`--name` sets the generated component's name (`Newsletter` becomes
`NewsletterEmail`; a name that already ends in `Email` is kept as-is);
it defaults to a name derived from the source document's own `<title>`.
`--package` sets the generated Go package name (default `emails`). See
"Import from existing HTML" above for the full contract.

### `gsxmail matrix refresh`

```sh
gsxmail matrix refresh
```

This is the one gsxmail command that calls the network. It downloads
[caniemail's dataset](https://www.caniemail.com/api/data.json), trims it
to the style properties and clients EM101 and EM102 need, and prints the
per-client support diffs. Then it rewrites `lint/snapshot.json`.

Run it from a gsxmail module checkout, and commit the result like any
other reviewed change. Nothing else in gsxmail touches the network.
Every test runs offline too.

## The library API

```go
package gsxmail

type Parts struct {
    HTML        string
    Text        string
    Diagnostics []Diagnostic // today, only ever an EM121 size warning
}

type Options struct {
    Theme        Theme
    Helpers      map[string]any
    MaxHTMLBytes int
    Outlook      string // "" / "ghost-tables" (hardened, default) | "off" (parity); a Shell's own outlook="..." attribute overrides this per template
    Dir          string // the real on-disk directory fsys is rooted at, when known — see "Props type resolution" below
}

func Load(fsys fs.FS, opts Options) (*Set, error)

func (s *Set) Render(name string, props any) (Parts, error)
func (s *Set) Names() []string
func (s *Set) Check() []Diagnostic

type Theme = renderhtml.Theme // gains DarkMode ("none"/"locked"/"adaptive") and Dark *DarkPalette (WP5.2)
type DarkPalette = renderhtml.DarkPalette
func DefaultTheme() Theme  // "Paper": DarkMode "none"
func TerminalTheme() Theme // dark, mono-forward: DarkMode "locked" (WP5.3)
func LedgerTheme() Theme   // warm, print-like light: DarkMode "adaptive" (WP5.3)

type SizeBudgetError struct{ Diagnostic Diagnostic } // Render's EM120
```

`Load` compiles every `*.gsx` file under `fsys`. It resolves each
template's declared props struct with `go/types`, reading the sibling
`*.go` files in the same directory. Then it runs the full email lint,
EM001 through EM112, before it lowers anything.

A template with an error-severity finding makes `Load` fail. `Load`
returns the complete diagnostic list, as a `*LintError`, and no `Set`.
Only a template set that clears the lint gets lowered to an internal
typed tree. `Render` then accepts a props struct, for library callers, or
a `map[string]any`, the shape `gsxmail render` decodes JSON into.

`Set.Check` returns every finding `Load` collected, including warnings,
for a `Set` that loaded successfully. Use it to surface EM102-style
partial-support warnings in your own CI, without loading twice.

gsxmail still fails closed at render time. An unknown props field, an
unsupported expression, or a disallowed href scheme is a returned error.
It is never a silently empty or unsafe value. See "Two layers, one
guarantee" below for why both checks exist.

### Props type resolution

`Load` resolves a template's declared props struct by parsing and
type-checking the `*.go` files beside it with `go/types`. When a props
file imports another package — this module, a third-party dependency,
even the standard library — that resolution needs to find the enclosing
Go module. **Set `Options.Dir` to the same real, on-disk directory string
you passed to `os.DirFS` to build `fsys`.** `gsxmail check` and `gsxmail
render` always do this for you; a library caller using `os.DirFS`
directly should do the same:

```go
dir := "emails"
set, err := gsxmail.Load(os.DirFS(dir), gsxmail.Options{Dir: dir})
```

Without `Options.Dir`, resolution falls back to interpreting the props
file's path as relative to the process's own current working directory —
it works when that happens to be inside the owning module and fails,
with a clear EM192 finding naming the real cause, everywhere else. This
matters most for a template `gsxmail import` generated: its `theme.go`
imports gsxmail itself, and checking that output from a directory outside
its module (a CI job whose working directory is the CI root, not the
generated package) needs `Options.Dir` to resolve correctly. `Options.Dir`
makes resolution work regardless of the calling process's own working
directory, for any import an ordinary `go build` from inside that module
would also resolve. It does not add general module-graph awareness
(`go/importer`'s source mode, not `go/packages`, is what resolves the
import) — a props file reachable only through an unusual layout a plain
`go build` also could not find would still fail, as EM192, never as a
misleading EM012.

An unresolvable props type is a Go-source or environment problem, not an
email-dialect violation: `Load` reports it as EM192, once per template,
with the real underlying error, and skips the per-field "no such field"
checks (EM012) that type would otherwise drive — those would only be
noise once the real cause is already known.

### The `importer` package

```go
package importer

type Options struct {
    PackageName  string // default "emails"
    TemplateName string // default: derived from the source's own <title>
}

type Result struct {
    TemplateName    string
    TemplateGSX     string
    PropsGo         string
    SamplePropsJSON string
    Report          *Report
}

func Import(html []byte, sourceName string, opts Options) (*Result, error)
```

`gsxmail import` (above) is a thin CLI wrapper around this one function.
Call it directly to drive the mapper from your own Go program — a
migration script importing a whole directory of legacy templates, for
instance — without shelling out. `Import` never returns an error for
malformed or unrecognized markup; the one error case is a byte stream
gotreesitter cannot parse into any tree at all. This package is the one
place outside a `_test.go` file that gsxmail imports gotreesitter; see
"Import from existing HTML" above.

## Two layers, one guarantee

gsxmail checks props twice, on purpose:

1. **Load time (`typesafe/`).** `Load` resolves each template's declared
   props struct with `go/types`. It then type-checks every expression
   against that struct: a missing field is EM012, and a non-scalar field
   interpolated as text is EM013. This catches almost every problem
   before a template ever renders.
2. **Render time (`doc.Resolve`, `renderhtml`).** `Render` still resolves
   every field by reflection, against the actual props value it receives.
   The HTML writer still re-checks every href scheme too. A
   `map[string]any` props value, or any mismatch between what `Load` saw
   and what `Render` receives, still fails closed.

Load-time checking is the fast, precise path: it gives a real diagnostic.
Render-time checking is the fallback guarantee, and it holds even when
Load-time checking cannot reach a value. `gsxmail render`'s CLI path is
the clearest example: it decodes JSON into a `map[string]any` with no
named Go type to check.

## gosx version window

gsxmail targets `m31labs.dev/gosx v0.42.2`. Earlier versions are
untested. A compatibility policy across gosx releases, and a CI matrix
that runs against both the pinned and the latest gosx version, land with
a later work package.

## The caniemail snapshot

EM101 and EM102 check style properties against an embedded, dated copy of
[caniemail's dataset](https://www.caniemail.com/api/data.json). The
embedded snapshot covers CSS-property features only, for one default
client set:

- Gmail: web, iOS, Android
- Apple Mail: macOS, iOS
- Outlook: Windows desktop, web
- Yahoo: web

The framework owner has not yet ratified this default. It is the design
spec's proposal only. Widen or narrow it with a `gsxmail matrix refresh`
code change.

Every test runs against the embedded snapshot. Only `gsxmail matrix
refresh` fetches fresh data.

## License

MIT. See [LICENSE](LICENSE).
