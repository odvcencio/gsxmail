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

## Status: WP5.1, bulletproof output contracts

The stdlib now covers `Shell`, `Signal`, `Headline`, `Panel`/`PanelRow`,
`CTA`, `PickList`/`Item`, `Footer`, `Note`, `Divider`, and
`StatTable`/`StatRow`. Templates can loop over a slice with `<Each>` and
branch on a bool with `<If>`; a raw-element `Custom` subtree stays
available as an escape hatch for markup the stdlib does not express. It
ships the `gsxmail render` and `gsxmail check` CLI verbs, `gsxmail matrix
refresh`, and the full `Load`/`Render`/`Check` library API. `Render`'s
HTML part is hardened, bulletproof markup by default, mechanically proven
by a structural verification pass — see "Output contracts" below.

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
anywhere outside a test.

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
    Outlook      string // "" / "ghost-tables" (hardened, default) | "off" (parity)
}

func Load(fsys fs.FS, opts Options) (*Set, error)

func (s *Set) Render(name string, props any) (Parts, error)
func (s *Set) Names() []string
func (s *Set) Check() []Diagnostic

type Theme = renderhtml.Theme
func DefaultTheme() Theme

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
