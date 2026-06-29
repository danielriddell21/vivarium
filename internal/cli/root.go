// Package cli wires together the root Cobra command and all subcommands.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/danielriddell21/vivarium/internal/sim"
)

// guiOpts holds the resolved command-line options for the GUI. changed reports
// whether a given flag was set explicitly (so config-file values are only
// overridden by flags the user actually passed).
type guiOpts struct {
	seed                           int64
	width, height                  float64
	plants, herbivores, carnivores int
	rescue                         bool
	configPath                     string
	printConfig                    bool
	loadPath, snapPath             string
	changed                        func(name string) bool
}

// Execute builds and runs the root command. Returns non-nil on error.
func Execute(version string) error {
	cfg := sim.DefaultConfig()
	var o guiOpts

	root := &cobra.Command{
		Use:           "vivarium",
		Short:         "Watch an evolving 2D ecosystem in a native window",
		Long:          "vivarium opens a window showing a 2D ecosystem in which agent behaviour evolves and is learned. The GUI requires a build with the \"ebiten\" tag; for headless batch runs use the 'vivarium headless' subcommand.",
		Version:       version,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.changed = cmd.Flags().Changed
			return runGame(o)
		},
	}

	f := root.Flags()
	f.Int64Var(&o.seed, "seed", 1, "random seed for reproducible runs")
	f.Float64Var(&o.width, "width", cfg.Width, "world width in pixels")
	f.Float64Var(&o.height, "height", cfg.Height, "world height in pixels")
	f.IntVar(&o.plants, "plants", cfg.Plants, "initial plant count")
	f.IntVar(&o.herbivores, "herbivores", cfg.Herbivores, "initial herbivore count")
	f.IntVar(&o.carnivores, "carnivores", cfg.Carnivores, "initial carnivore count")
	f.BoolVar(&o.rescue, "rescue", cfg.Rescue, "rescue effect: immigrants arrive when a tier nears extinction")
	f.StringVar(&o.configPath, "config", "", "path to a JSON config file overriding defaults (see --print-config)")
	f.BoolVar(&o.printConfig, "print-config", false, "print the default config as JSON and exit")
	f.StringVar(&o.loadPath, "load", "", "load a saved population snapshot instead of a fresh world")
	f.StringVar(&o.snapPath, "snapshot", "vivarium-snapshot.json", "file the 's' key saves the population to")

	root.AddCommand(headlessCmd())
	root.AddCommand(completionCmd())

	if err := root.Execute(); err != nil {
		return fmt.Errorf("vivarium: %w", err)
	}
	return nil
}
