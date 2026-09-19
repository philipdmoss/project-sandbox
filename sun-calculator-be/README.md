# sun-calculator-be

Small Go HTTP service for sun-phase times (sunrise, sunset, blue hour, golden
hour) at a location.

Run: `go run .` (listens on `:8080`, override with `PORT`).

## Endpoints

### `GET /health`
Liveness check.

### `GET /api/suntimes?lat=&lng=`
Sun times for **today** (UTC) at a location, as JSON: sunrise, sunset, solar
noon, day length and the four twilight windows. Computed locally with the NOAA
solar equations — no external API call. For an arbitrary date or a subset of
values, use `/api/solar` below.

### `GET /api/solar?lat=&lng=&date=&field=`
Individual sun values (or all of them) for a location on a date, as JSON,
computed locally. Backs the same solar helpers the calendar uses.

| param       | required | default     | notes |
|-------------|----------|-------------|-------|
| `lat`       | yes      | —           | −90..90 |
| `lng`       | yes      | —           | −180..180 |
| `date`      | no       | today (UTC) | `YYYY-MM-DD`, the local day at the location |
| `field`     | no       | all         | comma-separated subset (or `all`) |
| `elevation` | no       | —           | degrees in [−90, 90]; adds an `elevation` object with the rising/setting crossing times for that sun elevation |

`field` values: `sunrise`, `sunset`, `sunrise_azimuth`, `sunset_azimuth`,
`solar_noon`, `day_length`, `morning_blue_hour`, `morning_golden_hour`,
`evening_golden_hour`, `evening_blue_hour`, `civil_twilight`,
`nautical_twilight`, `astronomical_twilight`. Times are UTC RFC 3339; twilight
windows are `{start, end, duration}`; the `*_twilight` values are `{dawn, dusk}`
pairs; azimuths are degrees clockwise from true north; `day_length` is a
duration string. Returns only the requested keys.

Definitions (matching astral / suncalc / PhotoPills): sunrise/sunset are the
sun's centre at −0.833°; **blue hour** is the sun from −6° to −4°; **golden
hour** is −4° to +6°, so golden hour straddles the horizon and *contains*
sunrise/sunset. Civil/nautical/astronomical twilight are the −6/−12/−18°
depressions.

```
# one value
/api/solar?lat=42.3601&lng=-71.0589&field=sunrise
-> {"sunrise":"2026-06-21T09:07:30Z"}

# a combination
/api/solar?lat=48.8566&lng=2.3522&date=2026-06-21&field=day_length,morning_golden_hour

# everything (field omitted)
/api/solar?lat=-33.8688&lng=151.2093

# arbitrary sun elevation (e.g. when the sun is 10° up)
/api/solar?lat=42.3601&lng=-71.0589&field=sunrise&elevation=10
```

At high latitudes an event may not occur (polar day/night, or summer "white
nights" where the sun sets but never reaches −6°, so there is no civil
twilight). Only the values that actually occur are returned; solar noon always
does. A 422 is returned only when *none* of the requested values occur.

### `GET /api/position?lat=&lng=&time=`
The sun's position for an instant, as JSON: `{time, altitude, azimuth}`.
`time` is RFC 3339 (default now). Altitude is degrees above the horizon
(geometric); azimuth is degrees clockwise from true north (0=N, 90=E, 180=S).

```
/api/position?lat=42.3601&lng=-71.0589&time=2026-06-21T16:46:00Z
-> {"time":"2026-06-21T16:46:00Z","altitude":71.02,"azimuth":179.9}
```

### `GET /api/calendar?lat=&lng=&phases=&year=&tz=`
Generates a **full year** of events as a downloadable iCalendar (`.ics`) file
that can be imported into Google Calendar (Settings → Import & export), Apple
Calendar, etc. Sun times are computed locally with the NOAA solar equations, so
no per-day upstream calls are made.

| param    | required | default        | notes |
|----------|----------|----------------|-------|
| `lat`    | yes      | —              | −90..90 |
| `lng`    | yes      | —              | −180..180 |
| `phases` | no       | `all`          | comma-separated subset of `sunrise`, `sunset`, `blue_hour`, `golden_hour` (or `all`) |
| `year`   | no       | current year   | 1970..9999 |
| `tz`     | no       | UTC            | IANA name (e.g. `America/New_York`); see below |

**Phases** (elevation-based, as in `/api/solar` above)
- `sunrise` / `sunset`: the moment the sun's centre reaches −0.833°.
- `blue_hour`: morning and evening, sun −6° → −4°.
- `golden_hour`: morning and evening, sun −4° → +6° (straddles the horizon, so it contains sunrise/sunset).

**Events**: each day yields at most two events — the requested morning phases
merged into one, and the requested evening phases merged into one. A merged
event spans from the earliest requested piece's start to the latest piece's end
(morning order: blue hour → sunrise → golden hour; evening: golden hour →
sunset → blue hour), is titled with the pieces it contains
(e.g. `🌅 Blue Hour · Sunrise · Golden Hour`), and its description lists each
piece's time/range/duration. Pieces that don't occur that day (high-latitude
summer/winter) are left out; a morning containing only a sunrise renders as a
short block so it stays clickable.

**Timezone / display**
- No `tz`: event times are written as absolute UTC instants (`...Z`). The
  viewer's calendar localizes them.
- With `tz`: event times are written as *floating* local wall-clock times, so
  "golden hour 06:45" reads the same in every viewer's calendar. For a calendar
  about a specific place, pass that place's IANA zone.

Each day's events are anchored on that day's local solar noon, so sunrise,
sunset and the twilight windows are always grouped under the correct local date
worldwide (including far-east/far-west longitudes).

**Examples**
```
# Whole calendar for Boston, displayed in Eastern time
/api/calendar?lat=42.3601&lng=-71.0589&tz=America/New_York

# Just golden hours for Paris, 2027
/api/calendar?lat=48.8566&lng=2.3522&phases=golden_hour&year=2027

# Sunrise + sunset only (UTC)
/api/calendar?lat=-33.8688&lng=151.2093&phases=sunrise,sunset
```

**Frontend use**: CORS is open (`Access-Control-Allow-Origin: *`). Link to the
URL directly to trigger a download, or fetch it and offer it as a Blob.

### `GET /api/calendar/feed?lat=&lng=&phases=&tz=`
Same content as `/api/calendar`, but as a **subscribable** calendar: a rolling
window (current year plus next year), no download disposition, recomputed each
request. Point a calendar app's "subscribe from URL" (or `webcal://`) at it and
it stays current. Same `phases`/`tz` params (no `year`).

## Accuracy

Our engine is the NOAA/Meeus solar-position algorithm plus a refinement pass —
already better than the stock NOAA spreadsheet, and validated in `solar_test.go`
against captured `api.sunrise-sunset.org` reference values: civil twilight
agrees to within seconds, sunrise/sunset to within ~2 minutes. That ~2 min is a
horizon-refraction *convention* difference, not error — we use the standard
−0.833° (16′ solar semidiameter + 34′ mean refraction); the upstream behaves as
if it uses ~−1.1°. Real atmospheric refraction near the horizon varies far more
(0.4–2.1°) than any algorithm choice, and all of this is immaterial for planning
around the golden/blue-hour windows. Higher-accuracy algorithms (NREL SPA, full
Meeus) would change rise/set by well under a second — not worth the complexity.
