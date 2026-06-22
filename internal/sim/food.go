package sim

import "github.com/danielriddell21/vivarium/internal/geom"

// Food is a stationary plant patch. It holds a pool of energy that regrows over
// time toward a cap, modelling a regenerating food source; herbivores deplete it
// by eating. Its dynamics (regrowth, ripeness) live on World so they can be tuned
// via Params — see regrowFood / foodRipe.
// NumFoodTypes is how many distinct plant types exist. Herbivores evolve a Diet
// trait selecting which they digest efficiently, enabling dietary niches.
const NumFoodTypes = 2

type Food struct {
	Pos    geom.Vec2
	Energy float64
	Type   int // 0..NumFoodTypes-1
}
