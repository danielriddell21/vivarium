package sim

import (
	"math"

	"github.com/danielriddell21/vivarium/internal/geom"
	"github.com/danielriddell21/vivarium/internal/neural"
)

// Kind distinguishes the two trophic tiers of mobile agents. Plants are modelled
// separately as Food, not as agents.
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

// Sensing & neural network dimensions shared by every brain.
//
// Vision is a ring of VisionSectors equal-angle sectors covering the full 360°
// around the agent, oriented relative to its heading (sector 0 points straight
// ahead, sectors proceed counter-clockwise). Each sector reports a value for the
// nearest thing it sees, on three channels:
//
//	target channel — proximity of what the agent eats (plants for herbivores,
//	                 herbivores for carnivores)
//	threat channel — proximity of what eats it (carnivores for herbivores; always
//	                 empty for carnivores, which are apex predators)
//	voice channel  — the signal (-1..1) currently broadcast by the nearest
//	                 same-kind neighbour, i.e. communication between conspecifics
//
// Proximity is 0 (nothing / at the sense-radius edge) to 1 (adjacent). Inputs:
//
//	[0 .. V)     target-channel proximity per sector
//	[V .. 2V)    threat-channel proximity per sector
//	[2V .. 3V)   voice-channel signal per sector   (V = VisionSectors)
//	[3V]         own energy (normalised, ~0..1)
//
// Outputs:
//
//	0: turn   (-1..1, scaled to a max turn per tick)
//	1: speed  (-1..1, mapped to 0..MaxSpeed)
//	2: eat    (>0 means attempt to feed this tick)
//	3: signal (-1..1, broadcast to nearby conspecifics next tick)
const (
	VisionSectors = 6
	BrainInputs   = VisionSectors*3 + 1
	BrainHidden   = 12
	BrainOutputs  = 4
)

// basalCost returns the per-tick upkeep for the agent's kind (a tunable Param).
func (a *Agent) basalCost(w *World) float64 {
	if a.Kind == Carnivore {
		return w.params.CarnBasalCost
	}
	return w.params.HerbBasalCost
}

// Agent is a mobile organism driven by an evolved neural brain.
type Agent struct {
	ID         int
	Kind       Kind
	Pos        geom.Vec2
	Heading    float64 // radians
	Energy     float64
	Age        int
	Generation int

	// Lineage identity: LineageID is the founding ancestor shared by all
	// descendants; ParentID and BirthTick record genealogy for the lineage view.
	LineageID int
	ParentID  int
	BirthTick int

	Brain  *neural.Brain
	Traits Traits
	Alive  bool

	// Signal is the value this agent currently broadcasts to nearby conspecifics
	// (set from the brain's signal output). pendingSignal holds the value computed
	// this tick; it is promoted to Signal only after every agent has sensed, so all
	// agents hear the previous tick's signals (a consistent, order-independent
	// broadcast).
	Signal        float64
	pendingSignal float64

	// ReproCooldown is a gestation/maturation timer: an agent can only
	// reproduce when it reaches zero. It prevents a predator from converting one
	// large meal into an instant litter, which is the other half of the fix for
	// runaway carnivore booms.
	ReproCooldown int

	// Cached I/O from the most recent tick, surfaced by the inspector panel.
	// LastMemory is the brain's recurrent state (previous hidden activations).
	LastInputs  []float64
	LastOutputs []float64
	LastMemory  []float64

	// In-lifetime learning state. rewardBaseline is a running average of recent
	// reward; the learning rule is driven by the advantage (reward - baseline) so
	// routine ticks don't bias the weights. LastReward is the latest advantage,
	// surfaced by the inspector.
	rewardBaseline float64
	LastReward     float64

	// Curiosity / world-model state. worldModel predicts the next senses from the
	// current senses+action; the prediction error (LastSurprise) is the intrinsic
	// reward. lastFeatures holds the features whose prediction is checked next tick.
	worldModel   *neural.Predictor
	lastFeatures []float64
	featBuf      []float64
	LastSurprise float64
}

// learn turns this tick's reward into a reward-modulated Hebbian update of the
// agent's brain. The reward combines an extrinsic term (energy gained/lost) with
// an intrinsic curiosity term (how surprising the world was, scaled by the evolved
// Curiosity trait). Learning strength is the evolved Plasticity trait; with
// Plasticity 0 this is a no-op.
func (a *Agent) learn(w *World, deltaEnergy float64) {
	extrinsic := math.Tanh(deltaEnergy * w.params.RewardScale)
	intrinsic := a.Traits.Curiosity * math.Tanh(a.LastSurprise*w.params.CuriosityGain)
	total := extrinsic + intrinsic

	adv := total - a.rewardBaseline
	a.rewardBaseline += w.params.BaselineLR * (total - a.rewardBaseline)
	a.LastReward = adv
	a.Brain.Learn(adv, a.Traits.Plasticity)
}

// sense builds the brain input vector by casting the agent's vision over the
// world: every visible target and threat within the sense radius is binned into
// the directional sector it falls in, keeping the nearest (highest proximity) per
// sector. Candidates come from the spatial grid, so the scan cost depends on local
// density rather than total population. See the input-layout comment above.
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

// see records the proximity of point p (scaled by weight) into whichever vision
// sector it lies in, relative to the agent's heading, keeping the maximum per
// sector. radius is the effective sense radius this tick; points beyond it are
// ignored.
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

// hear records the signal broadcast by conspecific o into the voice channel,
// keeping the nearest neighbour's signal per sector. radius is the effective sense
// radius this tick.
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

// sectorOf returns the vision sector that the delta vector d falls in, relative to
// the agent's heading.
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

// think runs the brain on the current senses and caches the I/O for inspection.
// The brain is recurrent, so each call also advances its internal memory. It also
// drives the curiosity world-model: the new senses are scored against last tick's
// prediction (the surprise / intrinsic reward) and the model is trained, then a
// fresh prediction is staged from the current senses and chosen action.
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

// featureVec packs the senses and action into the world-model's input buffer
// (reused across ticks to avoid per-tick allocation).
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

// act applies the brain outputs: it turns, moves (paying energy), and reports
// whether the agent wants to eat this tick.
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
