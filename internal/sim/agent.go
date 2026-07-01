package sim

import (
	"math"

	"github.com/danielriddell21/vivarium/internal/geom"
	"github.com/danielriddell21/vivarium/internal/neural"
)

type Kind int

const (
	Herbivore Kind = iota
	Carnivore
)

func (k Kind) String() string {
	if k == Carnivore {
		return "Carnivore"
	}
	return "Herbivore"
}

const (
	VisionSectors = 6
	BrainInputs   = VisionSectors*3 + 1
	BrainHidden   = 12
	BrainOutputs  = 4
)

func (a *Agent) basalCost(w *World) float64 {
	if a.Kind == Carnivore {
		return w.params.CarnBasalCost
	}
	return w.params.HerbBasalCost
}

type Agent struct {
	ID         int
	Kind       Kind
	Pos        geom.Vec2
	Heading    float64
	Energy     float64
	Age        int
	Generation int

	LineageID int
	ParentID  int
	BirthTick int

	Brain  *neural.Brain
	Traits Traits
	Alive  bool

	Signal        float64
	pendingSignal float64

	ReproCooldown int

	LastInputs  []float64
	LastOutputs []float64
	LastMemory  []float64

	rewardBaseline float64
	LastReward     float64

	worldModel   *neural.Predictor
	lastFeatures []float64
	featBuf      []float64
	LastSurprise float64
}

func (a *Agent) learn(w *World, deltaEnergy float64) {
	extrinsic := math.Tanh(deltaEnergy * w.params.RewardScale)
	intrinsic := a.Traits.Curiosity * math.Tanh(a.LastSurprise*w.params.CuriosityGain)
	total := extrinsic + intrinsic

	adv := total - a.rewardBaseline
	a.rewardBaseline += w.params.BaselineLR * (total - a.rewardBaseline)
	a.LastReward = adv
	a.Brain.Learn(adv, a.Traits.Plasticity)
}

func (a *Agent) sense(w *World) []float64 {
	in := make([]float64, BrainInputs)
	in[3*VisionSectors] = clamp(a.Energy/w.params.ReproThreshold, 0, 1.5)

	// These sub-slices share backing storage with in, so writing to them fills
	// the corresponding input ranges directly.
	target := in[0:VisionSectors]
	threat := in[VisionSectors : 2*VisionSectors]
	voice := in[2*VisionSectors : 3*VisionSectors]
	// Effective vision shrinks at night, so the same agent perceives less in the
	// dark — pressure for caution, memory, and communication.
	r := a.Traits.SenseRadius * w.LightFactor()

	// voiceDist tracks the nearest conspecific per sector so voice carries the
	// closest neighbour's signal rather than an arbitrary one.
	var voiceDist [VisionSectors]float64
	for i := range voiceDist {
		voiceDist[i] = math.Inf(1)
	}

	// Herbivores forage on plants; each plant is perceived in proportion to how
	// edible it is for this agent's diet, so specialists are drawn to their type.
	if a.Kind == Herbivore {
		w.grid.forEachFoodNear(a.Pos, r, func(f *Food) {
			if w.foodRipe(f) {
				a.see(w, f.Pos, target, edibility(a.Traits.Diet, f.Type), r)
			}
		})
	}

	w.grid.forEachAgentNear(a.Pos, r, func(o *Agent) {
		if !o.Alive || o.ID == a.ID {
			return
		}
		switch {
		case a.Kind == Herbivore && o.Kind == Carnivore:
			a.see(w, o.Pos, threat, 1, r) // predators
		case a.Kind == Carnivore && o.Kind == Herbivore:
			a.see(w, o.Pos, target, 1, r) // prey
		}
		if o.Kind == a.Kind {
			a.hear(w, o, voice, voiceDist[:], r) // conspecific broadcast
		}
	})
	return in
}

func (a *Agent) see(w *World, p geom.Vec2, sectors []float64, weight, radius float64) {
	d := a.Pos.ShortestDelta(p, w.W, w.H)
	dist := d.Len()
	if dist == 0 || dist > radius {
		return
	}
	if w.occluded(a.Pos, d) {
		return // terrain blocks the line of sight
	}
	sec := a.sectorOf(d)
	if prox := (1 - dist/radius) * weight; prox > sectors[sec] {
		sectors[sec] = prox
	}
}

func (a *Agent) hear(w *World, o *Agent, voice, voiceDist []float64, radius float64) {
	d := a.Pos.ShortestDelta(o.Pos, w.W, w.H)
	dist := d.Len()
	if dist == 0 || dist > radius {
		return
	}
	sec := a.sectorOf(d)
	if dist < voiceDist[sec] {
		voiceDist[sec] = dist
		voice[sec] = o.Signal
	}
}

func (a *Agent) sectorOf(d geom.Vec2) int {
	rel := math.Mod(d.Angle()-a.Heading, 2*math.Pi)
	if rel < 0 {
		rel += 2 * math.Pi
	}
	sec := int(rel / (2 * math.Pi / VisionSectors))
	if sec >= VisionSectors { // guard the boundary case rel ≈ 2π
		sec = VisionSectors - 1
	}
	return sec
}

func (a *Agent) think(w *World) []float64 {
	in := a.sense(w)
	out := a.Brain.Forward(in)

	if a.lastFeatures != nil {
		// Surprise of the transition predicted last tick, and an SGD step toward it.
		a.LastSurprise = a.worldModel.Train(a.lastFeatures, in, w.params.WorldModelLR)
	}
	a.lastFeatures = a.featureVec(in, out)

	// Stage this tick's broadcast; it becomes visible to others next tick (see
	// World.Step), so all agents hear a consistent previous-tick signal.
	a.pendingSignal = out[3]

	a.LastInputs, a.LastOutputs, a.LastMemory = in, out, a.Brain.State()
	return out
}

func (a *Agent) featureVec(in, out []float64) []float64 {
	f := a.featBuf
	if cap(f) < len(in)+len(out) {
		f = make([]float64, len(in)+len(out))
	}
	f = f[:len(in)+len(out)]
	copy(f, in)
	copy(f[len(in):], out)
	a.featBuf = f
	return f
}

func (a *Agent) act(w *World, out []float64) (wantsEat bool) {
	a.Heading += out[0] * w.params.MaxTurn
	a.Heading = math.Mod(a.Heading, 2*math.Pi)

	speed := (out[1] + 1) / 2 * a.Traits.MaxSpeed
	if speed < 0 {
		speed = 0
	}
	proposed := a.Pos.Add(geom.FromAngle(a.Heading).Scale(speed)).WrapTo(w.W, w.H)
	a.Pos = w.resolveMove(a.Pos, proposed, a.Traits.Size)

	a.Energy -= a.basalCost(w) + w.params.MoveCost*speed
	if a.Energy > w.params.MaxEnergy {
		a.Energy = w.params.MaxEnergy
	}
	if a.ReproCooldown > 0 {
		a.ReproCooldown--
	}
	a.Age++
	return out[2] > 0
}
