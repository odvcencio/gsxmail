# Changelog

All notable changes to gsxmail are documented in this file.

## Unreleased

### Added

- `gosx.Compile` integration: `Load` compiles every `*.gsx` file under an
  `fs.FS` and lowers each declared component to a typed `EmailDoc` tree.
- The email.\* standard component set for this release: `Shell`, `Signal`,
  `Headline`, `Panel`/`PanelRow`, `CTA`, `PickList`/`Item`, and `Footer`.
- The HTML writer: theme tokens inline to styles, with a minimal escaper
  and an href scheme allowlist (https, http, mailto) for `CTA`.
- The text writer: the 72-column wrap and label/column rules, derived
  from the same tree the HTML writer reads, so the two parts cannot drift.
- `Set.Render`: renders one named template from a props struct or a
  `map[string]any`, deterministically.
- The `gsxmail render` CLI verb.
- `examples/quickstart`: a runnable welcome-email walkthrough.

### Known limitations

- `Set.Check` returns no diagnostics yet; the EM lint catalog is a later
  release.
- `gsxmail dev` and `gsxmail check` are not yet implemented.
- The Gmail clip size budget (`Options.MaxHTMLBytes`) is accepted but not
  enforced yet.
