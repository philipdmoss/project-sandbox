const BACKEND_URL = process.env.BACKEND_URL ?? "http://localhost:8080";

// Proxies the backend's /health so the main page's on-load ping actually wakes
// and checks the solar engine.
export async function GET() {
  try {
    const response = await fetch(`${BACKEND_URL}/health`, { cache: "no-store" });
    const body = await response.text();
    return new Response(body, {
      status: response.status,
      headers: { "content-type": "text/plain; charset=utf-8" },
    });
  } catch {
    return new Response("The solar engine is unreachable.\n", {
      status: 502,
      headers: { "content-type": "text/plain; charset=utf-8" },
    });
  }
}
