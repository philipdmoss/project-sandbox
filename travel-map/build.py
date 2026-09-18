#!/usr/bin/env python3
"""
build.py — Build the interactive travel-map demo from the three Google Photos
"Places" screenshots (one per year).

Single source of truth: the LOCATIONS table below. Running this script:
  1. Crops each location's tile out of its year screenshot (native `sips`, no deps)
     into photos/<slug>.jpg
  2. Writes data/locations.json (portable dataset)
  3. Renders index.html (self-contained Leaflet map, data injected inline)

No third-party Python packages required — cropping/resizing is done with macOS `sips`.
"""

import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
SHOTS = ROOT / "screenshots"
PHOTOS = ROOT / "photos"
DATA = ROOT / "data"

# Per-screenshot grid geometry (calibrated against the actual pixels):
#   x0,y0  = top-left of tile (row0,col0)
#   tw,th  = tile width/height to crop
#   sx,sy  = column / row stride (tile + gutter)
#   rows   = number of tile rows (cols is always 5)
GRID = {
    2019: dict(x0=14, y0=145, tw=466, th=466, sx=480, sy=480, rows=2),
    2020: dict(x0=14, y0=14,  tw=466, th=466, sx=480, sy=480, rows=3),
    2021: dict(x0=14, y0=14,  tw=460, th=460, sx=479, sy=470, rows=3),
}
COLS = 5
THUMB_MAX = 400  # output thumbnail max dimension (px)

