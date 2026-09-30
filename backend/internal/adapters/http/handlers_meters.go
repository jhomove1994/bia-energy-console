package httpapi

import (
	"errors"
	"net/http"
	"time"

	"energy/internal/app"
)

// writeServiceError maps service errors to the JSON error shape.
func (s *server) writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	var ve *app.ValidationError
	switch {
	case errors.As(err, &ve):
		writeError(w, http.StatusBadRequest, codeInvalidQuery, ve.Msg)
	case errors.Is(err, app.ErrNotFound):
		writeError(w, http.StatusNotFound, codeNotFound, err.Error())
	default:
		s.deps.Log.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		writeError(w, http.StatusInternalServerError, codeInternal, "internal server error")
	}
}

func (s *server) handleListMeters(w http.ResponseWriter, r *http.Request) {
	qp := r.URL.Query()
	q := qp.Get("q")
	if q == "" {
		q = qp.Get("search") // alias
	}
	rows, err := s.deps.Meters.List(r.Context(), app.ListQuery{
		Q: q, Status: qp.Get("status"), Sort: qp.Get("sort"), Order: qp.Get("order"),
	})
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	out := meterListDTO{Items: make([]meterRowDTO, len(rows))}
	for i, row := range rows {
		out.Items[i] = toMeterRowDTO(row)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) handleMeterDetail(w http.ResponseWriter, r *http.Request) {
	d, err := s.deps.Meters.Detail(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toMeterDetailDTO(d))
}

// parseTimeParam parses an optional RFC 3339 query parameter.
func parseTimeParam(r *http.Request, name string) (*time.Time, error) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, &app.ValidationError{Msg: "invalid " + name + ": expected an RFC 3339 timestamp"}
	}
	return &t, nil
}

func (s *server) handleMeterReadings(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	from, err := parseTimeParam(r, "from")
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	to, err := parseTimeParam(r, "to")
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	pts, err := s.deps.Meters.Readings(r.Context(), id, from, to)
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	out := readingsDTO{MeterID: id, Points: make([]readingPointDTO, len(pts))}
	for i, p := range pts {
		out.Points[i] = toReadingPointDTO(p)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) handleMeterEvents(w http.ResponseWriter, r *http.Request) {
	evs, err := s.deps.Meters.Events(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	out := eventListDTO{Items: make([]eventDTO, len(evs))}
	for i, e := range evs {
		out.Items[i] = toEventDTO(e)
	}
	writeJSON(w, http.StatusOK, out)
}
