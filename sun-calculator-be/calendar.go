package main

import (
	"fmt"
	"strings"
	"time"
)

// markerDuration is the visible length given to the sunrise and sunset events.
// They are moments in time, but a short block renders as a clickable event in
// calendar day/week views rather than a hard-to-see instant.
const markerDuration = 15 * time.Minute

// calendarEvent is a single VEVENT to be written into the ICS output.
type calendarEvent struct {
	uid     string
	summary string
	start   time.Time
	end     time.Time
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

// buildEvents produces every requested event for the whole given year at the
// location. Days where the requested phase does not occur (polar regions) are
// skipped.
func buildEvents(year int, lat, lng float64, phases phaseSet) []calendarEvent {
	var events []calendarEvent
	day := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(year+1, time.January, 1, 0, 0, 0, 0, time.UTC)

	for ; day.Before(end); day = day.AddDate(0, 0, 1) {
		s := computeSolarDay(day, lat, lng)
		if !s.ok {
			continue
		}
		stamp := day.Format("20060102")

		if phases.sunrise {
			events = append(events, calendarEvent{
				uid:     eventUID("sunrise", stamp, lat, lng),
				summary: "🌅 Sunrise",
				start:   s.Sunrise(),
				end:     s.Sunrise().Add(markerDuration),
			})
		}
		if phases.sunset {
			events = append(events, calendarEvent{
				uid:     eventUID("sunset", stamp, lat, lng),
				summary: "🌇 Sunset",
				start:   s.Sunset(),
				end:     s.Sunset().Add(markerDuration),
			})
		}
		if phases.blueHour {
			morning := s.MorningBlueHour()
			evening := s.EveningBlueHour()
			events = append(events, calendarEvent{
				uid:     eventUID("morning-blue", stamp, lat, lng),
				summary: "🔵 Morning Blue Hour",
				start:   morning.Start,
				end:     morning.End,
			})
			events = append(events, calendarEvent{
				uid:     eventUID("evening-blue", stamp, lat, lng),
				summary: "🔵 Evening Blue Hour",
				start:   evening.Start,
				end:     evening.End,
			})
		}
		if phases.goldenHour {
			morning := s.MorningGoldenHour()
			evening := s.EveningGoldenHour()
			events = append(events, calendarEvent{
				uid:     eventUID("morning-golden", stamp, lat, lng),
				summary: "🟡 Morning Golden Hour",
				start:   morning.Start,
				end:     morning.End,
			})
			events = append(events, calendarEvent{
				uid:     eventUID("evening-golden", stamp, lat, lng),
				summary: "🟡 Evening Golden Hour",
				start:   evening.Start,
				end:     evening.End,
			})
		}
	}
	return events
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
// terminating with CRLF.
func writeLine(b *strings.Builder, line string) {
	const limit = 75
	if len(line) <= limit {
		b.WriteString(line)
		b.WriteString("\r\n")
		return
	}
	b.WriteString(line[:limit])
	b.WriteString("\r\n")
	rest := line[limit:]
	for len(rest) > limit-1 {
		b.WriteString(" ")
		b.WriteString(rest[:limit-1])
		b.WriteString("\r\n")
		rest = rest[limit-1:]
	}
	b.WriteString(" ")
	b.WriteString(rest)
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
