package gui

import (
	"sort"

	"github.com/danielriddell21/vivarium/internal/sim"
)

const maxPhyloLeaves = 60

type phyloPos struct {
	x, y    float64
	lineage int
}

type phyloView struct {
	pos     map[int]phyloPos
	edges   [][2]int
	leafIDs []int
	leaves  int
	living  int
}

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
