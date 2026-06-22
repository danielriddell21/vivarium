package sim

// Params holds the simulation's tunable scalars. They were previously hard-coded
// constants; collecting them here lets a run be configured from a JSON file (see
// cmd/vivarium's -config / -print-config) without recompiling. Morphological trait
// ranges remain compile-time, since those are the substrate evolution searches.
type Params struct {
	// Metabolism. Carnivores have ~3x the upkeep of herbivores, so when prey is
	// scarce they starve quickly — the self-limitation that damps boom-bust.
	HerbBasalCost float64 `json:"herbBasalCost"`
	CarnBasalCost float64 `json:"carnBasalCost"`
	MoveCost      float64 `json:"moveCost"`  // energy per unit of speed
	MaxEnergy     float64 `json:"maxEnergy"` // hard cap so meals aren't hoarded
	MaxTurn       float64 `json:"maxTurn"`   // max heading change per tick (radians)

	// Reproduction.
	ReproThreshold  float64 `json:"reproThreshold"`
	StartEnergyHerb float64 `json:"startEnergyHerb"`
	StartEnergyCarn float64 `json:"startEnergyCarn"`
	GestationHerb   int     `json:"gestationHerb"`
	GestationCarn   int     `json:"gestationCarn"`

	// Predation.
	CarnivoreGain  float64 `json:"carnivoreGain"`  // fraction of prey energy absorbed
	CarnivoreBonus float64 `json:"carnivoreBonus"` // flat bonus per kill

	// Evolution (brain weight mutation).
	MutationRate float64 `json:"mutationRate"`
	MutationStd  float64 `json:"mutationStd"`

	// Food / plants.
	FoodMaxEnergy  float64 `json:"foodMaxEnergy"`
	FoodRegrowRate float64 `json:"foodRegrowRate"`
	FoodEatRadius  float64 `json:"foodEatRadius"`
	FoodBiteEnergy float64 `json:"foodBiteEnergy"`

	// In-lifetime learning.
	RewardScale   float64 `json:"rewardScale"`
	BaselineLR    float64 `json:"baselineLR"`
	WorldModelLR  float64 `json:"worldModelLR"`
	CuriosityGain float64 `json:"curiosityGain"`

	// Rescue effect.
	RescueChance float64 `json:"rescueChance"`
}

// DefaultParams returns the balanced defaults the simulation was tuned with.
func DefaultParams() Params {
	return Params{
		HerbBasalCost: 0.06, CarnBasalCost: 0.20, MoveCost: 0.04, MaxEnergy: 200, MaxTurn: 0.45,
		ReproThreshold: 120, StartEnergyHerb: 70, StartEnergyCarn: 90,
		GestationHerb: 45, GestationCarn: 110,
		CarnivoreGain: 0.5, CarnivoreBonus: 8,
		MutationRate: 0.18, MutationStd: 0.35,
		FoodMaxEnergy: 60, FoodRegrowRate: 0.18, FoodEatRadius: 8, FoodBiteEnergy: 14,
		RewardScale: 0.08, BaselineLR: 0.02, WorldModelLR: 0.03, CuriosityGain: 0.6,
		RescueChance: 0.05,
	}
}

// gestation returns the post-reproduction cooldown for the given kind.
func (w *World) gestation(k Kind) int {
	if k == Carnivore {
		return w.params.GestationCarn
	}
	return w.params.GestationHerb
}

// foodRipe reports whether a plant holds enough energy to be worth eating.
func (w *World) foodRipe(f *Food) bool { return f.Energy >= w.params.FoodBiteEnergy*0.5 }

// FoodRipe is the exported form used by the renderer to decide what to draw.
func (w *World) FoodRipe(f *Food) bool { return w.foodRipe(f) }

// regrowFood advances a plant's energy by one tick toward the cap.
func (w *World) regrowFood(f *Food) {
	if f.Energy < w.params.FoodMaxEnergy {
		f.Energy += w.params.FoodRegrowRate
		if f.Energy > w.params.FoodMaxEnergy {
			f.Energy = w.params.FoodMaxEnergy
		}
	}
}
