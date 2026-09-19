package main

import (
	"testing"
	"time"
)

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
	// These are western/equatorial locations whose sunrise falls in the UTC
	// morning, so the local day and the UTC day coincide and the captured
	// upstream timestamps are directly comparable. (Far-east longitudes like
	// Sydney file events under a different UTC day than the local calendar day;
	// they are covered by TestSouthernHemisphereConsistency instead.)
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
			if !got.ok {
				t.Fatalf("computeSolarDay returned ok=false")
			}
			assertWithin(t, "sunrise", got.sunrise, c.sunrise, riseSetTol)
			assertWithin(t, "sunset", got.sunset, c.sunset, riseSetTol)
			assertWithin(t, "civil-dawn", got.civilDawn, c.civilDawn, twilightTol)
			assertWithin(t, "civil-dusk", got.civilDusk, c.civilDusk, twilightTol)

			// Universal ordering invariant: dawn < sunrise < sunset < dusk.
			if !(got.civilDawn.Before(got.sunrise) &&
				got.sunrise.Before(got.sunset) &&
				got.sunset.Before(got.civilDusk)) {
				t.Errorf("events out of order: dawn=%s sunrise=%s sunset=%s dusk=%s",
					got.civilDawn, got.sunrise, got.sunset, got.civilDusk)
			}
		})
	}
}

// TestSouthernHemisphereConsistency checks the far-east, southern-hemisphere
// path (which the upstream-parity cases can't cover cleanly) via invariants:
// events are correctly ordered and grouped under the local day, and a December
// day in Sydney is a long summer day.
func TestSouthernHemisphereConsistency(t *testing.T) {
	d, _ := time.Parse("2006-01-02", "2026-12-21")
	got := computeSolarDay(d, -33.8688, 151.2093)
	if !got.ok {
		t.Fatalf("computeSolarDay returned ok=false")
	}
	if !(got.civilDawn.Before(got.sunrise) &&
		got.sunrise.Before(got.sunset) &&
		got.sunset.Before(got.civilDusk)) {
		t.Fatalf("events out of order: dawn=%s sunrise=%s sunset=%s dusk=%s",
			got.civilDawn, got.sunrise, got.sunset, got.civilDusk)
	}
	dayLength := got.sunset.Sub(got.sunrise)
	if dayLength < 14*time.Hour || dayLength > 15*time.Hour {
		t.Errorf("Sydney summer day length = %s, expected ~14.5h", dayLength)
	}
	// Sunrise and sunset must land on the same local (Sydney) calendar day.
	loc, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		t.Skipf("tz database unavailable: %v", err)
	}
	if got.sunrise.In(loc).Day() != got.sunset.In(loc).Day() {
		t.Errorf("sunrise %s and sunset %s not on same local day",
			got.sunrise.In(loc), got.sunset.In(loc))
	}
}

// TestSolarDayHelpers checks the per-value accessors and the twilight ranges.
func TestSolarDayHelpers(t *testing.T) {
	d, _ := time.Parse("2006-01-02", "2026-06-21")
	s := computeSolarDay(d, 42.3601, -71.0589)
	if !s.ok {
		t.Fatalf("computeSolarDay returned ok=false")
	}

	// Solar noon sits between sunrise and sunset, within a minute of their
	// midpoint (sunrise and sunset are symmetric about the transit).
	if !s.SolarNoon().After(s.Sunrise()) || !s.SolarNoon().Before(s.Sunset()) {
		t.Errorf("solar noon %s not between sunrise %s and sunset %s",
			s.SolarNoon(), s.Sunrise(), s.Sunset())
	}
	midpoint := s.Sunrise().Add(s.DayLength() / 2)
	if diff := s.SolarNoon().Sub(midpoint); diff > time.Minute || diff < -time.Minute {
		t.Errorf("solar noon %s far from midpoint %s (diff %s)", s.SolarNoon(), midpoint, diff)
	}

	if got := s.DayLength(); got != s.Sunset().Sub(s.Sunrise()) {
		t.Errorf("DayLength() = %s, want %s", got, s.Sunset().Sub(s.Sunrise()))
	}

	// Blue and golden hours meet at sunrise/sunset, and each golden hour spans
	// a mirror of the adjacent blue hour.
	mb, mg := s.MorningBlueHour(), s.MorningGoldenHour()
	eb, eg := s.EveningBlueHour(), s.EveningGoldenHour()
	if !mb.End.Equal(s.Sunrise()) || !mg.Start.Equal(s.Sunrise()) {
		t.Errorf("morning blue/golden should meet at sunrise: blueEnd=%s goldenStart=%s sunrise=%s",
			mb.End, mg.Start, s.Sunrise())
	}
	if !eg.End.Equal(s.Sunset()) || !eb.Start.Equal(s.Sunset()) {
		t.Errorf("evening golden/blue should meet at sunset: goldenEnd=%s blueStart=%s sunset=%s",
			eg.End, eb.Start, s.Sunset())
	}
	if mg.Duration() != mb.Duration() {
		t.Errorf("morning golden %s should mirror morning blue %s", mg.Duration(), mb.Duration())
	}
	if eg.Duration() != eb.Duration() {
		t.Errorf("evening golden %s should mirror evening blue %s", eg.Duration(), eb.Duration())
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
