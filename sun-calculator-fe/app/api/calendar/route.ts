import { NextRequest, NextResponse } from "next/server";

const BACKEND_URL = process.env.SUNCALC_BACKEND_URL ?? "http://localhost:8080";

export async function GET(request: NextRequest) {
  const upstream = `${BACKEND_URL}/api/calendar${request.nextUrl.search}`;

  let response: Response;
  try {
    response = await fetch(upstream, { cache: "no-store" });
  } catch {
    return NextResponse.json(
      { error: "Could not reach the calendar service." },
      { status: 502 },
    );
  }

  const headers = new Headers();
  const contentType = response.headers.get("content-type");
  if (contentType) headers.set("content-type", contentType);
  const disposition = response.headers.get("content-disposition");
  if (disposition) headers.set("content-disposition", disposition);

  return new Response(await response.arrayBuffer(), {
    status: response.status,
    headers,
  });
}
