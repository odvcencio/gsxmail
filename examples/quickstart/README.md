# gsxmail quickstart

This example renders one template, `WelcomeEmail`, from typed Go props to a
matched HTML and plain-text pair.

## Files

- `emails/welcome.gsx` — the template. It composes gsxmail's stdlib blocks:
  `Shell`, `Signal`, `Headline`, `Panel`, `CTA`, `PickList`, `Footer`.
- `emails/welcome.go` — `WelcomeProps`, the Go struct the template reads.
- `emails/welcome.props.json` — fixture props for this walkthrough.
- `main.go` — loads `emails/`, decodes the fixture, and renders.

## Run it

```sh
go run .
```

This writes `welcome.html` and `welcome.txt` in the current directory.
Open `welcome.html` in a browser to see the rendered email; open
`welcome.txt` to see its plain-text twin, derived from the same template.

## Change something

Edit `emails/welcome.gsx` or `emails/welcome.props.json` and run again. The
HTML and text parts always stay in sync: they render from one tree, so nothing
you change in one part can silently miss the other.
