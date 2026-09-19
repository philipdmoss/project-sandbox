package main

import (
	"testing"
	"time"
)

// mustEvent fetches an optional event accessor, failing the test if absent.
func mustEvent(t *testing.T, label string, f func() (time.Time, bool)) time.Time {
	t.Helper()
	tm, ok := f()
	if !ok {
		t.Fatalf("expected %s to be present", label)
	}
	return tm
}

// Ground-truth sunrise/sunset/civil-twilight instants (UTC) captured from the
// upstream api.sunrise-sunset.org reference for each location and date. Our
// locally computed NOAA values are expected to agree with these: civil
// twilight to within seconds, sunrise/sunset to within ~2 minutes (the two
// algorithms use slightly different horizon-refraction conventions, which is
// immaterial for a golden/blue-hour calendar). A sign flip or wrong constant
// would blow far past the tolerance and fail this test.
func TestComputeSolarDayMatchesReference(t *testing.T) {
	type ref struct {
		name                 string
		lat, lng             float64
		date                 string
		sunrise, sunset      string
		civilDawn, civilDusk string
	}
	// Western/equatorial locations whose sunrise falls in the UTC morning, so
	// the local day and the UTC day coincide and the captured upstream
	// timestamps are directly comparable. (Far-east longitudes like Sydney are
	// covered by TestSouthernHemisphereConsistency instead.)
	cases := []ref{
		{"boston-winter", 42.3601, -71.0589, "2026-01-15", "2026-01-15T12:08:59Z", "2026-01-15T21:38:25Z", "2026-01-15T11:39:37Z", "2026-01-15T22:07:48Z"},
		{"boston-summer", 42.3601, -71.0589, "2026-06-21", "2026-06-21T09:05:53Z", "2026-06-22T00:26:19Z", "2026-06-21T08:32:47Z", "2026-06-22T00:59:25Z"},
		{"paris-spring", 48.8566, 2.3522, "2026-03-20", "2026-03-20T05:51:33Z", "2026-03-20T18:04:32Z", "2026-03-20T05:21:41Z", "2026-03-20T18:34:24Z"},
		{"nairobi-equator", -1.2921, 36.8219, "2026-09-23", "2026-09-23T03:20:41Z", "2026-09-23T15:29:31Z", "2026-09-23T03:01:05Z", "2026-09-23T15:49:07Z"},
	}

	const riseSetTol = 2*time.Minute + 30*time.Second
	const twilightTol = time.Minute

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, _ := time.Parse("2006-01-02", c.date)
			got := computeSolarDay(d, c.lat, c.lng)

			sunrise := mustEvent(t, "sunrise", got.Sunrise)
			sunset := mustEvent(t, "sunset", got.Sunset)
			civilDawn := mustEvent(t, "civil dawn", got.CivilDawn)
			civilDusk := mustEvent(t, "civil dusk", got.CivilDusk)

			assertWithin(t, "sunrise", sunrise, c.sunrise, riseSetTol)
			assertWithin(t, "sunset", sunset, c.sunset, riseSetTol)
			assertWithin(t, "civil-dawn", civilDawn, c.civilDawn, twilightTol)
			assertWithin(t, "civil-dusk", civilDusk, c.civilDusk, twilightTol)

			// Universal ordering invariant: dawn < sunrise < sunset < dusk.
			if !(civilDawn.Before(sunrise) && sunrise.Before(sunset) && sunset.Before(civilDusk)) {
				t.Errorf("events out of order: dawn=%s sunrise=%s sunset=%s dusk=%s",
					civilDawn, sunrise, sunset, civilDusk)
			}
		})
	}
}

