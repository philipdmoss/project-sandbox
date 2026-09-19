package main

import (
	"math"
	"time"
)

// Solar elevation angles (degrees relative to the horizon) that define each
// event. Sunrise/sunset use −0.833° (the sun's ~16′ semidiameter plus ~34′ of
// mean atmospheric refraction — the standard "official sunrise" value). The
// golden and blue hours are the photographers' elevation bands (as used by
// astral, suncalc and PhotoPills): blue hour is −6°→−4°, golden hour is
// −4°→+6°, so golden hour straddles the horizon and contains sunrise/sunset.
// Civil/nautical/astronomical twilight are the standard −6/−12/−18° depressions.
const (
	sunriseSunsetAngle        = -0.833
	goldenHourAngle           = 6.0  // upper edge of golden hour (sun this far above horizon)
	blueGoldenBoundaryAngle   = -4.0 // boundary between blue hour and golden hour
	civilTwilightAngle        = -6.0 // civil twilight / lower edge of blue hour
	nauticalTwilightAngle     = -12.0
	astronomicalTwilightAngle = -18.0
	minutesPerHourAngle       = 4.0 // 4 minutes of time per degree of Earth rotation
)

// timeRange is a start/end interval, used for the twilight windows.
type timeRange struct {
	Start time.Time
	End   time.Time
}

// Duration returns how long the range lasts.
func (r timeRange) Duration() time.Duration { return r.End.Sub(r.Start) }

// solarDay represents one local day at a location. It computes each event's
// instant on demand from the day's mean solar noon, so any sun-elevation
// crossing can be requested (sunrise, twilight bands, arbitrary angles) and
// each is independently optional: at high latitudes the sun may never reach a
// given elevation (polar day/night, or summer "white nights" where it sets but
// never drops to −6°). Solar noon (the meridian transit) always exists.
type solarDay struct {
	meanNoon  time.Time
	lat       float64
	solarNoon time.Time
}

// computeSolarDay builds the solarDay for the given calendar date at a location.
//
// The date's year/month/day identify the *local* day at the location: events
// are anchored on that day's local solar noon, so they are always grouped under
// the correct local date even at far-east or far-west longitudes (anchoring on
// UTC midnight instead would file an early local sunrise under the previous UTC
// day).
func computeSolarDay(date time.Time, lat, lng float64) solarDay {
	// Local mean noon, expressed as a UTC instant: 12:00 local shifted by the
	// location's longitude (15 degrees per hour, east of Greenwich is earlier).
	meanNoon := time.Date(date.Year(), date.Month(), date.Day(), 12, 0, 0, 0, time.UTC).
		Add(time.Duration(-lng / 15 * float64(time.Hour)))

	// Apparent solar noon (transit) is mean noon corrected by the equation of
	// time evaluated at that moment. The transit always occurs.
	_, eqOfTimeAtNoon := solarParams(julianDay(meanNoon))
	solarNoon := meanNoon.Add(time.Duration(-eqOfTimeAtNoon * float64(time.Minute)))

	return solarDay{meanNoon: meanNoon, lat: lat, solarNoon: solarNoon}
}

// Crossing returns the UTC instant when the sun's centre reaches elevationDeg
// while rising (true) or setting (false), or false if it never does that day.
func (s solarDay) Crossing(elevationDeg float64, rising bool) (time.Time, bool) {
	return solarEvent(s.meanNoon, s.lat, elevationDeg, rising)
}

// Sunrise is the instant the sun's centre reaches −0.833°, if it does.
func (s solarDay) Sunrise() (time.Time, bool) { return s.Crossing(sunriseSunsetAngle, true) }

// Sunset is the instant the sun's centre drops to −0.833°, if it does.
func (s solarDay) Sunset() (time.Time, bool) { return s.Crossing(sunriseSunsetAngle, false) }

// SolarNoon is the instant the sun transits the local meridian. Always occurs.
func (s solarDay) SolarNoon() time.Time { return s.solarNoon }

// DayLength is the time between sunrise and sunset; false if either is absent.
func (s solarDay) DayLength() (time.Duration, bool) {
	rise, ok1 := s.Sunrise()
	set, ok2 := s.Sunset()
	if ok1 && ok2 {
		return set.Sub(rise), true
	}
	return 0, false
}

