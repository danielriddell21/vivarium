//go:build ebiten

package render

import (
	"fmt"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/danielriddell21/vivarium/internal/sim"
)

const maxPhyloLeaves = 60 // living agents sampled as leaves of the drawn tree

type phyloPos struct {
	x, y    float64 // normalised 0..1 (x = time, y = layout order)
	lineage int
}

// phyloView is a laid-out coalescent tree ready to draw.
type phyloView struct {
	pos     map[int]phyloPos
	edges   [][2]int // parentID, childID (both present in pos)
	leafIDs []int
	leaves  int
	living  int
}

// computePhylogeny builds the genealogy of a sample of the living population: it
// traces each sampled agent back through retained ancestors, then lays the induced
// tree out with time on the x-axis. Read-only; no RNG.
// samplePhyloLeaves returns the living agents that have a genealogy node,
// sorted and thinned with a stride to at most maxPhyloLeaves, plus the total
// living count before thinning.
func samplePhyloLeaves(w *sim.World, gen map[int]*sim.GenealogyNode) (leaves []int, living int) {
	for _, a := range w.Agents {
		if a.Alive {
			if _, ok := gen[a.ID]; ok {
				living++
				leaves = append(leaves, a.ID)
			}
		}
	}
	sort.Ints(leaves) // deterministic sampling/layout
	if len(leaves) > maxPhyloLeaves {
		stride := (len(leaves) + maxPhyloLeaves - 1) / maxPhyloLeaves
		var s []int
		for i := 0; i < len(leaves); i += stride {
			s = append(s, leaves[i])
		}
		leaves = s
	}
	return leaves, living
}

// inducedAncestors returns the union of the root-ward paths from each leaf.
func inducedAncestors(gen map[int]*sim.GenealogyNode, leaves []int) map[int]bool {
	induced := make(map[int]bool)
	for _, leaf := range leaves {
		for id := leaf; id != 0; {
			n := gen[id]
			if n == nil || induced[id] {
				break
			}
			induced[id] = true
			id = n.ParentID
		}
	}
	return induced
}

// buildInducedTree groups the induced nodes into parent->children and roots,
// sorts each for deterministic layout, and returns the birth-tick span.
func buildInducedTree(gen map[int]*sim.GenealogyNode, induced map[int]bool) (children map[int][]int, roots []int, minTick, maxTick int) {
	children = make(map[int][]int)
	minTick, maxTick = 1<<62, -(1 << 62)
	for id := range induced {
		n := gen[id]
		if n.BirthTick < minTick {
			minTick = n.BirthTick
		}
		if n.BirthTick > maxTick {
			maxTick = n.BirthTick
		}
		if induced[n.ParentID] {
			children[n.ParentID] = append(children[n.ParentID], id)
		} else {
			roots = append(roots, id)
		}
	}
	for _, ch := range children {
		sort.Ints(ch)
	}
	sort.Ints(roots)
	return children, roots, minTick, maxTick
}

func computePhylogeny(w *sim.World) *phyloView {
	gen := w.Genealogy()
	leaves, living := samplePhyloLeaves(w, gen)
	induced := inducedAncestors(gen, leaves)
	children, roots, minTick, maxTick := buildInducedTree(gen, induced)

	pv := &phyloView{pos: make(map[int]phyloPos, len(induced)), living: living}
	span := float64(maxTick - minTick)
	if span == 0 {
		span = 1
	}
	var nextLeaf float64
	// Post-order layout: leaves get successive rows; parents centre on children.
	var place func(id int) float64
	place = func(id int) float64 {
		ch := children[id]
		var y float64
		if len(ch) == 0 {
			y = nextLeaf
			nextLeaf++
			pv.leaves++
			pv.leafIDs = append(pv.leafIDs, id)
		} else {
			var sum float64
			for _, c := range ch {
				sum += place(c)
				pv.edges = append(pv.edges, [2]int{id, c})
			}
			y = sum / float64(len(ch))
		}
		pv.pos[id] = phyloPos{
			x:       float64(gen[id].BirthTick-minTick) / span,
			y:       y,
			lineage: gen[id].LineageID,
		}
		return y
	}
	for _, r := range roots {
		place(r)
	}
	// Normalise y to 0..1.
	if pv.leaves > 1 {
		for id, p := range pv.pos {
			p.y /= float64(pv.leaves - 1)
			pv.pos[id] = p
		}
	}
	return pv
}

// drawPhylogenyPanel draws the coalescent tree as a rectangular dendrogram with
// time increasing to the right, branches coloured by lineage.
func (g *Game) drawPhylogenyPanel(screen *ebiten.Image) {
	pv := g.phylo
	px := g.World.W * 0.10
	py := 64.0
	pw := g.World.W * 0.80
	ph := g.World.H - 130

	drawPanel(screen, px, py, pw, ph)
	drawText(screen, "phylogeny - coalescent tree of living population (older left, now right)", px+8, py+4, colText)

	if pv == nil || pv.leaves < 2 {
		drawText(screen, "building genealogy...", px+8, py+22, colText)
		return
	}

	plotX, plotY := px+12, py+26
	plotW, plotH := pw-24, ph-40
	for _, e := range pv.edges {
		p := pv.pos[e[0]]
		c := pv.pos[e[1]]
		x1 := float32(plotX + p.x*plotW)
		x2 := float32(plotX + c.x*plotW)
		y1 := float32(plotY + p.y*plotH)
		y2 := float32(plotY + c.y*plotH)
		clr := lineagePalette[c.lineage%len(lineagePalette)]
		// Rectangular elbow: vertical at the parent's time, then horizontal to child.
		vector.StrokeLine(screen, x1, y1, x1, y2, 1, clr, false)
		vector.StrokeLine(screen, x1, y2, x2, y2, 1, clr, false)
	}
	// Mark the living leaves.
	for _, id := range pv.leafIDs {
		p := pv.pos[id]
		clr := lineagePalette[p.lineage%len(lineagePalette)]
		vector.FillCircle(screen, float32(plotX+p.x*plotW), float32(plotY+p.y*plotH), 2, clr, false)
	}

	drawText(screen, fmt.Sprintf("%d leaves shown of %d living; %d nodes", pv.leaves, pv.living, len(pv.pos)), px+8, py+ph-16, colText)
}
