package engine

import "energy/internal/domain"

// synthV and synthPF are the fixed voltage and power factor of synthetic
// readings. The current follows the profile so that the electrical ratio
// kWh / (V*A*PF/1000) is 1.05 for an unmodified reading.
const (
	synthV     = 230.0
	synthPF    = 0.9
	synthRatio = 1.05
)

// flatProfile returns a profile with the same kWh in every hour of day.
func flatProfile(kwh float64) [24]float64 {
	var p [24]float64
	for h := range p {
		p[h] = kwh
	}
	return p
}

// shapedProfile returns a daytime-peaked profile whose hours differ enough to
// give a meaningful correlation.
func shapedProfile() [24]float64 {
	base := [24]float64{
		8, 7, 7, 6, 6, 7, 9, 12, 16, 20, 24, 27,
		28, 27, 25, 22, 19, 17, 15, 13, 11, 10, 9, 8,
	}
	return base
}

// synth builds days*24 healthy hourly readings for a synthetic meter. mutate,
// when not nil, may change any field of the reading at (day, hour) after the
// healthy values were set.
func synth(days int, profile [24]float64, mutate func(day, hour int, r *domain.Reading)) []domain.Reading {
	series := hourlySeries(days, func(_, h int) domain.Reading {
		kwh := profile[h]
		return domain.Reading{
			KWh:      kwh,
			VoltageV: synthV,
			CurrentA: kwh * 1000 / (synthRatio * synthV * synthPF),
			PF:       synthPF,
		}
	})
	if mutate != nil {
		// Applied after hourlySeries so the mutation can also change Status.
		for i := range series {
			mutate(i/24, i%24, &series[i])
		}
	}
	return series
}

// scaleLoad scales the energy and the current together so the electrical
// ratio of the reading is unchanged.
func scaleLoad(r *domain.Reading, f float64) {
	r.KWh *= f
	r.CurrentA *= f
}
