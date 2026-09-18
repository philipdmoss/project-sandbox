import { buildTimeline, formatClock, type SunTimesResponse } from "@/lib/solar";

interface Props {
  data: SunTimesResponse;
  now: Date;
  timeZone?: string;
}

const WIDTH = 800;
const HEIGHT = 300;
const HORIZON = 214;
const X0 = 60;
const X1 = 740;
const AMPLITUDE = 168;

function pointFor(fraction: number) {
  const clamped = Math.min(1, Math.max(0, fraction));
  return {
    x: X0 + clamped * (X1 - X0),
    y: HORIZON - Math.sin(clamped * Math.PI) * AMPLITUDE,
  };
}

function fractionFor(instant: Date, dawn: Date, dusk: Date) {
  return (
    (instant.getTime() - dawn.getTime()) / (dusk.getTime() - dawn.getTime())
  );
}

export default function SolarArc({ data, now, timeZone }: Props) {
  const timeline = buildTimeline(data);
  const dawn = timeline[0].at;
  const dusk = timeline[timeline.length - 1].at;

  const arcPoints = Array.from({ length: 121 }, (_, i) => pointFor(i / 120));
  const arcPath = arcPoints
    .map((p, i) => `${i === 0 ? "M" : "L"} ${p.x.toFixed(1)} ${p.y.toFixed(1)}`)
    .join(" ");
  const areaPath = `${arcPath} L ${X1} ${HORIZON} L ${X0} ${HORIZON} Z`;

  const markers = [timeline[0], timeline[1], timeline[3], timeline[5], timeline[6]];
  const nowFraction = fractionFor(now, dawn, dusk);
  const sunVisible = nowFraction >= 0 && nowFraction <= 1;
  const sun = pointFor(nowFraction);

  return (
    <svg
      viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
      className="h-auto w-full"
      role="img"
      aria-label="Sun path across the day"
    >
      <defs>
        <linearGradient id="arcFill" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="#ffffff" stopOpacity="0.28" />
          <stop offset="100%" stopColor="#ffffff" stopOpacity="0" />
        </linearGradient>
        <radialGradient id="sunGlow">
          <stop offset="0%" stopColor="#fff7e0" />
          <stop offset="100%" stopColor="#ffcf6b" />
        </radialGradient>
      </defs>

      <path d={areaPath} fill="url(#arcFill)" />
      <line
        x1={X0 - 20}
        y1={HORIZON}
        x2={X1 + 20}
        y2={HORIZON}
        stroke="#ffffff"
        strokeOpacity="0.35"
        strokeWidth="1"
      />
      <path
        d={arcPath}
        fill="none"
        stroke="#ffffff"
        strokeOpacity="0.7"
        strokeWidth="2"
        strokeDasharray="4 6"
      />

      {markers.map((marker) => {
        const point = pointFor(fractionFor(marker.at, dawn, dusk));
        return (
          <g key={marker.label}>
            <circle cx={point.x} cy={point.y} r="4" fill="#ffffff" />
            <text
              x={point.x}
              y={point.y - 14}
              textAnchor="middle"
              fill="#ffffff"
              fillOpacity="0.85"
              fontSize="13"
              fontWeight="500"
            >
              {marker.label}
            </text>
            <text
              x={point.x}
              y={point.y + 20}
              textAnchor="middle"
              fill="#ffffff"
              fillOpacity="0.6"
              fontSize="12"
            >
              {formatClock(marker.at, timeZone)}
            </text>
          </g>
        );
      })}

      {sunVisible && (
        <circle cx={sun.x} cy={sun.y} r="12" fill="url(#sunGlow)">
          <animate
            attributeName="r"
            values="11;13;11"
            dur="3s"
            repeatCount="indefinite"
          />
        </circle>
      )}
    </svg>
  );
}
