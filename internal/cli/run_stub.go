//go:build !ebiten

package cli

import "errors"

// runGame stands in when the GUI is not compiled in. The Ebiten window is gated
// behind the "ebiten" build tag so the default build needs no graphics or cgo
// dependencies.
func runGame(_ guiOpts) error {
	return errors.New("built without the GUI; rebuild with -tags ebiten, or use the 'vivarium headless' subcommand")
}
