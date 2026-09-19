package main

import (
	"strings"
	"testing"
	"time"
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
	// A full non-leap year at a mid-latitude location: every UTC day yields one
	// sunrise and one sunset, so "all" produces 6 events per day.
	events := buildEvents(2026, 42.3601, -71.0589, phaseSet{sunrise: true, sunset: true, blueHour: true, goldenHour: true})
	if len(events) != 365*6 {
		t.Fatalf("expected %d events, got %d", 365*6, len(events))
	}

	sunriseOnly := buildEvents(2026, 42.3601, -71.0589, phaseSet{sunrise: true})
	if len(sunriseOnly) != 365 {
		t.Fatalf("expected 365 sunrise events, got %d", len(sunriseOnly))
	}
}

func TestWriteICSStructure(t *testing.T) {
	events := buildEvents(2026, 42.3601, -71.0589, phaseSet{sunrise: true})
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
