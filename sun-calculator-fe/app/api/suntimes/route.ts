import { NextRequest, NextResponse } from "next/server";
import type { SunTimesResponse, TwilightWindow } from "@/lib/solar";

const SUN_API_ENDPOINT = "https://api.sunrise-sunset.org/json";

interface UpstreamResponse {
  status: string;
  results: {
    sunrise: string;
    sunset: string;
    solar_noon: string;
    day_length: number;
    civil_twilight_begin: string;
    civil_twilight_end: string;
  };
}

function formatDuration(totalSeconds: number): string {
  const seconds = Math.max(0, Math.round(totalSeconds));
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const remaining = seconds % 60;
  const parts: string[] = [];
  if (hours > 0) parts.push(`${hours}h`);
  if (minutes > 0) parts.push(`${minutes}m`);
  parts.push(`${remaining}s`);
  return parts.join(" ");
}

function windowBetween(start: Date, end: Date): TwilightWindow {
  const span = Math.abs(end.getTime() - start.getTime());
  return {
    start: start.toISOString(),
    end: end.toISOString(),
    duration: formatDuration(span / 1000),
  };
}

export async function GET(request: NextRequest) {
  const lat = request.nextUrl.searchParams.get("lat");
  const lng = request.nextUrl.searchParams.get("lng");
  if (!lat || !lng) {
    return NextResponse.json(
      { error: "Missing 'lat' or 'lng' parameters" },
      { status: 400 },
    );
  }

  const upstreamUrl = new URL(SUN_API_ENDPOINT);
  upstreamUrl.searchParams.set("lat", lat);
  upstreamUrl.searchParams.set("lng", lng);
  upstreamUrl.searchParams.set("formatted", "0");

  let upstream: UpstreamResponse;
  try {
    const response = await fetch(upstreamUrl, { cache: "no-store" });
    upstream = await response.json();
  } catch {
    return NextResponse.json(
      { error: "Failed to retrieve sun times" },
      { status: 502 },
    );
  }

  if (upstream.status !== "OK") {
    return NextResponse.json(
      { error: "Unexpected upstream response" },
      { status: 502 },
    );
  }

  const sunrise = new Date(upstream.results.sunrise);
  const sunset = new Date(upstream.results.sunset);
  const solarNoon = new Date(upstream.results.solar_noon);
  const civilTwilightBegin = new Date(upstream.results.civil_twilight_begin);
  const civilTwilightEnd = new Date(upstream.results.civil_twilight_end);

  const morningSpan = sunrise.getTime() - civilTwilightBegin.getTime();
  const eveningSpan = civilTwilightEnd.getTime() - sunset.getTime();

  const payload: SunTimesResponse = {
    status: "success",
    sunrise: sunrise.toISOString(),
    sunset: sunset.toISOString(),
    solar_noon: solarNoon.toISOString(),
    day_length: formatDuration(upstream.results.day_length),
    morning_blue_hour: windowBetween(civilTwilightBegin, sunrise),
    morning_golden_hour: windowBetween(
      sunrise,
      new Date(sunrise.getTime() + morningSpan),
    ),
    evening_golden_hour: windowBetween(
      new Date(sunset.getTime() - eveningSpan),
      sunset,
    ),
    evening_blue_hour: windowBetween(sunset, civilTwilightEnd),
  };

  return NextResponse.json(payload);
}