// TestSouthernHemisphereConsistency checks the far-east, southern-hemisphere
// path via invariants: events ordered and grouped under the local day, and a
// December day in Sydney is a long summer day.
func TestSouthernHemisphereConsistency(t *testing.T) {
	d, _ := time.Parse("2006-01-02", "2026-12-21")
	got := computeSolarDay(d, -33.8688, 151.2093)

	civilDawn := mustEvent(t, "civil dawn", got.CivilDawn)
	sunrise := mustEvent(t, "sunrise", got.Sunrise)
	sunset := mustEvent(t, "sunset", got.Sunset)
	civilDusk := mustEvent(t, "civil dusk", got.CivilDusk)

	if !(civilDawn.Before(sunrise) && sunrise.Before(sunset) && sunset.Before(civilDusk)) {
		t.Fatalf("events out of order: dawn=%s sunrise=%s sunset=%s dusk=%s",
			civilDawn, sunrise, sunset, civilDusk)
	}
	dayLength, ok := got.DayLength()
	if !ok || dayLength < 14*time.Hour || dayLength > 15*time.Hour {
		t.Errorf("Sydney summer day length = %s (ok=%v), expected ~14.5h", dayLength, ok)
	}
	loc, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		t.Skipf("tz database unavailable: %v", err)
	}
	if sunrise.In(loc).Day() != sunset.In(loc).Day() {
		t.Errorf("sunrise %s and sunset %s not on same local day", sunrise.In(loc), sunset.In(loc))
	}
}

// TestSolarDayHelpers checks the per-value accessors and the elevation-band
// twilight windows (blue −6→−4, golden −4→+6, so golden contains sunrise/sunset).
func TestSolarDayHelpers(t *testing.T) {
	d, _ := time.Parse("2006-01-02", "2026-06-21")
	s := computeSolarDay(d, 42.3601, -71.0589)

	sunrise := mustEvent(t, "sunrise", s.Sunrise)
	sunset := mustEvent(t, "sunset", s.Sunset)

	if !s.SolarNoon().After(sunrise) || !s.SolarNoon().Before(sunset) {
		t.Errorf("solar noon %s not between sunrise %s and sunset %s", s.SolarNoon(), sunrise, sunset)
	}
	dayLen, ok := s.DayLength()
	if !ok || dayLen != sunset.Sub(sunrise) {
		t.Errorf("DayLength() = %s (ok=%v), want %s", dayLen, ok, sunset.Sub(sunrise))
	}

	mb, mbOK := s.MorningBlueHour()
	mg, mgOK := s.MorningGoldenHour()
	eb, ebOK := s.EveningBlueHour()
	eg, egOK := s.EveningGoldenHour()
	if !mbOK || !mgOK || !ebOK || !egOK {
		t.Fatalf("expected all twilight windows present at Boston in summer")
	}

	// Each window is ordered start<end.
	for name, w := range map[string]timeRange{"mBlue": mb, "mGold": mg, "eGold": eg, "eBlue": eb} {
		if !w.End.After(w.Start) {
			t.Errorf("%s window not ordered: %s..%s", name, w.Start, w.End)
		}
	}

	// Bands are contiguous at the −4° boundary: morning blue ends where morning
	// golden begins; evening golden ends where evening blue begins.
	if !mb.End.Equal(mg.Start) {
		t.Errorf("morning blue end %s != morning golden start %s", mb.End, mg.Start)
	}
	if !eg.End.Equal(eb.Start) {
		t.Errorf("evening golden end %s != evening blue start %s", eg.End, eb.Start)
	}

	// Golden hour straddles the horizon, so sunrise/sunset fall *inside* it.
	if sunrise.Before(mg.Start) || sunrise.After(mg.End) {
		t.Errorf("sunrise %s not within morning golden hour %s..%s", sunrise, mg.Start, mg.End)
	}
	if sunset.Before(eg.Start) || sunset.After(eg.End) {
		t.Errorf("sunset %s not within evening golden hour %s..%s", sunset, eg.Start, eg.End)
	}

	// Blue hour sits fully below the horizon (ends at −4°, before sunrise).
	if !mb.End.Before(sunrise) {
		t.Errorf("morning blue hour should end before sunrise: end=%s sunrise=%s", mb.End, sunrise)
	}
}

