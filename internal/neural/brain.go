// Package neural implements a tiny hand-rolled recurrent neural network used as
// an agent's "brain". Behaviour is shaped by two processes:
//
//   - Evolution (between generations): offspring inherit the parent's GENOME
//     weights with small Gaussian mutations. There is no backpropagation.
//   - In-lifetime learning (within a generation): a brain adapts a working copy
//     of its weights from a scalar reward via a reward-modulated Hebbian rule.
//
// The two are kept separate (Baldwinian, not Lamarckian): only the genome is
// inherited, so what an individual learns during its life is NOT passed to its
// offspring — they re-grow the same plastic phenotype from the inherited genome
// and must learn again. This lets learning and evolution be studied independently.
//
// The implementation deliberately uses flat float64 slices and a few loops rather
// than any external ML/matrix library.
package neural

import (
	"math"
	"math/rand"
)

// weightCap bounds every working weight so the unsupervised Hebbian rule cannot
// run away to infinity over a long life.
const weightCap = 8.0

// weights is one full set of network parameters in flat row-major slices:
//
//	WIH has Hidden*In entries:     WIH[h*In + i]     connects input i  -> hidden h.
//	WCH has Hidden*Hidden entries: WCH[h*Hidden + c] connects context c -> hidden h.
//	WHO has Out*Hidden entries:    WHO[o*Hidden + h] connects hidden h -> output o.
type weights struct {
	WIH, WCH, BH, WHO, BO []float64
}

func newWeights(in, hidden, out int) weights {
	return weights{
		WIH: make([]float64, hidden*in),
		WCH: make([]float64, hidden*hidden),
		BH:  make([]float64, hidden),
		WHO: make([]float64, out*hidden),
		BO:  make([]float64, out),
	}
}

func (w weights) clone() weights {
	return weights{
		WIH: append([]float64(nil), w.WIH...),
		WCH: append([]float64(nil), w.WCH...),
		BH:  append([]float64(nil), w.BH...),
		WHO: append([]float64(nil), w.WHO...),
		BO:  append([]float64(nil), w.BO...),
	}
}

// Brain is a single-hidden-layer Elman recurrent network with tanh activations on
// both the hidden and output layers, so every output is bounded in (-1, 1). The
// previous tick's hidden activations are fed back into the hidden layer (the
// recurrent context, or memory), giving agents a one-step memory.
type Brain struct {
	In, Hidden, Out int

	genome weights // heritable initial weights; never modified after birth
	live   weights // working weights: used by Forward, adapted by Learn

	state []float64 // previous hidden activations (the recurrent context)

	// Activations cached from the most recent Forward, used by Learn.
	lastIn      []float64
	lastContext []float64
	lastHidden  []float64
	lastOut     []float64
}

// New returns a Brain with the given layer sizes and small random genome weights
// drawn from rng. The live weights start as a copy of the genome and the recurrent
// state starts blank.
func New(rng *rand.Rand, in, hidden, out int) *Brain {
	g := newWeights(in, hidden, out)
	for i := range g.WIH {
		g.WIH[i] = rng.NormFloat64()
	}
	for i := range g.WCH {
		g.WCH[i] = rng.NormFloat64()
	}
	for i := range g.WHO {
		g.WHO[i] = rng.NormFloat64()
	}
	// Biases start at zero; mutation will move them as needed.
	return &Brain{
		In: in, Hidden: hidden, Out: out,
		genome: g,
		live:   g.clone(),
		state:  make([]float64, hidden),
	}
}

