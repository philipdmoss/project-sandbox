import {
  buildTimeline,
  buildTwilightMarkers,
  formatClock,
  formatRange,
  type SunTimesResponse,
} from "@/lib/solar";

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
  const entries = buildTimeline(data);
  const twilight = buildTwilightMarkers(data);

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-7">
        {entries.map((entry, i) => {
          const styles = phaseStyles[entry.phase];
          return (
            <div
              key={i}
              className={`flex flex-col rounded-xl p-3 text-center ring-1 backdrop-blur ${styles.card}`}
            >
              {/* Fixed-height label area so every box's time aligns horizontally. */}
              <div className="flex min-h-[2.5rem] items-center justify-center gap-1">
                <span className={`inline-block h-1.5 w-1.5 shrink-0 rounded-full ${styles.dot}`} />
                <span className="text-[11px] uppercase leading-tight tracking-wide text-white/80">
                  {entry.label}
                </span>
              </div>
              <div className="mt-auto whitespace-nowrap font-mono text-xs text-white sm:text-sm">
                {formatRange(entry, timeZone)}
              </div>
            </div>
          );
        })}
      </div>

      {twilight.length > 0 && (
        <div className="rounded-xl bg-black/25 p-3 ring-1 ring-white/10 backdrop-blur">
          <div className="mb-2 text-[11px] uppercase tracking-widest text-white/50">
            Twilight
          </div>
          <div className="flex flex-wrap gap-x-5 gap-y-1 text-xs text-white/75">
            {twilight.map((marker) => (
              <span key={marker.label} className="whitespace-nowrap">
                {marker.label}{" "}
                <span className="font-mono text-white/90">
                  {formatClock(marker.at, timeZone)}
                </span>
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
