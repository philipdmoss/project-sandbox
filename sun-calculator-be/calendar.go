package main

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// markerDuration is the visible length given to an event that would otherwise
// be a single instant (e.g. a morning that contains only sunrise), so it
// renders as a clickable block in calendar day/week views.
const markerDuration = 15 * time.Minute

// calendarEvent is a single VEVENT to be written into the ICS output.
type calendarEvent struct {
	uid         string
	summary     string
	description string
	start       time.Time
	end         time.Time
}

// dayPiece is one requested phase on one day: a twilight range, or a point
// (sunrise/sunset) where start == end.
type dayPiece struct {
	name  string
	start time.Time
	end   time.Time
	point bool
}

// phaseSet records which categories of event the caller asked for.
type phaseSet struct {
	sunrise    bool
	sunset     bool
	blueHour   bool
	goldenHour bool
}

func (p phaseSet) any() bool {
	return p.sunrise || p.sunset || p.blueHour || p.goldenHour
}

// parsePhases turns the `phases` query value into a phaseSet. An empty value or
// "all" selects everything. Unknown tokens are reported so the caller can
// return a clear 400 rather than silently producing an empty calendar.
func parsePhases(raw string) (phaseSet, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" || raw == "all" {
		return phaseSet{sunrise: true, sunset: true, blueHour: true, goldenHour: true}, nil
	}

	var phases phaseSet
	for _, token := range strings.Split(raw, ",") {
		switch strings.TrimSpace(token) {
		case "sunrise":
			phases.sunrise = true
		case "sunset":
			phases.sunset = true
		case "blue_hour", "bluehour", "blue":
			phases.blueHour = true
		case "golden_hour", "goldenhour", "golden":
			phases.goldenHour = true
		case "":
			continue
		default:
			return phaseSet{}, fmt.Errorf("unknown phase %q", strings.TrimSpace(token))
		}
	}
	return phases, nil
}

// buildEvents produces the calendar for the whole given year at the location.
// Each day yields at most two events: the requested morning phases merged into
// one event and the requested evening phases merged into one. A merged event
// spans from the earliest requested piece's start to the latest piece's end,
// and its description lists each piece's times. Pieces that do not occur on a
// given day (high-latitude summer or winter) are simply left out, so a
// morning can still appear with only a sunrise when there is no twilight.
func buildEvents(year int, lat, lng float64, phases phaseSet, loc *time.Location) []calendarEvent {
	var events []calendarEvent
	day := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC)

	for ; day.Before(end); day = day.AddDate(0, 0, 1) {
		s := computeSolarDay(day, lat, lng)
		stamp := day.Format("20060102")

		if e, ok := combineHalf(morningPieces(s, phases), "🌅", "morning", stamp, lat, lng, loc); ok {
			events = append(events, e)
		}
		if e, ok := combineHalf(eveningPieces(s, phases), "🌇", "evening", stamp, lat, lng, loc); ok {
			events = append(events, e)
		}
	}
	return events
}

// morningPieces returns the requested, occurring morning phases in chronological
// order: blue hour, sunrise, golden hour.
func morningPieces(s solarDay, phases phaseSet) []dayPiece {
	var pieces []dayPiece
	if phases.blueHour {
		if r, ok := s.MorningBlueHour(); ok {
			pieces = append(pieces, dayPiece{name: "Blue Hour", start: r.Start, end: r.End})
		}
	}
	if phases.sunrise {
		if t, ok := s.Sunrise(); ok {
			pieces = append(pieces, dayPiece{name: "Sunrise", start: t, end: t, point: true})
		}
	}
	if phases.goldenHour {
		if r, ok := s.MorningGoldenHour(); ok {
			pieces = append(pieces, dayPiece{name: "Golden Hour", start: r.Start, end: r.End})
		}
	}
	return pieces
}

// eveningPieces returns the requested, occurring evening phases in chronological
// order: golden hour, sunset, blue hour.
func eveningPieces(s solarDay, phases phaseSet) []dayPiece {
	var pieces []dayPiece
	if phases.goldenHour {
		if r, ok := s.EveningGoldenHour(); ok {
			pieces = append(pieces, dayPiece{name: "Golden Hour", start: r.Start, end: r.End})
		}
	}
	if phases.sunset {
		if t, ok := s.Sunset(); ok {
			pieces = append(pieces, dayPiece{name: "Sunset", start: t, end: t, point: true})
		}
	}
	if phases.blueHour {
		if r, ok := s.EveningBlueHour(); ok {
			pieces = append(pieces, dayPiece{name: "Blue Hour", start: r.Start, end: r.End})
		}
	}
	return pieces
}

