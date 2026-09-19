import { NextRequest, NextResponse } from "next/server";

const BACKEND_URL = process.env.BACKEND_URL ?? "http://localhost:8080";

// Proxies the backend's downloadable .ics calendar, forwarding all query params
// (lat, lng, phases, year, tz) and preserving its content headers.
export async function GET(request: NextRequest) {
  const url = new URL(`${BACKEND_URL}/api/calendar`);
  request.nextUrl.searchParams.forEach((value, key) => url.searchParams.set(key, value));

  try {
    const response = await fetch(url, { cache: "no-store" });
    const body = await response.text();
    const headers = new Headers({
      "content-type": response.headers.get("content-type") ?? "text/calendar; charset=utf-8",
    });
    const disposition = response.headers.get("content-disposition");
    if (disposition) headers.set("content-disposition", disposition);
    return new NextResponse(body, { status: response.status, headers });
  } catch {
    return NextResponse.json(
      { error: "Failed to reach the solar engine" },
      { status: 502 },
    );
  }
}
