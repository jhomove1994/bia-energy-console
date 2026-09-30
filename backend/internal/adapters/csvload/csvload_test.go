package csvload

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"energy/internal/domain"
)

func testLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, nil))
}

const readingsHeader = "meter_id,timestamp,consumption_kwh,voltage_v,current_a,power_factor,status\n"

func TestParseReadingsTimestampFormats(t *testing.T) {
	in := readingsHeader +
		"X-1,2026-09-01 00:00:00,23.5,221.9,101.28,0.954,OK\n" +
		"X-1,2026-09-01 01:00,24.5,222.0,102.00,0.950,OK\n" +
		"X-1,2026-09-01T02:00:00Z,25.5,223.0,103.00,0.940,OK\n" +
		"X-1,2026-09-01T03:00:00,26.5,224.0,104.00,0.930,OK\n"
	var logs bytes.Buffer
	rows, skipped, err := ParseReadings(strings.NewReader(in), "readings.csv", testLogger(&logs))
	if err != nil {
		t.Fatalf("ParseReadings: %v", err)
	}
	if skipped != 0 || len(rows) != 4 {
		t.Fatalf("rows=%d skipped=%d, want 4/0", len(rows), skipped)
	}
	for i, r := range rows {
		want := time.Date(2026, 9, 1, i, 0, 0, 0, time.UTC)
		if !r.TS.Equal(want) || r.TS.Location() != time.UTC {
			t.Errorf("row %d TS = %v, want %v UTC", i, r.TS, want)
		}
	}
	first := rows[0]
	if first.MeterID != "X-1" || first.KWh != 23.5 || first.VoltageV != 221.9 || first.CurrentA != 101.28 || first.PF != 0.954 || first.Status != "OK" {
		t.Errorf("unexpected first row: %+v", first)
	}
}

func TestParseReadingsSkipsMalformedRows(t *testing.T) {
	in := readingsHeader +
		"X-1,2026-09-01 00:00:00,23.5,221.9,101.28,0.954,OK\n" +
		"X-1,2026-09-01 01:00:00,24.5,abc,102.00,0.950,OK\n" + // line 3: non-numeric voltage
		"X-1,2026-09-01 02:00:00,25.5,223.0\n" + // line 4: wrong column count
		"X-1,not-a-date,25.5,223.0,103.00,0.940,OK\n" + // line 5: bad timestamp
		"X-1,2026-09-01 03:00:00,26.5,224.0,104.00,0.930,SENSOR_FAULT\n"
	var logs bytes.Buffer
	rows, skipped, err := ParseReadings(strings.NewReader(in), "readings.csv", testLogger(&logs))
	if err != nil {
		t.Fatalf("ParseReadings: %v", err)
	}
	if len(rows) != 2 || skipped != 3 {
		t.Fatalf("rows=%d skipped=%d, want 2/3", len(rows), skipped)
	}
	if rows[1].Status != "SENSOR_FAULT" {
		t.Errorf("status must be preserved, got %q", rows[1].Status)
	}
	out := logs.String()
	for _, want := range []string{"readings.csv", "line=3", "line=4", "line=5", "voltage_v"} {
		if !strings.Contains(out, want) {
			t.Errorf("log missing %q:\n%s", want, out)
		}
	}
}

func TestParseEventsNormalizesTypes(t *testing.T) {
	in := "meter_id,event_timestamp,event_type,description\n" +
		"X-1,2026-09-11 00:00,OPERATIONAL_CHANGE,New production line activated\n" +
		"X-1,2026-09-12 00:00,FIRMWARE_UPDATE,Firmware v2 rollout\n" +
		"X-2,bad,DATA_QUALITY,skipped\n"
	var logs bytes.Buffer
	rows, skipped, err := ParseEvents(strings.NewReader(in), "events.csv", testLogger(&logs))
	if err != nil {
		t.Fatalf("ParseEvents: %v", err)
	}
	if len(rows) != 2 || skipped != 1 {
		t.Fatalf("rows=%d skipped=%d, want 2/1", len(rows), skipped)
	}
	if rows[0].Type != domain.EventOperationalChange {
		t.Errorf("row0 type = %q", rows[0].Type)
	}
	if rows[1].Type != domain.EventUnknown || rows[1].Description != "Firmware v2 rollout" {
		t.Errorf("unrecognized type must become UNKNOWN keeping description, got %+v", rows[1])
	}
	if !strings.Contains(logs.String(), "events.csv") || !strings.Contains(logs.String(), "line=4") {
		t.Errorf("log missing file/line:\n%s", logs.String())
	}
}
