import { buildTimeline, formatClock, type SunTimesResponse } from "@/lib/solar";

interface Props {
  data: SunTimesResponse;
  timeZone?: string;
}

export default function PhaseTimeline({ data, timeZone }: Props) {
  const markers = buildTimeline(data);

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-7">
      {markers.map((marker) => (
        <div
          key={marker.label}
          className="rounded-xl bg-white/10 p-3 text-center ring-1 ring-white/15 backdrop-blur"
        >
          <div className="text-[11px] uppercase tracking-wide text-white/60">
            {marker.label}
          </div>
          <div className="mt-1 font-mono text-sm text-white">
            {formatClock(marker.at, timeZone)}
          </div>
        </div>
      ))}
    </div>
  );
}
