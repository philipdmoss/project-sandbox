// Phase tokens accepted by the Go backend's /api/calendar endpoint
// (see sun-calculator-be/calendar.go parsePhases).
export const CALENDAR_PHASES = [
  { key: "sunrise", label: "Sunrise" },
  { key: "sunset", label: "Sunset" },
  { key: "golden_hour", label: "Golden hour" },
  { key: "blue_hour", label: "Blue hour" },
] as const;

export type CalendarPhaseKey = (typeof CALENDAR_PHASES)[number]["key"];
