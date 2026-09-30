package engine

import (
	"testing"

	"energy/internal/domain"
)

func TestRecommendMapsTypeSignAndKindToAnAction(t *testing.T) {
	cases := []struct {
		name string
		typ  domain.AnomalyType
		sign int
		kind domain.AnomalyKind
		want domain.ActionCode
	}{
		{"real level shift up", domain.TypeReal, 1, domain.KindLevelShiftUp, domain.ActionInspectLoadIncrease},
		{"real sustained increase", domain.TypeReal, 1, domain.KindSustainedIncrease, domain.ActionInspectLoadIncrease},
		{"real level shift down", domain.TypeReal, -1, domain.KindLevelShiftDown, domain.ActionCheckSupplyLoss},
		{"real sustained decrease", domain.TypeReal, -1, domain.KindSustainedDecrease, domain.ActionCheckSupplyLoss},
		{"real spike", domain.TypeReal, 1, domain.KindConsumptionSpike, domain.ActionReviewTransient},
		{"real dip", domain.TypeReal, -1, domain.KindConsumptionDip, domain.ActionReviewTransient},
		{"real pattern change", domain.TypeReal, 0, domain.KindPatternChange, domain.ActionReviewSchedule},
		{"data quality", domain.TypeDataQuality, 0, domain.KindDataInconsistency, domain.ActionOpenDQTicket},
		{"explainable up", domain.TypeExplainable, 1, domain.KindLevelShiftUp, domain.ActionUpdateBaseline},
		{"explainable down", domain.TypeExplainable, -1, domain.KindLevelShiftDown, domain.ActionUpdateBaseline},
		{"false positive", domain.TypeFalsePos, -1, domain.KindSustainedDecrease, domain.ActionAcknowledgeExpected},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := Recommend(tt.typ, tt.sign, tt.kind); got != tt.want {
				t.Fatalf("Recommend = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestRecommendCoversEveryTypeWithADistinctCode(t *testing.T) {
	seen := map[domain.ActionCode]domain.AnomalyType{}
	for _, typ := range []domain.AnomalyType{domain.TypeReal, domain.TypeDataQuality, domain.TypeExplainable, domain.TypeFalsePos} {
		code := Recommend(typ, 1, domain.KindLevelShiftUp)
		if code == "" {
			t.Fatalf("no action for %s", typ)
		}
		if other, dup := seen[code]; dup {
			t.Fatalf("%s and %s share the action %s", typ, other, code)
		}
		seen[code] = typ
	}
}
