export interface TwilightWindow {
  start: string;
  end: string;
  duration: string;
}

export interface Twilight {
  dawn?: string;
  dusk?: string;
}

// Mirrors the backend /api/solar (field=all) payload. Every field except
// solar_noon is optional: the backend omits values that do not occur that day
// (polar day/night, or summer "white nights" with no twilight).
export interface SunTimesResponse {
  status?: string;
  sunrise?: string;
  sunset?: string;
  sunrise_azimuth?: number;
  sunset_azimuth?: number;
  solar_noon: string;
  day_length?: string;
  morning_blue_hour?: TwilightWindow;
  morning_golden_hour?: TwilightWindow;
  evening_golden_hour?: TwilightWindow;
  evening_blue_hour?: TwilightWindow;
  civil_twilight?: Twilight;
  nautical_twilight?: Twilight;
  astronomical_twilight?: Twilight;
}

export interface PositionResponse {
  time: string;
  altitude: number;
  azimuth: number;
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
  phase: "blue" | "golden" | "day";
}

export interface Moment {
  phaseKey: PhaseKey;
  headline: string;
  detail: string;
  countdownTarget: Date | null;
}

// An anchor point for the solar arc: the sun's known elevation (degrees) at a
// given instant. Drawn from the backend's twilight/sunrise times (each at a
// fixed elevation) plus the peak altitude at solar noon.
export interface ElevationAnchor {
  at: Date;
  elevation: number;
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

const at = (value: string) => new Date(value);

export function buildSegments(data: SunTimesResponse): PhaseSegment[] {
  const segments: PhaseSegment[] = [];
  if (data.morning_blue_hour) {
    segments.push({
      key: "morningBlue",
      label: PHASE_LABELS.morningBlue,
      start: at(data.morning_blue_hour.start),
      end: at(data.morning_blue_hour.end),
    });
  }
  if (data.morning_golden_hour) {
    segments.push({
      key: "morningGolden",
      label: PHASE_LABELS.morningGolden,
      start: at(data.morning_golden_hour.start),
      end: at(data.morning_golden_hour.end),
    });
  }
  if (data.morning_golden_hour && data.evening_golden_hour) {
    segments.push({
      key: "day",
      label: PHASE_LABELS.day,
      start: at(data.morning_golden_hour.end),
      end: at(data.evening_golden_hour.start),
    });
  }
  if (data.evening_golden_hour) {
    segments.push({
      key: "eveningGolden",
      label: PHASE_LABELS.eveningGolden,
      start: at(data.evening_golden_hour.start),
      end: at(data.evening_golden_hour.end),
    });
  }
  if (data.evening_blue_hour) {
    segments.push({
      key: "eveningBlue",
      label: PHASE_LABELS.eveningBlue,
      start: at(data.evening_blue_hour.start),
      end: at(data.evening_blue_hour.end),
    });
  }
  return segments;
}

export function buildTimeline(data: SunTimesResponse): TimelineMarker[] {
  const markers: TimelineMarker[] = [];
  const push = (label: string, iso: string | undefined, phase: TimelineMarker["phase"]) => {
    if (iso) markers.push({ label, at: at(iso), phase });
  };
  push("Blue hour starts", data.morning_blue_hour?.start, "blue");
  push("Sunrise", data.sunrise, "golden");
  push("Golden hour ends", data.morning_golden_hour?.end, "golden");
  push("Solar noon", data.solar_noon, "day");
  push("Golden hour starts", data.evening_golden_hour?.start, "golden");
  push("Sunset", data.sunset, "golden");
  push("Blue hour ends", data.evening_blue_hour?.end, "blue");
  return markers;
}

// Twilight boundaries (civil/nautical/astronomical dawn & dusk) for a secondary
// detail row. Only the ones the backend returned are included.
export function buildTwilightMarkers(data: SunTimesResponse): TimelineMarker[] {
  const markers: TimelineMarker[] = [];
  const push = (label: string, iso: string | undefined) => {
    if (iso) markers.push({ label, at: at(iso), phase: "blue" });
  };
  push("Astronomical dawn", data.astronomical_twilight?.dawn);
  push("Nautical dawn", data.nautical_twilight?.dawn);
  push("Civil dawn", data.civil_twilight?.dawn);
  push("Civil dusk", data.civil_twilight?.dusk);
  push("Nautical dusk", data.nautical_twilight?.dusk);
  push("Astronomical dusk", data.astronomical_twilight?.dusk);
  return markers;
}

// buildElevationAnchors returns the (time, elevation) points that define the
// real shape of the day's sun arc, sorted by time. peakAltitude is the sun's
// altitude at solar noon (from /api/position); when unknown the noon peak is
// omitted and the arc tops out at the highest available anchor.
export function buildElevationAnchors(
  data: SunTimesResponse,
  peakAltitude: number | null,
): ElevationAnchor[] {
  const anchors: ElevationAnchor[] = [];
  const add = (iso: string | undefined, elevation: number) => {
    if (iso) anchors.push({ at: at(iso), elevation });
  };
  add(data.astronomical_twilight?.dawn, -18);
  add(data.nautical_twilight?.dawn, -12);
  add(data.civil_twilight?.dawn, -6);
  add(data.morning_blue_hour?.end, -4);
  add(data.sunrise, -0.833);
  add(data.morning_golden_hour?.end, 6);
  if (peakAltitude !== null) add(data.solar_noon, peakAltitude);
  add(data.evening_golden_hour?.start, 6);
  add(data.sunset, -0.833);
  add(data.evening_blue_hour?.start, -4);
  add(data.civil_twilight?.dusk, -6);
  add(data.nautical_twilight?.dusk, -12);
  add(data.astronomical_twilight?.dusk, -18);
  anchors.sort((a, b) => a.at.getTime() - b.at.getTime());
  return anchors;
}

// elevationAt linearly interpolates the sun's elevation at an instant from the
// sorted anchor points (clamped at the ends).
export function elevationAt(anchors: ElevationAnchor[], instant: Date): number {
  if (anchors.length === 0) return 0;
  const t = instant.getTime();
  if (t <= anchors[0].at.getTime()) return anchors[0].elevation;
  const last = anchors[anchors.length - 1];
  if (t >= last.at.getTime()) return last.elevation;
  for (let i = 1; i < anchors.length; i += 1) {
    const a = anchors[i - 1];
    const b = anchors[i];
    if (t <= b.at.getTime()) {
      const span = b.at.getTime() - a.at.getTime();
      const frac = span === 0 ? 0 : (t - a.at.getTime()) / span;
      return a.elevation + frac * (b.elevation - a.elevation);
    }
  }
  return last.elevation;
}

export function describeMoment(data: SunTimesResponse, now: Date): Moment {
  const segments = buildSegments(data);
  if (segments.length === 0) {
    return {
      phaseKey: "night",
      headline: PHASE_LABELS.night,
      detail: "The sun stays below the horizon here today",
      countdownTarget: null,
    };
  }
  const dawn = segments[0].start;
  const eveningGoldenStart = data.evening_golden_hour
    ? at(data.evening_golden_hour.start)
    : null;

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

  if (current.key === "day" && eveningGoldenStart) {
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

export function formatClock(
  date: Date,
  timeZone?: string,
  showSeconds = false,
  showZone = false,
): string {
  return new Intl.DateTimeFormat("en-US", {
    hour: "numeric",
    minute: "2-digit",
    ...(showSeconds && { second: "2-digit" }),
    ...(showZone && { timeZoneName: "short" }),
    timeZone,
  }).format(date);
}

// cardinal turns a compass azimuth (degrees clockwise from north) into a
// 16-point cardinal direction like "NE" or "WSW".
export function cardinal(azimuth: number): string {
  const dirs = [
    "N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE",
    "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW",
  ];
  const index = Math.round(((azimuth % 360) + 360) % 360 / 22.5) % 16;
  return dirs[index];
}
