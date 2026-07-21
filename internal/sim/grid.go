package sim

import (
	"math"

	"github.com/danielriddell21/crucible/geom"
)

const gridCellSize = 80.0

type spatialGrid struct {
	w, h       float64
	cellSize   float64
	cols, rows int

	foodCells  [][]*Food
	agentCells [][]*Agent
}

func newSpatialGrid(w, h, cellSize float64) *spatialGrid {
	cols := max(1, int(math.Ceil(w/cellSize)))
	rows := max(1, int(math.Ceil(h/cellSize)))
	return &spatialGrid{
		w: w, h: h, cellSize: cellSize, cols: cols, rows: rows,
		foodCells:  make([][]*Food, cols*rows),
		agentCells: make([][]*Agent, cols*rows),
	}
}

func (g *spatialGrid) cellIndex(p geom.Vec2) int {
	col := clamp(int(p.X/g.cellSize), 0, g.cols-1)
	row := clamp(int(p.Y/g.cellSize), 0, g.rows-1)
	return row*g.cols + col
}

func (g *spatialGrid) rebuild(foods []*Food, agents []*Agent) {
	for i := range g.foodCells {
		g.foodCells[i] = g.foodCells[i][:0]
	}
	for i := range g.agentCells {
		g.agentCells[i] = g.agentCells[i][:0]
	}
	for _, f := range foods {
		idx := g.cellIndex(f.Pos)
		g.foodCells[idx] = append(g.foodCells[idx], f)
	}
	for _, a := range agents {
		if !a.Alive {
			continue
		}
		idx := g.cellIndex(a.Pos)
		g.agentCells[idx] = append(g.agentCells[idx], a)
	}
}

func (g *spatialGrid) forEachCellNear(p geom.Vec2, radius float64, fn func(idx int)) {
	cmin := int(math.Floor((p.X - radius) / g.cellSize))
	cmax := int(math.Floor((p.X + radius) / g.cellSize))
	rmin := int(math.Floor((p.Y - radius) / g.cellSize))
	rmax := int(math.Floor((p.Y + radius) / g.cellSize))

	// Cap the span at the grid size so a wide query visits each column/row once.
	ncols := min(cmax-cmin+1, g.cols)
	nrows := min(rmax-rmin+1, g.rows)

	for dr := 0; dr < nrows; dr++ {
		row := mod(rmin+dr, g.rows)
		for dc := 0; dc < ncols; dc++ {
			col := mod(cmin+dc, g.cols)
			fn(row*g.cols + col)
		}
	}
}

func (g *spatialGrid) forEachFoodNear(p geom.Vec2, radius float64, fn func(*Food)) {
	g.forEachCellNear(p, radius, func(idx int) {
		for _, f := range g.foodCells[idx] {
			fn(f)
		}
	})
}

func (g *spatialGrid) forEachAgentNear(p geom.Vec2, radius float64, fn func(*Agent)) {
	g.forEachCellNear(p, radius, func(idx int) {
		for _, a := range g.agentCells[idx] {
			fn(a)
		}
	})
}

func mod(x, n int) int {
	x %= n
	if x < 0 {
		x += n
	}
	return x
}
