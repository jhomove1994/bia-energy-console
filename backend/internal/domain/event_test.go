package domain

import "testing"

func TestNormalizeEventType(t *testing.T) {
	tests := []struct {
		in   string
		want EventType
	}{
		{"SCHEDULED_OUTAGE", EventScheduledOutage},
		{"OPERATIONAL_CHANGE", EventOperationalChange},
		{"DATA_QUALITY", EventDataQuality},
		{"UNKNOWN", EventUnknown},
		{"FIRMWARE_UPDATE", EventUnknown},
		{"maintenance", EventScheduledOutage},
		{"SCHEDULED_MAINTENANCE", EventScheduledOutage},
		{"PLANNED_OUTAGE", EventScheduledOutage},
		{"  data_quality  ", EventDataQuality},
		{"", EventUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := NormalizeEventType(tt.in); got != tt.want {
				t.Errorf("NormalizeEventType(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
