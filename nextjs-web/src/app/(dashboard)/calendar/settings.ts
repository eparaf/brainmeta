"use client";

import { useCallback, useEffect, useState } from "react";

// Calendar customization — owned by the calendar module, persisted locally so each
// operator keeps their own view preferences. Light theme only (no dark mode), no emoji.

export type CalendarView = "day" | "week" | "month";
export type Density = "comfortable" | "compact";
export type AccentKey = "slate" | "blue" | "emerald" | "violet" | "rose" | "amber";

export type Accent = {
  label: string;
  dot: string; // strong color (event dot / today marker)
  soft: string; // tinted background for chips / cells
  ring: string; // border for chips / today outline
  text: string; // readable text on soft
};

export const ACCENTS: Record<AccentKey, Accent> = {
  slate: { label: "Antrasit", dot: "#475569", soft: "#f1f5f9", ring: "#cbd5e1", text: "#334155" },
  blue: { label: "Mavi", dot: "#1a73e8", soft: "#e8f0fe", ring: "#aecbfa", text: "#1967d2" },
  emerald: { label: "Yeşil", dot: "#059669", soft: "#ecfdf5", ring: "#a7f3d0", text: "#047857" },
  violet: { label: "Mor", dot: "#7c3aed", soft: "#f5f3ff", ring: "#ddd6fe", text: "#6d28d9" },
  rose: { label: "Gül", dot: "#e11d48", soft: "#fff1f2", ring: "#fecdd3", text: "#be123c" },
  amber: { label: "Kehribar", dot: "#d97706", soft: "#fffbeb", ring: "#fde68a", text: "#b45309" },
};

export type CalendarSettings = {
  defaultView: CalendarView;
  weekStart: 0 | 1; // 0 = Pazar, 1 = Pazartesi
  showWeekends: boolean;
  time24h: boolean;
  accent: AccentKey;
  density: Density;
  dayStartHour: number; // week view window
  dayEndHour: number;
  showProbability: boolean;
  showOverbook: boolean;
  showDoctor: boolean;
  showService: boolean;
};

export const DEFAULT_SETTINGS: CalendarSettings = {
  defaultView: "week",
  weekStart: 1,
  showWeekends: true,
  time24h: true,
  accent: "blue",
  density: "comfortable",
  dayStartHour: 8,
  dayEndHour: 20,
  showProbability: true,
  showOverbook: true,
  showDoctor: true,
  showService: true,
};

const STORAGE_KEY = "brainmeta.calendar.settings.v1";

export function useCalendarSettings() {
  const [settings, setSettings] = useState<CalendarSettings>(DEFAULT_SETTINGS);
  const [hydrated, setHydrated] = useState(false);

  // Load once on mount (client-only — avoids SSR hydration mismatch).
  /* eslint-disable react-hooks/set-state-in-effect -- one-shot localStorage hydration on mount */
  useEffect(() => {
    try {
      const raw = window.localStorage.getItem(STORAGE_KEY);
      if (raw) setSettings({ ...DEFAULT_SETTINGS, ...JSON.parse(raw) });
    } catch {
      /* corrupt value — fall back to defaults */
    }
    setHydrated(true);
  }, []);
  /* eslint-enable react-hooks/set-state-in-effect */

  const update = useCallback(<K extends keyof CalendarSettings>(key: K, value: CalendarSettings[K]) => {
    setSettings((prev) => {
      const next = { ...prev, [key]: value };
      try {
        window.localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
      } catch {
        /* storage unavailable — keep in-memory only */
      }
      return next;
    });
  }, []);

  const reset = useCallback(() => {
    setSettings(DEFAULT_SETTINGS);
    try {
      window.localStorage.removeItem(STORAGE_KEY);
    } catch {
      /* ignore */
    }
  }, []);

  return { settings, update, reset, hydrated };
}
