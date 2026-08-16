// Package em192 is EM192's fixture (launch-gate B3, point 1): a props
// package whose own compile fails because of an import path no module in
// any GOPATH or module graph provides. Resolver.Resolve must report this
// as its own error (deterministically — no environment makes this
// particular import path resolve), never as a silent nil that leaves
// every props.field read below reporting a misleading EM012.
package em192

import _ "no.such.module.example/does-not-exist"

// BadProps is otherwise an ordinary props type; its own package simply
// cannot type-check because of the blank import above.
type BadProps struct {
	Name string
}
