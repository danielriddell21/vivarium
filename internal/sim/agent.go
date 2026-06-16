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

// Neural network dimensions shared by every brain. Inputs:
//
//	0: own energy (normalised, ~0..1)
//	1: cos of angle to nearest target relative to heading
//	2: sin of angle to nearest target relative to heading
//	3: proximity of nearest target (0 = none/far, 1 = adjacent)
//	4: cos of angle to nearest threat relative to heading
//	5: sin of angle to nearest threat relative to heading
//	6: proximity of nearest threat (0 = none/far, 1 = adjacent)
//
// A "target" is what the agent eats (plants for herbivores, herbivores for
// carnivores); a "threat" is what eats it (carnivores for herbivores; none for
// carnivores). The proximity inputs double as the raycast-style distance sensors.
//
// Outputs:
//
//	0: turn   (-1..1, scaled to a max turn per tick)
//	1: speed  (-1..1, mapped to 0..MaxSpeed)
//	2: eat    (>0 means attempt to feed this tick)
const (
	BrainInputs  = 7
	BrainHidden  = 8
	BrainOutputs = 3
)

const (
	maxTurnPerTick = 0.45 // radians
	basalCost      = 0.06 // energy lost per tick just by being alive
	moveCost       = 0.04 // additional energy lost per unit of speed
)

// Agent is a mobile organism driven by an evolved neural brain.
type Agent struct {
	ID         int
	Kind       Kind
	Pos        geom.Vec2
	Heading    float64 // radians
	Energy     float64
	Age        int
	Generation int

	Brain  *neural.Brain
	Traits Traits
	Alive  bool

	// Cached I/O from the most recent tick, surfaced by the inspector panel.
	LastInputs  []float64
	LastOutputs []float64
}

// sense builds the brain input vector by scanning the world within the agent's
// sense radius for the nearest target and threat.
func (a *Agent) sense(w *World) []float64 {
	in := make([]float64, BrainInputs)
	in[0] = clamp(a.Energy/reproThreshold, 0, 1.5)

	var target, threat geom.Vec2
	var haveTarget, haveThreat bool
	var targetDist, threatDist float64

	switch a.Kind {
	case Herbivore:
		if f := w.nearestRipeFood(a.Pos, a.Traits.SenseRadius); f != nil {
			target, haveTarget = f.Pos, true
			targetDist = a.Pos.ToroidalDist(f.Pos, w.W, w.H)
		}
		if c := w.nearestAgentOfKind(a.Pos, Carnivore, a.Traits.SenseRadius, a.ID); c != nil {
			threat, haveThreat = c.Pos, true
			threatDist = a.Pos.ToroidalDist(c.Pos, w.W, w.H)
		}
	case Carnivore:
		if h := w.nearestAgentOfKind(a.Pos, Herbivore, a.Traits.SenseRadius, a.ID); h != nil {
			target, haveTarget = h.Pos, true
			targetDist = a.Pos.ToroidalDist(h.Pos, w.W, w.H)
		}
		// Carnivores are apex predators here: no threat channel.
	}

	if haveTarget {
		cos, sin, prox := a.relTo(w, target, targetDist)
		in[1], in[2], in[3] = cos, sin, prox
	}
	if haveThreat {
		cos, sin, prox := a.relTo(w, threat, threatDist)
		in[4], in[5], in[6] = cos, sin, prox
	}
	return in
}

// relTo returns the cosine and sine of the angle from the agent's heading to the
// point p, plus a 0..1 proximity value (1 when adjacent, 0 at the sense edge).
func (a *Agent) relTo(w *World, p geom.Vec2, dist float64) (cos, sin, prox float64) {
	d := a.Pos.ShortestDelta(p, w.W, w.H)
	rel := d.Angle() - a.Heading
	prox = 1 - dist/a.Traits.SenseRadius
	if prox < 0 {
		prox = 0
	}
	return math.Cos(rel), math.Sin(rel), prox
}

// think runs the brain on the current senses and caches the I/O for inspection.
func (a *Agent) think(w *World) []float64 {
	in := a.sense(w)
	out := a.Brain.Forward(in)
	a.LastInputs, a.LastOutputs = in, out
	return out
}

// act applies the brain outputs: it turns, moves (paying energy), and reports
// whether the agent wants to eat this tick.
func (a *Agent) act(w *World, out []float64) (wantsEat bool) {
	a.Heading += out[0] * maxTurnPerTick
	a.Heading = math.Mod(a.Heading, 2*math.Pi)

	speed := (out[1] + 1) / 2 * a.Traits.MaxSpeed
	if speed < 0 {
		speed = 0
	}
	a.Pos = a.Pos.Add(geom.FromAngle(a.Heading).Scale(speed)).WrapTo(w.W, w.H)

	a.Energy -= basalCost + moveCost*speed
	a.Age++
	return out[2] > 0
}
