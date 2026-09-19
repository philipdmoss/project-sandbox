import { formatClock, type Moment } from "@/lib/solar";

interface Props {
  moment: Moment;
  placeName: string;
  now: Date;
  timeZone?: string;
}

export default function StatusPanel({ moment, placeName, now, timeZone }: Props) {
  return (
    <div className="rounded-2xl bg-black/50 p-6 ring-1 ring-white/15 backdrop-blur">
      <div className="flex items-center justify-between text-sm text-white/70">
        <span className="font-medium">{placeName}</span>
        <span className="font-mono">
          {formatClock(now, { timeZone, showSeconds: true, showZone: true })}
        </span>
      </div>
      <p className="mt-4 text-xs uppercase tracking-widest text-white/50">
        Current status
      </p>
      <h2 className="mt-1 text-3xl font-semibold text-white">{moment.headline}</h2>
      <p className="mt-2 text-lg text-amber-200">{moment.detail}</p>
    </div>
  );
}
