package sim

import "math"

type Params struct {
	HerbBasalCost float64 `json:"herbBasalCost"`
	CarnBasalCost float64 `json:"carnBasalCost"`
	MoveCost      float64 `json:"moveCost"`
	MaxEnergy     float64 `json:"maxEnergy"`
	MaxTurn       float64 `json:"maxTurn"`

	ReproThreshold  float64 `json:"reproThreshold"`
	StartEnergyHerb float64 `json:"startEnergyHerb"`
	StartEnergyCarn float64 `json:"startEnergyCarn"`
	GestationHerb   int     `json:"gestationHerb"`
	GestationCarn   int     `json:"gestationCarn"`

	CarnivoreGain  float64 `json:"carnivoreGain"`
	CarnivoreBonus float64 `json:"carnivoreBonus"`

	MutationRate float64 `json:"mutationRate"`
	MutationStd  float64 `json:"mutationStd"`

	FoodMaxEnergy  float64 `json:"foodMaxEnergy"`
	FoodRegrowRate float64 `json:"foodRegrowRate"`
	FoodEatRadius  float64 `json:"foodEatRadius"`
	FoodBiteEnergy float64 `json:"foodBiteEnergy"`

	RewardScale   float64 `json:"rewardScale"`
	BaselineLR    float64 `json:"baselineLR"`
	WorldModelLR  float64 `json:"worldModelLR"`
	CuriosityGain float64 `json:"curiosityGain"`

	RescueChance float64 `json:"rescueChance"`

	SeasonLength    int     `json:"seasonLength"`
	SeasonAmplitude float64 `json:"seasonAmplitude"`

	DayLength   int     `json:"dayLength"`
	NightVision float64 `json:"nightVision"`

	Sexual     bool    `json:"sexual"`
	MateRadius float64 `json:"mateRadius"`
}

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
		Sexual: true, MateRadius: 45,
	}
}

func (w *World) LightFactor() float64 {
	if w.params.DayLength <= 0 {
		return 1
	}
	phase := 2 * math.Pi * float64(w.Tick) / float64(w.params.DayLength)
	day := 0.5 + 0.5*math.Sin(phase) // 0 at midnight, 1 at midday
	return w.params.NightVision + (1-w.params.NightVision)*day
}

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

func (w *World) gestation(k Kind) int {
	if k == Carnivore {
		return w.params.GestationCarn
	}
	return w.params.GestationHerb
}

func (w *World) foodRipe(f *Food) bool { return f.Energy >= w.params.FoodBiteEnergy*0.5 }

func (w *World) FoodRipe(f *Food) bool { return w.foodRipe(f) }

func (w *World) regrowFood(f *Food) {
	if f.Energy < w.params.FoodMaxEnergy {
		f.Energy += w.params.FoodRegrowRate * w.SeasonFactor()
		if f.Energy > w.params.FoodMaxEnergy {
			f.Energy = w.params.FoodMaxEnergy
		}
	}
}
