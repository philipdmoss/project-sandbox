export interface TwilightWindow {
  start: string;
  end: string;
  duration: string;
}

export interface SunTimesResponse {
  status: string;
  sunrise: string;
  sunset: string;
  solar_noon: string;
  day_length: string;
  morning_blue_hour: TwilightWindow;
  morning_golden_hour: TwilightWindow;
  evening_golden_hour: TwilightWindow;
  evening_blue_hour: TwilightWindow;
}

export type PhaseKey =
  | "night"
  | "morningBlue"
  | "morningGolden"
  | "day"
  | "eveningGolden"
  | "eveningBlue";

export interface PhaseSegment {
  key: PhaseKey;
  label: string;
  start: Date;
  end: Date;
}

export interface TimelineMarker {
  label: string;
  at: Date;
}

export interface Moment {
  phaseKey: PhaseKey;
  headline: string;
  detail: string;
  countdownTarget: Date | null;
}

export const PHASE_GRADIENTS: Record<PhaseKey, string> = {
  night: "linear-gradient(180deg, #060b1e 0%, #0f1630 55%, #1a2140 100%)",
  morningBlue: "linear-gradient(180deg, #12183a 0%, #3b2f6b 55%, #7d5b8c 100%)",
  morningGolden: "linear-gradient(180deg, #6a4a86 0%, #e2a036 58%, #e25822 100%)",
  day: "linear-gradient(180deg, #3f83c9 0%, #9cc4ec 55%, #f6ecd2 100%)",
  eveningGolden: "linear-gradient(180deg, #4a6aa5 0%, #e2a036 48%, #e25822 100%)",
  eveningBlue: "linear-gradient(180deg, #10163a 0%, #2a2b5e 52%, #7d5b8c 100%)",
};

export const PHASE_LABELS: Record<PhaseKey, string> = {
  night: "Night",
  morningBlue: "Morning Blue Hour",
  morningGolden: "Morning Golden Hour",
  day: "Daytime",
  eveningGolden: "Evening Golden Hour",
  eveningBlue: "Evening Blue Hour",
};

export function buildSegments(data: SunTimesResponse): PhaseSegment[] {
  const at = (value: string) => new Date(value);
  return [
    {
      key: "morningBlue",
      label: PHASE_LABELS.morningBlue,
      start: at(data.morning_blue_hour.start),
      end: at(data.morning_blue_hour.end),
    },
    {
      key: "morningGolden",
      label: PHASE_LABELS.morningGolden,
      start: at(data.morning_golden_hour.start),
      end: at(data.morning_golden_hour.end),
    },
    {
      key: "day",
      label: PHASE_LABELS.day,
      start: at(data.morning_golden_hour.end),
      end: at(data.evening_golden_hour.start),
    },
    {
      key: "eveningGolden",
      label: PHASE_LABELS.eveningGolden,
      start: at(data.evening_golden_hour.start),
      end: at(data.evening_golden_hour.end),
    },
    {
      key: "eveningBlue",
      label: PHASE_LABELS.eveningBlue,
      start: at(data.evening_blue_hour.start),
      end: at(data.evening_blue_hour.end),
    },
  ];
}

export function buildTimeline(data: SunTimesResponse): TimelineMarker[] {
  return [
    { label: "Dawn", at: new Date(data.morning_blue_hour.start) },
    { label: "Sunrise", at: new Date(data.sunrise) },
    { label: "Golden ends", at: new Date(data.morning_golden_hour.end) },
    { label: "Solar noon", at: new Date(data.solar_noon) },
    { label: "Golden starts", at: new Date(data.evening_golden_hour.start) },
    { label: "Sunset", at: new Date(data.sunset) },
    { label: "Dusk", at: new Date(data.evening_blue_hour.end) },
  ];
}

export function describeMoment(data: SunTimesResponse, now: Date): Moment {
  const segments = buildSegments(data);
  const dawn = segments[0].start;
  const dusk = segments[segments.length - 1].end;
  const sunrise = new Date(data.sunrise);
  const sunset = new Date(data.sunset);
  const eveningGoldenStart = new Date(data.evening_golden_hour.start);

  const current = segments.find((s) => now >= s.start && now < s.end);

  if (!current) {
    if (now < dawn) {
      return {
        phaseKey: "night",
        headline: PHASE_LABELS.night,
        detail: `Dawn in ${formatCountdown(dawn.getTime() - now.getTime())}`,
        countdownTarget: dawn,
      };
    }
    return {
      phaseKey: "night",
      headline: PHASE_LABELS.night,
      detail: "Next dawn tomorrow",
      countdownTarget: null,
    };
  }

  if (current.key === "day") {
    return {
      phaseKey: "day",
      headline: PHASE_LABELS.day,
      detail: `Evening golden hour in ${formatCountdown(eveningGoldenStart.getTime() - now.getTime())}`,
      countdownTarget: eveningGoldenStart,
    };
  }

  const remaining = formatCountdown(current.end.getTime() - now.getTime());
  return {
    phaseKey: current.key,
    headline: current.label,
    detail: `Ends in ${remaining}`,
    countdownTarget: current.end,
  };
}

export function formatCountdown(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const seconds = total % 60;
  if (hours > 0) return `${hours}h ${minutes}m`;
  if (minutes > 0) return `${minutes}m ${seconds}s`;
  return `${seconds}s`;
}

export function formatClock(date: Date, timeZone?: string): string {
  return new Intl.DateTimeFormat("en-US", {
    hour: "numeric",
    minute: "2-digit",
    timeZone,
  }).format(date);
}
