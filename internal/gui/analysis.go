package gui

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/danielriddell21/vivarium/internal/analytics"
	"github.com/danielriddell21/vivarium/internal/sim"
)

const (
	numClusters           = 5
	analysisRefreshFrames = 20
	analysisSampleCap     = 500
	kmeansIters           = 25
)

var clusterPalette = []color.RGBA{
	{0xe6, 0x55, 0x55, 0xff},
	{0x55, 0xc0, 0xe6, 0xff},
	{0x9b, 0xe0, 0x55, 0xff},
	{0xe0, 0xa0, 0x40, 0xff},
	{0xc0, 0x7c, 0xe6, 0xff},
}

func clusterColor(i int) color.RGBA { return clusterPalette[i%len(clusterPalette)] }

type analysis struct {
	k                      int
	sizes                  []int
	clusterOf              map[*sim.Agent]int
	coords                 map[*sim.Agent][2]float64
	centroids              [][]float64
	pcaOK                  bool
	minX, maxX, minY, maxY float64
	sampled, total         int
}

func computeAnalysis(w *sim.World, prev *analysis) *analysis {
	// Collect living agents, sampling with a stride if the population is large.
	var agents []*sim.Agent
	for _, a := range w.Agents {
		if a.Alive {
			agents = append(agents, a)
		}
	}
	total := len(agents)
	if total < 2 {
		return &analysis{total: total}
	}
	stride := 1
	if total > analysisSampleCap {
		stride = (total + analysisSampleCap - 1) / analysisSampleCap
	}
	var sample []*sim.Agent
	feats := make([][]float64, 0, analysisSampleCap)
	for i := 0; i < total; i += stride {
		sample = append(sample, agents[i])
		feats = append(feats, agents[i].Brain.Genome())
	}

	rng := rand.New(rand.NewSource(1)) // local: does not touch the sim RNG
	k := numClusters
	if k > len(sample) {
		k = len(sample)
	}
	var warm [][]float64
	if prev != nil {
		warm = prev.centroids // reused only if it has the right shape (k x dim)
	}
	assign, centroids := analytics.KMeans(feats, k, kmeansIters, rng, warm)
	coords, ok := analytics.Project2D(feats, rand.New(rand.NewSource(2)))

	res := &analysis{
		k:         k,
		sizes:     make([]int, k),
		clusterOf: make(map[*sim.Agent]int, len(sample)),
		coords:    make(map[*sim.Agent][2]float64, len(sample)),
		centroids: centroids,
		pcaOK:     ok,
		total:     total,
		sampled:   len(sample),
		minX:      math.Inf(1), minY: math.Inf(1),
		maxX: math.Inf(-1), maxY: math.Inf(-1),
	}
	for i, a := range sample {
		res.clusterOf[a] = assign[i]
		res.sizes[assign[i]]++
		if ok {
			res.coords[a] = coords[i]
			res.minX, res.maxX = math.Min(res.minX, coords[i][0]), math.Max(res.maxX, coords[i][0])
			res.minY, res.maxY = math.Min(res.minY, coords[i][1]), math.Max(res.maxY, coords[i][1])
		}
	}
	return res
}

// clusterBodyColor recolours agents by the genome cluster they fell into, so
// the world matches the species panel's legend.
func clusterBodyColor(a *analysis) func(*sim.Agent) (color.RGBA, bool) {
	if a == nil {
		return nil
	}
	return func(ag *sim.Agent) (color.RGBA, bool) {
		if c, ok := a.clusterOf[ag]; ok {
			return clusterColor(c), true
		}
		return color.RGBA{R: 0x70, G: 0x70, B: 0x70, A: 0xff}, true // sampled-out / newly born
	}
}
