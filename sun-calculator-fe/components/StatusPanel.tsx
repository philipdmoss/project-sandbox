import { cardinal, formatClock, type Moment, type SunTimesResponse } from "@/lib/solar";

interface Props {
  moment: Moment;
  data: SunTimesResponse;
  placeName: string;
  now: Date;
  timeZone?: string;
}

export default function StatusPanel({ moment, data, placeName, now, timeZone }: Props) {
  const stats: { label: string; value: string }[] = [];
  if (data.day_length) {
    stats.push({ label: "Day length", value: data.day_length });
  }
  if (data.sunrise_azimuth !== undefined) {
    stats.push({
      label: "Sun rises",
      value: `${cardinal(data.sunrise_azimuth)} · ${Math.round(data.sunrise_azimuth)}°`,
    });
  }
  if (data.sunset_azimuth !== undefined) {
    stats.push({
      label: "Sun sets",
      value: `${cardinal(data.sunset_azimuth)} · ${Math.round(data.sunset_azimuth)}°`,
    });
  }

  return (
    <div className="rounded-2xl bg-black/50 p-6 ring-1 ring-white/15 backdrop-blur">
      <div className="flex items-center justify-between text-sm text-white/70">
        <span className="font-medium">{placeName}</span>
        <span className="font-mono">{formatClock(now, timeZone, true, true)}</span>
      </div>
      <p className="mt-4 text-xs uppercase tracking-widest text-white/50">
        Current status
      </p>
      <h2 className="mt-1 text-3xl font-semibold text-white">{moment.headline}</h2>
      <p className="mt-2 text-lg text-amber-200">{moment.detail}</p>

      {stats.length > 0 && (
        <dl className="mt-5 grid grid-cols-3 gap-3 border-t border-white/10 pt-4">
          {stats.map((stat) => (
            <div key={stat.label}>
              <dt className="text-[11px] uppercase tracking-wide text-white/50">
                {stat.label}
              </dt>
              <dd className="mt-0.5 text-sm font-medium text-white">{stat.value}</dd>
            </div>
          ))}
        </dl>
      )}
    </div>
  );
}
