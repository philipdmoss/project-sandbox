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
const HEIGHT = 396;
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
  const [smooth, setSmooth] = useState(true);

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

  // Golden/blue-hour highlights and their side-placed labels. The label sits on
  // the outer side of each highlight: morning blue on the left / golden on the
  // right, and the reverse in the evening.
  const bandDefs = [
    { win: data.morning_blue_hour, color: "#5b6cd6", name: "Blue Hour", labelSide: "left" as const },
    { win: data.morning_golden_hour, color: "#e2a036", name: "Golden Hour", labelSide: "right" as const },
    { win: data.evening_golden_hour, color: "#e2a036", name: "Golden Hour", labelSide: "left" as const },
    { win: data.evening_blue_hour, color: "#5b6cd6", name: "Blue Hour", labelSide: "right" as const },
  ]
    .filter((d): d is { win: TwilightWindow; color: string; name: string; labelSide: "left" | "right" } => Boolean(d.win))
    .map((d) => {
      const x = xFor(new Date(d.win.start).getTime());
      const w = xFor(new Date(d.win.end).getTime()) - x;
      const labelX = d.labelSide === "left" ? x - 8 : x + w + 8;
      return { ...d, x, w, labelX, range: rangeText(d.win) };
    });

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

  // Twilight labels, styled like the key markers (name + time, two lines).
  // Dawns stack in the bottom-left, dusks in the bottom-right.
  const twilightLeft = [
    data.astronomical_twilight?.dawn && { label: "Astronomical dawn", iso: data.astronomical_twilight.dawn },
    data.nautical_twilight?.dawn && { label: "Nautical dawn", iso: data.nautical_twilight.dawn },
  ].filter(Boolean) as { label: string; iso: string }[];
  const twilightRight = [
    data.nautical_twilight?.dusk && { label: "Nautical dusk", iso: data.nautical_twilight.dusk },
    data.astronomical_twilight?.dusk && { label: "Astronomical dusk", iso: data.astronomical_twilight.dusk },
  ].filter(Boolean) as { label: string; iso: string }[];

  const twilightLabel = (c: { label: string; iso: string }, x: number, y: number, anchor: "start" | "end", key: string) => (
    <g key={key}>
      <text x={x} y={y} textAnchor={anchor} fill="#ffffff" fillOpacity="0.95" fontSize="14" fontWeight="600" {...outline}>
        {c.label}
      </text>
      <text x={x} y={y + 16} textAnchor={anchor} fill="#ffffff" fillOpacity="0.7" fontSize="13" {...outline}>
        {formatClock(new Date(c.iso), timeZone)}
      </text>
    </g>
  );

  return (
    <div>
      <div className="mb-2 flex items-center justify-end gap-1 text-xs">
        <span className="mr-1 text-white/50">Arc</span>
        {(["smooth", "actual"] as const).map((mode) => {
          const active = (mode === "smooth") === smooth;
          return (
            <button
              key={mode}
              type="button"
              onClick={() => setSmooth(mode === "smooth")}
              className={`rounded-md px-2.5 py-1 transition ${
                active
                  ? "bg-white/90 text-slate-900 ring-1 ring-white"
                  : "bg-white/10 text-white/80 ring-1 ring-white/20 hover:bg-white/20"
              }`}
            >
              {mode === "smooth" ? "Smooth" : "Actual height"}
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

        {bandDefs.map((b, i) => (
          <rect key={`band${i}`} x={b.x} y={TOP} width={b.w} height={BOTTOM - TOP} fill={b.color} fillOpacity="0.22" />
        ))}

        <path d={areaPath} fill="url(#arcFill)" />
        <line x1={X0 - 30} y1={horizonY} x2={X1 + 30} y2={horizonY} stroke="#ffffff" strokeOpacity="0.35" strokeWidth="1" />
        <text x={X1 + 32} y={horizonY - 4} textAnchor="end" fill="#ffffff" fillOpacity="0.4" fontSize="11">
          horizon
        </text>
        <path d={arcPath} fill="none" stroke="#ffffff" strokeOpacity="0.75" strokeWidth="2" strokeDasharray="4 6" />

        {/* Golden/blue-hour labels, beside their highlights */}
        {bandDefs.map((b, i) => {
          const anchor = b.labelSide === "left" ? "end" : "start";
          return (
            <g key={`bl${i}`}>
              <text x={b.labelX} y={96} textAnchor={anchor} fill="#ffffff" fillOpacity="0.95" fontSize="14" fontWeight="600" {...outline}>
                {b.name}
              </text>
              <text x={b.labelX} y={112} textAnchor={anchor} fill="#ffffff" fillOpacity="0.7" fontSize="12" {...outline}>
                {b.range}
              </text>
            </g>
          );
        })}

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

        {/* Dots on the arc for each twilight point */}
        {[...twilightLeft, ...twilightRight].map((c, i) => {
          const t = new Date(c.iso).getTime();
          return (
            <circle key={`td${i}`} cx={xFor(t)} cy={yFor(elevationForMode(t))} r="3.5" fill="#ffffff" fillOpacity="0.85" />
          );
        })}

        {twilightLeft.map((c, i) => twilightLabel(c, X0 - 10, BOTTOM + 30 + i * 38, "start", `tl${i}`))}
        {twilightRight.map((c, i) => twilightLabel(c, X1 + 10, BOTTOM + 30 + i * 38, "end", `tr${i}`))}
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
