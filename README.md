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

## Status: WP2, the lint layer

This release compiles one template shape end to end: `Shell`, `Signal`,
`Headline`, `Panel`/`PanelRow`, `CTA`, `PickList`/`Item`, and `Footer`. It
ships the `gsxmail render` and `gsxmail check` CLI verbs, `gsxmail matrix
refresh`, and the full `Load`/`Render`/`Check` library API.

`Load` now runs the full email lint catalog (EM001 through EM112) before
it lowers anything. A missing props field, an expression outside the
email dialect, a disallowed HTML element, or an unsupported style
property all fail `Load` closed. Each failure carries an exact
diagnostic, not a runtime surprise.

`gsxmail dev` (the live preview server) lands in a later release. Until
then, validate a template with `gsxmail check` and by rendering it.

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
    HTML string
    Text string
}

type Options struct {
    Theme        Theme
    Helpers      map[string]any
    MaxHTMLBytes int
}

func Load(fsys fs.FS, opts Options) (*Set, error)

func (s *Set) Render(name string, props any) (Parts, error)
func (s *Set) Names() []string
func (s *Set) Check() []Diagnostic

type Theme = renderhtml.Theme
func DefaultTheme() Theme
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
