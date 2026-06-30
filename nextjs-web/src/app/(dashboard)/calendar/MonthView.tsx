"use client";

import { useMemo } from "react";
import type { Appointment } from "@/lib/brain/schemas";
import type { Accent, CalendarSettings } from "./settings";
import type { Draft, Selection } from "./panels";
import {
  dayKey,
  formatTime,
  isSameDay,
  isWeekend,
  monthGrid,
  TR_WEEKDAYS_SHORT,
  weekdayOrder,
} from "./dates";

export function MonthView({
  anchor,
  appointments,
  drafts,
  settings,
  accent,
  onSelect,
  onPickDay,
}: {
  anchor: Date;
  appointments: Appointment[];
  drafts: Draft[];
  settings: CalendarSettings;
  accent: Accent;
  onSelect: (s: Selection) => void;
  onPickDay: (d: Date) => void;
}) {
  const today = useMemo(() => new Date(), []);

  const byDay = useMemo(() => {
    const m = new Map<string, { id: string; when: string; name: string; kind: "appt" | "draft"; appt?: Appointment }[]>();
    const push = (when: string, item: { id: string; when: string; name: string; kind: "appt" | "draft"; appt?: Appointment }) => {
      const k = dayKey(new Date(when));
      const list = m.get(k) ?? [];
      list.push(item);
      m.set(k, list);
    };
    for (const a of appointments) push(a.when, { id: a.id, when: a.when, name: a.name || "İsimsiz", kind: "appt", appt: a });
    for (const d of drafts) push(d.when, { id: d.id, when: d.when, name: d.name || "Taslak", kind: "draft" });
    for (const list of m.values()) list.sort((a, b) => a.when.localeCompare(b.when));
    return m;
  }, [appointments, drafts]);

  const order = weekdayOrder(settings.weekStart).filter(
    (g) => settings.showWeekends || (g !== 0 && g !== 6),
  );
  const cells = monthGrid(anchor, settings.weekStart).filter(
    (d) => settings.showWeekends || !isWeekend(d),
  );
  const cols = order.length;

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {/* weekday header */}
      <div
        className="grid shrink-0 border-b border-zinc-200 bg-white"
        style={{ gridTemplateColumns: `repeat(${cols}, minmax(0,1fr))` }}
      >
        {order.map((g) => (
          <div
            key={g}
            className="px-3 py-2 text-center text-[11px] font-semibold uppercase tracking-wide text-zinc-400"
          >
            {TR_WEEKDAYS_SHORT[g]}
          </div>
        ))}
      </div>

      {/* day cells */}
      <div
        className="grid min-h-0 flex-1"
        style={{
          gridTemplateColumns: `repeat(${cols}, minmax(0,1fr))`,
          gridAutoRows: "minmax(0, 1fr)",
        }}
      >
        {cells.map((d, i) => {
          const inMonth = d.getMonth() === anchor.getMonth();
          const isToday = isSameDay(d, today);
          const items = byDay.get(dayKey(d)) ?? [];
          const shown = items.slice(0, 3);
          const overflow = items.length - shown.length;
          return (
            <div
              key={i}
              onClick={() => onPickDay(d)}
              className={`flex cursor-pointer flex-col gap-1 overflow-hidden border-b border-r border-zinc-100 p-1.5 transition-colors hover:bg-zinc-50 ${
                inMonth ? "" : "bg-zinc-50/50"
              }`}
            >
              <div className="flex items-center justify-between px-0.5">
                <span
                  className={`grid h-6 min-w-6 place-items-center rounded-full px-1 text-xs font-semibold tabular-nums ${
                    inMonth ? "text-zinc-700" : "text-zinc-300"
                  }`}
                  style={isToday ? { backgroundColor: accent.dot, color: "#fff" } : undefined}
                >
                  {d.getDate()}
                </span>
              </div>
              <div className="flex flex-col gap-0.5 overflow-hidden">
                {shown.map((it) => (
                  <button
                    key={it.id}
                    onClick={(e) => {
                      e.stopPropagation();
                      onSelect(it.kind === "appt" ? { kind: "appt", data: it.appt! } : { kind: "draft", id: it.id });
                    }}
                    className="flex items-center gap-1 truncate rounded px-1.5 py-0.5 text-left text-[11px] font-medium"
                    style={
                      it.kind === "draft"
                        ? { color: "#3c4043", border: `1px dashed ${accent.dot}`, backgroundColor: "#fff" }
                        : { backgroundColor: accent.dot, color: "#fff" }
                    }
                  >
                    <span className="text-[10px] tabular-nums opacity-80">
                      {formatTime(it.when, settings.time24h)}
                    </span>
                    <span className="truncate">{it.name}</span>
                  </button>
                ))}
                {overflow > 0 && (
                  <span className="px-1 text-[10px] font-semibold text-zinc-400">+{overflow} daha</span>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
