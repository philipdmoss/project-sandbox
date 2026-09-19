import { NextRequest, NextResponse } from "next/server";
import { backendFetch } from "@/lib/backend";

export async function GET(request: NextRequest) {
  let response: Response;
  try {
    response = await backendFetch(`/api/calendar${request.nextUrl.search}`);
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

  return new Response(response.body, {
    status: response.status,
    headers,
  });
}
