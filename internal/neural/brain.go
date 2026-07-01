package neural

import (
	"math"
	"math/rand"
	"slices"
)

const weightCap = 8.0

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
		WIH: slices.Clone(w.WIH),
		WCH: slices.Clone(w.WCH),
		BH:  slices.Clone(w.BH),
		WHO: slices.Clone(w.WHO),
		BO:  slices.Clone(w.BO),
	}
}

type Brain struct {
	In, Hidden, Out int

	genome weights
	live   weights

	state []float64

	lastIn      []float64
	lastContext []float64
	lastHidden  []float64
	lastOut     []float64
}

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

func (b *Brain) Reset() {
	for i := range b.state {
		b.state[i] = 0
	}
}

func (b *Brain) State() []float64 {
	return append([]float64(nil), b.state...)
}

func (b *Brain) Genome() []float64 {
	g := make([]float64, 0, len(b.genome.WIH)+len(b.genome.WCH)+len(b.genome.BH)+len(b.genome.WHO)+len(b.genome.BO))
	g = append(g, b.genome.WIH...)
	g = append(g, b.genome.WCH...)
	g = append(g, b.genome.BH...)
	g = append(g, b.genome.WHO...)
	g = append(g, b.genome.BO...)
	return g
}

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

func (b *Brain) Clone() *Brain {
	g := b.genome.clone()
	return &Brain{
		In: b.In, Hidden: b.Hidden, Out: b.Out,
		genome: g,
		live:   g.clone(),
		state:  make([]float64, b.Hidden),
	}
}

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

func resize(s []float64, n int) []float64 {
	if cap(s) >= n {
		return s[:n]
	}
	return make([]float64, n)
}

func appendTo(dst, src []float64) []float64 {
	dst = resize(dst, len(src))
	copy(dst, src)
	return dst
}
