package sim

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"slices"

	"github.com/danielriddell21/crucible/geom"
	"github.com/danielriddell21/crucible/ring"

	"github.com/danielriddell21/vivarium/internal/neural"
)

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

func NewWorldFromSnapshot(rng *rand.Rand, s Snapshot) *World {
	w := &World{
		W: s.Config.Width, H: s.Config.Height,
		rng:            rng,
		params:         s.Config.Params,
		targetPlants:   s.Config.TargetPlants,
		rescue:         s.Config.Rescue,
		minHerb:        s.Config.MinHerbivores,
		minCarn:        s.Config.MinCarnivores,
		Tick:           s.Tick,
		obstacles:      slices.Clone(s.Obstacles),
		Foods:          slices.Clone(s.Foods),
		history:        ring.New[Counts](maxHistory),
		lineageHistory: ring.New[map[int]int](maxHistory),
	}
	discarded := 0
	for _, as := range s.Agents {
		brain := neural.FromGenome(s.BrainIn, s.BrainHid, s.BrainOut, as.Genome)
		if brain == nil {
			// Genome shape no longer matches the configured brain (e.g. a
			// snapshot from an incompatible build); start this agent fresh.
			brain = neural.New(rng, BrainInputs, BrainHidden, BrainOutputs)
			discarded++
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
	if discarded > 0 {
		fmt.Fprintf(os.Stderr, "vivarium: snapshot genome mismatch — reinitialised %d of %d agents with fresh brains\n", discarded, len(s.Agents))
	}
	w.reindex()
	w.sampleHistory()
	return w
}

func SaveSnapshot(path string, s Snapshot) error {
	data, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return fmt.Errorf("sim: marshal snapshot: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("sim: write snapshot: %w", err)
	}
	return nil
}

func LoadSnapshotFile(path string) (Snapshot, error) {
	var s Snapshot
	data, err := os.ReadFile(path)
	if err != nil {
		return s, fmt.Errorf("sim: read snapshot: %w", err)
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("sim: parse snapshot: %w", err)
	}
	return s, nil
}
