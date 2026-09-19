import { backendFetch } from "@/lib/backend";

export async function GET() {
  try {
    const response = await backendFetch("/health", 5_000);
    return new Response(await response.text(), {
      status: response.status,
      headers: {
        "content-type":
          response.headers.get("content-type") ?? "text/plain; charset=utf-8",
      },
    });
  } catch {
    return new Response("solar engine unreachable", { status: 502 });
  }
}