# Each location: year, name, slug, lat, lng, row, col, reflection.
# row/col index into that year's screenshot grid (0-based, row-major).
# Coordinates curated from photo content; `needs_review` flags a guess.
LOCATIONS = [
    # ---------------- 2019 (screenshot 2x5) ----------------
    dict(year=2019, name="Washington", slug="washington", lat=38.8977, lng=-77.0365, row=0, col=0,
         reflection="Stood at the fence like every other tourist. Still got me, seeing it in person."),
    dict(year=2019, name="Durham", slug="durham", lat=36.0016, lng=-78.9403, row=0, col=1,
         reflection="The chapel with the blossoms out front. Spring showed up right on time that year."),
    dict(year=2019, name="Duke University", slug="duke_university", lat=36.0014, lng=-78.9382, row=0, col=2,
         reflection="Sat inside the chapel and mostly just looked up. Photos don't do the ceiling justice."),
    dict(year=2019, name="New York", slug="new_york", lat=40.7484, lng=-73.9857, row=0, col=3,
         reflection="Saw the skyline from up high for the first time. The Chrysler Building holds up in person."),
    dict(year=2019, name="Chapel Hill", slug="chapel_hill", lat=35.9132, lng=-79.0558, row=0, col=4,
         reflection="Out in the crowd after dark. One of those nights the whole town was in the street."),
    dict(year=2019, name="South Bend", slug="south_bend", lat=41.6764, lng=-86.2520, row=1, col=0,
         reflection="Post-flight spread on the counter. Sweet tea and whatever was left out — exactly what I needed."),
    dict(year=2019, name="Arlington", slug="arlington", lat=38.8783, lng=-77.0687, row=1, col=1,
         reflection="Quiet morning at the amphitheater. Some places make you lower your voice without thinking about it."),
    dict(year=2019, name="Duke East Campus", slug="duke_east_campus", lat=36.0098, lng=-78.9139, row=1, col=2,
         reflection="Bare trees and scaffolding, blue sky over all of it. Campus in the in-between season."),
    dict(year=2019, name="University of North Carolina at Chapel Hill", slug="unc_chapel_hill", lat=35.9120, lng=-79.0510, row=1, col=3,
         reflection="Made it to the Old Well. Had to do the walk everyone tells you about."),
    dict(year=2019, name="Tallahassee", slug="tallahassee", lat=30.4383, lng=-84.2807, row=1, col=4,
         reflection="Rained most of the day. Stood under the oaks and waited it out."),

    # ---------------- 2020 (screenshot 3x5) ----------------
    dict(year=2020, name="Merritt Island", slug="merritt_island", lat=28.3600, lng=-80.7000, row=0, col=0,
         reflection="Christmas on the island. Gifts everywhere and nowhere we had to be."),
    dict(year=2020, name="New Orleans", slug="new_orleans", lat=29.9450, lng=-90.0600, row=0, col=1,
         reflection="Grey sky over the river, bridge running off into it. Didn't need the sun for this one."),
    dict(year=2020, name="Cochabamba", slug="cochabamba", lat=-17.3895, lng=-66.1568, row=0, col=2,
         reflection="Afternoon in the field, no plan to speak of. Some of the best days go like that."),
    dict(year=2020, name="Atlanta", slug="atlanta", lat=33.7907, lng=-84.3733, row=0, col=3,
         reflection="The garden had that giant living sculpture. Stood there longer than I'll admit."),
    dict(year=2020, name="Edmonston", slug="edmonston", lat=38.9457, lng=-76.9316, row=0, col=4,
         reflection="Burger and fried plantains, which is the only right way to do it."),
    dict(year=2020, name="Hyattsville", slug="hyattsville", lat=38.9559, lng=-76.9455, row=1, col=0,
         reflection="In the car before heading out. Fresh braids, long day ahead."),
    dict(year=2020, name="Downtown", slug="downtown", lat=35.9917, lng=-78.9028, row=1, col=1, needs_review=True,
         reflection="Ballpark at dusk, nothing riding on the game. Just a good place to sit for a while."),
    dict(year=2020, name="Tysons", slug="tysons", lat=38.9187, lng=-77.2311, row=1, col=2,
         reflection="Courts were empty under a grey sky. Had the whole place to myself."),
    dict(year=2020, name="Jinja", slug="jinja", lat=0.4244, lng=33.2041, row=1, col=3,
         reflection="Cake by the Nile. Still can't believe that sentence is true."),
    dict(year=2020, name="Chicago", slug="chicago", lat=41.9742, lng=-87.9073, row=1, col=4,
         reflection="Just passing through, like always with this airport. Watched the terminal move for a bit."),
    dict(year=2020, name="Silver Spring", slug="silver_spring", lat=38.9907, lng=-77.0261, row=2, col=0,
         reflection="Evening on the courts, sun going down behind the pines. Didn't want to head in."),
    dict(year=2020, name="Dublin", slug="dublin", lat=53.3478, lng=-6.2306, row=2, col=1,
         reflection="Grey the whole time and I didn't mind. Walked the Liffey until my feet gave out."),
    dict(year=2020, name="Baltimore", slug="baltimore", lat=39.2904, lng=-76.6122, row=2, col=2,
         reflection="Barbecue out of the container, mac and beans on the side. No plates, no complaints."),
    dict(year=2020, name="Reston", slug="reston", lat=38.9586, lng=-77.3570, row=2, col=3,
         reflection="Sun in the mirror on the way out. Nothing to the photo, but I remember the drive."),
    dict(year=2020, name="Nassau", slug="nassau", lat=25.0443, lng=-77.3504, row=2, col=4,
         reflection="Ended the day out on the water. Didn't get a good shot until the sun did the work for me."),

    # ---------------- 2021 (screenshot 3x5) ----------------
    dict(year=2021, name="Columbia", slug="columbia", lat=39.2037, lng=-76.8610, row=0, col=0,
         reflection="Something going on behind the fence, music in the distance. Summer starting to open back up."),
    dict(year=2021, name="Cambridge", slug="cambridge", lat=42.3736, lng=-71.1097, row=0, col=1,
         reflection="Rooftops and the skyline past them, everything turning. Fall came in fast that year."),
    dict(year=2021, name="Chillum", slug="chillum", lat=38.9640, lng=-76.9830, row=0, col=2,
         reflection="Went looking for a teapot and left with more than that. Half the fun is the browsing."),
    dict(year=2021, name="Njeru", slug="njeru", lat=0.4530, lng=33.1560, row=0, col=3,
         reflection="Open field, one person way off in the distance. Quiet in a way I still think about."),
    dict(year=2021, name="Raleigh", slug="raleigh", lat=35.7796, lng=-78.6382, row=0, col=4,
         reflection="Pumpkins with the masks still on. That was the fall we had, and it was still a good one."),
    dict(year=2021, name="North Myrtle Beach", slug="north_myrtle_beach", lat=33.8160, lng=-78.6800, row=1, col=0,
         reflection="Sun going down over the low roofs. Stood on the balcony till it was gone."),
    dict(year=2021, name="Boston", slug="boston", lat=42.3551, lng=-71.0656, row=1, col=1,
         reflection="Cut through the Common with the skyline ahead. Grey day, good walk."),
    dict(year=2021, name="Mishawaka", slug="mishawaka", lat=41.6620, lng=-86.1586, row=1, col=2,
         reflection="Meet day on the red track. Cold in the stands, worth it."),
    dict(year=2021, name="Tropical Park", slug="tropical_park", lat=25.7290, lng=-80.3290, row=1, col=3,
         reflection="Holiday plate in Miami, the table packed. These are the ones I hold onto."),
    dict(year=2021, name="College Park", slug="college_park", lat=38.9807, lng=-76.9370, row=1, col=4,
         reflection="Milkshake in the car, red shirt, no reason. Small good moment on an ordinary day."),
    dict(year=2021, name="Riverdale Park", slug="riverdale_park", lat=38.9640, lng=-76.9280, row=2, col=0,
         reflection="Walked the pup through the frost. She was less into the cold than I was."),
    dict(year=2021, name="Philadelphia", slug="philadelphia", lat=39.9526, lng=-75.1652, row=2, col=1,
         reflection="Brunch that took over the whole table. Bagels, coffee, and no rush to be anywhere."),
    dict(year=2021, name="Somerville", slug="somerville", lat=42.3876, lng=-71.0995, row=2, col=2,
         reflection="Cookie the size of my hand, eaten on the walk. Sometimes that's the whole outing."),
    dict(year=2021, name="Cardiff", slug="cardiff", lat=51.4816, lng=-3.1791, row=2, col=3,
         reflection="Pizza and a glass of red after a long day. Nothing fancy, and that was the point."),
    dict(year=2021, name="Orlando", slug="orlando", lat=28.5383, lng=-81.3792, row=2, col=4,
         reflection="French toast and fries because why choose. Slow morning, exactly right."),
]


