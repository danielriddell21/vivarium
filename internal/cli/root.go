package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/danielriddell21/vivarium/internal/gui"
	"github.com/danielriddell21/vivarium/internal/sim"
)

func Execute(version string) error {
	cfg := sim.DefaultConfig()
	var o gui.Config

	root := &cobra.Command{
		Use:           "vivarium",
		Short:         "Watch an evolving 2D ecosystem in a native window",
		Long:          "vivarium opens a window showing a 2D ecosystem in which agent behaviour evolves and is learned. The GUI requires a build with the \"ebiten\" tag; for headless batch runs use the 'vivarium headless' subcommand.",
		Version:       version,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Changed = cmd.Flags().Changed
			if err := gui.Run(o); err != nil {
				return fmt.Errorf("run gui: %w", err)
			}
			return nil
		},
	}

	f := root.Flags()
	f.Int64Var(&o.Seed, "seed", 1, "random seed for reproducible runs")
	f.Float64Var(&o.Width, "width", cfg.Width, "world width in pixels")
	f.Float64Var(&o.Height, "height", cfg.Height, "world height in pixels")
	f.IntVar(&o.Plants, "plants", cfg.Plants, "initial plant count")
	f.IntVar(&o.Herbivores, "herbivores", cfg.Herbivores, "initial herbivore count")
	f.IntVar(&o.Carnivores, "carnivores", cfg.Carnivores, "initial carnivore count")
	f.BoolVar(&o.Rescue, "rescue", cfg.Rescue, "rescue effect: immigrants arrive when a tier nears extinction")
	f.StringVar(&o.ConfigPath, "config", "", "path to a JSON config file overriding defaults (see --print-config)")
	f.BoolVar(&o.PrintConfig, "print-config", false, "print the default config as JSON and exit")
	f.StringVar(&o.LoadPath, "load", "", "load a saved population snapshot instead of a fresh world")
	f.StringVar(&o.SnapPath, "snapshot", "vivarium-snapshot.json", "file the 's' key saves the population to")
	f.StringVar(&o.RecordPath, "record", "", "record the run to this GIF path, then exit")
	f.IntVar(&o.RecordFPS, "record-fps", 30, "recording frames per second")
	f.IntVar(&o.RecordScale, "record-scale", 2, "downscale factor for the recording")
	f.IntVar(&o.RecordFrames, "record-frames", 600, "frames to capture before exiting")

	root.AddCommand(headlessCmd())
	root.AddCommand(completionCmd())

	if err := root.Execute(); err != nil {
		return fmt.Errorf("vivarium: %w", err)
	}
	return nil
}