// TestHighLatitudeRegimes verifies the independently-optional events: polar day
// (nothing), a winter day with civil twilight but no sunrise, and white nights
// (sunrise/sunset but no civil twilight, hence no blue hour).
func TestHighLatitudeRegimes(t *testing.T) {
	summer, _ := time.Parse("2006-01-02", "2026-06-21")
	winter, _ := time.Parse("2006-01-02", "2026-12-21")

	// Svalbard midsummer: polar day — the sun never rises or sets.
	polar := computeSolarDay(summer, 78.22, 15.63)
	if _, ok := polar.Sunrise(); ok {
		t.Errorf("expected no sunrise at Svalbard midsummer (polar day)")
	}
	if _, ok := polar.Sunset(); ok {
		t.Errorf("expected no sunset at Svalbard midsummer (polar day)")
	}

	// 70N midwinter: the sun clears −6° (civil twilight) but never reaches
	// −0.833°, so there is twilight but no sunrise/sunset.
	polarWinter := computeSolarDay(winter, 70, 25)
	if _, ok := polarWinter.Sunrise(); ok {
		t.Errorf("expected no sunrise at 70N midwinter")
	}
	if _, ok := polarWinter.CivilDawn(); !ok {
		t.Errorf("expected civil twilight at 70N midwinter (sun reaches -6 but not -0.833)")
	}

	// White nights: a mid-high latitude in summer with sunrise/sunset but no
	// civil twilight (and therefore no blue hour). Sweep to stay robust.
	foundWhiteNight := false
	for _, lat := range []float64{60, 62, 63, 64, 65} {
		s := computeSolarDay(summer, lat, 10)
		_, hasRise := s.Sunrise()
		_, hasSet := s.Sunset()
		_, hasDawn := s.CivilDawn()
		if hasRise && hasSet && !hasDawn {
			foundWhiteNight = true
			if _, ok := s.MorningBlueHour(); ok {
				t.Errorf("white night at %.0fN should have no morning blue hour", lat)
			}
			break
		}
	}
	if !foundWhiteNight {
		t.Errorf("expected a white-night regime (sunrise/sunset, no civil twilight) at some 60-65N latitude")
	}
}

// TestSunPosition sanity-checks altitude/azimuth: at solar noon the sun is near
// due south (northern hemisphere) at its highest, and lower a few hours later.
func TestSunPosition(t *testing.T) {
	d, _ := time.Parse("2006-01-02", "2026-06-21")
	s := computeSolarDay(d, 42.3601, -71.0589)
	noon := s.SolarNoon()

	altNoon, azNoon := sunPosition(noon, 42.3601, -71.0589)
	if altNoon < 60 || altNoon > 75 {
		t.Errorf("Boston summer-solstice noon altitude = %.1f°, expected ~71°", altNoon)
	}
	if azNoon < 175 || azNoon > 185 {
		t.Errorf("noon azimuth = %.1f°, expected ~due south (180°)", azNoon)
	}

	// Sunrise: sun near the horizon, azimuth in the north-east (summer).
	if sr, ok := s.Sunrise(); ok {
		altRise, azRise := sunPosition(sr, 42.3601, -71.0589)
		if altRise < -2 || altRise > 2 {
			t.Errorf("sunrise altitude = %.1f°, expected ~0°", altRise)
		}
		if azRise < 30 || azRise > 90 {
			t.Errorf("summer sunrise azimuth = %.1f°, expected north-east (30-90°)", azRise)
		}
	}
}

func assertWithin(t *testing.T, label string, got time.Time, wantRFC3339 string, tol time.Duration) {
	t.Helper()
	want, err := time.Parse(time.RFC3339, wantRFC3339)
	if err != nil {
		t.Fatalf("bad reference time %q: %v", wantRFC3339, err)
	}
	diff := got.Sub(want)
	if diff < 0 {
		diff = -diff
	}
	if diff > tol {
		t.Errorf("%s: got %s want %s (diff %s > tol %s)", label,
			got.UTC().Format(time.RFC3339), wantRFC3339, diff, tol)
	}
}