// Forward runs the network on inputs for one tick using the live weights, returns
// a freshly allocated slice of Out activations, and advances the recurrent state.
// It caches the activations so a subsequent Learn call can adapt the weights. If
// len(inputs) != In the input is treated as zero-padded or truncated.
func (b *Brain) Forward(inputs []float64) []float64 {
	// Cache the (padded) inputs and the context that feeds this pass.
	in := resize(b.lastIn, b.In)
	for i := range in {
		if i < len(inputs) {
			in[i] = inputs[i]
		} else {
			in[i] = 0
		}
	}
	b.lastIn = in
	b.lastContext = appendTo(b.lastContext, b.state)

	hidden := make([]float64, b.Hidden)
	for h := 0; h < b.Hidden; h++ {
		sum := b.live.BH[h]
		ibase := h * b.In
		for i := 0; i < b.In; i++ {
			sum += b.live.WIH[ibase+i] * in[i]
		}
		cbase := h * b.Hidden
		for c := 0; c < b.Hidden; c++ {
			sum += b.live.WCH[cbase+c] * b.state[c]
		}
		hidden[h] = math.Tanh(sum)
	}
	copy(b.state, hidden) // the new hidden activations become next tick's context

	out := make([]float64, b.Out)
	for o := 0; o < b.Out; o++ {
		sum := b.live.BO[o]
		base := o * b.Hidden
		for h := 0; h < b.Hidden; h++ {
			sum += b.live.WHO[base+h] * hidden[h]
		}
		out[o] = math.Tanh(sum)
	}

	b.lastHidden = hidden
	b.lastOut = out
	return out
}

// Learn applies one reward-modulated Hebbian update to the live weights using the
// activations cached by the most recent Forward. The signed modulator scales the
// update: a connection between two co-active units is strengthened when the
// modulator is positive and weakened when negative — so behaviour that preceded
// reward is reinforced. rate is the per-agent learning rate (0 disables learning).
// Weights are clamped to ±weightCap.
func (b *Brain) Learn(modulator, rate float64) {
	if rate == 0 || modulator == 0 || b.lastHidden == nil {
		return
	}
	k := rate * modulator

	for h := 0; h < b.Hidden; h++ {
		post := b.lastHidden[h]
		ibase := h * b.In
		for i := 0; i < b.In; i++ {
			b.live.WIH[ibase+i] = clampW(b.live.WIH[ibase+i] + k*b.lastIn[i]*post)
		}
		cbase := h * b.Hidden
		for c := 0; c < b.Hidden; c++ {
			b.live.WCH[cbase+c] = clampW(b.live.WCH[cbase+c] + k*b.lastContext[c]*post)
		}
		b.live.BH[h] = clampW(b.live.BH[h] + k*post)
	}
	for o := 0; o < b.Out; o++ {
		post := b.lastOut[o]
		base := o * b.Hidden
		for h := 0; h < b.Hidden; h++ {
			b.live.WHO[base+h] = clampW(b.live.WHO[base+h] + k*b.lastHidden[h]*post)
		}
		b.live.BO[o] = clampW(b.live.BO[o] + k*post)
	}
}

// Reset clears the recurrent memory back to a blank slate.
func (b *Brain) Reset() {
	for i := range b.state {
		b.state[i] = 0
	}
}

// State returns a copy of the current recurrent memory (the previous hidden
// activations), for inspection/visualisation.
func (b *Brain) State() []float64 {
	return append([]float64(nil), b.state...)
}

// Genome returns a flat copy of all heritable weights (the brain's identity in
// genome space), in a fixed order. Brains with the same layer sizes return
// vectors of the same length, so they can be compared/clustered.
func (b *Brain) Genome() []float64 {
	g := make([]float64, 0, len(b.genome.WIH)+len(b.genome.WCH)+len(b.genome.BH)+len(b.genome.WHO)+len(b.genome.BO))
	g = append(g, b.genome.WIH...)
	g = append(g, b.genome.WCH...)
	g = append(g, b.genome.BH...)
	g = append(g, b.genome.WHO...)
	g = append(g, b.genome.BO...)
	return g
}

