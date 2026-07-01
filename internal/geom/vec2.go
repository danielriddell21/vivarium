package geom

import "math"

type Vec2 struct {
	X, Y float64
}

func (v Vec2) Add(o Vec2) Vec2 { return Vec2{v.X + o.X, v.Y + o.Y} }

func (v Vec2) Sub(o Vec2) Vec2 { return Vec2{v.X - o.X, v.Y - o.Y} }

func (v Vec2) Scale(s float64) Vec2 { return Vec2{v.X * s, v.Y * s} }

func (v Vec2) Len() float64 { return math.Hypot(v.X, v.Y) }

func (v Vec2) Normalize() Vec2 {
	l := v.Len()
	if l == 0 {
		return v
	}
	return Vec2{v.X / l, v.Y / l}
}

func (v Vec2) Angle() float64 { return math.Atan2(v.Y, v.X) }

func FromAngle(a float64) Vec2 { return Vec2{math.Cos(a), math.Sin(a)} }

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

func (v Vec2) ShortestDelta(o Vec2, w, h float64) Vec2 {
	dx := wrapDelta(o.X-v.X, w)
	dy := wrapDelta(o.Y-v.Y, h)
	return Vec2{dx, dy}
}

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
