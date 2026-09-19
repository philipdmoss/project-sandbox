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
const HEIGHT = 340;
const X0 = 60;
const X1 = 740;
const TOP = 34;
const BOTTOM = 300;

export default function SolarArc({ data, now, timeZone, peakAltitude }: Props) {
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
  const arcPath = anchors
    .map((a, i) => `${i === 0 ? "M" : "L"} ${xFor(a.at.getTime()).toFixed(1)} ${yFor(a.elevation).toFixed(1)}`)
    .join(" ");
  const areaPath = `${arcPath} L ${xFor(last).toFixed(1)} ${horizonY.toFixed(1)} L ${xFor(first).toFixed(1)} ${horizonY.toFixed(1)} Z`;

  // Golden/blue-hour periods drawn as tinted vertical bands under the arc.
  const bandRect = (win: TwilightWindow | undefined) => {
    if (!win) return null;
    const x = xFor(new Date(win.start).getTime());
    const w = xFor(new Date(win.end).getTime()) - x;
    return { x, w };
  };
  const goldBands = [data.morning_golden_hour, data.evening_golden_hour]
    .map(bandRect)
    .filter((b): b is { x: number; w: number } => b !== null);
  const blueBands = [data.morning_blue_hour, data.evening_blue_hour]
    .map(bandRect)
    .filter((b): b is { x: number; w: number } => b !== null);

  const nowMs = now.getTime();
  const sunVisible = nowMs >= first && nowMs <= last;
  const sunElevation = elevationAt(anchors, now);
  const sunX = xFor(nowMs);
  const sunY = yFor(sunElevation);

  const markers = [
    { label: "Sunrise", iso: data.sunrise },
    { label: "Solar noon", iso: data.solar_noon },
    { label: "Sunset", iso: data.sunset },
  ].filter((m): m is { label: string; iso: string } => Boolean(m.iso));

  return (
    <div>
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

        {/* Golden-hour and blue-hour periods */}
        {goldBands.map((b, i) => (
          <rect key={`g${i}`} x={b.x} y={TOP} width={b.w} height={BOTTOM - TOP} fill="#e2a036" fillOpacity="0.22" />
        ))}
        {blueBands.map((b, i) => (
          <rect key={`b${i}`} x={b.x} y={TOP} width={b.w} height={BOTTOM - TOP} fill="#5b6cd6" fillOpacity="0.22" />
        ))}

        {/* Area + horizon + arc */}
        <path d={areaPath} fill="url(#arcFill)" />
        <line x1={X0 - 20} y1={horizonY} x2={X1 + 20} y2={horizonY} stroke="#ffffff" strokeOpacity="0.35" strokeWidth="1" />
        <text x={X1 + 22} y={horizonY - 4} textAnchor="end" fill="#ffffff" fillOpacity="0.4" fontSize="11">
          horizon
        </text>
        <path d={arcPath} fill="none" stroke="#ffffff" strokeOpacity="0.75" strokeWidth="2" strokeDasharray="4 6" />

        {/* Now sun */}
        {sunVisible && (
          <g>
            <circle cx={sunX} cy={sunY} r="16" fill="url(#sunGlow)">
              <animate attributeName="r" values="15;18;15" dur="3s" repeatCount="indefinite" />
            </circle>
            <text
              x={sunX}
              y={sunY - 24}
              textAnchor="middle"
              fill="#ffffff"
              fontSize="12"
              stroke="#0b1024"
              strokeWidth="3"
              strokeOpacity="0.55"
              paintOrder="stroke"
            >
              {`${Math.round(sunElevation)}°`}
            </text>
          </g>
        )}

        {/* Key markers */}
        {markers.map((m) => {
          const t = new Date(m.iso).getTime();
          const e = elevationAt(anchors, new Date(m.iso));
          const x = xFor(t);
          const y = yFor(e);
          const below = e < 3;
          const labelY = below ? horizonY + 26 : y - 26;
          const timeY = below ? horizonY + 42 : y - 12;
          return (
            <g key={m.label}>
              <circle cx={x} cy={y} r="4" fill="#ffffff" />
              <text x={x} y={labelY} textAnchor="middle" fill="#ffffff" fillOpacity="0.95" fontSize="14" fontWeight="600" stroke="#0b1024" strokeWidth="3" strokeOpacity="0.55" paintOrder="stroke">
                {m.label}
              </text>
              <text x={x} y={timeY} textAnchor="middle" fill="#ffffff" fillOpacity="0.7" fontSize="13" stroke="#0b1024" strokeWidth="3" strokeOpacity="0.55" paintOrder="stroke">
                {formatClock(new Date(m.iso), timeZone)}
              </text>
            </g>
          );
        })}
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
