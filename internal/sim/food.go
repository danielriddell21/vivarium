package sim

import "github.com/danielriddell21/crucible/geom"

const NumFoodTypes = 2

type Food struct {
	Pos    geom.Vec2
	Energy float64
	Type   int
}
