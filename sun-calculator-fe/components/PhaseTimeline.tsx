import { buildTimeline, formatClock, type SunTimesResponse } from "@/lib/solar";

interface Props {
  data: SunTimesResponse;
  timeZone?: string;
}

const phaseStyles = {
  blue: {
    card: "bg-blue-950/60 ring-blue-400/30",
    dot: "bg-blue-400",
  },
  golden: {
    card: "bg-amber-950/60 ring-amber-400/30",
    dot: "bg-amber-400",
  },
  day: {
    card: "bg-sky-950/50 ring-sky-400/20",
    dot: "bg-sky-300",
  },
};

export default function PhaseTimeline({ data, timeZone }: Props) {
  const markers = buildTimeline(data);

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-7">
      {markers.map((marker) => {
        const styles = phaseStyles[marker.phase];
        return (
          <div
            key={marker.label}
            className={`rounded-xl p-3 text-center ring-1 backdrop-blur ${styles.card}`}
          >
            <div className="flex items-center justify-center gap-1 mb-1">
              <span className={`inline-block h-1.5 w-1.5 rounded-full ${styles.dot}`} />
              <div className="text-[11px] uppercase tracking-wide text-white/80">
                {marker.label}
              </div>
            </div>
            <div className="font-mono text-sm text-white">
              {formatClock(marker.at, timeZone)}
            </div>
          </div>
        );
      })}
    </div>
  );
}
