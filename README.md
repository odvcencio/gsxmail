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

## Status: WP1, the thin vertical slice

This release compiles one template shape end to end: `Shell`, `Signal`,
`Headline`, `Panel`/`PanelRow`, `CTA`, `PickList`/`Item`, and `Footer`. It
ships the `gsxmail render` CLI verb and the full `Load`/`Render` library
API.

`gsxmail dev` (the live preview server) and `gsxmail check` (the lint
catalog and the caniemail client-support snapshot) are landing in v0.2.
Until then, validate a template by rendering it and reviewing the output
directly.

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

`Load` compiles every `*.gsx` file under `fsys` and lowers each declared
component to an internal typed tree. `Render` accepts a props struct (for
library callers) or a `map[string]any` (the shape `gsxmail render`
decodes JSON into, ahead of the go/types-resolved decoding a later release
adds).

`Set.Check` returns an empty diagnostic list in this release: the EM lint
catalog described in the design spec is part of the v0.2 work package.
gsxmail still fails closed at render time today — an unknown props field,
an unsupported expression, or a disallowed href scheme is a returned
error, never a silently empty or unsafe value.

## gosx version window

gsxmail targets `m31labs.dev/gosx v0.42.2`. Earlier versions are
untested. A compatibility policy across gosx releases, and a CI matrix
that runs against both the pinned and the latest gosx version, land with
the v0.2 work package.

## License

MIT. See [LICENSE](LICENSE).
