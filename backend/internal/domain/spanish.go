package domain

// Spanish text fragments for user-facing copy the backend produces (numbers,
// dates and event names): neutral Spanish with Colombian formatting, that is
// decimal comma, dot as thousands separator and UTC timestamps.

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FormatNumber formats x with a fixed number of decimals, for example 2.180,5.
func FormatNumber(x float64, decimals int) string {
	raw := strconv.FormatFloat(x, 'f', decimals, 64)
	negative := strings.HasPrefix(raw, "-")
	raw = strings.TrimPrefix(raw, "-")
	whole, frac, _ := strings.Cut(raw, ".")

	var sb strings.Builder
	for i, digit := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			sb.WriteByte('.')
		}
		sb.WriteRune(digit)
	}
	out := sb.String()
	if frac != "" {
		out += "," + frac
	}
	if negative && strings.Trim(out, "0.,") != "" {
		out = "-" + out
	}
	return out
}

// FormatSigned is FormatNumber with an explicit plus for positive values.
func FormatSigned(x float64, decimals int) string {
	out := FormatNumber(x, decimals)
	if !strings.HasPrefix(out, "-") && strings.Trim(out, "0.,") != "" {
		return "+" + out
	}
	return out
}

var months = [...]string{"ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sept", "oct", "nov", "dic"}

// FormatStamp renders an instant as "12 sept 2026, 14:00 UTC".
func FormatStamp(t time.Time) string {
	t = t.UTC()
	return fmt.Sprintf("%d %s %d, %s UTC", t.Day(), months[t.Month()-1], t.Year(), t.Format("15:04"))
}

// EventTypeName names an operational event type; unknown values read as "Desconocido".
func EventTypeName(t EventType) string {
	switch t {
	case EventOperationalChange:
		return "Cambio operativo"
	case EventScheduledOutage:
		return "Parada programada"
	case EventDataQuality:
		return "Calidad de datos"
	default:
		return "Desconocido"
	}
}
