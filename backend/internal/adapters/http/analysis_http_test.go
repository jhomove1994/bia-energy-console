package httpapi

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"energy/internal/adapters/csvload"
	"energy/internal/adapters/explain"
	"energy/internal/adapters/sqlite"
	"energy/internal/app"
	"energy/internal/engine"
	"energy/internal/platform"
)

// newDemoFixture seeds the demo dataset from data/*.csv into a temp database
// and wires the full analysis stack around it.
func newDemoFixture(t *testing.T, opts ...app.AnalysisOption) *fixture {
	t.Helper()
	ctx := context.Background()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "demo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := sqlite.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	var seedLogs bytes.Buffer
	if _, err := csvload.Seed(ctx, db, filepath.Join("..", "..", "..", "..", "data"), slog.New(slog.NewTextHandler(&seedLogs, nil))); err != nil {
		t.Fatal(err)
	}

	meters, readings, events := sqlite.NewMeterStore(db), sqlite.NewReadingStore(db), sqlite.NewEventStore(db)
	analyses, anomalies := sqlite.NewAnalysisStore(db), sqlite.NewAnomalyStore(db)
	cfg := engine.DefaultConfig()
	var logs bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logs, nil))
	tokens := NewTokens(testSecret, time.Hour, time.Now)
	h := NewRouter(Deps{
		DemoUser: testUser, DemoPassword: testPass, Tokens: tokens, Log: log,
		Meters:    app.NewMeterService(meters, readings, events, app.NewAnalysisStatuses(analyses, anomalies), cfg),
		Dashboard: app.NewDashboardService(meters, readings, analyses),
		Analysis:  app.NewAnalysisService(analyses, readings, events, explain.New(), platform.UUIDGenerator{}, cfg, opts...),
		Anomalies: app.NewAnomalyService(analyses, anomalies),
	})
	token, _ := tokens.Issue(testUser)
	return &fixture{Handler: h, Token: token, Logs: &logs}
}

func (f *fixture) post(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	return do(f.Handler, "POST", path, "", f.Token)
}

type analysisResp struct {
	ID         string  `json:"id"`
	Status     string  `json:"status"`
	Stage      string  `json:"stage"`
	StageIndex int     `json:"stage_index"`
	StageTotal int     `json:"stage_total"`
	Progress   int     `json:"progress"`
	StartedAt  *string `json:"started_at"`
	FinishedAt *string `json:"finished_at"`
	Error      *string `json:"error"`
	Summary    *struct {
		Total             int    `json:"total"`
		PriorityAttention int    `json:"priority_attention"`
		Text              string `json:"text"`
	} `json:"summary"`
}

// runAnalysis starts an analysis and waits until it is done.
func (f *fixture) runAnalysis(t *testing.T) string {
	t.Helper()
	rec := f.post(t, "/api/ai/analyze")
	if rec.Code != 202 {
		t.Fatalf("analyze status = %d body %s", rec.Code, rec.Body)
	}
	id := decode[struct{ ID string }](t, rec).ID
	f.waitFinal(t, id)
	return id
}

