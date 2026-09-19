# sun-calculator-be

Small Go HTTP service for sun-phase times (sunrise, sunset, blue hour, golden
hour) at a location.

Run: `go run .` (listens on `:8080`, override with `PORT`).

## Endpoints

### `GET /health`
Liveness check.

### `GET /api/suntimes?lat=&lng=`
Sun times for **today** at a location, as JSON. Proxies
`api.sunrise-sunset.org` and derives the twilight windows.

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

**Phases → events**
- `sunrise` / `sunset`: short 15-minute blocks starting at the event, so they render as clickable events in day/week views.
- `blue_hour`: morning (civil dawn → sunrise) and evening (sunset → civil dusk) spans.
- `golden_hour`: morning and evening spans, each mirroring the adjacent blue-hour span — matching the definitions used by `/api/suntimes`.

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
URL directly to trigger a download, or fetch it and offer it as a Blob. A
"Subscribe" experience can point Google Calendar's "From URL" at this endpoint.

## Accuracy

Computed sunrise/sunset agree with `api.sunrise-sunset.org` to within ~2 minutes
(the two algorithms use slightly different horizon-refraction conventions);
civil twilight agrees to within seconds. This is validated in `calendar_test.go`
against captured reference values. The variance is immaterial for planning
around the ~30–60 minute golden/blue-hour windows.
