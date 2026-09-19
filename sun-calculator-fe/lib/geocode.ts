export interface GeocodeResult {
  name: string;
  latitude: number;
  longitude: number;
  timezone: string;
}

const GEOCODE_ENDPOINT = "https://geocoding-api.open-meteo.com/v1/search";

export async function geocodePlace(query: string): Promise<GeocodeResult> {
  const url = new URL(GEOCODE_ENDPOINT);
  url.searchParams.set("name", query);
  url.searchParams.set("count", "1");
  url.searchParams.set("language", "en");
  url.searchParams.set("format", "json");

  const response = await fetch(url);
  if (!response.ok) {
    throw new Error("Could not reach the location search service.");
  }

  const body = await response.json();
  const match = body.results?.[0];
  if (!match) {
    throw new Error(`No place found for "${query}".`);
  }

  const parts = [match.name, match.admin1, match.country].filter(Boolean);
  return {
    name: parts.join(", "),
    latitude: match.latitude,
    longitude: match.longitude,
    timezone: match.timezone,
  };
}

const REVERSE_ENDPOINT = "https://api.bigdatacloud.net/data/reverse-geocode-client";

// reverseGeocode turns coordinates into a human place name (city, state,
// country). Uses BigDataCloud's keyless, CORS-enabled client endpoint. The
// timezone comes from the browser (the viewer is at this location).
export async function reverseGeocode(
  latitude: number,
  longitude: number,
): Promise<string> {
  const url = new URL(REVERSE_ENDPOINT);
  url.searchParams.set("latitude", String(latitude));
  url.searchParams.set("longitude", String(longitude));
  url.searchParams.set("localityLanguage", "en");

  const response = await fetch(url);
  if (!response.ok) {
    throw new Error("reverse geocode failed");
  }
  const body = await response.json();
  const parts = [
    body.city || body.locality,
    body.principalSubdivision,
    body.countryName,
  ].filter(Boolean);
  return parts.join(", ");
}
