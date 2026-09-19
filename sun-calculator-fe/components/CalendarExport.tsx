"use client";

import { useState } from "react";

interface Props {
  lat: number;
  lng: number;
  timeZone?: string;
}

const PHASES = [
  { key: "sunrise", label: "Sunrise" },
  { key: "sunset", label: "Sunset" },
  { key: "blue_hour", label: "Blue hour" },
  { key: "golden_hour", label: "Golden hour" },
];

export default function CalendarExport({ lat, lng, timeZone }: Props) {
  const [selected, setSelected] = useState<string[]>(PHASES.map((p) => p.key));
  const [copied, setCopied] = useState(false);

  const toggle = (key: string) =>
    setSelected((current) =>
      current.includes(key) ? current.filter((k) => k !== key) : [...current, key],
    );

  const phasesParam = selected.length === PHASES.length ? "all" : selected.join(",");
  const disabled = selected.length === 0;

  const params = new URLSearchParams({
    lat: String(lat),
    lng: String(lng),
    phases: phasesParam,
  });
  if (timeZone) params.set("tz", timeZone);
  const query = params.toString();
  const downloadHref = `/api/calendar?${query}`;

  const subscribe = () => {
    const { host } = window.location;
    window.location.href = `webcal://${host}/api/calendar/feed?${query}`;
  };

  const copyFeed = async () => {
    const { origin } = window.location;
    try {
      await navigator.clipboard.writeText(`${origin}/api/calendar/feed?${query}`);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // clipboard unavailable — ignore
    }
  };

  return (
    <div className="rounded-2xl bg-black/25 p-4 ring-1 ring-white/15 backdrop-blur">
      <div className="text-xs uppercase tracking-widest text-white/50">
        Add to calendar
      </div>
      <p className="mt-1 text-sm text-white/70">
        A year of these times as a calendar (.ics), or subscribe to stay current.
      </p>

      <div className="mt-3 flex flex-wrap gap-2">
        {PHASES.map((phase) => {
          const on = selected.includes(phase.key);
          return (
            <button
              key={phase.key}
              type="button"
              onClick={() => toggle(phase.key)}
              aria-pressed={on}
              className={`rounded-full px-3 py-1 text-sm ring-1 transition ${
                on
                  ? "bg-white/90 text-slate-900 ring-white"
                  : "bg-white/10 text-white/70 ring-white/20 hover:bg-white/20"
              }`}
            >
              {phase.label}
            </button>
          );
        })}
      </div>

      <div className="mt-4 flex flex-wrap items-center gap-2">
        <a
          href={disabled ? undefined : downloadHref}
          aria-disabled={disabled}
          className={`rounded-xl px-4 py-2 text-sm font-medium transition ${
            disabled
              ? "pointer-events-none bg-white/20 text-white/40"
              : "bg-white/90 text-slate-900 hover:bg-white"
          }`}
        >
          Download .ics
        </a>
        <button
          type="button"
          onClick={subscribe}
          disabled={disabled}
          className="rounded-xl bg-white/10 px-4 py-2 text-sm text-white ring-1 ring-white/20 transition hover:bg-white/20 disabled:opacity-50"
        >
          Subscribe
        </button>
        <button
          type="button"
          onClick={copyFeed}
          disabled={disabled}
          className="rounded-xl px-3 py-2 text-sm text-white/70 transition hover:text-white disabled:opacity-50"
        >
          {copied ? "Copied!" : "Copy feed URL"}
        </button>
      </div>
    </div>
  );
}
