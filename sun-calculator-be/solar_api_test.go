package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseSolarFields(t *testing.T) {
	all, err := parseSolarFields("")
	if err != nil || !all.any() || !all.sunrise || !all.eveningBlue {
		t.Fatalf("empty should select all: %+v err=%v", all, err)
	}
	if word, _ := parseSolarFields("all"); word != all {
		t.Fatalf("\"all\" should equal empty selection")
	}
	combo, err := parseSolarFields("sunrise, day_length , morning_golden_hour")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !combo.sunrise || !combo.dayLength || !combo.morningGolden ||
		combo.sunset || combo.solarNoon || combo.morningBlue {
		t.Fatalf("combo parsed wrong: %+v", combo)
	}
	if _, err := parseSolarFields("sunrise,banana"); err == nil {
		t.Fatalf("expected error for unknown field")
	}
	if empty, _ := parseSolarFields(","); empty.any() {
		t.Fatalf("expected empty selection to report any()==false")
	}
}

func TestSolarHandlerSingleField(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet,
		"/api/solar?lat=42.3601&lng=-71.0589&date=2026-06-21&field=sunrise", nil)
	rec := httptest.NewRecorder()
	solarHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(body) != 1 {
		t.Fatalf("expected exactly one field, got %v", body)
	}
	if _, ok := body["sunrise"].(string); !ok {
		t.Fatalf("expected sunrise string, got %v", body)
	}
}

func TestSolarHandlerAllFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet,
		"/api/solar?lat=42.3601&lng=-71.0589&date=2026-06-21", nil)
	rec := httptest.NewRecorder()
	solarHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, key := range []string{
		"sunrise", "sunset", "solar_noon", "day_length",
		"morning_blue_hour", "morning_golden_hour", "evening_golden_hour", "evening_blue_hour",
	} {
		if _, ok := body[key]; !ok {
			t.Errorf("missing field %q in response %v", key, body)
		}
	}
	// A twilight window carries start/end/duration.
	win, ok := body["morning_blue_hour"].(map[string]any)
	if !ok || win["start"] == nil || win["end"] == nil || win["duration"] == nil {
		t.Errorf("morning_blue_hour not a full window: %v", body["morning_blue_hour"])
	}
}

func TestSolarHandlerErrors(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want int
	}{
		{"missing coords", "/api/solar", http.StatusBadRequest},
		{"bad lat", "/api/solar?lat=999&lng=1", http.StatusBadRequest},
		{"NaN coords", "/api/solar?lat=NaN&lng=NaN&field=sunrise", http.StatusBadRequest},
		{"bad date", "/api/solar?lat=1&lng=1&date=nope", http.StatusBadRequest},
		{"bad field", "/api/solar?lat=1&lng=1&field=banana", http.StatusBadRequest},
		{"empty field", "/api/solar?lat=1&lng=1&field=,", http.StatusBadRequest},
		{"polar night", "/api/solar?lat=89&lng=0&date=2026-12-21&field=sunrise", http.StatusUnprocessableEntity},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, c.url, nil)
			rec := httptest.NewRecorder()
			solarHandler(rec, req)
			if rec.Code != c.want {
				t.Fatalf("got %d want %d: %s", rec.Code, c.want, rec.Body.String())
			}
		})
	}
}
