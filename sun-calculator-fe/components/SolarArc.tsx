import { buildTimeline, formatClock, type SunTimesResponse } from "@/lib/solar";

interface Props {
  data: SunTimesResponse;
  now: Date;
  timeZone?: string;
}

const WIDTH = 800;
const HEIGHT = 340;
const HORIZON = 245;
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

  const shownMarkers = [timeline[0], timeline[1], timeline[3], timeline[5], timeline[6]];
  const nowFraction = fractionFor(now, dawn, dusk);
  const sunVisible = nowFraction >= 0 && nowFraction <= 1;
  const sun = pointFor(nowFraction);

  // Lay out labels: endpoints drop below the horizon (empty space, clear of the
  // clustered interior markers), while interior labels sit above the arc and
  // stagger upward whenever they land too close in x to the previous one.
  const MIN_X_GAP = 96;
  const STAGGER = 30;
  const lastIndex = shownMarkers.length - 1;
  let prevX = -Infinity;
  let level = 0;
  const laidOut = shownMarkers.map((marker, i) => {
    const point = pointFor(fractionFor(marker.at, dawn, dusk));
    const isEndpoint = i === 0 || i === lastIndex;
    let labelY: number;
    let timeY: number;
    if (isEndpoint) {
      labelY = HORIZON + 30;
      timeY = HORIZON + 46;
    } else {
      level = point.x - prevX < MIN_X_GAP ? level + 1 : 0;
      prevX = point.x;
      labelY = point.y - 28 - level * STAGGER;
      timeY = point.y - 14 - level * STAGGER;
    }
    return { marker, point, labelY, timeY, isEndpoint };
  });

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

      {sunVisible && (
        <circle cx={sun.x} cy={sun.y} r="18" fill="url(#sunGlow)">
          <animate
            attributeName="r"
            values="17;20;17"
            dur="3s"
            repeatCount="indefinite"
          />
        </circle>
      )}

      {laidOut.map(({ marker, point, labelY, timeY, isEndpoint }) => {
        const leaderY = isEndpoint ? labelY - 12 : timeY + 4;
        const hasLeader = isEndpoint || Math.abs(point.y - timeY) > 20;
        return (
          <g key={marker.label}>
            {hasLeader && (
              <line
                x1={point.x}
                y1={point.y}
                x2={point.x}
                y2={leaderY}
                stroke="#ffffff"
                strokeOpacity="0.3"
                strokeWidth="1"
              />
            )}
            <circle cx={point.x} cy={point.y} r="4" fill="#ffffff" />
            <text
              x={point.x}
              y={labelY}
              textAnchor="middle"
              fill="#ffffff"
              fillOpacity="0.95"
              fontSize="14"
              fontWeight="600"
              fontFamily="var(--font-geist-sans), system-ui, sans-serif"
              stroke="#0b1024"
              strokeWidth="3"
              strokeOpacity="0.55"
              strokeLinejoin="round"
              paintOrder="stroke"
            >
              {marker.label}
            </text>
            <text
              x={point.x}
              y={timeY}
              textAnchor="middle"
              fill="#ffffff"
              fillOpacity="0.7"
              fontSize="13"
              fontFamily="var(--font-geist-sans), system-ui, sans-serif"
              stroke="#0b1024"
              strokeWidth="3"
              strokeOpacity="0.55"
              strokeLinejoin="round"
              paintOrder="stroke"
            >
              {formatClock(marker.at, timeZone)}
            </text>
          </g>
        );
      })}
    </svg>
  );
}
