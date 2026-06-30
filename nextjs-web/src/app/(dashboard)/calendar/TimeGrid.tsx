"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import type { Appointment } from "@/lib/brain/schemas";
import type { Accent, CalendarSettings } from "./settings";
import { DEFAULT_DURATION_MIN, type Draft, type Selection } from "./panels";
import { packDay, type Placed } from "./pack";
import { dayKey, formatTime, isSameDay, TR_WEEKDAYS_LONG, TR_WEEKDAYS_SHORT } from "./dates";

const GUTTER = 56;
const NOW_RED = "#ea4335";

export type SlotHit = { day: Date; minutes: number; x: number; y: number };
export type ActiveSlot = { dayKey: string; minutes: number } | null;
type DragMode = "move" | "resize-top" | "resize-bottom";

type Block =
  | { kind: "appt"; id: string; start: number; end: number; data: Appointment }
  | { kind: "draft"; id: string; start: number; end: number; data: Draft };

export function TimeGrid({
  days,
  appointments,
  drafts,
  settings,
  accent,
  selection,
  activeSlot,
  onSelect,
  onSlot,
  onMoveDraft,
  onResizeDraft,
  onCreateRange,
}: {
  days: Date[];
  appointments: Appointment[];
  drafts: Draft[];
  settings: CalendarSettings;
  accent: Accent;
  selection: Selection;
  activeSlot: ActiveSlot;
  onSelect: (s: Selection) => void;
  onSlot: (hit: SlotHit) => void;
  onMoveDraft: (id: string, day: Date, minutes: number) => void;
  onResizeDraft: (id: string, durationMin: number, startMinutes?: number) => void;
  onCreateRange: (day: Date, minutes: number, durationMin: number) => void;
}) {
  const today = useMemo(() => new Date(), []);
  const rowH = settings.density === "compact" ? 44 : 56;
  const startMin = settings.dayStartHour * 60;
  const endMin = settings.dayEndHour * 60;
  const hours = useMemo(
    () => Array.from({ length: settings.dayEndHour - settings.dayStartHour }, (_, i) => settings.dayStartHour + i),
    [settings.dayStartHour, settings.dayEndHour],
  );
  const pxPerMin = rowH / 60;
  const bodyH = (settings.dayEndHour - settings.dayStartHour) * rowH;
  const single = days.length === 1;

  const [nowMin, setNowMin] = useState(today.getHours() * 60 + today.getMinutes());
  useEffect(() => {
    const t = setInterval(() => {
      const n = new Date();
      setNowMin(n.getHours() * 60 + n.getMinutes());
    }, 60_000);
    return () => clearInterval(t);
  }, []);

  const scrollRef = useRef<HTMLDivElement>(null);
  const colsRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    if (nowMin >= startMin && nowMin <= endMin) {
      el.scrollTop = Math.max(0, (nowMin - startMin) * pxPerMin - el.clientHeight / 3);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- mount-only scroll
  }, []);

  const [ghost, setGhost] = useState<{ dayIndex: number; start: number; dur: number } | null>(null);

  // ---- pointer → grid coordinate helpers (shared by every drag) ----
  const snap = (m: number) => Math.round(m / 15) * 15;
  const minuteAt = (clientY: number) => {
    const r = colsRef.current!.getBoundingClientRect();
    return clamp(startMin + (clientY - r.top) / pxPerMin, startMin, endMin);
  };
  const dayIndexAt = (clientX: number) => {
    const r = colsRef.current!.getBoundingClientRect();
    return clamp(Math.floor((clientX - r.left) / (r.width / days.length)), 0, days.length - 1);
  };

  // ---- move / resize an existing draft block ----
  const beginBlockDrag = (e: React.PointerEvent, item: Block, mode: DragMode) => {
    if (e.button !== 0) return;
    e.preventDefault();
    e.stopPropagation();
    const start0 = item.start;
    const end0 = item.end;
    const dur0 = end0 - start0;
    const grab = minuteAt(e.clientY) - start0;
    let moved = false;
    const move = (ev: PointerEvent) => {
      if (Math.abs(ev.clientY - e.clientY) + Math.abs(ev.clientX - e.clientX) > 4) moved = true;
      const pm = minuteAt(ev.clientY);
      if (mode === "move") {
        const ns = clamp(snap(pm - grab), startMin, endMin - dur0);
        onMoveDraft(item.id, days[dayIndexAt(ev.clientX)], ns);
      } else if (mode === "resize-bottom") {
        const ne = clamp(snap(pm), start0 + 15, endMin);
        onResizeDraft(item.id, ne - start0);
      } else {
        const ns = clamp(snap(pm), startMin, end0 - 15);
        onResizeDraft(item.id, end0 - ns, ns);
      }
    };
    const up = () => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
      if (!moved) onSelect({ kind: "draft", id: item.id });
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
  };

  // ---- drag on empty space → create a ranged event (or a click → slot popover) ----
  const beginCreate = (e: React.PointerEvent, day: Date, dayIndex: number) => {
    if (e.button !== 0) return;
    const anchor = snap(minuteAt(e.clientY));
    let moved = false;
    const range = (ev: PointerEvent) => {
      const cur = snap(minuteAt(ev.clientY));
      const start = Math.min(anchor, cur);
      let end = Math.max(anchor, cur);
      if (end <= start) end = start + 15;
      return { start, end: Math.min(end, endMin) };
    };
    const move = (ev: PointerEvent) => {
      if (Math.abs(ev.clientY - e.clientY) > 4) moved = true;
      const { start, end } = range(ev);
      setGhost({ dayIndex, start, dur: end - start });
    };
    const up = (ev: PointerEvent) => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
      setGhost(null);
      if (moved) {
        const { start, end } = range(ev);
        onCreateRange(day, start, Math.max(15, end - start));
      } else {
        onSlot({ day, minutes: anchor, x: e.clientX, y: e.clientY });
      }
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
  };

  const colPct = 100 / days.length;

  return (
    <div ref={scrollRef} className="h-full select-none overflow-y-auto overscroll-contain bg-white">
      <div className="min-w-full">
        {/* Sticky header */}
        <div className="sticky top-0 z-50 bg-white">
          <div className="flex">
            <div className="shrink-0" style={{ width: GUTTER }} />
            {days.map((d, i) => {
              const isToday = isSameDay(d, today);
              return (
                <div key={i} className="flex flex-1 flex-col items-center gap-1 border-l border-zinc-100 pb-1.5 pt-2">
                  <span
                    className="text-[11px] font-medium uppercase tracking-wide"
                    style={{ color: isToday ? accent.dot : "#70757a" }}
                  >
                    {single ? TR_WEEKDAYS_LONG[d.getDay()] : TR_WEEKDAYS_SHORT[d.getDay()]}
                  </span>
                  <span
                    className="grid h-9 min-w-9 place-items-center rounded-full px-1 text-[22px] font-normal tabular-nums"
                    style={isToday ? { backgroundColor: accent.dot, color: "#fff" } : { color: "#3c4043" }}
                  >
                    {d.getDate()}
                  </span>
                </div>
              );
            })}
          </div>
          {/* all-day strip (GCal layout — no all-day data, kept slim) */}
          <div className="flex border-y border-zinc-100 bg-white">
            <div
              className="flex shrink-0 items-center justify-end pr-2 text-[10px] font-medium text-zinc-400"
              style={{ width: GUTTER, height: 24 }}
            >
              Tüm gün
            </div>
            {days.map((_, i) => (
              <div key={i} className="flex-1 border-l border-zinc-100" style={{ height: 24 }} />
            ))}
          </div>
        </div>

        {/* Body */}
        <div className="flex" style={{ height: bodyH }}>
          {/* Hour gutter */}
          <div className="relative shrink-0" style={{ width: GUTTER }}>
            {hours.map((h, i) => (
              <div
                key={h}
                className="absolute right-2 -translate-y-1/2 text-[10px] font-medium tabular-nums text-zinc-400"
                style={{ top: i * rowH }}
              >
                {i === 0 ? "" : `${String(h).padStart(2, "0")}:00`}
              </div>
            ))}
          </div>

          {/* Day columns */}
          <div ref={colsRef} className="relative flex flex-1">
            {days.map((d, i) => (
              <DayColumn
                key={i}
                day={d}
                index={i}
                today={today}
                rowH={rowH}
                hours={hours}
                startMin={startMin}
                endMin={endMin}
                pxPerMin={pxPerMin}
                nowMin={nowMin}
                settings={settings}
                accent={accent}
                appointments={appointments}
                drafts={drafts}
                selection={selection}
                activeSlot={activeSlot}
                onSelect={onSelect}
                onSlotContext={(e) => onSlot({ day: d, minutes: snap(minuteAt(e.clientY)), x: e.clientX, y: e.clientY })}
                onEmptyDown={(e) => beginCreate(e, d, i)}
                onBlockDrag={beginBlockDrag}
              />
            ))}

            {/* create ghost */}
            {ghost && (
              <div
                className="pointer-events-none absolute z-40 rounded-md border-2 border-dashed px-2 py-1 text-[11px] font-semibold"
                style={{
                  left: `calc(${ghost.dayIndex * colPct}% + 2px)`,
                  width: `calc(${colPct}% - 4px)`,
                  top: (ghost.start - startMin) * pxPerMin,
                  height: ghost.dur * pxPerMin,
                  borderColor: accent.dot,
                  backgroundColor: accent.soft,
                  color: accent.text,
                }}
              >
                {fmtMin(ghost.start, settings.time24h)} – {fmtMin(ghost.start + ghost.dur, settings.time24h)}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function DayColumn({
  day,
  index,
  today,
  rowH,
  hours,
  startMin,
  endMin,
  pxPerMin,
  nowMin,
  settings,
  accent,
  appointments,
  drafts,
  selection,
  activeSlot,
  onSelect,
  onSlotContext,
  onEmptyDown,
  onBlockDrag,
}: {
  day: Date;
  index: number;
  today: Date;
  rowH: number;
  hours: number[];
  startMin: number;
  endMin: number;
  pxPerMin: number;
  nowMin: number;
  settings: CalendarSettings;
  accent: Accent;
  appointments: Appointment[];
  drafts: Draft[];
  selection: Selection;
  activeSlot: ActiveSlot;
  onSelect: (s: Selection) => void;
  onSlotContext: (e: React.MouseEvent) => void;
  onEmptyDown: (e: React.PointerEvent) => void;
  onBlockDrag: (e: React.PointerEvent, item: Block, mode: DragMode) => void;
}) {
  const k = dayKey(day);
  const isToday = isSameDay(day, today);

  const blocks = useMemo<Block[]>(() => {
    const inDay = (iso: string) => dayKey(new Date(iso)) === k;
    const mins = (iso: string) => {
      const d = new Date(iso);
      return d.getHours() * 60 + d.getMinutes();
    };
    const evs: Block[] = appointments
      .filter((a) => inDay(a.when))
      .map((a) => {
        const s = mins(a.when);
        return { kind: "appt", id: a.id, start: s, end: s + DEFAULT_DURATION_MIN, data: a } as Block;
      });
    const drs: Block[] = drafts
      .filter((d) => inDay(d.when))
      .map((d) => {
        const s = mins(d.when);
        return { kind: "draft", id: d.id, start: s, end: s + d.durationMin, data: d } as Block;
      });
    return [...evs, ...drs].filter((b) => b.start >= startMin && b.start < endMin);
  }, [appointments, drafts, k, startMin, endMin]);

  const placed = useMemo(() => packDay(blocks), [blocks]);
  const showActive = activeSlot && activeSlot.dayKey === k;

  return (
    <div
      onPointerDown={onEmptyDown}
      onContextMenu={(e) => {
        e.preventDefault();
        onSlotContext(e);
      }}
      className={`relative flex-1 border-l border-zinc-100 ${isToday ? "bg-blue-50/30" : ""}`}
    >
      {hours.map((h, i) => (
        <div key={h} className="absolute inset-x-0 border-t border-zinc-100" style={{ top: i * rowH }} />
      ))}

      {showActive && (
        <div
          className="pointer-events-none absolute inset-x-1 z-[5] rounded-md border-2 border-dashed"
          style={{
            top: (activeSlot!.minutes - startMin) * pxPerMin,
            height: DEFAULT_DURATION_MIN * pxPerMin - 2,
            borderColor: accent.dot,
            backgroundColor: accent.soft,
          }}
        />
      )}

      {isToday && nowMin >= startMin && nowMin <= endMin && (
        <div className="pointer-events-none absolute inset-x-0 z-20 flex items-center" style={{ top: (nowMin - startMin) * pxPerMin }}>
          {index === 0 && <span className="-ml-1.5 h-3 w-3 rounded-full" style={{ backgroundColor: NOW_RED }} />}
          <span className="h-[2px] flex-1" style={{ backgroundColor: NOW_RED }} />
        </div>
      )}

      {placed.map((p) => (
        <BlockCard
          key={p.item.id}
          placed={p}
          startMin={startMin}
          pxPerMin={pxPerMin}
          settings={settings}
          accent={accent}
          selected={
            (selection?.kind === "appt" && p.item.kind === "appt" && selection.data.id === p.item.id) ||
            (selection?.kind === "draft" && p.item.kind === "draft" && selection.id === p.item.id)
          }
          onSelect={onSelect}
          onBlockDrag={onBlockDrag}
        />
      ))}
    </div>
  );
}

function BlockCard({
  placed,
  startMin,
  pxPerMin,
  settings,
  accent,
  selected,
  onSelect,
  onBlockDrag,
}: {
  placed: Placed<Block>;
  startMin: number;
  pxPerMin: number;
  settings: CalendarSettings;
  accent: Accent;
  selected: boolean;
  onSelect: (s: Selection) => void;
  onBlockDrag: (e: React.PointerEvent, item: Block, mode: DragMode) => void;
}) {
  const { item, col, cols } = placed;
  const top = (item.start - startMin) * pxPerMin;
  const height = Math.max((item.end - item.start) * pxPerMin - 2, 20);
  const gapPct = 100 / cols;
  const isDraft = item.kind === "draft";
  const over = !isDraft && settings.showOverbook && item.data.overbook;

  const base: React.CSSProperties = {
    top,
    height,
    left: `calc(${col * gapPct}% + 2px)`,
    width: `calc(${gapPct}% - 4px)`,
    zIndex: selected ? 30 : 10,
  };

  if (isDraft) {
    // tentative: white card with dashed accent border (distinct from confirmed)
    return (
      <div
        onPointerDown={(e) => onBlockDrag(e, item, "move")}
        className="group absolute cursor-grab touch-none overflow-hidden rounded-md border border-dashed bg-white active:cursor-grabbing"
        style={{
          ...base,
          borderColor: accent.dot,
          boxShadow: selected ? `0 1px 8px rgba(60,64,67,0.3), 0 0 0 2px ${accent.dot}` : "0 1px 2px rgba(60,64,67,0.15)",
        }}
      >
        <ResizeHandle position="top" onPointerDown={(e) => onBlockDrag(e, item, "resize-top")} />
        <div className="px-2 py-1">
          <div className="truncate text-[12px] font-semibold" style={{ color: accent.text }}>
            {item.data.name || "Taslak"}
          </div>
          {height > 30 && (
            <div className="text-[10px] tabular-nums text-zinc-500">{formatTime(item.data.when, settings.time24h)}</div>
          )}
        </div>
        <ResizeHandle position="bottom" onPointerDown={(e) => onBlockDrag(e, item, "resize-bottom")} />
      </div>
    );
  }

  // confirmed appointment: GCal-style solid fill + white text, read-only
  const fill = over ? "#e8710a" : accent.dot;
  return (
    <button
      onPointerDown={(e) => e.stopPropagation()}
      onClick={(e) => {
        e.stopPropagation();
        onSelect({ kind: "appt", data: item.data });
      }}
      className="absolute overflow-hidden rounded-md px-2 py-1 text-left"
      style={{
        ...base,
        backgroundColor: fill,
        color: "#fff",
        boxShadow: selected ? `0 1px 10px rgba(60,64,67,0.45), 0 0 0 2px #fff, 0 0 0 4px ${fill}` : "none",
      }}
    >
      <div className="truncate text-[12px] font-semibold leading-tight">{item.data.name || "İsimsiz"}</div>
      {height > 30 && (
        <div className="text-[10px] tabular-nums text-white/85">{formatTime(item.data.when, settings.time24h)}</div>
      )}
    </button>
  );
}

function ResizeHandle({ position, onPointerDown }: { position: "top" | "bottom"; onPointerDown: (e: React.PointerEvent) => void }) {
  return (
    <span
      onPointerDown={onPointerDown}
      className={`absolute inset-x-0 z-10 h-1.5 cursor-ns-resize ${position === "top" ? "top-0" : "bottom-0"}`}
    />
  );
}

function fmtMin(min: number, time24h: boolean) {
  const d = new Date();
  d.setHours(Math.floor(min / 60), min % 60, 0, 0);
  return formatTime(d.toISOString(), time24h);
}

function clamp(v: number, min: number, max: number) {
  return Math.min(max, Math.max(min, v));
}
