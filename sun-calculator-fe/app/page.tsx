"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { describeMoment, type SunTimesResponse } from "@/lib/solar";
import { geocodePlace } from "@/lib/geocode";
import PhaseBackground from "@/components/PhaseBackground";
import SearchBar from "@/components/SearchBar";
import StatusPanel from "@/components/StatusPanel";
import SolarArc from "@/components/SolarArc";
import PhaseTimeline from "@/components/PhaseTimeline";
import CalendarExport from "@/components/CalendarExport";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE ?? "";

export default function Home() {
  const [data, setData] = useState<SunTimesResponse | null>(null);
  const [placeName, setPlaceName] = useState("");
  const [timeZone, setTimeZone] = useState<string | undefined>(undefined);
  const [coords, setCoords] = useState<{ lat: number; lng: number } | null>(
    null,
  );
  const [now, setNow] = useState<Date>(() => new Date());
  const [loading, setLoading] = useState(false);
  const [waking, setWaking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const warmed = useRef(false);

  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(id);
  }, []);

  const warm = useCallback(() => {
    if (warmed.current) return;
    warmed.current = true;
    fetch(`${API_BASE}/api/health`).catch(() => {});
  }, []);

  useEffect(() => {
    warm();
  }, [warm]);

  const loadSunTimes = useCallback(
    async (lat: number, lng: number, name: string, tz?: string) => {
      setLoading(true);
      setError(null);
      const wakingTimer = setTimeout(() => setWaking(true), 1500);
      try {
        const response = await fetch(
          `${API_BASE}/api/suntimes?lat=${lat}&lng=${lng}`,
        );
        if (!response.ok) {
          throw new Error("The solar engine could not compute this location.");
        }
        const payload: SunTimesResponse = await response.json();
        setData(payload);
        setPlaceName(name);
        setTimeZone(tz);
        setCoords({ lat, lng });
      } catch (err) {
        setError(err instanceof Error ? err.message : "Something went wrong.");
      } finally {
        clearTimeout(wakingTimer);
        setWaking(false);
        setLoading(false);
      }
    },
    [],
  );

  const handleSearch = useCallback(
    async (query: string) => {
      setLoading(true);
      setError(null);
      try {
        const place = await geocodePlace(query);
        await loadSunTimes(
          place.latitude,
          place.longitude,
          place.name,
          place.timezone,
        );
      } catch (err) {
        setError(err instanceof Error ? err.message : "Location search failed.");
        setLoading(false);
      }
    },
    [loadSunTimes],
  );

  const handleUseLocation = useCallback(() => {
    if (!navigator.geolocation) {
      setError("Geolocation isn't available in this browser.");
      return;
    }
    setLoading(true);
    setError(null);
    navigator.geolocation.getCurrentPosition(
      (position) => {
        const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
        loadSunTimes(
          position.coords.latitude,
          position.coords.longitude,
          "Your location",
          tz,
        );
      },
      () => {
        setError("Couldn't get your location.");
        setLoading(false);
      },
    );
  }, [loadSunTimes]);

  const moment = useMemo(
    () => (data ? describeMoment(data, now) : null),
    [data, now],
  );
  const phase = moment?.phaseKey ?? "night";

  return (
    <main className="relative min-h-screen text-white">
      <PhaseBackground phase={phase} />
      <div className="mx-auto w-full max-w-4xl px-4 py-12 sm:py-16">
        <header className="mb-8">
          <h1 className="font-playfair text-3xl font-semibold tracking-tight sm:text-4xl">
            Sun Times
          </h1>
          <p className="mt-1 text-sm text-white/70">
            Golden hours, blue hours, and the day&apos;s light for any place.
          </p>
        </header>

        <SearchBar
          onSearch={handleSearch}
          onUseLocation={handleUseLocation}
          onWarm={warm}
          busy={loading}
        />

        {waking && (
          <p className="mt-3 animate-pulse text-sm text-amber-300">
            Waking up the solar engine… hang tight.
          </p>
        )}
        {error && <p className="mt-3 text-sm text-rose-300">{error}</p>}

        {data && moment && (
          <div className="mt-8 space-y-6">
            <StatusPanel
              moment={moment}
              placeName={placeName}
              now={now}
              timeZone={timeZone}
            />
            <div className="rounded-2xl bg-black/25 p-4 ring-1 ring-white/15 backdrop-blur">
              <SolarArc data={data} now={now} timeZone={timeZone} />
            </div>
            <PhaseTimeline data={data} timeZone={timeZone} />
            {coords && (
              <CalendarExport coords={coords} timeZone={timeZone} />
            )}
          </div>
        )}

        {!data && !loading && (
          <p className="mt-12 text-center text-white/60">
            Search a city or use your location to see today&apos;s light.
          </p>
        )}
      </div>
    </main>
  );
}
