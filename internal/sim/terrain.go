package sim

import "github.com/danielriddell21/crucible/geom"

const (
	obstacleMinR = 14.0
	obstacleMaxR = 40.0
)

type Obstacle struct {
	Pos    geom.Vec2
	Radius float64
}

func (w *World) Obstacles() []Obstacle { return w.obstacles }

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
