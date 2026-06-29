package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/danielriddell21/vivarium/internal/sim"
)

// headlessCmd advances the ecosystem simulation without any GUI, for long
// offline experiments and reproducible batch runs. It imports only the
// simulation core (no Ebiten), so it needs no display and starts instantly. It
// emits a CSV row of population and trait statistics every N ticks to stdout.
//
//	vivarium headless --seed 1 --ticks 20000 --every 200 > run.csv
//	vivarium headless --config myconfig.json --ticks 50000 > run.csv
//	vivarium headless --print-config > config.json
func headlessCmd() *cobra.Command {
	cfg := sim.DefaultConfig()
	var (
		seed                           int64
		ticks, every                   int
		configPath                     string
		printConfig                    bool
		plants, herbivores, carnivores int
		width, height                  float64
		rescue                         bool
		loadPath, savePath             string
	)

	cmd := &cobra.Command{
		Use:           "headless",
		Short:         "Headless batch runner for the Vivarium ecosystem simulation",
		Long:          "headless advances the ecosystem simulation without a GUI and emits CSV population/trait statistics, for long offline experiments and reproducible batch runs.",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if printConfig {
				return printDefaultConfig()
			}
			resolved, err := resolveConfig(cmd.Flags(), cfg, ov{
				configPath: configPath,
				plants:     plants, herbivores: herbivores, carnivores: carnivores,
				width: width, height: height, rescue: rescue,
			})
			if err != nil {
				return err
			}
			w, err := buildWorld(rand.New(rand.NewSource(seed)), resolved, loadPath)
			if err != nil {
				return err
			}
			if err := simulate(w, ticks, every); err != nil {
				return err
			}
			return saveFinal(w, savePath)
		},
	}

	f := cmd.Flags()
	f.Int64Var(&seed, "seed", 1, "random seed for reproducible runs")
	f.IntVar(&ticks, "ticks", 10000, "number of ticks to simulate")
	f.IntVar(&every, "every", 200, "emit a stats row every N ticks")
	f.StringVar(&configPath, "config", "", "path to a JSON config file overriding defaults")
	f.BoolVar(&printConfig, "print-config", false, "print the default config as JSON and exit")
	f.IntVar(&plants, "plants", cfg.Plants, "initial plant count")
	f.IntVar(&herbivores, "herbivores", cfg.Herbivores, "initial herbivore count")
	f.IntVar(&carnivores, "carnivores", cfg.Carnivores, "initial carnivore count")
	f.Float64Var(&width, "width", cfg.Width, "world width")
	f.Float64Var(&height, "height", cfg.Height, "world height")
	f.BoolVar(&rescue, "rescue", cfg.Rescue, "rescue effect on/off")
	f.StringVar(&loadPath, "load", "", "start from a saved population snapshot")
	f.StringVar(&savePath, "save", "", "write the final population snapshot to this file")

	return cmd
}

// ov bundles the flag values that can override the config when set explicitly.
type ov struct {
	configPath                     string
	plants, herbivores, carnivores int
	width, height                  float64
	rescue                         bool
}

// resolveConfig applies precedence defaults < config file < explicitly-set flags.
func resolveConfig(fl *pflag.FlagSet, cfg sim.Config, o ov) (sim.Config, error) {
	if o.configPath != "" {
		c, err := sim.LoadConfig(o.configPath)
		if err != nil {
			return cfg, fmt.Errorf("load config: %w", err)
		}
		cfg = c
	}
	if fl.Changed("plants") {
		cfg.Plants = o.plants
	}
	if fl.Changed("herbivores") {
		cfg.Herbivores = o.herbivores
	}
	if fl.Changed("carnivores") {
		cfg.Carnivores = o.carnivores
	}
	if fl.Changed("width") {
		cfg.Width = o.width
	}
	if fl.Changed("height") {
		cfg.Height = o.height
	}
	if fl.Changed("rescue") {
		cfg.Rescue = o.rescue
	}
	if cfg.TargetPlants < cfg.Plants {
		cfg.TargetPlants = cfg.Plants
	}
	return cfg, nil
}

// printDefaultConfig writes the default config as indented JSON to stdout.
func printDefaultConfig() error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(sim.DefaultConfig()); err != nil {
		return fmt.Errorf("print config: %w", err)
	}
	return nil
}

// buildWorld constructs the world, loading from a snapshot when loadPath is set.
func buildWorld(rng *rand.Rand, cfg sim.Config, loadPath string) (*sim.World, error) {
	if loadPath != "" {
		snap, err := sim.LoadSnapshotFile(loadPath)
		if err != nil {
			return nil, fmt.Errorf("load snapshot: %w", err)
		}
		return sim.NewWorldFromSnapshot(rng, snap), nil
	}
	return sim.NewWorld(rng, cfg), nil
}

// simulate advances the world, emitting a CSV stats row every `every` ticks.
func simulate(w *sim.World, ticks, every int) error {
	out := bufio.NewWriter(os.Stdout)
	fmt.Fprintln(out, "tick,plants,herbivores,carnivores,meanGen,maxGen,meanSize,meanSpeed,meanSense,meanPlast,meanCurio,meanDrift,lineages")
	for t := 0; t <= ticks; t++ {
		if t%every == 0 {
			writeStats(out, w, t)
		}
		w.Step()
	}
	if err := out.Flush(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}
	return nil
}

// saveFinal writes the final population snapshot when savePath is set.
func saveFinal(w *sim.World, savePath string) error {
	if savePath == "" {
		return nil
	}
	if err := sim.SaveSnapshot(savePath, w.Snapshot()); err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}
	fmt.Fprintf(os.Stderr, "saved population (%d agents) to %s\n", len(w.Snapshot().Agents), savePath)
	return nil
}

// writeStats emits one CSV row summarising the living population.
func writeStats(out *bufio.Writer, w *sim.World, tick int) {
	c := w.CountKinds()
	var n, sumGen, maxGen int
	var size, speed, sense, plast, curio, drift float64
	lineages := make(map[int]struct{})
	for _, a := range w.Agents {
		if !a.Alive {
			continue
		}
		n++
		sumGen += a.Generation
		if a.Generation > maxGen {
			maxGen = a.Generation
		}
		size += a.Traits.Size
		speed += a.Traits.MaxSpeed
		sense += a.Traits.SenseRadius
		plast += a.Traits.Plasticity
		curio += a.Traits.Curiosity
		drift += a.Brain.LearnedDrift()
		lineages[a.LineageID] = struct{}{}
	}
	mean := func(sum float64) float64 {
		if n == 0 {
			return 0
		}
		return sum / float64(n)
	}
	meanGen := 0.0
	if n > 0 {
		meanGen = float64(sumGen) / float64(n)
	}
	fmt.Fprintf(out, "%d,%d,%d,%d,%.2f,%d,%.2f,%.2f,%.1f,%.4f,%.3f,%.4f,%d\n",
		tick, c.Plants, c.Herbivores, c.Carnivores,
		meanGen, maxGen, mean(size), mean(speed), mean(sense), mean(plast), mean(curio), mean(drift),
		len(lineages))
}
