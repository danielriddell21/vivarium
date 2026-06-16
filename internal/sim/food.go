package sim

import "github.com/danielriddell21/vivarium/internal/geom"

// Food is a stationary plant patch. It holds a pool of energy that regrows over
// time toward a cap, modelling a regenerating food source. Herbivores deplete it
// by eating.
type Food struct {
	Pos    geom.Vec2
	Energy float64
}

const (
	foodMaxEnergy  = 60.0 // cap a single plant can hold
	foodRegrowRate = 0.18 // energy regained per tick
	foodEatRadius  = 8.0  // herbivore must be within size+this to feed
	foodBiteEnergy = 14.0 // energy transferred per successful bite
)

// regrow advances the plant's energy by one tick, never exceeding the cap.
func (f *Food) regrow() {
	if f.Energy < foodMaxEnergy {
		f.Energy += foodRegrowRate
		if f.Energy > foodMaxEnergy {
			f.Energy = foodMaxEnergy
		}
	}
}

// Ripe reports whether the plant currently has enough energy to be worth eating.
func (f *Food) Ripe() bool { return f.Energy >= foodBiteEnergy*0.5 }
