"use client";

import { useState } from "react";

interface Props {
  onSearch: (query: string) => void;
  onUseLocation: () => void;
  onWarm: () => void;
  busy: boolean;
}

export default function SearchBar({ onSearch, onUseLocation, onWarm, busy }: Props) {
  const [value, setValue] = useState("");

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        const trimmed = value.trim();
        if (trimmed) onSearch(trimmed);
      }}
      className="flex flex-col gap-3 sm:flex-row"
    >
      <input
        value={value}
        onChange={(event) => setValue(event.target.value)}
        onFocus={onWarm}
        placeholder="Search a city…"
        aria-label="Search a city"
        className="flex-1 rounded-xl bg-white/10 px-4 py-3 text-white placeholder-white/50 outline-none ring-1 ring-white/20 backdrop-blur transition focus:ring-white/60"
      />
      <div className="flex gap-2">
        <button
          type="submit"
          onMouseEnter={onWarm}
          disabled={busy}
          className="rounded-xl bg-white/90 px-6 py-3 font-medium text-slate-900 transition hover:bg-white disabled:opacity-60"
        >
          {busy ? "Loading…" : "Search"}
        </button>
        <button
          type="button"
          onClick={onUseLocation}
          disabled={busy}
          title="Use my location"
          aria-label="Use my location"
          className="rounded-xl bg-white/10 px-4 py-3 text-white ring-1 ring-white/20 backdrop-blur transition hover:bg-white/20 disabled:opacity-60"
        >
          📍
        </button>
      </div>
    </form>
  );
}
