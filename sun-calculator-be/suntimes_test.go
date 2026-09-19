package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSunTimesHandlerLocal(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/suntimes?lat=42.3601&lng=-71.0589", nil)
	rec := httptest.NewRecorder()
	sunTimesHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp SunTimesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Status != "success" {
		t.Errorf("status = %q, want success", resp.Status)
	}
	for name, val := range map[string]string{
		"sunrise": resp.Sunrise, "sunset": resp.Sunset,
		"solar_noon": resp.SolarNoon, "day_length": resp.DayLength,
	} {
		if val == "" {
			t.Errorf("%s is empty", name)
		}
	}
	for name, win := range map[string]*twilightWindow{
		"morning_blue_hour": resp.MorningBlueHour, "morning_golden_hour": resp.MorningGoldenHour,
		"evening_golden_hour": resp.EveningGoldenHour, "evening_blue_hour": resp.EveningBlueHour,
	} {
		if win == nil || win.Start == "" || win.End == "" || win.Duration == "" {
			t.Errorf("%s window incomplete: %+v", name, win)
		}
	}
}

func TestSunTimesHandlerErrors(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want int
	}{
		{"missing coords", "/api/suntimes", http.StatusBadRequest},
		{"bad lat", "/api/suntimes?lat=999&lng=1", http.StatusBadRequest},
		{"NaN coords", "/api/suntimes?lat=NaN&lng=NaN", http.StatusBadRequest},
		{"Inf coords", "/api/suntimes?lat=Inf&lng=0", http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, c.url, nil)
			rec := httptest.NewRecorder()
			sunTimesHandler(rec, req)
			if rec.Code != c.want {
				t.Fatalf("got %d want %d: %s", rec.Code, c.want, rec.Body.String())
			}
		})
	}
}
