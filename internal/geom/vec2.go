// Package geom provides minimal 2D vector math for the simulation, including
// helpers for a toroidal (wrap-around) world where the left edge connects to the
// right and the top to the bottom.
package geom

import "math"

// Vec2 is a 2D vector / point.
type Vec2 struct {
	X, Y float64
}

// Add returns v + o.
func (v Vec2) Add(o Vec2) Vec2 { return Vec2{v.X + o.X, v.Y + o.Y} }

// Sub returns v - o.
func (v Vec2) Sub(o Vec2) Vec2 { return Vec2{v.X - o.X, v.Y - o.Y} }

// Scale returns v scaled by s.
func (v Vec2) Scale(s float64) Vec2 { return Vec2{v.X * s, v.Y * s} }

// Len returns the Euclidean length of v.
func (v Vec2) Len() float64 { return math.Hypot(v.X, v.Y) }

// Normalize returns a unit vector in the direction of v. The zero vector is
// returned unchanged.
func (v Vec2) Normalize() Vec2 {
	l := v.Len()
	if l == 0 {
		return v
	}
	return Vec2{v.X / l, v.Y / l}
}

// Angle returns the heading of v in radians, in the range (-pi, pi].
func (v Vec2) Angle() float64 { return math.Atan2(v.Y, v.X) }

// FromAngle returns a unit vector pointing along the given angle (radians).
func FromAngle(a float64) Vec2 { return Vec2{math.Cos(a), math.Sin(a)} }

// WrapTo wraps v into the rectangle [0,w) x [0,h) using modular arithmetic, so
// that positions leaving one edge re-enter on the opposite edge.
func (v Vec2) WrapTo(w, h float64) Vec2 {
	x := math.Mod(v.X, w)
	if x < 0 {
		x += w
	}
	y := math.Mod(v.Y, h)
	if y < 0 {
		y += h
	}
	return Vec2{x, y}
}

// ShortestDelta returns the shortest vector from v to o on a toroidal world of
// size w x h. Each component is wrapped to the half-open range [-w/2, w/2) (and
// likewise for h) so that distances respect the wrap-around topology.
func (v Vec2) ShortestDelta(o Vec2, w, h float64) Vec2 {
	dx := wrapDelta(o.X-v.X, w)
	dy := wrapDelta(o.Y-v.Y, h)
	return Vec2{dx, dy}
}

// ToroidalDist returns the shortest distance between v and o on a w x h torus.
func (v Vec2) ToroidalDist(o Vec2, w, h float64) float64 {
	return v.ShortestDelta(o, w, h).Len()
}

func wrapDelta(d, size float64) float64 {
	d = math.Mod(d, size)
	if d < -size/2 {
		d += size
	} else if d >= size/2 {
		d -= size
	}
	return d
}