def crop_photo(loc):
    """Crop one location's tile from its year screenshot into photos/<slug>.jpg."""
    g = GRID[loc["year"]]
    x = g["x0"] + loc["col"] * g["sx"]
    y = g["y0"] + loc["row"] * g["sy"]
    src = SHOTS / f"{loc['year']}.png"
    out = PHOTOS / f"{loc['slug']}.jpg"
    # NOTE: do not combine -c with -Z in one sips call — sips resizes the full
    # image before cropping, so the offset lands out of bounds. Crop only.
    subprocess.run(
        ["sips",
         "-c", str(g["th"]), str(g["tw"]),      # crop to height width
         "--cropOffset", str(y), str(x),        # offset Y X
         "-s", "format", "jpeg",
         str(src), "--out", str(out)],
        check=True, capture_output=True,
    )


def main():
    PHOTOS.mkdir(exist_ok=True)
    DATA.mkdir(exist_ok=True)

    print(f"Cropping {len(LOCATIONS)} photos...")
    for loc in LOCATIONS:
        crop_photo(loc)
    print(f"  {len(list(PHOTOS.glob('*.jpg')))} thumbnails in photos/")

    # Portable dataset
    records = [
        {
            "name": l["name"], "year": l["year"], "lat": l["lat"], "lng": l["lng"],
            "photo": f"photos/{l['slug']}.jpg", "reflection": l["reflection"],
            **({"needs_review": True} if l.get("needs_review") else {}),
        }
        for l in LOCATIONS
    ]
    (DATA / "locations.json").write_text(json.dumps(records, indent=2, ensure_ascii=False), encoding="utf-8")
    print(f"  data/locations.json ({len(records)} locations)")

    # Render self-contained index.html with data injected inline
    html = INDEX_TEMPLATE.replace(
        "__LOCATIONS__", json.dumps(records, separators=(",", ":"), ensure_ascii=False)
    )
    (ROOT / "index.html").write_text(html, encoding="utf-8")
    print("  index.html")

    by_year = {}
    for r in records:
        by_year[r["year"]] = by_year.get(r["year"], 0) + 1
    print("\nDone. Per year:", by_year)
    flagged = [r["name"] for r in records if r.get("needs_review")]
    if flagged:
        print("Needs review (guessed coordinates):", ", ".join(flagged))


INDEX_TEMPLATE = r"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Travels 2019–2021</title>
<link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css"
  integrity="sha256-p4NxAoJBhIIN+hmNHrzRCf9tD/miZyoHS5obTRR9BMY=" crossorigin=""/>
<script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"
  integrity="sha256-20nQCchB9co0qIjJZRGuk2/Z9VM+kNiyxNV1lvTlZBo=" crossorigin=""></script>
