/*
Vivarium is a small 2D ecosystem simulation in which behaviour evolves rather
than being programmed.

The world is a toroidal plane with three trophic tiers (plants, herbivores,
carnivores). Each agent is steered by a tiny recurrent neural network that both
evolves across generations and learns within a lifetime; agents reproduce,
mutate, and communicate, so viable strategies emerge over time.

The default invocation opens the GUI. The Ebiten window is compiled only under
the "ebiten" build tag, so the default build (and CI) needs no graphics or cgo
dependencies; without the tag the command explains how to get the GUI or run
headless. For offline batch runs use the "vivarium headless" subcommand.
*/
package main

import (
	"fmt"
	"os"

	"github.com/danielriddell21/vivarium/internal/cli"
)

// version is the build version, overridden at release time via
// -ldflags "-X main.version=...". It defaults to "dev" for local builds.
var version = "dev"

func main() {
	if err := cli.Execute(version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