// FromGenome rebuilds a brain of the given layer sizes from a flat genome produced
// by Genome (same field order). The live weights start as a copy of the genome and
// the memory is blank. It returns nil if the genome length does not match the
// requested dimensions.
func FromGenome(in, hidden, out int, genome []float64) *Brain {
	g := newWeights(in, hidden, out)
	need := len(g.WIH) + len(g.WCH) + len(g.BH) + len(g.WHO) + len(g.BO)
	if len(genome) != need {
		return nil
	}
	off := 0
	take := func(dst []float64) {
		copy(dst, genome[off:off+len(dst)])
		off += len(dst)
	}
	take(g.WIH)
	take(g.WCH)
	take(g.BH)
	take(g.WHO)
	take(g.BO)
	return &Brain{
		In: in, Hidden: hidden, Out: out,
		genome: g,
		live:   g.clone(),
		state:  make([]float64, hidden),
	}
}

// LearnedDrift reports the mean absolute difference between the live weights and
// the genome — i.e. how far in-lifetime learning has moved the phenotype away from
// the inherited starting point. It is 0 at birth and for non-plastic agents.
func (b *Brain) LearnedDrift() float64 {
	var sum float64
	var n int
	drift := func(live, gen []float64) {
		for i := range live {
			sum += math.Abs(live[i] - gen[i])
		}
		n += len(live)
	}
	drift(b.live.WIH, b.genome.WIH)
	drift(b.live.WCH, b.genome.WCH)
	drift(b.live.BH, b.genome.BH)
	drift(b.live.WHO, b.genome.WHO)
	drift(b.live.BO, b.genome.BO)
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

// Clone returns a new brain that inherits this brain's GENOME (its birth weights,
// unaffected by any learning). The offspring's live weights start as a copy of
// that genome and its memory is blank — nothing learned in life is inherited.
func (b *Brain) Clone() *Brain {
	g := b.genome.clone()
	return &Brain{
		In: b.In, Hidden: b.Hidden, Out: b.Out,
		genome: g,
		live:   g.clone(),
		state:  make([]float64, b.Hidden),
	}
}

// CrossoverWith returns a child brain whose GENOME is a uniform (per-weight) mix
// of this brain's and other's genomes — each weight inherited at random from one
// parent. The child's live weights start as a copy of that genome and its memory
// is blank. Both parents must share the same layer sizes.
func (b *Brain) CrossoverWith(rng *rand.Rand, other *Brain) *Brain {
	g := newWeights(b.In, b.Hidden, b.Out)
	mix := func(dst, x, y []float64) {
		for i := range dst {
			if rng.Float64() < 0.5 {
				dst[i] = x[i]
			} else {
				dst[i] = y[i]
			}
		}
	}
	mix(g.WIH, b.genome.WIH, other.genome.WIH)
	mix(g.WCH, b.genome.WCH, other.genome.WCH)
	mix(g.BH, b.genome.BH, other.genome.BH)
	mix(g.WHO, b.genome.WHO, other.genome.WHO)
	mix(g.BO, b.genome.BO, other.genome.BO)
	return &Brain{
		In: b.In, Hidden: b.Hidden, Out: b.Out,
		genome: g,
		live:   g.clone(),
		state:  make([]float64, b.Hidden),
	}
}

// Mutate perturbs the genome in place (each weight, with probability rate, nudged
// by Gaussian noise scaled by std) and resets the live weights to match. This is
// the sole mechanism by which inherited behaviour changes between generations.
func (b *Brain) Mutate(rng *rand.Rand, rate, std float64) {
	mut := func(s []float64) {
		for i := range s {
			if rng.Float64() < rate {
				s[i] += rng.NormFloat64() * std
			}
		}
	}
	mut(b.genome.WIH)
	mut(b.genome.WCH)
	mut(b.genome.BH)
	mut(b.genome.WHO)
	mut(b.genome.BO)
	b.live = b.genome.clone()
}

func clampW(v float64) float64 {
	if v > weightCap {
		return weightCap
	}
	if v < -weightCap {
		return -weightCap
	}
	return v
}

// resize returns a slice of length n, reusing s's backing array when possible.
func resize(s []float64, n int) []float64 {
	if cap(s) >= n {
		return s[:n]
	}
	return make([]float64, n)
}

// appendTo copies src into a reused dst slice.
func appendTo(dst, src []float64) []float64 {
	dst = resize(dst, len(src))
	copy(dst, src)
	return dst
}
