package sim

import "github.com/danielriddell21/vivarium/internal/geom"

// Food is a stationary plant patch. It holds a pool of energy that regrows over
// time toward a cap, modelling a regenerating food source; herbivores deplete it
// by eating. Its dynamics (regrowth, ripeness) live on World so they can be tuned
// via Params — see regrowFood / foodRipe.
type Food struct {
	Pos    geom.Vec2
	Energy float64
}
