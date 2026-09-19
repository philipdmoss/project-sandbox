package main

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Solar elevation angles (degrees below the horizon) used to define each event.
// These mirror the definitions the single-day endpoint gets from the upstream
// API: sunrise/sunset account for atmospheric refraction and the sun's radius,
// civil twilight (blue hour boundary) is 6 degrees below the horizon.
const (
	sunriseSunsetAngle  = -0.833
	civilTwilightAngle  = -6.0
	minutesPerHourAngle = 4.0 // 4 minutes of time per degree of Earth rotation
)

// markerDuration is the visible length given to the sunrise and sunset events.
// They are moments in time, but a short block renders as a clickable event in
// calendar day/week views rather than a hard-to-see instant.
const markerDuration = 15 * time.Minute

// solarDay holds the four boundary instants (all UTC) that every twilight
// window and marker for a single calendar day is derived from.
type solarDay struct {
	civilDawn time.Time
	sunrise   time.Time
	sunset    time.Time
	civilDusk time.Time
	// ok is false on days where the sun never reaches an angle (polar day or
	// polar night); such days are skipped rather than producing bogus events.
	ok bool
}

// computeSolarDay calculates sunrise, sunset and civil twilight for the given
// calendar date at a location using the NOAA solar position equations. All
// returned times are UTC instants.
//
// The date's year/month/day identify the *local* day at the location: every
// event is anchored on that day's local solar noon, so sunrise, sunset and the
// twilight windows are always grouped under the same local day even at far-east
// or far-west longitudes (anchoring on UTC midnight instead would file an early
// local sunrise under the previous UTC day).
func computeSolarDay(date time.Time, lat, lng float64) solarDay {
	// Local mean noon, expressed as a UTC instant: 12:00 local shifted by the
	// location's longitude (15 degrees per hour, east of Greenwich is earlier).
	meanNoon := time.Date(date.Year(), date.Month(), date.Day(), 12, 0, 0, 0, time.UTC).
		Add(time.Duration(-lng / 15 * float64(time.Hour)))

	sunrise, riseOK := solarEvent(meanNoon, lat, sunriseSunsetAngle, true)
	sunset, setOK := solarEvent(meanNoon, lat, sunriseSunsetAngle, false)
	civilDawn, dawnOK := solarEvent(meanNoon, lat, civilTwilightAngle, true)
	civilDusk, duskOK := solarEvent(meanNoon, lat, civilTwilightAngle, false)
	if !riseOK || !setOK || !dawnOK || !duskOK {
		return solarDay{ok: false}
	}

	return solarDay{
		civilDawn: civilDawn,
		sunrise:   sunrise,
		sunset:    sunset,
		civilDusk: civilDusk,
		ok:        true,
	}
}

// solarEvent returns the UTC instant when the sun's centre crosses elevationDeg
// while rising (or setting) on the local day whose mean solar noon is meanNoon.
// ok is false when the sun never reaches that elevation (polar day or night).
//
// The sun's declination and the equation of time drift over the hours between
// noon and the event, so we estimate the event using the sun's position at
// noon, then refine once by re-evaluating at that estimate. A single refinement
// brings the result to within a few seconds of the upstream reference globally.
func solarEvent(meanNoon time.Time, lat, elevationDeg float64, rising bool) (time.Time, bool) {
	estimate, ok := solarEventPass(meanNoon, meanNoon, lat, elevationDeg, rising)
	if !ok {
		return time.Time{}, false
	}
	return solarEventPass(meanNoon, estimate, lat, elevationDeg, rising)
}

// solarEventPass computes one estimate of the event instant, evaluating the
// sun's position at refAt. Apparent solar noon is the mean noon corrected by
// the equation of time; the event is that noon offset by the hour angle.
func solarEventPass(meanNoon, refAt time.Time, lat, elevationDeg float64, rising bool) (time.Time, bool) {
	jd := float64(refAt.Unix())/86400.0 + 2440587.5
	declination, eqOfTime := solarParams(jd)

	hourAngle, ok := hourAngleDegrees(lat, declination, elevationDeg)
	if !ok {
		return time.Time{}, false
	}
	transit := meanNoon.Add(time.Duration(-eqOfTime * float64(time.Minute)))
	offset := time.Duration(minutesPerHourAngle * hourAngle * float64(time.Minute))
	if rising {
		return transit.Add(-offset), true
	}
	return transit.Add(offset), true
}