// CivilDawn/CivilDusk/... are the standard twilight instants (sun crossing
// −6/−12/−18° rising or setting).
func (s solarDay) CivilDawn() (time.Time, bool) { return s.Crossing(civilTwilightAngle, true) }
func (s solarDay) CivilDusk() (time.Time, bool) { return s.Crossing(civilTwilightAngle, false) }
func (s solarDay) NauticalDawn() (time.Time, bool) {
	return s.Crossing(nauticalTwilightAngle, true)
}
func (s solarDay) NauticalDusk() (time.Time, bool) {
	return s.Crossing(nauticalTwilightAngle, false)
}
func (s solarDay) AstronomicalDawn() (time.Time, bool) {
	return s.Crossing(astronomicalTwilightAngle, true)
}
func (s solarDay) AstronomicalDusk() (time.Time, bool) {
	return s.Crossing(astronomicalTwilightAngle, false)
}

// band returns the timeRange between two rising (or two setting) elevation
// crossings, present only if both crossings occur.
func (s solarDay) band(lower, upper float64, rising bool) (timeRange, bool) {
	a, ok1 := s.Crossing(lower, rising)
	b, ok2 := s.Crossing(upper, rising)
	if !ok1 || !ok2 {
		return timeRange{}, false
	}
	if a.Before(b) {
		return timeRange{Start: a, End: b}, true
	}
	return timeRange{Start: b, End: a}, true
}

// MorningBlueHour runs from −6° to −4° while the sun is rising.
func (s solarDay) MorningBlueHour() (timeRange, bool) {
	return s.band(civilTwilightAngle, blueGoldenBoundaryAngle, true)
}

// MorningGoldenHour runs from −4° up to +6° while the sun is rising; it
// straddles the horizon, so sunrise falls inside it.
func (s solarDay) MorningGoldenHour() (timeRange, bool) {
	return s.band(blueGoldenBoundaryAngle, goldenHourAngle, true)
}

// EveningGoldenHour runs from +6° down to −4° while the sun is setting.
func (s solarDay) EveningGoldenHour() (timeRange, bool) {
	return s.band(blueGoldenBoundaryAngle, goldenHourAngle, false)
}

// EveningBlueHour runs from −4° down to −6° while the sun is setting.
func (s solarDay) EveningBlueHour() (timeRange, bool) {
	return s.band(civilTwilightAngle, blueGoldenBoundaryAngle, false)
}

// sunPosition returns the sun's apparent altitude and azimuth (both degrees) at
// an instant for a location, using the NOAA equations. Altitude is degrees
// above the horizon (geometric, no refraction correction); azimuth is degrees
// clockwise from true north (0=N, 90=E, 180=S, 270=W).
func sunPosition(t time.Time, lat, lng float64) (altitude, azimuth float64) {
	declination, eqOfTime := solarParams(julianDay(t))

	utc := t.UTC()
	utcMinutes := float64(utc.Hour())*60 + float64(utc.Minute()) +
		float64(utc.Second())/60 + float64(utc.Nanosecond())/6e10
	// True solar time (minutes), normalized to [0, 1440); hour angle 0 at solar
	// noon, positive in the afternoon.
	trueSolarMin := math.Mod(utcMinutes+eqOfTime+4*lng, 1440)
	if trueSolarMin < 0 {
		trueSolarMin += 1440
	}
	hourAngle := trueSolarMin/4 - 180 // degrees, [-180, 180)

	latRad, declRad, haRad := rad(lat), rad(declination), rad(hourAngle)
	zenithCos := clampUnit(math.Sin(latRad)*math.Sin(declRad) +
		math.Cos(latRad)*math.Cos(declRad)*math.Cos(haRad))
	zenith := math.Acos(zenithCos)
	altitude = 90 - deg(zenith)

	sinZenith := math.Sin(zenith)
	if math.Abs(sinZenith) < 1e-9 {
		return altitude, 0 // sun at zenith/nadir: azimuth undefined
	}
	azCos := clampUnit((math.Sin(latRad)*math.Cos(zenith) - math.Sin(declRad)) /
		(math.Cos(latRad) * sinZenith))
	azCore := deg(math.Acos(azCos))
	if hourAngle > 0 {
		azimuth = math.Mod(azCore+180, 360)
	} else {
		azimuth = math.Mod(540-azCore, 360)
	}
	return altitude, azimuth
}

func clampUnit(x float64) float64 {
	if x > 1 {
		return 1
	}
	if x < -1 {
		return -1
	}
	return x
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