func (f *fixture) waitFinal(t *testing.T, id string) analysisResp {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		a := decode[analysisResp](t, f.get(t, "/api/ai/analysis/"+id))
		if a.Status == "done" || a.Status == "failed" {
			return a
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("analysis %s did not finish", id)
	return analysisResp{}
}

type anomalyResp struct {
	ID                string  `json:"id"`
	AnalysisID        string  `json:"analysis_id"`
	MeterID           string  `json:"meter_id"`
	Anomaly           string  `json:"anomaly"`
	Kind              string  `json:"kind"`
	Type              string  `json:"type"`
	Severity          string  `json:"severity"`
	Confidence        float64 `json:"confidence"`
	ConfidenceLabel   string  `json:"confidence_label"`
	PriorityScore     float64 `json:"priority_score"`
	PriorityRank      int     `json:"priority_rank"`
	Reason            string  `json:"reason"`
	RecommendedAction string  `json:"recommended_action"`
	ActionCode        string  `json:"action_code"`
	WindowStart       string  `json:"window_start"`
	WindowEnd         string  `json:"window_end"`
	RelatedEvent      *struct {
		ID          int64  `json:"id"`
		Type        string `json:"type"`
		Description string `json:"description"`
	} `json:"related_event"`
	Evidence []struct {
		Metric   string   `json:"metric"`
		Label    string   `json:"label"`
		Baseline *float64 `json:"baseline"`
		Observed float64  `json:"observed"`
		DeltaPct *float64 `json:"delta_pct"`
		Unit     string   `json:"unit"`
	} `json:"evidence"`
}

type anomalyListResp struct {
	AnalysisID  *string       `json:"analysis_id"`
	SummaryText *string       `json:"summary_text"`
	Items       []anomalyResp `json:"items"`
}

func meterIDs(items []anomalyResp) []string {
	var out []string
	for _, a := range items {
		out = append(out, a.MeterID)
	}
	return out
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestAnalyzeStartsOnceWhileActive(t *testing.T) {
	f := newDemoFixture(t, app.WithStageDelay(40*time.Millisecond))
	first := f.post(t, "/api/ai/analyze")
	if first.Code != http.StatusAccepted {
		t.Fatalf("status = %d body %s", first.Code, first.Body)
	}
	id := decode[map[string]any](t, first)
	if len(id) != 1 || id["id"] == "" || id["id"] == nil {
		t.Fatalf("body must be exactly {id}: %v", id)
	}
	again := f.post(t, "/api/ai/analyze")
	if again.Code != 202 || decode[map[string]any](t, again)["id"] != id["id"] {
		t.Fatalf("repeat while active = %d %s, want same id %v", again.Code, again.Body, id["id"])
	}

	// Poll: at least two distinct stages, progress never decreases, nothing visible before done.
	stages, last := map[string]bool{}, -1
	for {
		a := decode[analysisResp](t, f.get(t, "/api/ai/analysis/"+id["id"].(string)))
		if a.Progress < last {
			t.Fatalf("progress decreased: %d after %d", a.Progress, last)
		}
		last = a.Progress
		if a.Stage != "" {
			stages[a.Stage] = true
		}
		if a.Status == "done" {
			if a.Progress != 100 || a.FinishedAt == nil || a.StartedAt == nil || a.Summary == nil || a.StageTotal != 7 {
				t.Fatalf("done payload = %+v", a)
			}
			break
		}
		if a.Status == "failed" {
			t.Fatalf("analysis failed: %+v", a)
		}
		mid := decode[anomalyListResp](t, f.get(t, "/api/anomalies"))
		if len(mid.Items) != 0 {
			t.Fatalf("anomalies visible before done: %d", len(mid.Items))
		}
		time.Sleep(5 * time.Millisecond)
	}
	if len(stages) < 2 {
		t.Fatalf("observed stages %v, want at least two", stages)
	}
	next := f.post(t, "/api/ai/analyze")
	nextID := decode[map[string]any](t, next)["id"]
	if next.Code != 202 || nextID == id["id"] {
		t.Fatalf("new run after done must get a new id: %s", next.Body)
	}
	f.waitFinal(t, nextID.(string)) // do not close the database under a running job
}

func TestAnalysisStatusEndpoints(t *testing.T) {
	f := newDemoFixture(t)
	if rec := f.get(t, "/api/ai/analysis/latest"); rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("latest before any run = %d %q, want 204 empty", rec.Code, rec.Body)
	}
	if rec := f.get(t, "/api/ai/analysis/no-such-id"); rec.Code != 404 || errorCode(t, rec) != "not_found" {
		t.Fatalf("unknown id = %d %s", rec.Code, rec.Body)
	}

	id := f.runAnalysis(t)
	a := decode[analysisResp](t, f.get(t, "/api/ai/analysis/"+id))
	if a.ID != id || a.Status != "done" || a.Progress != 100 || a.Error == nil || *a.Error != "" || a.StartedAt == nil || a.FinishedAt == nil {
		t.Fatalf("done payload = %+v", a)
	}
	if a.Summary == nil || a.Summary.Total != 4 || a.Summary.PriorityAttention != 2 || a.Summary.Text != "4 anomalías detectadas · 2 requieren atención prioritaria" {
		t.Fatalf("summary = %+v", a.Summary)
	}
	latest := f.get(t, "/api/ai/analysis/latest")
	if latest.Code != 200 || decode[analysisResp](t, latest).ID != id {
		t.Fatalf("latest = %d %s", latest.Code, latest.Body)
	}
}

func TestAnomaliesEmptyBeforeAnyAnalysis(t *testing.T) {
	f := newDemoFixture(t)
	rec := f.get(t, "/api/anomalies")
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	got := decode[anomalyListResp](t, rec)
	if got.Items == nil || len(got.Items) != 0 || got.AnalysisID != nil || got.SummaryText != nil {
		t.Fatalf("list = %+v, want empty items and null analysis", got)
	}
}

func TestAnomaliesOnDemoDataset(t *testing.T) {
	f := newDemoFixture(t)
	id := f.runAnalysis(t)

	got := decode[anomalyListResp](t, f.get(t, "/api/anomalies"))
	if got.AnalysisID == nil || *got.AnalysisID != id || got.SummaryText == nil || *got.SummaryText != "4 anomalías detectadas · 2 requieren atención prioritaria" {
		t.Fatalf("envelope = %+v", got)
	}
	if want := []string{"M-109", "M-112", "M-104", "M-106"}; !sameStrings(meterIDs(got.Items), want) {
		t.Fatalf("order = %v, want %v", meterIDs(got.Items), want)
	}
	for i, a := range got.Items {
		if a.PriorityRank != i+1 || a.ID == "" || a.AnalysisID != id || a.Anomaly == "" || a.Kind == "" || a.ConfidenceLabel == "" ||
			a.Reason == "" || a.RecommendedAction == "" || len(a.Evidence) == 0 || a.WindowStart == "" || a.WindowEnd == "" || a.Confidence <= 0 {
			t.Fatalf("item %d incomplete: %+v", i, a)
		}
	}
	if top := got.Items[0]; top.Type != "REAL_ANOMALY" || top.Severity != "HIGH" || top.Anomaly != "Level shift up" || top.ActionCode != "INSPECT_LOAD_INCREASE" {
		t.Fatalf("top = %+v", top)
	}

	dq := decode[anomalyListResp](t, f.get(t, "/api/anomalies?type=DATA_QUALITY"))
	if !sameStrings(meterIDs(dq.Items), []string{"M-112"}) {
		t.Fatalf("type filter = %v", meterIDs(dq.Items))
	}
	high := decode[anomalyListResp](t, f.get(t, "/api/anomalies?severity=HIGH"))
	if !sameStrings(meterIDs(high.Items), []string{"M-109", "M-112"}) {
		t.Fatalf("severity filter = %v", meterIDs(high.Items))
	}
	one := decode[anomalyListResp](t, f.get(t, "/api/anomalies?meter_id=M-104&analysis_id="+id))
	if !sameStrings(meterIDs(one.Items), []string{"M-104"}) {
		t.Fatalf("meter filter = %v", meterIDs(one.Items))
	}

	for path, want := range map[string]int{
		"/api/anomalies?analysis_id=ghost": 404,
		"/api/anomalies?type=BOGUS":        400,
		"/api/anomalies?severity=EXTREME":  400,
	} {
		if rec := f.get(t, path); rec.Code != want {
			t.Fatalf("GET %s = %d, want %d (%s)", path, rec.Code, want, rec.Body)
		}
	}
}

func TestAnomalyDetail(t *testing.T) {
	f := newDemoFixture(t)
	f.runAnalysis(t)
	list := decode[anomalyListResp](t, f.get(t, "/api/anomalies"))

	rec := f.get(t, "/api/anomalies/"+list.Items[0].ID)
	if rec.Code != 200 {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body)
	}
	a := decode[anomalyResp](t, rec)
	if a.MeterID != "M-109" || a.Reason == "" || a.RecommendedAction == "" {
		t.Fatalf("detail = %+v", a)
	}
	metrics := map[string]bool{}
	for _, e := range a.Evidence {
		metrics[e.Metric] = true
		if e.Label == "" || e.Metric == "" {
			t.Fatalf("evidence item incomplete: %+v", e)
		}
	}
	if !metrics["consumption_kwh"] || !metrics["current_a"] {
		t.Fatalf("evidence metrics = %v, want consumption and current", metrics)
	}
	if a.RelatedEvent == nil || a.RelatedEvent.Type != "UNKNOWN" || a.RelatedEvent.ID == 0 {
		t.Fatalf("related_event = %+v, want the UNKNOWN event as context", a.RelatedEvent)
	}
	// An anomaly with no event serializes related_event as null.
	for _, it := range list.Items {
		if it.MeterID == "M-112" && it.RelatedEvent == nil {
			t.Fatal("M-112 is explained by its DATA_QUALITY event")
		}
	}
	if rec := f.get(t, "/api/anomalies/ghost"); rec.Code != 404 || errorCode(t, rec) != "not_found" {
		t.Fatalf("unknown anomaly = %d %s", rec.Code, rec.Body)
	}
}

