package sim

import "github.com/danielriddell21/vivarium/internal/geom"

// obstacle size range (world units).
const (
	obstacleMinR = 14.0
	obstacleMaxR = 40.0
)

// Obstacle is an impassable circular region of terrain (a rock/barrier). Agents
// cannot move into one, which carves the open plain into spaces to navigate.
type Obstacle struct {
	Pos    geom.Vec2
	Radius float64
}

// Obstacles returns the world's static terrain (read-only) for rendering.
func (w *World) Obstacles() []Obstacle { return w.obstacles }

// penetration returns how far a body of the given size at pos overlaps the most
// overlapping obstacle (0 if it is clear of all of them).
func (w *World) penetration(pos geom.Vec2, size float64) float64 {
	var maxPen float64
	for i := range w.obstacles {
		o := &w.obstacles[i]
		pen := (o.Radius + size) - pos.ToroidalDist(o.Pos, w.W, w.H)
		if pen > maxPen {
			maxPen = pen
		}
	}
	return maxPen
}

// resolveMove returns where an agent of the given size may move to: the proposed
// position if it does not push further into an obstacle, otherwise its current
// position. Allowing moves that *reduce* penetration means an agent that starts
// (or is pushed) inside terrain can always walk back out.
func (w *World) resolveMove(cur, proposed geom.Vec2, size float64) geom.Vec2 {
	if len(w.obstacles) == 0 {
		return proposed
	}
	newPen := w.penetration(proposed, size)
	if newPen > 0 && newPen >= w.penetration(cur, size) {
		return cur
	}
	return proposed
}

// occluded reports whether the line of sight from an observer to a target — given
// as the shortest delta vector from observer to target — is blocked by terrain.
// An obstacle blocks only if the segment passes through it strictly between the
// two endpoints, so terrain at or beyond the target (or behind the observer) does
// not occlude.
func (w *World) occluded(from geom.Vec2, delta geom.Vec2) bool {
	if len(w.obstacles) == 0 {
		return false
	}
	l2 := delta.X*delta.X + delta.Y*delta.Y
	if l2 == 0 {
		return false
	}
	for i := range w.obstacles {
		o := &w.obstacles[i]
		c := from.ShortestDelta(o.Pos, w.W, w.H) // obstacle centre relative to observer
		t := (c.X*delta.X + c.Y*delta.Y) / l2
		if t <= 0 || t >= 1 {
			continue // closest approach is not between observer and target
		}
		dx := delta.X*t - c.X
		dy := delta.Y*t - c.Y
		if dx*dx+dy*dy < o.Radius*o.Radius {
			return true
		}
	}
	return false
}
