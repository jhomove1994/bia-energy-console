package engine

import "energy/internal/domain"

// Recommend maps the type, direction and kind of an anomaly to the action to
// take. Only REAL anomalies depend on the direction and the kind: a transient
// or a pattern change is reviewed, a sustained increase is inspected on site and
// a sustained decrease is checked for a supply loss.
func Recommend(t domain.AnomalyType, sign int, kind domain.AnomalyKind) domain.ActionCode {
	switch t {
	case domain.TypeDataQuality:
		return domain.ActionOpenDQTicket
	case domain.TypeExplainable:
		return domain.ActionUpdateBaseline
	case domain.TypeFalsePos:
		return domain.ActionAcknowledgeExpected
	}
	switch {
	case kind == domain.KindPatternChange:
		return domain.ActionReviewSchedule
	case kind == domain.KindConsumptionSpike || kind == domain.KindConsumptionDip:
		return domain.ActionReviewTransient
	case sign < 0:
		return domain.ActionCheckSupplyLoss
	}
	return domain.ActionInspectLoadIncrease
}
