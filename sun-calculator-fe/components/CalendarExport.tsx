"use client";

import { useState } from "react";
import { CALENDAR_PHASES, type CalendarPhaseKey } from "@/lib/calendar";

interface Props {
  coords: { lat: number; lng: number };
  timeZone?: string;
}

function currentYear(timeZone?: string): number {
  const formatted = new Intl.DateTimeFormat("en-US", {
    timeZone,
    year: "numeric",
  }).format(new Date());
  return Number(formatted);
}

export default function CalendarExport({ coords, timeZone }: Props) {
  const [selected, setSelected] = useState<Record<CalendarPhaseKey, boolean>>({
    sunrise: true,
    sunset: true,
    golden_hour: true,
    blue_hour: true,
  });
  const [downloading, setDownloading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const chosen = CALENDAR_PHASES.filter((p) => selected[p.key]).map((p) => p.key);
  const year = currentYear(timeZone);

  async function download() {
    if (!chosen.length || downloading) return;
    setDownloading(true);
    setError(null);

    const params = new URLSearchParams({
      lat: String(coords.lat),
      lng: String(coords.lng),
      phases: chosen.join(","),
      year: String(year),
    });
    if (timeZone) params.set("tz", timeZone);

    try {
      const response = await fetch(`/api/calendar?${params.toString()}`);
      if (!response.ok) {
        const body = await response.json().catch(() => null);
        throw new Error(body?.error ?? "Couldn't build the calendar.");
      }
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = `sun-phases-${year}.ics`;
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      URL.revokeObjectURL(url);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Couldn't build the calendar.",
      );
    } finally {
      setDownloading(false);
    }
  }

  return (
    <div className="rounded-2xl bg-black/40 p-6 ring-1 ring-white/15 backdrop-blur">
      <p className="text-xs uppercase tracking-widest text-white/50">
        Add to calendar
      </p>
      <p className="mt-1 text-sm text-white/70">
        Download a full year of sun phases as an .ics file for {year}.
      </p>
      <div className="mt-4 flex flex-wrap gap-2">
        {CALENDAR_PHASES.map((p) => (
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
      <button
        type="button"
        onClick={download}
        disabled={!chosen.length || downloading}
        className="mt-4 inline-block rounded-xl bg-white px-4 py-2 text-sm font-medium text-slate-900 transition hover:bg-white/90 disabled:cursor-not-allowed disabled:bg-white/20 disabled:text-white/40"
      >
        {downloading ? "Preparing…" : "Download .ics"}
      </button>
      {error && <p className="mt-3 text-sm text-rose-300">{error}</p>}
    </div>
  );
}
