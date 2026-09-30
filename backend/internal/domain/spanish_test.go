package domain

import (
	"testing"
	"time"
)

func TestFormatNumberUsesDecimalCommaAndDotThousands(t *testing.T) {
	cases := []struct {
		in   float64
		dec  int
		want string
	}{
		{2180.04, 0, "2.180"},
		{2180.456, 1, "2.180,5"},
		{155250.85, 1, "155.250,9"},
		{103.74, 1, "103,7"},
		{0.94, 2, "0,94"},
		{58, 0, "58"},
		{-79.84, 1, "-79,8"},
		{-1234567.5, 1, "-1.234.567,5"},
		{999.96, 1, "1.000,0"},
		{0, 1, "0,0"},
	}
	for _, c := range cases {
		if got := FormatNumber(c.in, c.dec); got != c.want {
			t.Errorf("FormatNumber(%v, %d) = %q, want %q", c.in, c.dec, got, c.want)
		}
	}
}

func TestFormatSignedAlwaysShowsASign(t *testing.T) {
	if got := FormatSigned(103.74, 1); got != "+103,7" {
		t.Errorf("Signed up = %q", got)
	}
	if got := FormatSigned(-22.26, 1); got != "-22,3" {
		t.Errorf("Signed down = %q", got)
	}
}

func TestFormatStampIsSpanishAndUTC(t *testing.T) {
	at := time.Date(2026, 9, 12, 14, 0, 0, 0, time.UTC)
	if got := FormatStamp(at); got != "12 sept 2026, 14:00 UTC" {
		t.Errorf("Stamp = %q", got)
	}
	jan := time.Date(2026, 1, 3, 9, 5, 0, 0, time.UTC)
	if got := FormatStamp(jan); got != "3 ene 2026, 09:05 UTC" {
		t.Errorf("Stamp = %q", got)
	}
	bogota := time.FixedZone("COT", -5*3600)
	if got := FormatStamp(time.Date(2026, 9, 12, 9, 0, 0, 0, bogota)); got != "12 sept 2026, 14:00 UTC" {
		t.Errorf("Stamp must convert to UTC: %q", got)
	}
}

func TestEventTypeNames(t *testing.T) {
	cases := map[EventType]string{
		EventOperationalChange: "Cambio operativo",
		EventScheduledOutage:   "Parada programada",
		EventDataQuality:       "Calidad de datos",
		EventUnknown:           "Desconocido",
		"SOMETHING_NEW":        "Desconocido",
	}
	for in, want := range cases {
		if got := EventTypeName(in); got != want {
			t.Errorf("EventTypeName(%s) = %q, want %q", in, got, want)
		}
	}
}
