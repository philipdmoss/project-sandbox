package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
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

type twilightWindow struct {
	Start    string `json:"start"`
	End      string `json:"end"`
	Duration string `json:"duration"`
}

// SunTimesResponse is the /api/suntimes payload. Solar noon always occurs;
// every other field is omitted when the sun does not produce it that day (high
// latitudes), so consumers see only values that actually exist.
type SunTimesResponse struct {
	Status            string          `json:"status"`
	Sunrise           string          `json:"sunrise,omitempty"`
	Sunset            string          `json:"sunset,omitempty"`
	SolarNoon         string          `json:"solar_noon"`
	DayLength         string          `json:"day_length,omitempty"`
	MorningBlueHour   *twilightWindow `json:"morning_blue_hour,omitempty"`
	MorningGoldenHour *twilightWindow `json:"morning_golden_hour,omitempty"`
	EveningGoldenHour *twilightWindow `json:"evening_golden_hour,omitempty"`
	EveningBlueHour   *twilightWindow `json:"evening_blue_hour,omitempty"`
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
	// ParseFloat accepts "NaN"/"Inf", and every range comparison against NaN is
	// false, so reject non-finite values explicitly before the range check.
	if latErr != nil || lngErr != nil ||
		math.IsNaN(lat) || math.IsNaN(lng) || math.IsInf(lat, 0) || math.IsInf(lng, 0) ||
		lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return 0, 0, errors.New("Invalid 'lat' or 'lng' parameters")
	}
	return lat, lng, nil
}

// writeJSONError sends {"error": msg} with the correct JSON content type and
// status. Using the encoder keeps the body valid JSON even if msg contains
// characters that need escaping.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	principle := unitarianUniversalistPrinciples[rand.Intn(len(unitarianUniversalistPrinciples))]
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Yes, I'm here.\n\n%s\n", principle)
}

// sunTimesHandler returns today's sun times for a location as JSON, computed
// locally with the NOAA solar equations (no external API call).
func sunTimesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	lat, lng, err := parseCoords(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	day := computeSolarDay(time.Now().UTC(), lat, lng)
	sunrise, hasSunrise := day.Sunrise()
	sunset, hasSunset := day.Sunset()
	if !hasSunrise && !hasSunset {
		writeJSONError(w, http.StatusUnprocessableEntity,
			"The sun does not rise and set at this location today")
		return
	}

	response := SunTimesResponse{
		Status:    "success",
		SolarNoon: day.SolarNoon().UTC().Format(time.RFC3339),
	}
	if hasSunrise {
		response.Sunrise = sunrise.UTC().Format(time.RFC3339)
	}
	if hasSunset {
		response.Sunset = sunset.UTC().Format(time.RFC3339)
	}
	if d, ok := day.DayLength(); ok {
		response.DayLength = d.Round(time.Second).String()
	}
	if win, ok := day.MorningBlueHour(); ok {
		response.MorningBlueHour = ptrWindow(win)
	}
	if win, ok := day.MorningGoldenHour(); ok {
		response.MorningGoldenHour = ptrWindow(win)
	}
	if win, ok := day.EveningGoldenHour(); ok {
		response.EveningGoldenHour = ptrWindow(win)
	}
	if win, ok := day.EveningBlueHour(); ok {
		response.EveningBlueHour = ptrWindow(win)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("failed to encode suntimes response for lat=%f lng=%f: %v", lat, lng, err)
	}
}

func ptrWindow(r timeRange) *twilightWindow {
	w := rangeWindow(r)
	return &w
}

// parseCalendarParams reads the lat/lng, phases, and optional tz shared by the
// calendar endpoints. The returned error message is safe to send to the client.
func parseCalendarParams(r *http.Request) (lat, lng float64, phases phaseSet, loc *time.Location, err error) {
	lat, lng, err = parseCoords(r)
	if err != nil {
		return
	}
	phases, err = parsePhases(r.URL.Query().Get("phases"))
	if err != nil {
		return
	}
	if !phases.any() {
		err = errors.New("No phases selected")
		return
	}
	if tz := r.URL.Query().Get("tz"); tz != "" {
		loc, err = time.LoadLocation(tz)
		if err != nil {
			err = errors.New("Invalid 'tz' parameter, expected an IANA name like America/New_York")
			return
		}
	}
	return
}

