import { NextRequest, NextResponse } from "next/server";

const BACKEND_URL = process.env.BACKEND_URL ?? "http://localhost:8080";

// Proxies the backend's subscribable calendar feed (rolling window, no download
// disposition) so a calendar app can subscribe to this URL.
export async function GET(request: NextRequest) {
  const url = new URL(`${BACKEND_URL}/api/calendar/feed`);
  request.nextUrl.searchParams.forEach((value, key) => url.searchParams.set(key, value));

  try {
    const response = await fetch(url, { cache: "no-store" });
    const body = await response.text();
    return new NextResponse(body, {
      status: response.status,
      headers: {
        "content-type": response.headers.get("content-type") ?? "text/calendar; charset=utf-8",
      },
    });
  } catch {
    return NextResponse.json(
      { error: "Failed to reach the solar engine" },
      { status: 502 },
    );
  }
}
