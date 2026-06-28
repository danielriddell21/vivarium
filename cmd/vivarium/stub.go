//go:build !ebiten

// Command vivarium is the GUI front-end for the ecosystem simulation. The
// Ebiten-backed window is compiled only under the "ebiten" build tag so that
// the default build (and CI lint) needs no graphics or cgo dependencies. This
// stub stands in when the tag is absent and points at the real entry points.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "vivarium: built without the GUI; rebuild with -tags ebiten, or use vivarium-headless")
	os.Exit(1)
}