// calendarHandler serves one year (default current) as a downloadable .ics file.
func calendarHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	lat, lng, phases, loc, err := parseCalendarParams(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	year := time.Now().UTC().Year()
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		parsed, perr := strconv.Atoi(yearStr)
		if perr != nil || parsed < 1970 || parsed > 9999 {
			writeJSONError(w, http.StatusBadRequest, "Invalid 'year' parameter")
			return
		}
		year = parsed
	}

	events := buildEvents(year, lat, lng, phases, loc)
	calName := fmt.Sprintf("Sun Times %d (%.4f, %.4f)", year, lat, lng)
	ics := writeICS(calName, events, loc)

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="sun-times-%d.ics"`, year))
	fmt.Fprint(w, ics)
}

// calendarFeedHandler serves a rolling window (current year plus next year) as a
// subscribable calendar — no attachment header, recomputed each request — so a
// calendar app pointed at this URL (e.g. via webcal://) always stays current.
func calendarFeedHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	lat, lng, phases, loc, err := parseCalendarParams(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	thisYear := time.Now().UTC().Year()
	var events []calendarEvent
	for year := thisYear; year <= thisYear+1; year++ {
		events = append(events, buildEvents(year, lat, lng, phases, loc)...)
	}
	calName := fmt.Sprintf("Sun Times (%.4f, %.4f)", lat, lng)
	ics := writeICS(calName, events, loc)

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	fmt.Fprint(w, ics)
}

// fieldSet records which solar values the caller asked /api/solar to return.
type fieldSet struct {
	sunrise        bool
	sunset         bool
	sunriseAzimuth bool
	sunsetAzimuth  bool
	solarNoon      bool
	dayLength      bool
	morningBlue    bool
	morningGolden  bool
	eveningGolden  bool
	eveningBlue    bool
	civil          bool
	nautical       bool
	astronomical   bool
}

func (f fieldSet) any() bool {
	return f.sunrise || f.sunset || f.sunriseAzimuth || f.sunsetAzimuth ||
		f.solarNoon || f.dayLength || f.morningBlue || f.morningGolden ||
		f.eveningGolden || f.eveningBlue || f.civil || f.nautical || f.astronomical
}

func allFields() fieldSet {
	return fieldSet{
		sunrise: true, sunset: true, sunriseAzimuth: true, sunsetAzimuth: true,
		solarNoon: true, dayLength: true, morningBlue: true, morningGolden: true,
		eveningGolden: true, eveningBlue: true, civil: true, nautical: true, astronomical: true,
	}
}

// parseSolarFields turns the `field` query value into a fieldSet. An empty value
// or "all" selects everything. Unknown tokens are reported so the caller can
// return a clear 400.
func parseSolarFields(raw string) (fieldSet, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" || raw == "all" {
		return allFields(), nil
	}

	var f fieldSet
	for _, token := range strings.Split(raw, ",") {
		switch strings.TrimSpace(token) {
		case "sunrise":
			f.sunrise = true
		case "sunset":
			f.sunset = true
		case "sunrise_azimuth":
			f.sunriseAzimuth = true
		case "sunset_azimuth":
			f.sunsetAzimuth = true
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
		case "civil_twilight":
			f.civil = true
		case "nautical_twilight":
			f.nautical = true
		case "astronomical_twilight":
			f.astronomical = true
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

// twilightTimes is the dawn/dusk pair for a twilight depression. Either instant
// is omitted when it does not occur that day.
type twilightTimes struct {
	Dawn string `json:"dawn,omitempty"`
	Dusk string `json:"dusk,omitempty"`
}

// twilightPair builds a twilightTimes from a dawn/dusk accessor pair, and
// reports whether either instant occurred.
func twilightPair(dawn, dusk func() (time.Time, bool)) (twilightTimes, bool) {
	var tt twilightTimes
	present := false
	if t, ok := dawn(); ok {
		tt.Dawn = t.UTC().Format(time.RFC3339)
		present = true
	}
	if t, ok := dusk(); ok {
		tt.Dusk = t.UTC().Format(time.RFC3339)
		present = true
	}
	return tt, present
}

func round2(x float64) float64 { return math.Round(x*100) / 100 }

// solarHandler returns individual sun values (or all of them) for a location on
// a given date, computed locally. Select values with `field` (comma-separated:
// sunrise, sunset, solar_noon, day_length, morning_blue_hour,
// morning_golden_hour, evening_golden_hour, evening_blue_hour); omit it for all.
func solarHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	lat, lng, err := parseCoords(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	date := time.Now().UTC()
	if ds := r.URL.Query().Get("date"); ds != "" {
		parsed, perr := time.Parse("2006-01-02", ds)
		if perr != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid 'date' parameter, expected YYYY-MM-DD")
			return
		}
		date = parsed
	}

	fields, err := parseSolarFields(r.URL.Query().Get("field"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !fields.any() {
		writeJSONError(w, http.StatusBadRequest, "No fields selected")
		return
	}

	// Optional arbitrary-elevation query (?elevation=<deg>), independent of the
	// field filter: returns the rising/setting crossing times for that angle.
	var elevation *float64
	if es := r.URL.Query().Get("elevation"); es != "" {
		v, perr := strconv.ParseFloat(es, 64)
		if perr != nil || v < -90 || v > 90 {
			writeJSONError(w, http.StatusBadRequest, "Invalid 'elevation' parameter, expected degrees in [-90, 90]")
			return
		}
		elevation = &v
	}

	day := computeSolarDay(date, lat, lng)

	// Include only the requested values that actually occur on this day; a
	// requested twilight window or sunrise may be absent at high latitudes.
	out := map[string]any{}
	if fields.sunrise {
		if t, ok := day.Sunrise(); ok {
			out["sunrise"] = t.UTC().Format(time.RFC3339)
		}
	}
	if fields.sunset {
		if t, ok := day.Sunset(); ok {
			out["sunset"] = t.UTC().Format(time.RFC3339)
		}
	}
	if fields.sunriseAzimuth {
		if t, ok := day.Sunrise(); ok {
			_, az := sunPosition(t, lat, lng)
			out["sunrise_azimuth"] = round2(az)
		}
	}
	if fields.sunsetAzimuth {
		if t, ok := day.Sunset(); ok {
			_, az := sunPosition(t, lat, lng)
			out["sunset_azimuth"] = round2(az)
		}
	}
	if fields.solarNoon {
		out["solar_noon"] = day.SolarNoon().UTC().Format(time.RFC3339)
	}
	if fields.dayLength {
		if d, ok := day.DayLength(); ok {
			out["day_length"] = d.Round(time.Second).String()
		}
	}
	if fields.morningBlue {
		if win, ok := day.MorningBlueHour(); ok {
			out["morning_blue_hour"] = rangeWindow(win)
		}
	}
	if fields.morningGolden {
		if win, ok := day.MorningGoldenHour(); ok {
			out["morning_golden_hour"] = rangeWindow(win)
		}
	}
	if fields.eveningGolden {
		if win, ok := day.EveningGoldenHour(); ok {
			out["evening_golden_hour"] = rangeWindow(win)
		}
	}
	if fields.eveningBlue {
		if win, ok := day.EveningBlueHour(); ok {
			out["evening_blue_hour"] = rangeWindow(win)
		}
	}
	if fields.civil {
		if tt, ok := twilightPair(day.CivilDawn, day.CivilDusk); ok {
			out["civil_twilight"] = tt
		}
	}
	if fields.nautical {
		if tt, ok := twilightPair(day.NauticalDawn, day.NauticalDusk); ok {
			out["nautical_twilight"] = tt
		}
	}
	if fields.astronomical {
		if tt, ok := twilightPair(day.AstronomicalDawn, day.AstronomicalDusk); ok {
			out["astronomical_twilight"] = tt
		}
	}
	if elevation != nil {
		cross := map[string]any{"degrees": *elevation}
		if t, ok := day.Crossing(*elevation, true); ok {
			cross["rising"] = t.UTC().Format(time.RFC3339)
		}
		if t, ok := day.Crossing(*elevation, false); ok {
			cross["setting"] = t.UTC().Format(time.RFC3339)
		}
		out["elevation"] = cross
	}

	if len(out) == 0 {
		writeJSONError(w, http.StatusUnprocessableEntity,
			"None of the requested values occur at this location on this date")
		return
	}

	if err := json.NewEncoder(w).Encode(out); err != nil {
		log.Printf("failed to encode solar response for lat=%f lng=%f: %v", lat, lng, err)
	}
}

// positionHandler returns the sun's altitude and azimuth at a location for an
// instant (`time` RFC3339, default now). Azimuth is degrees clockwise from north.
func positionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	lat, lng, err := parseCoords(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	t := time.Now().UTC()
	if ts := r.URL.Query().Get("time"); ts != "" {
		parsed, perr := time.Parse(time.RFC3339, ts)
		if perr != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid 'time' parameter, expected RFC3339")
			return
		}
		t = parsed
	}

	altitude, azimuth := sunPosition(t, lat, lng)
	out := map[string]any{
		"time":     t.UTC().Format(time.RFC3339),
		"altitude": round2(altitude),
		"azimuth":  round2(azimuth),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(out); err != nil {
		log.Printf("failed to encode position response for lat=%f lng=%f: %v", lat, lng, err)
	}
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/suntimes", sunTimesHandler)
	http.HandleFunc("/api/calendar", calendarHandler)
	http.HandleFunc("/api/calendar/feed", calendarFeedHandler)
	http.HandleFunc("/api/solar", solarHandler)
	http.HandleFunc("/api/position", positionHandler)

	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}

	// Explicit timeouts so a slow or idle client cannot hold a connection open
	// indefinitely (the zero-value http.Server has none).
	srv := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("sun-calculator-be listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}
