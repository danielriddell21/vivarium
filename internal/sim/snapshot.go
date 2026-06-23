package sim

import (
	"encoding/json"
	"math/rand"
	"os"

	"github.com/danielriddell21/vivarium/internal/geom"
	"github.com/danielriddell21/vivarium/internal/neural"
)

// Snapshot is a serialisable capture of a world's population and terrain. It is
// self-contained: combined with a fresh RNG it rebuilds a runnable world (see
// NewWorldFromSnapshot). Loading is not a bit-exact resume — transient state (the
// RNG stream, recurrent memory, learned weights, reward baselines) is intentionally
// not stored — but it preserves the evolved genomes, traits, and genealogy so a
// population can be saved, shared, and used to seed further runs.
type Snapshot struct {
	Config    Config          `json:"config"`
	Tick      int             `json:"tick"`
	BrainIn   int             `json:"brainIn"`
	BrainHid  int             `json:"brainHidden"`
	BrainOut  int             `json:"brainOut"`
	Obstacles []Obstacle      `json:"obstacles"`
	Foods     []*Food         `json:"foods"`
	Agents    []AgentSnapshot `json:"agents"`
}

// AgentSnapshot is the heritable and identity state of one agent.
type AgentSnapshot struct {
	Kind          Kind      `json:"kind"`
	Pos           geom.Vec2 `json:"pos"`
	Heading       float64   `json:"heading"`
	Energy        float64   `json:"energy"`
	Age           int       `json:"age"`
	Generation    int       `json:"generation"`
	LineageID     int       `json:"lineage"`
	ParentID      int       `json:"parent"`
	BirthTick     int       `json:"birthTick"`
	ReproCooldown int       `json:"reproCooldown"`
	Traits        Traits    `json:"traits"`
	Genome        []float64 `json:"genome"`
}

// Snapshot captures the world's current living population and terrain.
func (w *World) Snapshot() Snapshot {
	s := Snapshot{
		Config:    w.config(),
		Tick:      w.Tick,
		BrainIn:   BrainInputs,
		BrainHid:  BrainHidden,
		BrainOut:  BrainOutputs,
		Obstacles: append([]Obstacle(nil), w.obstacles...),
		Foods:     append([]*Food(nil), w.Foods...),
	}
	for _, a := range w.Agents {
		if !a.Alive {
			continue
		}
		s.Agents = append(s.Agents, AgentSnapshot{
			Kind: a.Kind, Pos: a.Pos, Heading: a.Heading, Energy: a.Energy,
			Age: a.Age, Generation: a.Generation, LineageID: a.LineageID,
			ParentID: a.ParentID, BirthTick: a.BirthTick, ReproCooldown: a.ReproCooldown,
			Traits: a.Traits, Genome: a.Brain.Genome(),
		})
	}
	return s
}

// config reconstructs a Config describing the world's rules (the initial-population
// counts are irrelevant once a snapshot is loaded and are left at the live counts).
func (w *World) config() Config {
	c := w.CountKinds()
	return Config{
		Width: w.W, Height: w.H,
		Plants: c.Plants, Herbivores: c.Herbivores, Carnivores: c.Carnivores,
		TargetPlants: w.targetPlants, Obstacles: len(w.obstacles),
		Rescue: w.rescue, MinHerbivores: w.minHerb, MinCarnivores: w.minCarn,
		Params: w.params,
	}
}

// NewWorldFromSnapshot rebuilds a world from a snapshot, reconstructing each agent
// from its stored genome. New IDs are assigned; lineage IDs are preserved (with the
// next-ID counter advanced past them) so the lineage/phylogeny views stay coherent.
func NewWorldFromSnapshot(rng *rand.Rand, s Snapshot) *World {
	w := &World{
		W: s.Config.Width, H: s.Config.Height,
		rng:          rng,
		params:       s.Config.Params,
		targetPlants: s.Config.TargetPlants,
		rescue:       s.Config.Rescue,
		minHerb:      s.Config.MinHerbivores,
		minCarn:      s.Config.MinCarnivores,
		Tick:         s.Tick,
		obstacles:    append([]Obstacle(nil), s.Obstacles...),
		Foods:        append([]*Food(nil), s.Foods...),
	}
	for _, as := range s.Agents {
		brain := neural.FromGenome(s.BrainIn, s.BrainHid, s.BrainOut, as.Genome)
		if brain == nil {
			brain = neural.New(rng, BrainInputs, BrainHidden, BrainOutputs)
		}
		w.nextID++
		a := &Agent{
			ID: w.nextID, Kind: as.Kind, Pos: as.Pos, Heading: as.Heading,
			Energy: as.Energy, Age: as.Age, Generation: as.Generation,
			LineageID: as.LineageID, ParentID: as.ParentID, BirthTick: as.BirthTick,
			ReproCooldown: as.ReproCooldown, Traits: as.Traits, Brain: brain, Alive: true,
			worldModel: neural.NewPredictor(BrainInputs+BrainOutputs, BrainInputs),
		}
		if as.LineageID > w.nextLineageID {
			w.nextLineageID = as.LineageID
		}
		w.Agents = append(w.Agents, a)
		w.recordBirth(a)
	}
	w.reindex()
	w.sampleHistory()
	return w
}

// SaveSnapshot writes a snapshot to path as indented JSON.
func SaveSnapshot(path string, s Snapshot) error {
	data, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// LoadSnapshotFile reads a snapshot from a JSON file.
func LoadSnapshotFile(path string) (Snapshot, error) {
	var s Snapshot
	data, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(data, &s)
	return s, err
}
