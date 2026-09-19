"use client";

import { useState } from "react";
import {
  buildElevationAnchors,
  elevationAt,
  formatClock,
  type SunTimesResponse,
  type TwilightWindow,
} from "@/lib/solar";

interface Props {
  data: SunTimesResponse;
  now: Date;
  timeZone?: string;
  peakAltitude: number | null;
}

const WIDTH = 800;
const HEIGHT = 360;
const X0 = 70;
const X1 = 730;
const TOP = 40;
const BOTTOM = 300;

// Shared text styling matching the existing arc labels (white fill, dark
// outline via paint-order stroke).
const outline = {
  stroke: "#0b1024",
  strokeWidth: 3,
  strokeOpacity: 0.55,
  strokeLinejoin: "round" as const,
  paintOrder: "stroke" as const,
};

export default function SolarArc({ data, now, timeZone, peakAltitude }: Props) {
  const [smooth, setSmooth] = useState(false);

  const anchors = buildElevationAnchors(data, peakAltitude);
  if (anchors.length < 2) {
    return (
      <p className="py-8 text-center text-sm text-white/60">
        The sun doesn&apos;t rise and set here today.
      </p>
    );
  }

  const first = anchors[0].at.getTime();
  const last = anchors[anchors.length - 1].at.getTime();
  const span = last - first || 1;
  const elevations = anchors.map((a) => a.elevation);
  const eMax = Math.max(...elevations, 6);
  const eMin = Math.min(...elevations, -6);

  const xFor = (t: number) => X0 + ((t - first) / span) * (X1 - X0);
  const yFor = (e: number) => BOTTOM - ((e - eMin) / (eMax - eMin)) * (BOTTOM - TOP);
  const horizonY = yFor(0);

  // Smooth mode: a clean symmetric bell peaking at solar noon. Actual mode: the
  // true elevation at each instant.
  const smoothElevation = (t: number) => {
    const f = Math.min(1, Math.max(0, (t - first) / span));
    return eMin + (eMax - eMin) * Math.sin(f * Math.PI);
  };
  const elevationForMode = (t: number) =>
    smooth ? smoothElevation(t) : elevationAt(anchors, new Date(t));

  const arcPoints = smooth
    ? Array.from({ length: 121 }, (_, i) => {
        const t = first + (i / 120) * span;
        return { x: xFor(t), y: yFor(smoothElevation(t)) };
      })
    : anchors.map((a) => ({ x: xFor(a.at.getTime()), y: yFor(a.elevation) }));
  const arcPath = arcPoints
    .map((p, i) => `${i === 0 ? "M" : "L"} ${p.x.toFixed(1)} ${p.y.toFixed(1)}`)
    .join(" ");
  const areaPath = `${arcPath} L ${xFor(last).toFixed(1)} ${horizonY.toFixed(1)} L ${xFor(first).toFixed(1)} ${horizonY.toFixed(1)} Z`;

  const rangeText = (win: TwilightWindow) =>
    `${formatClock(new Date(win.start), timeZone)} – ${formatClock(new Date(win.end), timeZone)}`;
  const band = (win: TwilightWindow | undefined) =>
    win ? { x: xFor(new Date(win.start).getTime()), w: xFor(new Date(win.end).getTime()) - xFor(new Date(win.start).getTime()) } : null;

  const goldBands = [data.morning_golden_hour, data.evening_golden_hour]
    .map(band)
    .filter((b): b is { x: number; w: number } => b !== null);
  const blueBands = [data.morning_blue_hour, data.evening_blue_hour]
    .map(band)
    .filter((b): b is { x: number; w: number } => b !== null);

  // Band legend labels: morning stacked top-left, evening stacked top-right.
  const morningLabels = [
    data.morning_blue_hour && { name: "Blue Hour", range: rangeText(data.morning_blue_hour) },
    data.morning_golden_hour && { name: "Golden Hour", range: rangeText(data.morning_golden_hour) },
  ].filter(Boolean) as { name: string; range: string }[];
  const eveningLabels = [
    data.evening_golden_hour && { name: "Golden Hour", range: rangeText(data.evening_golden_hour) },
    data.evening_blue_hour && { name: "Blue Hour", range: rangeText(data.evening_blue_hour) },
  ].filter(Boolean) as { name: string; range: string }[];

  const nowMs = now.getTime();
  const sunVisible = nowMs >= first && nowMs <= last;
  const sunX = xFor(nowMs);
  const sunY = yFor(elevationForMode(nowMs));
  const sunAltitude = Math.round(elevationAt(anchors, now));

  const keyMarkers = [
    { label: "Sunrise", iso: data.sunrise },
    { label: "Solar noon", iso: data.solar_noon },
    { label: "Sunset", iso: data.sunset },
  ].filter((m): m is { label: string; iso: string } => Boolean(m.iso));

  const twilightCaptions = {
    left: [
      data.astronomical_twilight?.dawn && { label: "Astronomical dawn", iso: data.astronomical_twilight.dawn },
      data.nautical_twilight?.dawn && { label: "Nautical dawn", iso: data.nautical_twilight.dawn },
    ].filter(Boolean) as { label: string; iso: string }[],
    right: [
      data.nautical_twilight?.dusk && { label: "Nautical dusk", iso: data.nautical_twilight.dusk },
      data.astronomical_twilight?.dusk && { label: "Astronomical dusk", iso: data.astronomical_twilight.dusk },
    ].filter(Boolean) as { label: string; iso: string }[],
  };

  return (
    <div>
      <div className="mb-2 flex items-center justify-end gap-1 text-xs">
        <span className="mr-1 text-white/50">Arc</span>
        {(["actual", "smooth"] as const).map((mode) => {
          const active = (mode === "smooth") === smooth;
          return (
            <button
              key={mode}
              type="button"
              onClick={() => setSmooth(mode === "smooth")}
              className={`rounded-md px-2.5 py-1 capitalize ring-1 transition ${
                active
                  ? "bg-white/90 text-slate-900 ring-white"
                  : "bg-white/10 text-white/80 ring-white/20 hover:bg-white/20"
              }`}
            >
              {mode === "actual" ? "Actual height" : "Smooth"}
            </button>
          );
        })}
      </div>

      <svg
        viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
        className="h-auto w-full"
        role="img"
        aria-label="Sun altitude across the day, with golden and blue hours marked"
      >
        <defs>
          <linearGradient id="arcFill" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#ffffff" stopOpacity="0.22" />
            <stop offset="100%" stopColor="#ffffff" stopOpacity="0" />
          </linearGradient>
          <radialGradient id="sunGlow">
            <stop offset="0%" stopColor="#fff7e0" />
            <stop offset="100%" stopColor="#ffcf6b" />
          </radialGradient>
        </defs>

        {goldBands.map((b, i) => (
          <rect key={`g${i}`} x={b.x} y={TOP} width={b.w} height={BOTTOM - TOP} fill="#e2a036" fillOpacity="0.22" />
        ))}
        {blueBands.map((b, i) => (
          <rect key={`b${i}`} x={b.x} y={TOP} width={b.w} height={BOTTOM - TOP} fill="#5b6cd6" fillOpacity="0.24" />
        ))}

        <path d={areaPath} fill="url(#arcFill)" />
        <line x1={X0 - 30} y1={horizonY} x2={X1 + 30} y2={horizonY} stroke="#ffffff" strokeOpacity="0.35" strokeWidth="1" />
        <text x={X1 + 32} y={horizonY - 4} textAnchor="end" fill="#ffffff" fillOpacity="0.4" fontSize="11">
          horizon
        </text>
        <path d={arcPath} fill="none" stroke="#ffffff" strokeOpacity="0.75" strokeWidth="2" strokeDasharray="4 6" />

        {/* Band legend labels — morning left, evening right */}
        {morningLabels.map((l, i) => (
          <g key={`ml${i}`}>
            <text x={X0} y={TOP + 4 + i * 40} textAnchor="start" fill="#ffffff" fillOpacity="0.95" fontSize="14" fontWeight="600" {...outline}>
              {l.name}
            </text>
            <text x={X0} y={TOP + 20 + i * 40} textAnchor="start" fill="#ffffff" fillOpacity="0.7" fontSize="12" {...outline}>
              {l.range}
            </text>
          </g>
        ))}
        {eveningLabels.map((l, i) => (
          <g key={`el${i}`}>
            <text x={X1} y={TOP + 4 + i * 40} textAnchor="end" fill="#ffffff" fillOpacity="0.95" fontSize="14" fontWeight="600" {...outline}>
              {l.name}
            </text>
            <text x={X1} y={TOP + 20 + i * 40} textAnchor="end" fill="#ffffff" fillOpacity="0.7" fontSize="12" {...outline}>
              {l.range}
            </text>
          </g>
        ))}

        {sunVisible && (
          <g>
            <circle cx={sunX} cy={sunY} r="16" fill="url(#sunGlow)">
              <animate attributeName="r" values="15;18;15" dur="3s" repeatCount="indefinite" />
            </circle>
            <text x={sunX} y={sunY - 24} textAnchor="middle" fill="#ffffff" fontSize="12" {...outline}>
              {`${sunAltitude}°`}
            </text>
          </g>
        )}

        {keyMarkers.map((m) => {
          const t = new Date(m.iso).getTime();
          const x = xFor(t);
          const y = yFor(elevationForMode(t));
          const below = elevationAt(anchors, new Date(m.iso)) < 3;
          const labelY = below ? horizonY + 26 : y - 26;
          const timeY = below ? horizonY + 42 : y - 12;
          return (
            <g key={m.label}>
              <circle cx={x} cy={y} r="4" fill="#ffffff" />
              <text x={x} y={labelY} textAnchor="middle" fill="#ffffff" fillOpacity="0.95" fontSize="14" fontWeight="600" {...outline}>
                {m.label}
              </text>
              <text x={x} y={timeY} textAnchor="middle" fill="#ffffff" fillOpacity="0.7" fontSize="13" {...outline}>
                {formatClock(new Date(m.iso), timeZone)}
              </text>
            </g>
          );
        })}

        {/* Twilight labels — dawns bottom-left, dusks bottom-right */}
        {twilightCaptions.left.map((c, i) => (
          <text key={`tl${i}`} x={X0 - 20} y={BOTTOM + 22 + i * 16} textAnchor="start" fill="#ffffff" fillOpacity="0.65" fontSize="11" {...outline}>
            {c.label} {formatClock(new Date(c.iso), timeZone)}
          </text>
        ))}
        {twilightCaptions.right.map((c, i) => (
          <text key={`tr${i}`} x={X1 + 20} y={BOTTOM + 22 + i * 16} textAnchor="end" fill="#ffffff" fillOpacity="0.65" fontSize="11" {...outline}>
            {c.label} {formatClock(new Date(c.iso), timeZone)}
          </text>
        ))}
      </svg>

      <div className="mt-2 flex items-center justify-center gap-4 text-xs text-white/70">
        <span className="flex items-center gap-1.5">
          <span className="inline-block h-2.5 w-2.5 rounded-sm" style={{ backgroundColor: "#e2a036" }} />
          Golden hour
        </span>
        <span className="flex items-center gap-1.5">
          <span className="inline-block h-2.5 w-2.5 rounded-sm" style={{ backgroundColor: "#5b6cd6" }} />
          Blue hour
        </span>
      </div>
    </div>
  );
}
