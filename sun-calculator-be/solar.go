package main

import (
	"math"
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

// timeRange is a start/end interval, used for the twilight windows.
type timeRange struct {
	Start time.Time
	End   time.Time
}

// Duration returns how long the range lasts.
func (r timeRange) Duration() time.Duration { return r.End.Sub(r.Start) }

// optionalTime is an instant that may not occur on a given day: at high
// latitudes the sun can fail to reach a required elevation (polar day/night, or
// the summer "white nights" where the sun sets but never drops to −6°).
type optionalTime struct {
	Time    time.Time
	Present bool
}

// solarDay holds the boundary instants (all UTC) for a single local day, from
// which every twilight window and marker is derived. Sunrise, sunset and civil
// twilight are each independently optional; solar noon (the meridian transit)
// always exists, even when the sun stays entirely above or below the horizon.
type solarDay struct {
	civilDawn optionalTime
	sunrise   optionalTime
	solarNoon time.Time
	sunset    optionalTime
	civilDusk optionalTime
}

// Sunrise is the instant the sun's upper limb clears the horizon, if it does.
func (s solarDay) Sunrise() (time.Time, bool) { return s.sunrise.Time, s.sunrise.Present }

// Sunset is the instant the sun's upper limb drops below the horizon, if it does.
func (s solarDay) Sunset() (time.Time, bool) { return s.sunset.Time, s.sunset.Present }

// SolarNoon is the instant the sun transits the local meridian (its highest
// point of the day). It always occurs.
func (s solarDay) SolarNoon() time.Time { return s.solarNoon }

// DayLength is the time between sunrise and sunset; false if either is absent.
func (s solarDay) DayLength() (time.Duration, bool) {
	if s.sunrise.Present && s.sunset.Present {
		return s.sunset.Time.Sub(s.sunrise.Time), true
	}
	return 0, false
}

// MorningBlueHour runs from civil dawn to sunrise; false if either is absent.
func (s solarDay) MorningBlueHour() (timeRange, bool) {
	if s.civilDawn.Present && s.sunrise.Present {
		return timeRange{Start: s.civilDawn.Time, End: s.sunrise.Time}, true
	}
	return timeRange{}, false
}

// MorningGoldenHour runs from sunrise for a span mirroring the morning blue
// hour, matching the definition used by the single-day endpoint. It requires
// both civil dawn and sunrise.
func (s solarDay) MorningGoldenHour() (timeRange, bool) {
	if s.civilDawn.Present && s.sunrise.Present {
		span := s.sunrise.Time.Sub(s.civilDawn.Time)
		return timeRange{Start: s.sunrise.Time, End: s.sunrise.Time.Add(span)}, true
	}
	return timeRange{}, false
}

// EveningGoldenHour ends at sunset, spanning a mirror of the evening blue hour.
// It requires both sunset and civil dusk.
func (s solarDay) EveningGoldenHour() (timeRange, bool) {
	if s.sunset.Present && s.civilDusk.Present {
		span := s.civilDusk.Time.Sub(s.sunset.Time)
		return timeRange{Start: s.sunset.Time.Add(-span), End: s.sunset.Time}, true
	}
	return timeRange{}, false
}

// EveningBlueHour runs from sunset to civil dusk; false if either is absent.
func (s solarDay) EveningBlueHour() (timeRange, bool) {
	if s.sunset.Present && s.civilDusk.Present {
		return timeRange{Start: s.sunset.Time, End: s.civilDusk.Time}, true
	}
	return timeRange{}, false
}

// computeSolarDay calculates sunrise, sunset, solar noon and civil twilight for
// the given calendar date at a location using the NOAA solar position
// equations. All returned times are UTC instants.
//
// The date's year/month/day identify the *local* day at the location: every
// event is anchored on that day's local solar noon, so sunrise, sunset and the
// twilight windows are always grouped under the same local day even at far-east
// or far-west longitudes (anchoring on UTC midnight instead would file an early
// local sunrise under the previous UTC day).
//
// Sunrise/sunset (the −0.833° crossing) and civil twilight (−6°) are computed
// independently: a high-latitude summer day can have a sunrise and sunset with
// no civil twilight in between, and a high-latitude winter day can have civil
// twilight but no sunrise. Absent events are marked not-present rather than
// invalidating the whole day.
func computeSolarDay(date time.Time, lat, lng float64) solarDay {
	// Local mean noon, expressed as a UTC instant: 12:00 local shifted by the
	// location's longitude (15 degrees per hour, east of Greenwich is earlier).
	meanNoon := time.Date(date.Year(), date.Month(), date.Day(), 12, 0, 0, 0, time.UTC).
		Add(time.Duration(-lng / 15 * float64(time.Hour)))

	// Apparent solar noon (transit) is mean noon corrected by the equation of
	// time evaluated at that moment. The transit always occurs.
	_, eqOfTimeAtNoon := solarParams(julianDay(meanNoon))
	solarNoon := meanNoon.Add(time.Duration(-eqOfTimeAtNoon * float64(time.Minute)))

	sunrise, riseOK := solarEvent(meanNoon, lat, sunriseSunsetAngle, true)
	sunset, setOK := solarEvent(meanNoon, lat, sunriseSunsetAngle, false)
	civilDawn, dawnOK := solarEvent(meanNoon, lat, civilTwilightAngle, true)
	civilDusk, duskOK := solarEvent(meanNoon, lat, civilTwilightAngle, false)

	return solarDay{
		civilDawn: optionalTime{Time: civilDawn, Present: dawnOK},
		sunrise:   optionalTime{Time: sunrise, Present: riseOK},
		solarNoon: solarNoon,
		sunset:    optionalTime{Time: sunset, Present: setOK},
		civilDusk: optionalTime{Time: civilDusk, Present: duskOK},
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
	declination, eqOfTime := solarParams(julianDay(refAt))

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

// julianDay converts an instant to its Julian Day number.
func julianDay(t time.Time) float64 {
	return float64(t.Unix())/86400.0 + 2440587.5
}

func rad(deg float64) float64 { return deg * math.Pi / 180.0 }
func deg(rad float64) float64 { return rad * 180.0 / math.Pi }
