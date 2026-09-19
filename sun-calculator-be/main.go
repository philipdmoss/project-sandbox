package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var unitarianUniversalistPrinciples = []string{
	"The inherent worth and dignity of every person",
	"Justice, equity and compassion in human relations",
	"Acceptance of one another and encouragement to spiritual growth in our congregations",
	"A free and responsible search for truth and meaning",
	"The right of conscience and the use of the democratic process within our congregations and in society at large",
	"The goal of world community with peace, liberty, and justice for all",
	"Respect for the interdependent web of all existence of which we are a part",
}

type upstreamResponse struct {
	Status  string `json:"status"`
	Results struct {
		Sunrise            string `json:"sunrise"`
		Sunset             string `json:"sunset"`
		SolarNoon          string `json:"solar_noon"`
		DayLength          int    `json:"day_length"`
		CivilTwilightBegin string `json:"civil_twilight_begin"`
		CivilTwilightEnd   string `json:"civil_twilight_end"`
	} `json:"results"`
}

type twilightWindow struct {
	Start    string `json:"start"`
	End      string `json:"end"`
	Duration string `json:"duration"`
}

type SunTimesResponse struct {
	Status            string         `json:"status"`
	Sunrise           string         `json:"sunrise"`
	Sunset            string         `json:"sunset"`
	SolarNoon         string         `json:"solar_noon"`
	DayLength         string         `json:"day_length"`
	MorningBlueHour   twilightWindow `json:"morning_blue_hour"`
	MorningGoldenHour twilightWindow `json:"morning_golden_hour"`
	EveningGoldenHour twilightWindow `json:"evening_golden_hour"`
	EveningBlueHour   twilightWindow `json:"evening_blue_hour"`
}

const sunApiEndpoint = "https://api.sunrise-sunset.org/json"

func fetchSunTimes(lat, lng string) (upstreamResponse, error) {
	var parsed upstreamResponse

	query := url.Values{}
	query.Set("lat", lat)
	query.Set("lng", lng)
	query.Set("formatted", "0")
	requestURL := sunApiEndpoint + "?" + query.Encode()

	resp, err := http.Get(requestURL)
	if err != nil {
		return parsed, fmt.Errorf("contacting upstream API: %w", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return parsed, fmt.Errorf("decoding upstream response: %w", err)
	}
	if parsed.Status != "OK" {
		return parsed, fmt.Errorf("upstream API returned status %q", parsed.Status)
	}
	return parsed, nil
}

func newTwilightWindow(start, end time.Time) twilightWindow {
	span := end.Sub(start)
	if span < 0 {
		span = -span
	}
	return twilightWindow{
		Start:    start.Format(time.RFC3339),
		End:      end.Format(time.RFC3339),
		Duration: span.String(),
	}
}

func morningGoldenHour(sunrise, civilTwilightBegin time.Time) twilightWindow {
	span := sunrise.Sub(civilTwilightBegin)
	return newTwilightWindow(sunrise, sunrise.Add(span))
}

func eveningGoldenHour(sunset, civilTwilightEnd time.Time) twilightWindow {
	span := civilTwilightEnd.Sub(sunset)
	return newTwilightWindow(sunset.Add(-span), sunset)
}

// parseCoords reads and validates the `lat`/`lng` query parameters as decimal
// degrees. The returned error message is safe to send back to the client.
func parseCoords(r *http.Request) (lat, lng float64, err error) {
	latStr := r.URL.Query().Get("lat")
	lngStr := r.URL.Query().Get("lng")
	if latStr == "" || lngStr == "" {
		return 0, 0, errors.New("Missing 'lat' or 'lng' parameters")
	}
	lat, latErr := strconv.ParseFloat(latStr, 64)
	lng, lngErr := strconv.ParseFloat(lngStr, 64)
	if latErr != nil || lngErr != nil || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return 0, 0, errors.New("Invalid 'lat' or 'lng' parameters")
	}
	return lat, lng, nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	principle := unitarianUniversalistPrinciples[rand.Intn(len(unitarianUniversalistPrinciples))]
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Yes, I'm here.\n\n%s\n", principle)
}

func sunTimesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	lat := r.URL.Query().Get("lat")
	lng := r.URL.Query().Get("lng")
	if lat == "" || lng == "" {
		http.Error(w, `{"error": "Missing 'lat' or 'lng' parameters"}`, http.StatusBadRequest)
		return
	}

	upstream, err := fetchSunTimes(lat, lng)
	if err != nil {
		log.Printf("sun times lookup failed for lat=%s lng=%s: %v", lat, lng, err)
		http.Error(w, `{"error": "Failed to retrieve sun times"}`, http.StatusBadGateway)
		return
	}

	sunrise, sunriseErr := time.Parse(time.RFC3339, upstream.Results.Sunrise)
	sunset, sunsetErr := time.Parse(time.RFC3339, upstream.Results.Sunset)
	solarNoon, solarNoonErr := time.Parse(time.RFC3339, upstream.Results.SolarNoon)
	civilTwilightBegin, civilBeginErr := time.Parse(time.RFC3339, upstream.Results.CivilTwilightBegin)
	civilTwilightEnd, civilEndErr := time.Parse(time.RFC3339, upstream.Results.CivilTwilightEnd)
	if sunriseErr != nil || sunsetErr != nil || solarNoonErr != nil || civilBeginErr != nil || civilEndErr != nil {
		log.Printf("failed to parse upstream timestamps for lat=%s lng=%s: sunrise=%v sunset=%v solarNoon=%v civilBegin=%v civilEnd=%v",
			lat, lng, sunriseErr, sunsetErr, solarNoonErr, civilBeginErr, civilEndErr)
		http.Error(w, `{"error": "Unexpected upstream response format"}`, http.StatusBadGateway)
		return
	}

	response := SunTimesResponse{
		Status:            "success",
		Sunrise:           sunrise.Format(time.RFC3339),
		Sunset:            sunset.Format(time.RFC3339),
		SolarNoon:         solarNoon.Format(time.RFC3339),
		DayLength:         (time.Duration(upstream.Results.DayLength) * time.Second).String(),
		MorningBlueHour:   newTwilightWindow(civilTwilightBegin, sunrise),
		MorningGoldenHour: morningGoldenHour(sunrise, civilTwilightBegin),
		EveningGoldenHour: eveningGoldenHour(sunset, civilTwilightEnd),
		EveningBlueHour:   newTwilightWindow(sunset, civilTwilightEnd),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("failed to encode response for lat=%s lng=%s: %v", lat, lng, err)
	}
}

func calendarHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	lat, lng, err := parseCoords(r)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusBadRequest)
		return
	}

	phases, err := parsePhases(r.URL.Query().Get("phases"))
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusBadRequest)
		return
	}
	if !phases.any() {
		http.Error(w, `{"error": "No phases selected"}`, http.StatusBadRequest)
		return
	}

	year := time.Now().UTC().Year()
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		parsed, err := strconv.Atoi(yearStr)
		if err != nil || parsed < 1970 || parsed > 9999 {
			http.Error(w, `{"error": "Invalid 'year' parameter"}`, http.StatusBadRequest)
			return
		}
		year = parsed
	}

	var loc *time.Location
	if tz := r.URL.Query().Get("tz"); tz != "" {
		loc, err = time.LoadLocation(tz)
		if err != nil {
			http.Error(w, `{"error": "Invalid 'tz' parameter, expected an IANA name like America/New_York"}`, http.StatusBadRequest)
			return
		}
	}

	events := buildEvents(year, lat, lng, phases)
	calName := fmt.Sprintf("Sun Times %d (%.4f, %.4f)", year, lat, lng)
	ics := writeICS(calName, events, loc)

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="sun-times-%d.ics"`, year))
	fmt.Fprint(w, ics)
}

// fieldSet records which solar values the caller asked /api/solar to return.
type fieldSet struct {
	sunrise       bool
	sunset        bool
	solarNoon     bool
	dayLength     bool
	morningBlue   bool
	morningGolden bool
	eveningGolden bool
	eveningBlue   bool
}

func (f fieldSet) any() bool {
	return f.sunrise || f.sunset || f.solarNoon || f.dayLength ||
		f.morningBlue || f.morningGolden || f.eveningGolden || f.eveningBlue
}

// parseSolarFields turns the `field` query value into a fieldSet. An empty value
// or "all" selects everything. Unknown tokens are reported so the caller can
// return a clear 400.
func parseSolarFields(raw string) (fieldSet, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" || raw == "all" {
		return fieldSet{true, true, true, true, true, true, true, true}, nil
	}

	var f fieldSet
	for _, token := range strings.Split(raw, ",") {
		switch strings.TrimSpace(token) {
		case "sunrise":
			f.sunrise = true
		case "sunset":
			f.sunset = true
		case "solar_noon", "solarnoon":
			f.solarNoon = true
		case "day_length", "daylength":
			f.dayLength = true
		case "morning_blue_hour":
			f.morningBlue = true
		case "morning_golden_hour":
			f.morningGolden = true
		case "evening_golden_hour":
			f.eveningGolden = true
		case "evening_blue_hour":
			f.eveningBlue = true
		case "":
			continue
		default:
			return fieldSet{}, fmt.Errorf("unknown field %q", strings.TrimSpace(token))
		}
	}
	return f, nil
}

// rangeWindow renders a timeRange as the JSON window shape used across the API.
// The duration is rounded to whole seconds; sub-second precision from the solar
// math is noise at this scale.
func rangeWindow(r timeRange) twilightWindow {
	return twilightWindow{
		Start:    r.Start.UTC().Format(time.RFC3339),
		End:      r.End.UTC().Format(time.RFC3339),
		Duration: r.Duration().Round(time.Second).String(),
	}
}

// solarHandler returns individual sun values (or all of them) for a location on
// a given date, computed locally. Select values with `field` (comma-separated:
// sunrise, sunset, solar_noon, day_length, morning_blue_hour,
// morning_golden_hour, evening_golden_hour, evening_blue_hour); omit it for all.
func solarHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	lat, lng, err := parseCoords(r)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusBadRequest)
		return
	}

	date := time.Now().UTC()
	if ds := r.URL.Query().Get("date"); ds != "" {
		parsed, perr := time.Parse("2006-01-02", ds)
		if perr != nil {
			http.Error(w, `{"error": "Invalid 'date' parameter, expected YYYY-MM-DD"}`, http.StatusBadRequest)
			return
		}
		date = parsed
	}

	fields, err := parseSolarFields(r.URL.Query().Get("field"))
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusBadRequest)
		return
	}
	if !fields.any() {
		http.Error(w, `{"error": "No fields selected"}`, http.StatusBadRequest)
		return
	}

	day := computeSolarDay(date, lat, lng)
	if !day.ok {
		http.Error(w, `{"error": "The sun does not rise and set at this location on this date"}`, http.StatusUnprocessableEntity)
		return
	}

	out := map[string]any{}
	if fields.sunrise {
		out["sunrise"] = day.Sunrise().UTC().Format(time.RFC3339)
	}
	if fields.sunset {
		out["sunset"] = day.Sunset().UTC().Format(time.RFC3339)
	}
	if fields.solarNoon {
		out["solar_noon"] = day.SolarNoon().UTC().Format(time.RFC3339)
	}
	if fields.dayLength {
		out["day_length"] = day.DayLength().Round(time.Second).String()
	}
	if fields.morningBlue {
		out["morning_blue_hour"] = rangeWindow(day.MorningBlueHour())
	}
	if fields.morningGolden {
		out["morning_golden_hour"] = rangeWindow(day.MorningGoldenHour())
	}
	if fields.eveningGolden {
		out["evening_golden_hour"] = rangeWindow(day.EveningGoldenHour())
	}
	if fields.eveningBlue {
		out["evening_blue_hour"] = rangeWindow(day.EveningBlueHour())
	}

	if err := json.NewEncoder(w).Encode(out); err != nil {
		log.Printf("failed to encode solar response for lat=%f lng=%f: %v", lat, lng, err)
	}
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/suntimes", sunTimesHandler)
	http.HandleFunc("/api/calendar", calendarHandler)
	http.HandleFunc("/api/solar", solarHandler)

	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}
	log.Printf("sun-calculator-be listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
