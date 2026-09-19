export const BACKEND_URL =
  process.env.SUNCALC_BACKEND_URL ?? "http://localhost:8080";

export function backendFetch(path: string, timeoutMs = 10_000): Promise<Response> {
  return fetch(`${BACKEND_URL}${path}`, {
    cache: "no-store",
    signal: AbortSignal.timeout(timeoutMs),
  });
}