// combineHalf merges the pieces of one half-day into a single event spanning
// their full extent, titled with the piece names and described line by line.
// Returns false when there are no pieces.
func combineHalf(pieces []dayPiece, emoji, kind, stamp string, lat, lng float64, loc *time.Location) (calendarEvent, bool) {
	if len(pieces) == 0 {
		return calendarEvent{}, false
	}

	start, end := pieces[0].start, pieces[0].end
	names := make([]string, 0, len(pieces))
	lines := make([]string, 0, len(pieces))
	for _, p := range pieces {
		if p.start.Before(start) {
			start = p.start
		}
		if p.end.After(end) {
			end = p.end
		}
		names = append(names, p.name)
		lines = append(lines, describePiece(p, loc))
	}
	// A half with only point pieces (e.g. sunrise alone) has zero duration; give
	// it a visible block.
	if !end.After(start) {
		end = start.Add(markerDuration)
	}

	return calendarEvent{
		uid:         eventUID(kind, stamp, lat, lng),
		summary:     emoji + " " + strings.Join(names, " · "),
		description: strings.Join(lines, "\n"),
		start:       start,
		end:         end,
	}, true
}

// describePiece renders one piece for the event description, e.g.
// "Blue Hour: 05:22–05:51 (28m)" or "Sunrise: 05:51".
func describePiece(p dayPiece, loc *time.Location) string {
	if p.point {
		return fmt.Sprintf("%s: %s", p.name, clockTime(p.start, loc))
	}
	return fmt.Sprintf("%s: %s–%s (%s)", p.name,
		clockTime(p.start, loc), clockTime(p.end, loc), compactDuration(p.end.Sub(p.start)))
}

// clockTime formats an instant as HH:MM in the display zone (loc if given, else
// UTC), matching how the event's start/end are written.
func clockTime(t time.Time, loc *time.Location) string {
	if loc != nil {
		return t.In(loc).Format("15:04")
	}
	return t.UTC().Format("15:04")
}

// compactDuration formats a duration rounded to the minute, e.g. "28m" or "1h5m".
func compactDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func eventUID(kind, stamp string, lat, lng float64) string {
	return fmt.Sprintf("%s-%s-%.4f-%.4f@sun-calculator", kind, stamp, lat, lng)
}

// writeICS renders the events as an RFC 5545 iCalendar document. When loc is
// non-nil, event times are written as floating local wall-clock times in that
// zone; otherwise they are written as absolute UTC instants (with a Z suffix).
func writeICS(calName string, events []calendarEvent, loc *time.Location) string {
	var b strings.Builder
	writeLine(&b, "BEGIN:VCALENDAR")
	writeLine(&b, "VERSION:2.0")
	writeLine(&b, "PRODID:-//sun-calculator//Sun Times Calendar//EN")
	writeLine(&b, "CALSCALE:GREGORIAN")
	writeLine(&b, "METHOD:PUBLISH")
	writeLine(&b, "X-WR-CALNAME:"+escapeText(calName))

	stamp := time.Now().UTC().Format("20060102T150405Z")
	for _, e := range events {
		writeLine(&b, "BEGIN:VEVENT")
		writeLine(&b, "UID:"+e.uid)
		writeLine(&b, "DTSTAMP:"+stamp)
		writeLine(&b, "DTSTART"+icsTime(e.start, loc))
		writeLine(&b, "DTEND"+icsTime(e.end, loc))
		writeLine(&b, "SUMMARY:"+escapeText(e.summary))
		if e.description != "" {
			writeLine(&b, "DESCRIPTION:"+escapeText(e.description))
		}
		writeLine(&b, "TRANSP:TRANSPARENT")
		writeLine(&b, "END:VEVENT")
	}
	writeLine(&b, "END:VCALENDAR")
	return b.String()
}

// icsTime formats the property suffix and value for a DTSTART/DTEND line,
// e.g. ":20260101T113000Z" (UTC). We emit floating local time (no TZID) when a
// location is supplied so the wall-clock reads identically in every viewer's
// calendar.
func icsTime(t time.Time, loc *time.Location) string {
	if loc != nil {
		return ":" + t.In(loc).Format("20060102T150405")
	}
	return ":" + t.UTC().Format("20060102T150405Z")
}

// writeLine appends a content line, folding at 75 octets per RFC 5545 and
// terminating with CRLF. Folding happens on rune boundaries (never mid-UTF-8),
// and each continuation line begins with a space that counts toward its 75.
func writeLine(b *strings.Builder, line string) {
	const limit = 75
	if len(line) <= limit {
		b.WriteString(line)
		b.WriteString("\r\n")
		return
	}
	lineLen := 0
	for i, r := range line {
		rl := utf8.RuneLen(r)
		if rl < 1 {
			rl = 1
		}
		if lineLen+rl > limit {
			b.WriteString("\r\n ")
			lineLen = 1 // the leading space
		}
		b.WriteString(line[i : i+rl])
		lineLen += rl
	}
	b.WriteString("\r\n")
}

// escapeText escapes the characters that carry meaning in ICS text values.
func escapeText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
