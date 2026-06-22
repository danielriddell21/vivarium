package sim

import "math"

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

	// Seasons: a sinusoidal cycle that scales plant regrowth between scarcity and
	// plenty. SeasonLength is the period in ticks (0 disables); SeasonAmplitude is
	// the 0..1 swing around the baseline regrowth rate.
	SeasonLength    int     `json:"seasonLength"`
	SeasonAmplitude float64 `json:"seasonAmplitude"`

	// Day/night: a cycle that scales effective vision range. DayLength is the
	// period in ticks (0 disables); NightVision (0..1) is the fraction of sense
	// radius retained at the darkest point of night.
	DayLength   int     `json:"dayLength"`
	NightVision float64 `json:"nightVision"`
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
		SeasonLength: 3000, SeasonAmplitude: 0.6,
		DayLength: 1200, NightVision: 0.45,
	}
}

// LightFactor returns the current daylight level, scaling effective vision range
// from NightVision (deepest night) to 1 (full day). Deterministic in the tick.
func (w *World) LightFactor() float64 {
	if w.params.DayLength <= 0 {
		return 1
	}
	phase := 2 * math.Pi * float64(w.Tick) / float64(w.params.DayLength)
	day := 0.5 + 0.5*math.Sin(phase) // 0 at midnight, 1 at midday
	return w.params.NightVision + (1-w.params.NightVision)*day
}

// SeasonFactor returns the current plant-regrowth multiplier from the seasonal
// cycle: 1 at the equinoxes, up to 1+amplitude in summer and down to 1-amplitude
// (floored at 0) in winter. It is a deterministic function of the tick.
func (w *World) SeasonFactor() float64 {
	if w.params.SeasonLength <= 0 {
		return 1
	}
	phase := 2 * math.Pi * float64(w.Tick) / float64(w.params.SeasonLength)
	f := 1 + w.params.SeasonAmplitude*math.Sin(phase)
	if f < 0 {
		f = 0
	}
	return f
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

// regrowFood advances a plant's energy by one tick toward the cap, scaled by the
// current season (plants grow faster in summer, slower in winter).
func (w *World) regrowFood(f *Food) {
	if f.Energy < w.params.FoodMaxEnergy {
		f.Energy += w.params.FoodRegrowRate * w.SeasonFactor()
		if f.Energy > w.params.FoodMaxEnergy {
			f.Energy = w.params.FoodMaxEnergy
		}
	}
}