<style>
  html, body { margin: 0; height: 100%; font-family: -apple-system, system-ui, sans-serif; }
  #map { position: absolute; inset: 0; background: #f5f5f3; }

  .title-card {
    position: absolute; top: 16px; left: 16px; z-index: 1000;
    background: rgba(255,255,255,.92); backdrop-filter: blur(6px);
    padding: 12px 16px; border-radius: 12px; box-shadow: 0 2px 12px rgba(0,0,0,.12);
  }
  .title-card h1 { margin: 0; font-size: 1.05rem; font-weight: 650; color: #1a1a1a; letter-spacing: -.01em; }
  .title-card p { margin: 3px 0 0; font-size: .78rem; color: #777; }

  .year-filter {
    position: absolute; top: 16px; right: 16px; z-index: 1000; display: flex; gap: 6px;
    background: rgba(255,255,255,.92); backdrop-filter: blur(6px);
    padding: 6px; border-radius: 12px; box-shadow: 0 2px 12px rgba(0,0,0,.12);
  }
  .year-filter button {
    border: none; background: transparent; padding: 6px 12px; border-radius: 8px;
    font-size: .82rem; font-weight: 600; color: #555; cursor: pointer; transition: all .12s;
  }
  .year-filter button:hover { background: #eee; }
  .year-filter button.active { color: #fff; }

  .legend {
    position: absolute; bottom: 20px; left: 16px; z-index: 1000;
    background: rgba(255,255,255,.92); backdrop-filter: blur(6px);
    padding: 10px 14px; border-radius: 10px; box-shadow: 0 2px 12px rgba(0,0,0,.12);
    font-size: .78rem; color: #444;
  }
  .legend .row { display: flex; align-items: center; gap: 8px; margin: 3px 0; }
  .legend .dot { width: 11px; height: 11px; border-radius: 50%; border: 2px solid #fff; box-shadow: 0 0 0 1px rgba(0,0,0,.15); }

  .leaflet-popup-content-wrapper { border-radius: 14px; padding: 0; overflow: hidden; box-shadow: 0 6px 24px rgba(0,0,0,.18); }
  .leaflet-popup-content { margin: 0; width: 260px !important; }
  .pop-img { width: 100%; height: 180px; object-fit: cover; display: block; }
  .pop-body { padding: 12px 14px 14px; }
  .pop-name { font-size: 1rem; font-weight: 650; color: #1a1a1a; line-height: 1.2; }
  .pop-year { display: inline-block; margin-top: 6px; padding: 2px 9px; border-radius: 20px;
              font-size: .72rem; font-weight: 700; color: #fff; }
  .pop-reflection { margin: 9px 0 0; font-size: .86rem; line-height: 1.45; color: #444; }
</style>
</head>
<body>
<div id="map"></div>

<div class="title-card">
  <h1>Travels 2019–2021</h1>
  <p>Click a pin for the photo and a note.</p>
</div>

<div class="year-filter" id="yearFilter"></div>

<div class="legend" id="legend"></div>

<script>
const LOCATIONS = __LOCATIONS__;

const YEAR_COLORS = { 2019: "#0d9488", 2020: "#d97706", 2021: "#e11d48" };
const YEARS = [2019, 2020, 2021];

const map = L.map("map", { zoomControl: false, worldCopyJump: true });
L.control.zoom({ position: "bottomright" }).addTo(map);

L.tileLayer("https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png", {
  attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> &copy; <a href="https://carto.com/attributions">CARTO</a>',
  subdomains: "abcd", maxZoom: 19,
}).addTo(map);

function popupHtml(loc) {
  const color = YEAR_COLORS[loc.year];
  return `<img class="pop-img" src="${loc.photo}" alt="${loc.name}">
    <div class="pop-body">
      <div class="pop-name">${loc.name}</div>
      <span class="pop-year" style="background:${color}">${loc.year}</span>
      <p class="pop-reflection">${loc.reflection}</p>
    </div>`;
}

// One layer group per year
const layers = {};
YEARS.forEach(y => layers[y] = L.layerGroup());

LOCATIONS.forEach(loc => {
  const marker = L.circleMarker([loc.lat, loc.lng], {
    radius: 8, weight: 2, color: "#fff",
    fillColor: YEAR_COLORS[loc.year], fillOpacity: 1,
  }).bindPopup(popupHtml(loc), { closeButton: true, maxWidth: 260 });
  marker.on("mouseover", function () { this.setStyle({ radius: 11 }); });
  marker.on("mouseout", function () { this.setStyle({ radius: 8 }); });
  layers[loc.year].addLayer(marker);
});

function visibleBounds(activeYears) {
  const pts = LOCATIONS.filter(l => activeYears.includes(l.year)).map(l => [l.lat, l.lng]);
  return L.latLngBounds(pts);
}

let current = "all";
function render(sel) {
  current = sel;
  const active = sel === "all" ? YEARS : [sel];
  YEARS.forEach(y => {
    if (active.includes(y)) layers[y].addTo(map);
    else map.removeLayer(layers[y]);
  });
  map.fitBounds(visibleBounds(active), { padding: [60, 60], maxZoom: 8 });
  // button states
  document.querySelectorAll("#yearFilter button").forEach(b => {
    const on = b.dataset.sel === String(sel);
    b.classList.toggle("active", on);
    b.style.background = on ? (sel === "all" ? "#1a1a1a" : YEAR_COLORS[sel]) : "";
  });
}

// Build filter buttons
const filter = document.getElementById("yearFilter");
[["all", "All"], [2019, "2019"], [2020, "2020"], [2021, "2021"]].forEach(([sel, label]) => {
  const b = document.createElement("button");
  b.textContent = label; b.dataset.sel = String(sel);
  b.onclick = () => render(sel === "all" ? "all" : Number(sel));
  filter.appendChild(b);
});

// Legend
document.getElementById("legend").innerHTML = YEARS.map(y =>
  `<div class="row"><span class="dot" style="background:${YEAR_COLORS[y]}"></span>${y}</div>`
).join("");

render("all");
</script>
</body>
</html>"""


if __name__ == "__main__":
    main()
