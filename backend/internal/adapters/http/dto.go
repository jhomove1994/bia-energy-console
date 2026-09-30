package httpapi

import (
	"time"

	"energy/internal/app"
	"energy/internal/domain"
)

// Public field names follow the specs (see the tasks naming resolutions).

type meterRowDTO struct {
	MeterID       string     `json:"meter_id"`
	Status        string     `json:"status"`
	LastReadingAt *time.Time `json:"last_reading_at"`
	TotalKWh      float64    `json:"total_kwh"`
	VariationPct  *float64   `json:"variation_pct"`
}

type meterDetailDTO struct {
	meterRowDTO
	AnomalyCount int `json:"anomaly_count"`
}

type meterListDTO struct {
	Items []meterRowDTO `json:"items"`
}

type readingPointDTO struct {
	Timestamp           time.Time `json:"timestamp"`
	ConsumptionKWh      float64   `json:"consumption_kwh"`
	VoltageV            float64   `json:"voltage_v"`
	CurrentA            float64   `json:"current_a"`
	PowerFactor         float64   `json:"power_factor"`
	Status              string    `json:"status"`
	BaselineKWh         float64   `json:"baseline_kwh"`
	BaselineVoltageV    float64   `json:"baseline_voltage_v"`
	BaselineCurrentA    float64   `json:"baseline_current_a"`
	BaselinePowerFactor float64   `json:"baseline_power_factor"`
}

type readingsDTO struct {
	MeterID string            `json:"meter_id"`
	Points  []readingPointDTO `json:"points"`
}

type eventDTO struct {
	ID          int64     `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
}

type eventListDTO struct {
	Items []eventDTO `json:"items"`
}

func toMeterRowDTO(r app.MeterRow) meterRowDTO {
	return meterRowDTO{
		MeterID: r.MeterID, Status: string(r.Status), LastReadingAt: r.LastReadingAt,
		TotalKWh: r.TotalKWh, VariationPct: r.VariationPct,
	}
}

func toMeterDetailDTO(d app.MeterDetail) meterDetailDTO {
	return meterDetailDTO{meterRowDTO: toMeterRowDTO(d.MeterRow), AnomalyCount: d.FindingsCount}
}

func toReadingPointDTO(p app.ReadingPoint) readingPointDTO {
	return readingPointDTO{
		Timestamp: p.TS, ConsumptionKWh: p.KWh, VoltageV: p.VoltageV, CurrentA: p.CurrentA,
		PowerFactor: p.PF, Status: p.Status,
		BaselineKWh: p.Baseline.KWh, BaselineVoltageV: p.Baseline.VoltageV,
		BaselineCurrentA: p.Baseline.CurrentA, BaselinePowerFactor: p.Baseline.PF,
	}
}

func toEventDTO(e domain.Event) eventDTO {
	return eventDTO{ID: e.ID, Timestamp: e.TS, Type: string(e.Type), Description: e.Description}
}
