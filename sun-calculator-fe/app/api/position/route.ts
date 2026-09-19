import { NextRequest, NextResponse } from "next/server";

const BACKEND_URL = process.env.BACKEND_URL ?? "http://localhost:8080";

// Proxies the backend's /api/position (sun altitude + azimuth at an instant).
export async function GET(request: NextRequest) {
  const lat = request.nextUrl.searchParams.get("lat");
  const lng = request.nextUrl.searchParams.get("lng");
  if (!lat || !lng) {
    return NextResponse.json(
      { error: "Missing 'lat' or 'lng' parameters" },
      { status: 400 },
    );
  }

  const url = new URL(`${BACKEND_URL}/api/position`);
  url.searchParams.set("lat", lat);
  url.searchParams.set("lng", lng);
  const time = request.nextUrl.searchParams.get("time");
  if (time) url.searchParams.set("time", time);

  try {
    const response = await fetch(url, { cache: "no-store" });
    const body = await response.text();
    return new NextResponse(body, {
      status: response.status,
      headers: { "content-type": "application/json" },
    });
  } catch {
    return NextResponse.json(
      { error: "Failed to reach the solar engine" },
      { status: 502 },
    );
  }
}
