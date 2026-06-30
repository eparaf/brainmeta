"use client";

import { useState } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import type { Accent, CalendarSettings } from "./settings";
import {
  addMonths,
  isSameDay,
  monthGrid,
  startOfDay,
  TR_MONTHS,
  TR_WEEKDAYS_SHORT,
  weekdayOrder,
} from "./dates";

// Compact month picker shown in a popover under the toolbar title.
export function MiniMonth({
  value,
  settings,
  accent,
  onPick,
}: {
  value: Date;
  settings: CalendarSettings;
  accent: Accent;
  onPick: (d: Date) => void;
}) {
  const [view, setView] = useState(() => startOfDay(value));
  const today = new Date();
  const order = weekdayOrder(settings.weekStart);
  const cells = monthGrid(view, settings.weekStart);

  return (
    <div className="w-64 rounded-2xl border border-zinc-200 bg-white p-3 shadow-xl ring-1 ring-zinc-900/5">
      <div className="mb-2 flex items-center justify-between px-1">
        <span className="text-sm font-semibold text-zinc-900">
          {TR_MONTHS[view.getMonth()]} {view.getFullYear()}
        </span>
        <div className="flex items-center gap-1">
          <button
            onClick={() => setView((v) => addMonths(v, -1))}
            className="grid h-7 w-7 place-items-center rounded-lg text-zinc-500 hover:bg-zinc-100"
            aria-label="Önceki ay"
          >
            <ChevronLeft className="h-4 w-4" />
          </button>
          <button
            onClick={() => setView((v) => addMonths(v, 1))}
            className="grid h-7 w-7 place-items-center rounded-lg text-zinc-500 hover:bg-zinc-100"
            aria-label="Sonraki ay"
          >
            <ChevronRight className="h-4 w-4" />
          </button>
        </div>
      </div>

      <div className="grid grid-cols-7 gap-0.5">
        {order.map((g) => (
          <div key={g} className="py-1 text-center text-[10px] font-semibold uppercase text-zinc-400">
            {TR_WEEKDAYS_SHORT[g][0]}
          </div>
        ))}
        {cells.map((d, i) => {
          const inMonth = d.getMonth() === view.getMonth();
          const isToday = isSameDay(d, today);
          const isSel = isSameDay(d, value);
          return (
            <button
              key={i}
              onClick={() => onPick(d)}
              className={`grid h-8 place-items-center rounded-lg text-xs font-medium tabular-nums transition-colors ${
                inMonth ? "text-zinc-700 hover:bg-zinc-100" : "text-zinc-300 hover:bg-zinc-50"
              }`}
              style={
                isSel
                  ? { backgroundColor: accent.dot, color: "#fff" }
                  : isToday
                    ? { color: accent.text, backgroundColor: accent.soft }
                    : undefined
              }
            >
              {d.getDate()}
            </button>
          );
        })}
      </div>
    </div>
  );
}
