/*
vivarium-headless runs the ecosystem simulation without any GUI, for long offline
experiments and reproducible batch runs. It imports only the simulation core (no
Ebiten), so it needs no display and starts instantly.

It advances the world for a number of ticks and emits a CSV row of population and
trait statistics every N ticks to stdout, which is easy to redirect to a file and
plot. Configuration uses the same JSON files and flags as the GUI binary.

	go run ./cmd/vivarium-headless -seed 1 -ticks 20000 -every 200 > run.csv
	go run ./cmd/vivarium-headless -config myconfig.json -ticks 50000 > run.csv
	go run ./cmd/vivarium-headless -print-config > config.json
*/
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/danielriddell21/vivarium/internal/sim"
)

func main() {
	cfg := sim.DefaultConfig()
	seed := flag.Int64("seed", 1, "random seed for reproducible runs")
	ticks := flag.Int("ticks", 10000, "number of ticks to simulate")
	every := flag.Int("every", 200, "emit a stats row every N ticks")
	configPath := flag.String("config", "", "path to a JSON config file overriding defaults")
	printConfig := flag.Bool("print-config", false, "print the default config as JSON and exit")
	plants := flag.Int("plants", cfg.Plants, "initial plant count")
	herbivores := flag.Int("herbivores", cfg.Herbivores, "initial herbivore count")
	carnivores := flag.Int("carnivores", cfg.Carnivores, "initial carnivore count")
	width := flag.Float64("width", cfg.Width, "world width")
	height := flag.Float64("height", cfg.Height, "world height")
	rescue := flag.Bool("rescue", cfg.Rescue, "rescue effect on/off")
	loadPath := flag.String("load", "", "start from a saved population snapshot")
	savePath := flag.String("save", "", "write the final population snapshot to this file")
	flag.Parse()

	if *printConfig {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(sim.DefaultConfig()); err != nil {
			log.Fatal(err)
		}
		return
	}

	// Precedence: defaults < config file < explicitly-set flags.
	if *configPath != "" {
		c, err := sim.LoadConfig(*configPath)
		if err != nil {
			log.Fatalf("config: %v", err)
		}
		cfg = c
	}
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "plants":
			cfg.Plants = *plants
		case "herbivores":
			cfg.Herbivores = *herbivores
		case "carnivores":
			cfg.Carnivores = *carnivores
		case "width":
			cfg.Width = *width
		case "height":
			cfg.Height = *height
		case "rescue":
			cfg.Rescue = *rescue
		}
	})
	if cfg.TargetPlants < cfg.Plants {
		cfg.TargetPlants = cfg.Plants
	}

	rng := rand.New(rand.NewSource(*seed))
	var w *sim.World
	if *loadPath != "" {
		snap, err := sim.LoadSnapshotFile(*loadPath)
		if err != nil {
			log.Fatalf("load snapshot: %v", err)
		}
		w = sim.NewWorldFromSnapshot(rng, snap)
	} else {
		w = sim.NewWorld(rng, cfg)
	}
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	fmt.Fprintln(out, "tick,plants,herbivores,carnivores,meanGen,maxGen,meanSize,meanSpeed,meanSense,meanPlast,meanCurio,meanDrift,lineages")
	for t := 0; t <= *ticks; t++ {
		if t%*every == 0 {
			writeStats(out, w, t)
		}
		w.Step()
	}
	if *savePath != "" {
		out.Flush()
		if err := sim.SaveSnapshot(*savePath, w.Snapshot()); err != nil {
			log.Fatalf("save snapshot: %v", err)
		}
		fmt.Fprintf(os.Stderr, "saved population (%d agents) to %s\n", len(w.Snapshot().Agents), *savePath)
	}
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