// hourAngleDegrees returns the sun's hour angle (degrees from local noon) at the
// moment it sits at elevationDeg. ok is false when the sun never reaches it.
func hourAngleDegrees(lat, declination, elevationDeg float64) (float64, bool) {
	cosHourAngle := (math.Sin(rad(elevationDeg)) - math.Sin(rad(lat))*math.Sin(rad(declination))) /
		(math.Cos(rad(lat)) * math.Cos(rad(declination)))
	if cosHourAngle > 1 || cosHourAngle < -1 {
		return 0, false
	}
	return deg(math.Acos(cosHourAngle)), true
}

// solarParams returns the sun's declination (degrees) and the equation of time
// (minutes) for the given Julian Day, using the NOAA solar position equations.
func solarParams(jd float64) (declination, eqOfTime float64) {
	t := (jd - 2451545.0) / 36525.0 // Julian centuries since J2000.0

	meanLong := math.Mod(280.46646+t*(36000.76983+0.0003032*t), 360.0)
	meanAnom := 357.52911 + t*(35999.05029-0.0001537*t)
	eccentricity := 0.016708634 - t*(0.000042037+0.0000001267*t)

	center := math.Sin(rad(meanAnom))*(1.914602-t*(0.004817+0.000014*t)) +
		math.Sin(rad(2*meanAnom))*(0.019993-0.000101*t) +
		math.Sin(rad(3*meanAnom))*0.000289
	trueLong := meanLong + center
	appLong := trueLong - 0.00569 - 0.00478*math.Sin(rad(125.04-1934.136*t))

	meanObliq := 23.0 + (26.0+(21.448-t*(46.815+t*(0.00059-t*0.001813)))/60.0)/60.0
	obliqCorr := meanObliq + 0.00256*math.Cos(rad(125.04-1934.136*t))

	declination = deg(math.Asin(math.Sin(rad(obliqCorr)) * math.Sin(rad(appLong))))

	y := math.Tan(rad(obliqCorr/2)) * math.Tan(rad(obliqCorr/2))
	eqOfTime = 4 * deg(y*math.Sin(2*rad(meanLong))-
		2*eccentricity*math.Sin(rad(meanAnom))+
		4*eccentricity*y*math.Sin(rad(meanAnom))*math.Cos(2*rad(meanLong))-
		0.5*y*y*math.Sin(4*rad(meanLong))-
		1.25*eccentricity*eccentricity*math.Sin(2*rad(meanAnom)))
	return declination, eqOfTime
}

func rad(deg float64) float64 { return deg * math.Pi / 180.0 }
func deg(rad float64) float64 { return rad * 180.0 / math.Pi }

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
// skipped. Golden-hour spans mirror the adjacent blue-hour span, matching the
// definition used by the single-day endpoint.
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
				start:   s.sunrise,
				end:     s.sunrise.Add(markerDuration),
			})
		}
		if phases.sunset {
			events = append(events, calendarEvent{
				uid:     eventUID("sunset", stamp, lat, lng),
				summary: "🌇 Sunset",
				start:   s.sunset,
				end:     s.sunset.Add(markerDuration),
			})
		}
		if phases.blueHour {
			events = append(events, calendarEvent{
				uid:     eventUID("morning-blue", stamp, lat, lng),
				summary: "🔵 Morning Blue Hour",
				start:   s.civilDawn,
				end:     s.sunrise,
			})
			events = append(events, calendarEvent{
				uid:     eventUID("evening-blue", stamp, lat, lng),
				summary: "🔵 Evening Blue Hour",
				start:   s.sunset,
				end:     s.civilDusk,
			})
		}
		if phases.goldenHour {
			morningSpan := s.sunrise.Sub(s.civilDawn)
			eveningSpan := s.civilDusk.Sub(s.sunset)
			events = append(events, calendarEvent{
				uid:     eventUID("morning-golden", stamp, lat, lng),
				summary: "🟡 Morning Golden Hour",
				start:   s.sunrise,
				end:     s.sunrise.Add(morningSpan),
			})
			events = append(events, calendarEvent{
				uid:     eventUID("evening-golden", stamp, lat, lng),
				summary: "🟡 Evening Golden Hour",
				start:   s.sunset.Add(-eveningSpan),
				end:     s.sunset,
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
// e.g. ":20260101T113000Z" (UTC) or ";TZID=...:20260101T063000" style. We emit
// floating local time (no TZID) when a location is supplied so the wall-clock
// reads identically in every viewer's calendar.
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
