"use client";

import { useState } from "react";

interface Props {
  coords: { lat: number; lng: number };
  timeZone?: string;
}

const PHASES = [
  { key: "sunrise", label: "Sunrise" },
  { key: "sunset", label: "Sunset" },
  { key: "golden_hour", label: "Golden hour" },
  { key: "blue_hour", label: "Blue hour" },
];

export default function CalendarExport({ coords, timeZone }: Props) {
  const [selected, setSelected] = useState<Record<string, boolean>>({
    sunrise: true,
    sunset: true,
    golden_hour: true,
    blue_hour: true,
  });

  const chosen = PHASES.filter((p) => selected[p.key]).map((p) => p.key);
  const year = new Date().getFullYear();

  const params = new URLSearchParams({
    lat: String(coords.lat),
    lng: String(coords.lng),
    phases: chosen.join(","),
    year: String(year),
  });
  if (timeZone) params.set("tz", timeZone);
  const href = `/api/calendar?${params.toString()}`;

  return (
    <div className="rounded-2xl bg-black/40 p-6 ring-1 ring-white/15 backdrop-blur">
      <p className="text-xs uppercase tracking-widest text-white/50">
        Add to calendar
      </p>
      <p className="mt-1 text-sm text-white/70">
        Download a full year of sun phases as an .ics file for {year}.
      </p>
      <div className="mt-4 flex flex-wrap gap-2">
        {PHASES.map((p) => (
          <button
            key={p.key}
            type="button"
            aria-pressed={selected[p.key]}
            onClick={() =>
              setSelected((s) => ({ ...s, [p.key]: !s[p.key] }))
            }
            className={`rounded-full px-3 py-1 text-sm ring-1 transition ${
              selected[p.key]
                ? "bg-amber-300/90 text-slate-900 ring-amber-200"
                : "bg-white/5 text-white/70 ring-white/20 hover:bg-white/10"
            }`}
          >
            {p.label}
          </button>
        ))}
      </div>
      <a
        href={chosen.length ? href : undefined}
        download
        aria-disabled={chosen.length === 0}
        className={`mt-4 inline-block rounded-xl px-4 py-2 text-sm font-medium transition ${
          chosen.length
            ? "bg-white text-slate-900 hover:bg-white/90"
            : "pointer-events-none bg-white/20 text-white/40"
        }`}
      >
        Download .ics
      </a>
    </div>
  );
}
