//go:build !ebiten

package gui

import "errors"

// Available reports whether the Ebiten window is compiled in.
func Available() bool { return false }

// Run stands in when the GUI is not compiled in. The Ebiten window is gated
// behind the "ebiten" build tag so the default build needs no graphics or cgo
// dependencies.
func Run(_ Config) error {
	return errors.New("built without the GUI; rebuild with -tags ebiten, or use the 'vivarium headless' subcommand")
}
