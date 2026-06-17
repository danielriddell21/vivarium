package sim

import (
	"math"

	"github.com/danielriddell21/vivarium/internal/geom"
)

// gridCellSize is the side length of a spatial bucket, in world units. It is a
// balance: large enough that a typical sense query touches only a handful of
// cells, small enough that each cell holds few entities.
const gridCellSize = 80.0

// spatialGrid buckets foods and agents by cell so that neighbour queries cost
// roughly O(local density) instead of O(total population). The grid is rebuilt
// once per tick from the authoritative entity slices, so bucket contents are in a
// deterministic order (slice order) — important for reproducible runs.
//
// Queries are toroidal: the cell window wraps around the world edges, matching the
// wrap-around topology used everywhere else.
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

// cellIndex maps a position to its flat bucket index.
func (g *spatialGrid) cellIndex(p geom.Vec2) int {
	col := clampInt(int(p.X/g.cellSize), 0, g.cols-1)
	row := clampInt(int(p.Y/g.cellSize), 0, g.rows-1)
	return row*g.cols + col
}

// rebuild clears the grid and re-buckets the given entities. Dead agents are
// skipped. Backing slices are reused to avoid per-tick allocation.
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

// forEachCellNear visits, exactly once each, the cells of the toroidal window that
// covers the square [p-radius, p+radius]. Cells are visited in a fixed (row, col)
// order so callers see a deterministic sequence.
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

// forEachFoodNear calls fn for every food in cells overlapping the query window.
// Candidates may lie slightly outside radius (cells are square); callers that need
// an exact radius must re-check distance.
func (g *spatialGrid) forEachFoodNear(p geom.Vec2, radius float64, fn func(*Food)) {
	g.forEachCellNear(p, radius, func(idx int) {
		for _, f := range g.foodCells[idx] {
			fn(f)
		}
	})
}

// forEachAgentNear calls fn for every agent in cells overlapping the query window.
func (g *spatialGrid) forEachAgentNear(p geom.Vec2, radius float64, fn func(*Agent)) {
	g.forEachCellNear(p, radius, func(idx int) {
		for _, a := range g.agentCells[idx] {
			fn(a)
		}
	})
}

// mod returns a value in [0, n) for any integer x (Go's % keeps the sign).
func mod(x, n int) int {
	x %= n
	if x < 0 {
		x += n
	}
	return x
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
