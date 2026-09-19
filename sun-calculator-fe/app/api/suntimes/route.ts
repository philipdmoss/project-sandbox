import { NextRequest, NextResponse } from "next/server";
import { backendFetch } from "@/lib/backend";
import type { SunTimesResponse } from "@/lib/solar";

export async function GET(request: NextRequest) {
  const lat = request.nextUrl.searchParams.get("lat");
  const lng = request.nextUrl.searchParams.get("lng");
  if (!lat || !lng) {
    return NextResponse.json(
      { error: "Missing 'lat' or 'lng' parameters" },
      { status: 400 },
    );
  }

  const params = new URLSearchParams({ lat, lng, field: "all" });

  let response: Response;
  try {
    response = await backendFetch(`/api/solar?${params.toString()}`);
  } catch {
    return NextResponse.json(
      { error: "Could not reach the solar engine." },
      { status: 502 },
    );
  }

  const data = await response.json().catch(() => null);
  if (!response.ok || !data) {
    return NextResponse.json(
      { error: data?.error ?? "Failed to compute sun times." },
      { status: response.ok ? 502 : response.status },
    );
  }

  const payload: SunTimesResponse = { status: "success", ...data };
  return NextResponse.json(payload);
}
