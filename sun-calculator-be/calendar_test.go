package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestParsePhases(t *testing.T) {
	all, err := parsePhases("")
	if err != nil || !all.sunrise || !all.sunset || !all.blueHour || !all.goldenHour {
		t.Fatalf("empty should select all: %+v err=%v", all, err)
	}
	allWord, _ := parsePhases("all")
	if allWord != all {
		t.Fatalf("\"all\" should equal empty selection")
	}
	combo, err := parsePhases("sunrise, golden_hour")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !combo.sunrise || !combo.goldenHour || combo.sunset || combo.blueHour {
		t.Fatalf("combo parsed wrong: %+v", combo)
	}
	if _, err := parsePhases("sunrise,banana"); err == nil {
		t.Fatalf("expected error for unknown phase")
	}
	// A selection that resolves to nothing is reported as empty by any().
	empty, err := parsePhases(",")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if empty.any() {
		t.Fatalf("expected empty selection to report any()==false")
	}
}

func TestBuildEventsCounts(t *testing.T) {
	// A full non-leap year at a mid-latitude location: morning and evening
	// phases each merge into one event, so "all" produces 2 events per day.
	events := buildEvents(2026, 42.3601, -71.0589,
		phaseSet{sunrise: true, sunset: true, blueHour: true, goldenHour: true}, nil)
	if len(events) != 365*2 {
		t.Fatalf("expected %d events, got %d", 365*2, len(events))
	}

	// Sunrise is a morning-only phase, so a sunrise-only calendar is one event
	// per day (no evening event).
	sunriseOnly := buildEvents(2026, 42.3601, -71.0589, phaseSet{sunrise: true}, nil)
	if len(sunriseOnly) != 365 {
		t.Fatalf("expected 365 sunrise events, got %d", len(sunriseOnly))
	}
}

func TestBuildEventsMerging(t *testing.T) {
	// Boston, midsummer: all phases present. Morning event should span civil
	// dawn -> morning golden-hour end and list all three pieces.
	d, _ := time.Parse("2006-01-02", "2026-06-21")
	s := computeSolarDay(d, 42.3601, -71.0589)
	all := phaseSet{sunrise: true, sunset: true, blueHour: true, goldenHour: true}

	events := buildEvents(2026, 42.3601, -71.0589, all, nil)
	// Find the June 21 morning event by UID.
	stamp := d.Format("20060102")
	morningUID := eventUID("morning", stamp, 42.3601, -71.0589)
	var morning *calendarEvent
	for i := range events {
		if events[i].uid == morningUID {
			morning = &events[i]
			break
		}
	}
	if morning == nil {
		t.Fatalf("no morning event for %s", stamp)
	}

	mb, _ := s.MorningBlueHour()
	mg, _ := s.MorningGoldenHour()
	if !morning.start.Equal(mb.Start) {
		t.Errorf("morning start = %s, want civil dawn %s", morning.start, mb.Start)
	}
	if !morning.end.Equal(mg.End) {
		t.Errorf("morning end = %s, want golden-hour end %s", morning.end, mg.End)
	}
	if want := "🌅 Blue Hour · Sunrise · Golden Hour"; morning.summary != want {
		t.Errorf("summary = %q, want %q", morning.summary, want)
	}
	for _, piece := range []string{"Blue Hour:", "Sunrise:", "Golden Hour:"} {
		if !strings.Contains(morning.description, piece) {
			t.Errorf("description missing %q: %q", piece, morning.description)
		}
	}

	// Evening blue+sunset should span sunset -> civil dusk (the example that
	// prompted the feature).
	eve := buildEvents(2026, 42.3601, -71.0589, phaseSet{blueHour: true, sunset: true}, nil)
	eveningUID := eventUID("evening", stamp, 42.3601, -71.0589)
	var evening *calendarEvent
	for i := range eve {
		if eve[i].uid == eveningUID {
			evening = &eve[i]
			break
		}
	}
	if evening == nil {
		t.Fatalf("no evening event for %s", stamp)
	}
	sunset, _ := s.Sunset()
	eb, _ := s.EveningBlueHour()
	if !evening.start.Equal(sunset) {
		t.Errorf("evening start = %s, want sunset %s", evening.start, sunset)
	}
	if !evening.end.Equal(eb.End) {
		t.Errorf("evening end = %s, want civil dusk %s", evening.end, eb.End)
	}
}

