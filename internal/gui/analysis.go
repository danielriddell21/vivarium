//go:build ebiten

package gui

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/danielriddell21/vivarium/internal/analytics"
	"github.com/danielriddell21/vivarium/internal/sim"
)

const (
	numClusters           = 5   // emergent "species" to look for
	analysisRefreshFrames = 20  // recompute the clustering every N frames
	analysisSampleCap     = 500 // cap genomes fed to k-means/PCA per refresh
	kmeansIters           = 25
)

// clusterPalette colours each discovered species. Index by cluster number.
var clusterPalette = []color.RGBA{
	{0xe6, 0x55, 0x55, 0xff}, // red
	{0x55, 0xc0, 0xe6, 0xff}, // cyan
	{0x9b, 0xe0, 0x55, 0xff}, // green
	{0xe0, 0xa0, 0x40, 0xff}, // orange
	{0xc0, 0x7c, 0xe6, 0xff}, // purple
}

func clusterColor(i int) color.RGBA { return clusterPalette[i%len(clusterPalette)] }

// analysis holds the result of one clustering/projection pass over the population.
type analysis struct {
	k                      int
	sizes                  []int
	clusterOf              map[*sim.Agent]int
	coords                 map[*sim.Agent][2]float64
	centroids              [][]float64 // kept to warm-start the next pass (stable labels)
	pcaOK                  bool
	minX, maxX, minY, maxY float64
	sampled, total         int
}

// computeAnalysis clusters the living population in brain-genome space and projects
// it to 2D. It is read-only over the simulation and uses its own RNG, so it never
// affects the run's determinism. prev is the previous result (or nil); its
// centroids warm-start k-means so cluster labels — and therefore the colours —
// stay attached to the same groups across refreshes instead of flashing.
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

// drawAnalysisPanel renders a bottom-right panel with the cluster legend and a 2D
// PCA scatter of brain-genome space, coloured by species.
func (g *Game) drawAnalysisPanel(screen *ebiten.Image) {
	a := g.analysis
	const pw, ph = 232.0, 196.0
	px := g.World.W - pw - 6
	py := g.World.H - ph - 6
	drawPanel(screen, px, py, pw, ph)
	drawText(screen, "species (genome k-means + PCA)", px+6, py+4, colText)

	if a == nil || a.total < 2 {
		drawText(screen, "population too small", px+6, py+22, colText)
		return
	}
	drawText(screen, fmt.Sprintf("K=%d   n=%d/%d", a.k, a.sampled, a.total), px+6, py+20, colText)

	// Legend: a coloured swatch and size per cluster.
	ly := py + 36
	for c := 0; c < a.k; c++ {
		vector.FillRect(screen, float32(px+8), float32(ly), 8, 8, clusterColor(c), false)
		drawText(screen, fmt.Sprintf("sp%d  %d", c, a.sizes[c]), px+22, ly-3, colText)
		ly += 14
	}

	// PCA scatter to the right of the legend.
	plotX, plotY := px+110, py+34
	plotW, plotH := pw-116, ph-42
	vector.StrokeRect(screen, float32(plotX), float32(plotY), float32(plotW), float32(plotH), 1, colPanel, false)
	if !a.pcaOK {
		return
	}
	spanX := a.maxX - a.minX
	spanY := a.maxY - a.minY
	if spanX == 0 {
		spanX = 1
	}
	if spanY == 0 {
		spanY = 1
	}
	for agent, xy := range a.coords {
		fx := plotX + (xy[0]-a.minX)/spanX*plotW
		fy := plotY + plotH - (xy[1]-a.minY)/spanY*plotH
		vector.FillCircle(screen, float32(fx), float32(fy), 1.5, clusterColor(a.clusterOf[agent]), false)
	}
}

// agentBodyColor returns the colour to draw an agent's body in. The lineage and
// species views recolour agents; otherwise nil signals the default kind colour.
func (g *Game) agentBodyColor(a *sim.Agent) color.Color {
	if g.lineageOn && g.lineageView != nil {
		if ci, ok := g.lineageView.colorOf[a.LineageID]; ok {
			return lineagePalette[ci]
		}
		return colOther
	}
	if g.analysisOn && g.analysis != nil {
		if c, ok := g.analysis.clusterOf[a]; ok {
			return clusterColor(c)
		}
		return color.RGBA{0x70, 0x70, 0x70, 0xff} // sampled-out / newly born
	}
	return nil
}