func TestMeterStatusesAfterAnalysis(t *testing.T) {
	f := newDemoFixture(t)
	before := decode[listResp](t, f.get(t, "/api/meters"))
	for _, it := range before.Items {
		if it.Status != "OK" {
			t.Fatalf("%s is %s before any analysis", it.MeterID, it.Status)
		}
	}
	f.runAnalysis(t)
	after := decode[listResp](t, f.get(t, "/api/meters"))
	want := map[string]string{"M-109": "Critical", "M-112": "Alert", "M-104": "Alert"}
	for _, it := range after.Items {
		exp := want[it.MeterID]
		if exp == "" {
			exp = "OK"
		}
		if it.Status != exp {
			t.Fatalf("%s = %s, want %s", it.MeterID, it.Status, exp)
		}
	}
	detail := decode[struct {
		Status       string `json:"status"`
		AnomalyCount int    `json:"anomaly_count"`
	}](t, f.get(t, "/api/meters/M-109"))
	if detail.Status != "Critical" || detail.AnomalyCount != 1 {
		t.Fatalf("M-109 detail = %+v", detail)
	}
}

func TestDashboardAfterAnalysis(t *testing.T) {
	f := newDemoFixture(t)
	f.runAnalysis(t)
	d := decode[struct {
		Anomalies  int      `json:"anomalies_count"`
		High       int      `json:"high_priority_count"`
		Confidence *float64 `json:"aggregate_confidence"`
		Last       *struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"last_analysis"`
		Text *string `json:"summary_text"`
	}](t, f.get(t, "/api/dashboard/summary"))
	if d.Anomalies != 4 || d.High != 2 || d.Text == nil || *d.Text != "4 anomalías detectadas · 2 requieren atención prioritaria" ||
		d.Last == nil || d.Last.Status != "done" || d.Confidence == nil || *d.Confidence < 0.8 || *d.Confidence > 0.95 {
		t.Fatalf("dashboard = %+v", d)
	}
}

func TestAnalysisRoutesRequireAuth(t *testing.T) {
	f := newDemoFixture(t)
	for _, r := range [][2]string{
		{"POST", "/api/ai/analyze"}, {"GET", "/api/ai/analysis/latest"}, {"GET", "/api/ai/analysis/x"},
		{"GET", "/api/anomalies"}, {"GET", "/api/anomalies/x"},
	} {
		rec := do(f.Handler, r[0], r[1], "", "")
		if rec.Code != 401 || errorCode(t, rec) != "unauthorized" {
			t.Fatalf("%s %s without token = %d %s", r[0], r[1], rec.Code, rec.Body)
		}
	}
}