func TestWriteLineFolding(t *testing.T) {
	// A line well over 75 octets, built from 4-byte emoji so a naive byte-slice
	// fold would split a rune.
	long := "SUMMARY:" + strings.Repeat("🌅", 40) // 8 + 160 = 168 octets
	var b strings.Builder
	writeLine(&b, long)
	out := b.String()

	for _, line := range strings.Split(strings.TrimRight(out, "\r\n"), "\r\n") {
		if len(line) > 75 {
			t.Errorf("folded line exceeds 75 octets: %d", len(line))
		}
	}
	// Unfolding (drop each CRLF + leading space) reconstructs the original.
	unfolded := strings.TrimRight(strings.ReplaceAll(out, "\r\n ", ""), "\r\n")
	if unfolded != long {
		t.Errorf("unfold mismatch:\n got %q\nwant %q", unfolded, long)
	}
	if !utf8.ValidString(unfolded) {
		t.Errorf("folding split a multi-byte rune")
	}
}

func TestCalendarHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet,
			"/api/calendar?lat=42.3601&lng=-71.0589&phases=sunrise&year=2026&tz=America/New_York", nil)
		rec := httptest.NewRecorder()
		calendarHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/calendar") {
			t.Errorf("Content-Type = %q, want text/calendar", ct)
		}
		if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "sun-times-2026.ics") {
			t.Errorf("Content-Disposition = %q", cd)
		}
		if !strings.HasPrefix(rec.Body.String(), "BEGIN:VCALENDAR\r\n") {
			t.Errorf("body is not an iCalendar document")
		}
	})

	errorCases := []struct {
		name string
		url  string
		want int
	}{
		{"missing coords", "/api/calendar", http.StatusBadRequest},
		{"NaN coords", "/api/calendar?lat=NaN&lng=NaN", http.StatusBadRequest},
		{"bad phase", "/api/calendar?lat=1&lng=1&phases=banana", http.StatusBadRequest},
		{"empty phases", "/api/calendar?lat=1&lng=1&phases=,", http.StatusBadRequest},
		{"bad year", "/api/calendar?lat=1&lng=1&year=abc", http.StatusBadRequest},
		{"year out of range", "/api/calendar?lat=1&lng=1&year=1000", http.StatusBadRequest},
		{"bad tz", "/api/calendar?lat=1&lng=1&tz=Mars/Olympus", http.StatusBadRequest},
	}
	for _, c := range errorCases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, c.url, nil)
			rec := httptest.NewRecorder()
			calendarHandler(rec, req)
			if rec.Code != c.want {
				t.Fatalf("got %d want %d: %s", rec.Code, c.want, rec.Body.String())
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				t.Errorf("error Content-Type = %q, want application/json", ct)
			}
		})
	}
}

func TestWriteICSStructure(t *testing.T) {
	events := buildEvents(2026, 42.3601, -71.0589, phaseSet{sunrise: true}, nil)
	ics := writeICS("Test", events, nil)

	if !strings.HasPrefix(ics, "BEGIN:VCALENDAR\r\n") {
		t.Fatalf("missing calendar header")
	}
	if !strings.HasSuffix(ics, "END:VCALENDAR\r\n") {
		t.Fatalf("missing calendar footer")
	}
	if strings.Count(ics, "BEGIN:VEVENT") != 365 {
		t.Fatalf("expected 365 VEVENTs, got %d", strings.Count(ics, "BEGIN:VEVENT"))
	}
	// UTC instants carry a trailing Z, and every event has a DTSTART and DTEND.
	if !strings.Contains(ics, "DTSTART:2026") || !strings.Contains(ics, "Z\r\n") {
		t.Fatalf("expected UTC DTSTART values")
	}
	if strings.Count(ics, "DTSTART") != 365 || strings.Count(ics, "DTEND") != 365 {
		t.Fatalf("expected 365 DTSTART and 365 DTEND, got %d/%d",
			strings.Count(ics, "DTSTART"), strings.Count(ics, "DTEND"))
	}

	// With a location, times are written as floating local wall-clock (no Z).
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("tz database unavailable: %v", err)
	}
	localICS := writeICS("Test", events, loc)
	for _, line := range strings.Split(localICS, "\r\n") {
		if strings.HasPrefix(line, "DTSTART:") && strings.HasSuffix(line, "Z") {
			t.Fatalf("local calendar should not emit UTC Z times: %q", line)
		}
	}
}
