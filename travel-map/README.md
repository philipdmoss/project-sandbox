# Travels 2019–2021 — interactive map demo

A minimalist, embeddable travel map built from three Google Photos "Places" screenshots
(one per year). Click a pin to open a clean popup with the location's photo and a short note.

## Preview

Open `index.html` in any browser (double-click works — data is inline, photos are local
files). The map needs internet for the Leaflet library and the CARTO base-map tiles, as any
web map does.

- 40 locations across **2019** (10), **2020** (15), **2021** (15)
- Pins colored by year (2019 teal · 2020 amber · 2021 coral); legend bottom-left
- Year filter top-right (`All · 2019 · 2020 · 2021`) — toggles pins and re-frames the map
- Click a pin → photo + location + year + reflection

## Files

| Path | What it is |
|---|---|
| `index.html` | Self-contained map (location data injected inline; photos referenced locally) |
| `build.py` | Single source of truth — crops photos, writes `data/locations.json`, renders `index.html` |
| `data/locations.json` | Portable 40-location dataset (name, year, lat, lng, photo, reflection) |
| `photos/*.jpg` | 40 tiles cropped from the screenshots (466×466) |
| `screenshots/2019–2021.png` | Original source screenshots (crop source + provenance) |

## Editing

Everything lives in the `LOCATIONS` table in `build.py` — coordinates, reflections, and the
`(shot, row, col)` grid position each photo is cropped from. Edit there, then rerun:

```bash
python3 build.py
```

This re-crops photos, rewrites `data/locations.json`, and regenerates `index.html`. Cropping
uses the native macOS `sips` tool, so there are no Python dependencies to install.

## Embedding in Webflow

The map is static files, so any host works:

1. Upload this folder to any static host (Netlify, GitHub Pages, S3, your own server).
2. In Webflow, drop an **Embed** element where you want the map and iframe the hosted page:
   ```html
   <iframe src="https://your-host/travel-map/index.html"
           style="width:100%;height:600px;border:0;border-radius:12px"></iframe>
   ```

For a fully native Webflow build later, `data/locations.json` maps cleanly onto a "Travel
Logs" CMS Collection (Location, Year, Coordinates, Photo, Reflection).

## Known items to review

- **`Downtown` (2020)** — the screenshot label is just "Downtown" (a dusk ballpark photo).
  Coordinates are a best guess (Durham, NC) and flagged `needs_review` in `locations.json`.
  Update the lat/lng in `build.py` if it belongs somewhere else.
- Reflections are drafted in your voice from the photos — edit any that don't ring true.
- Ambiguous city names were placed from photo cues (e.g. Washington = DC, Cambridge = MA,
  Dublin = Ireland, Cardiff = Wales, the MD suburbs in PG County).
