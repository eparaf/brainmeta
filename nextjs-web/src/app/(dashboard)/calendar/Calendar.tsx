"use client";

import { useEffect, useMemo, useState } from "react";
import { Roboto } from "next/font/google";
import { CalendarDays, CalendarPlus, ChevronLeft, ChevronRight, Settings2, Trash2, X } from "lucide-react";
import type { Appointment } from "@/lib/brain/schemas";
import { ACCENTS, type Accent, type CalendarSettings, type CalendarView } from "./settings";
import { type Compose, DEFAULT_DURATION_MIN, DetailPanel, type Draft, type NewEvent, type Selection } from "./panels";
import { TimeGrid, type SlotHit } from "./TimeGrid";
import { MonthView } from "./MonthView";
import { MiniMonth } from "./MiniMonth";
import {
  addDays,
  addMonths,
  dayKey,
  formatTime,
  isWeekend,
  startOfDay,
  TR_MONTHS,
  TR_WEEKDAYS_LONG,
  weekGrid,
} from "./dates";

// Roboto = Google Calendar's UI font; scoped to the calendar subtree only.
const roboto = Roboto({ subsets: ["latin"], weight: ["400", "500", "700"], display: "swap" });

const VIEWS: { value: CalendarView; label: string }[] = [
  { value: "day", label: "Gün" },
  { value: "week", label: "Hafta" },
  { value: "month", label: "Ay" },
];

export function Calendar({
  appointments,
  settings,
  onCustomize,
  clinicName,
}: {
  appointments: Appointment[];
  settings: CalendarSettings;
  onCustomize: () => void;
  clinicName?: string;
}) {
  const accent = ACCENTS[settings.accent];
  const today = useMemo(() => startOfDay(new Date()), []);
  const [anchor, setAnchor] = useState<Date>(today);
  const [view, setView] = useState<CalendarView>(settings.defaultView);
  const [drafts, setDrafts] = useState<Draft[]>([]);
  const [selected, setSelected] = useState<Selection>(null);
  const [composing, setComposing] = useState<Compose>(null);
  const [slot, setSlot] = useState<SlotHit | null>(null);
  const [pickerOpen, setPickerOpen] = useState(false);

  const weekDays = useMemo(
    () => weekGrid(anchor, settings.weekStart).filter((d) => settings.showWeekends || !isWeekend(d)),
    [anchor, settings.weekStart, settings.showWeekends],
  );
  const days = view === "day" ? [startOfDay(anchor)] : weekDays;

  const step = (dir: -1 | 1) => {
    if (view === "day") setAnchor((a) => addDays(a, dir));
    else if (view === "week") setAnchor((a) => addDays(a, dir * 7));
    else setAnchor((a) => addMonths(a, dir));
  };

  const title = useMemo(() => {
    if (view === "day") {
      return `${anchor.getDate()} ${TR_MONTHS[anchor.getMonth()]} ${anchor.getFullYear()} · ${
        TR_WEEKDAYS_LONG[anchor.getDay()]
      }`;
    }
    if (view === "month") return `${TR_MONTHS[anchor.getMonth()]} ${anchor.getFullYear()}`;
    const a = weekDays[0];
    const b = weekDays[weekDays.length - 1];
    if (a.getMonth() === b.getMonth())
      return `${a.getDate()}–${b.getDate()} ${TR_MONTHS[a.getMonth()]} ${a.getFullYear()}`;
    return `${a.getDate()} ${TR_MONTHS[a.getMonth()]} – ${b.getDate()} ${TR_MONTHS[b.getMonth()]}`;
  }, [view, anchor, weekDays]);

  // Click/right-click an empty slot → show a small popover (no instant add).
  const onSlot = (hit: SlotHit) => {
    setSelected(null);
    setComposing(null);
    setSlot(hit);
  };
  // From the popover (click) or a drag-create, open the form with a duration.
  const startCompose = (day: Date, minutes: number, durationMin = DEFAULT_DURATION_MIN) => {
    setSlot(null);
    setSelected(null);
    setComposing({ day, minutes, durationMin });
  };
  const commitNew = (data: NewEvent) => {
    const id = `draft-${data.when}-${drafts.length}`;
    setDrafts((d) => [...d, { id, ...data }]);
    setComposing(null);
    setSelected({ kind: "draft", id });
  };
  const updateDraft = (id: string, patch: Partial<Draft>) =>
    setDrafts((d) => d.map((x) => (x.id === id ? { ...x, ...patch } : x)));
  // Move (drag body) — supports a different day (week view) + start minute.
  const moveDraft = (id: string, day: Date, minutes: number) =>
    setDrafts((d) =>
      d.map((x) => {
        if (x.id !== id) return x;
        const n = new Date(day);
        n.setHours(Math.floor(minutes / 60), minutes % 60, 0, 0);
        return { ...x, when: n.toISOString() };
      }),
    );
  // Resize (drag edge) — change duration, and optionally the start minute (top edge).
  const resizeDraft = (id: string, durationMin: number, startMinutes?: number) =>
    setDrafts((d) =>
      d.map((x) => {
        if (x.id !== id) return x;
        if (startMinutes === undefined) return { ...x, durationMin };
        const n = new Date(x.when);
        n.setHours(Math.floor(startMinutes / 60), startMinutes % 60, 0, 0);
        return { ...x, when: n.toISOString(), durationMin };
      }),
    );
  const removeDraft = (id: string) => {
    setDrafts((d) => d.filter((x) => x.id !== id));
    setSelected((s) => (s?.kind === "draft" && s.id === id ? null : s));
  };
  const closePanel = () => {
    setSelected(null);
    setComposing(null);
    setSlot(null);
  };

  // Keyboard: Delete/Backspace removes the selected draft (not while typing in a field).
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement)?.tagName;
      if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return;
      if ((e.key === "Delete" || e.key === "Backspace") && selected?.kind === "draft") {
        e.preventDefault();
        removeDraft(selected.id);
      }
      if (e.key === "Escape") closePanel();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [selected]);

  const selectedDraft =
    selected?.kind === "draft" ? drafts.find((d) => d.id === selected.id) ?? null : null;
  const panelSelection =
    selected?.kind === "appt"
      ? selected
      : selectedDraft
        ? ({ kind: "draft", draft: selectedDraft } as const)
        : null;

  return (
    <div className={`${roboto.className} flex h-full flex-col bg-white`}>
      {/* Toolbar */}
      <header className="flex shrink-0 flex-wrap items-center justify-between gap-3 border-b border-zinc-200 px-4 py-3">
        <div className="flex items-center gap-3">
          <button
            onClick={() => setAnchor(today)}
            className="rounded-lg border border-zinc-200 bg-white px-3 py-1.5 text-xs font-semibold text-zinc-700 transition-colors hover:bg-zinc-50"
          >
            Bugün
          </button>
          <div className="flex items-center">
            <button
              onClick={() => step(-1)}
              className="grid h-8 w-8 place-items-center rounded-lg text-zinc-500 transition-colors hover:bg-zinc-100"
              aria-label="Önceki"
            >
              <ChevronLeft className="h-4.5 w-4.5" />
            </button>
            <button
              onClick={() => step(1)}
              className="grid h-8 w-8 place-items-center rounded-lg text-zinc-500 transition-colors hover:bg-zinc-100"
              aria-label="Sonraki"
            >
              <ChevronRight className="h-4.5 w-4.5" />
            </button>
          </div>

          <div className="relative">
            <button
              onClick={() => setPickerOpen((v) => !v)}
              className="flex items-center gap-2 rounded-lg px-2 py-1 text-left transition-colors hover:bg-zinc-50"
            >
              <CalendarDays className="h-4 w-4 text-zinc-400" />
              <span className="text-xl font-normal tracking-tight text-zinc-800">{title}</span>
            </button>
            {pickerOpen && (
              <>
                <div className="fixed inset-0 z-30" onClick={() => setPickerOpen(false)} />
                <div className="absolute left-0 top-[calc(100%+6px)] z-40">
                  <MiniMonth
                    value={anchor}
                    settings={settings}
                    accent={accent}
                    onPick={(d) => {
                      setAnchor(startOfDay(d));
                      setPickerOpen(false);
                    }}
                  />
                </div>
              </>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2">
          {drafts.length > 0 && (
            <button
              onClick={() => {
                setDrafts([]);
                closePanel();
              }}
              className="flex items-center gap-1.5 rounded-lg border border-zinc-200 bg-white px-2.5 py-1.5 text-xs font-medium text-zinc-600 transition-colors hover:bg-zinc-50"
              title="Tüm taslakları sil"
            >
              <Trash2 className="h-3.5 w-3.5" /> {drafts.length} taslak
            </button>
          )}
          {clinicName && (
            <span className="hidden items-center gap-1.5 rounded-full border border-zinc-200 bg-white px-2.5 py-1 text-[11px] font-semibold text-zinc-600 lg:inline-flex">
              <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: accent.dot }} />
              {clinicName}
            </span>
          )}
          <div className="inline-flex rounded-lg bg-zinc-100 p-1">
            {VIEWS.map((v) => (
              <button
                key={v.value}
                onClick={() => setView(v.value)}
                className={`rounded-md px-3 py-1 text-xs font-semibold transition-all ${
                  view === v.value
                    ? "bg-white text-zinc-900 shadow-sm ring-1 ring-zinc-900/5"
                    : "text-zinc-500 hover:text-zinc-800"
                }`}
              >
                {v.label}
              </button>
            ))}
          </div>
          <button
            onClick={onCustomize}
            className="grid h-9 w-9 place-items-center rounded-lg border border-zinc-200 bg-white text-zinc-500 transition-colors hover:bg-zinc-50 hover:text-zinc-900"
            aria-label="Özelleştir"
          >
            <Settings2 className="h-4 w-4" />
          </button>
        </div>
      </header>

      {/* Body */}
      <div className="min-h-0 flex-1">
        {view === "month" ? (
          <MonthView
            anchor={anchor}
            appointments={appointments}
            drafts={drafts}
            settings={settings}
            accent={accent}
            onSelect={(s) => {
              setComposing(null);
              setSelected(s);
            }}
            onPickDay={(d) => {
              setAnchor(startOfDay(d));
              setView("day");
            }}
          />
        ) : (
          <TimeGrid
            days={days}
            appointments={appointments}
            drafts={drafts}
            settings={settings}
            accent={accent}
            selection={selected}
            activeSlot={slot ? { dayKey: dayKey(slot.day), minutes: slot.minutes } : null}
            onSelect={(s) => {
              setComposing(null);
              setSlot(null);
              setSelected(s);
            }}
            onSlot={onSlot}
            onMoveDraft={moveDraft}
            onResizeDraft={resizeDraft}
            onCreateRange={(day, minutes, durationMin) => startCompose(day, minutes, durationMin)}
          />
        )}
      </div>

      {slot && (
        <SlotMenu
          slot={slot}
          settings={settings}
          accent={accent}
          onAdd={() => startCompose(slot.day, slot.minutes)}
          onClose={() => setSlot(null)}
        />
      )}

      <DetailPanel
        selection={panelSelection}
        composing={composing}
        settings={settings}
        accent={accent}
        onClose={closePanel}
        onUpdateDraft={updateDraft}
        onRemoveDraft={removeDraft}
        onCommitNew={commitNew}
      />
    </div>
  );
}

function SlotMenu({
  slot,
  settings,
  accent,
  onAdd,
  onClose,
}: {
  slot: SlotHit;
  settings: CalendarSettings;
  accent: Accent;
  onAdd: () => void;
  onClose: () => void;
}) {
  const when = new Date(slot.day);
  when.setHours(Math.floor(slot.minutes / 60), slot.minutes % 60, 0, 0);
  const heading = `${slot.day.getDate()} ${TR_MONTHS[slot.day.getMonth()]} ${
    TR_WEEKDAYS_LONG[slot.day.getDay()]
  }`;
  return (
    <>
      <div className="fixed inset-0 z-40" onClick={onClose} onContextMenu={(e) => { e.preventDefault(); onClose(); }} />
      <div
        className="fixed z-50 w-60 overflow-hidden rounded-2xl border border-zinc-200 bg-white shadow-xl ring-1 ring-zinc-900/5"
        style={{
          left: Math.min(slot.x, window.innerWidth - 260),
          top: Math.min(slot.y, window.innerHeight - 160),
        }}
      >
        <div className="flex items-start justify-between gap-2 border-b border-zinc-100 px-4 py-3">
          <div>
            <div className="text-sm font-semibold text-zinc-900">{heading}</div>
            <div className="font-mono text-xs tabular-nums text-zinc-400">
              {formatTime(when.toISOString(), settings.time24h)}
            </div>
          </div>
          <button
            onClick={onClose}
            className="grid h-7 w-7 shrink-0 place-items-center rounded-full text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700"
            aria-label="Kapat"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
        <div className="p-2">
          <button
            onClick={onAdd}
            className="flex w-full items-center gap-2.5 rounded-xl px-3 py-2.5 text-left text-sm font-semibold text-white transition-opacity hover:opacity-90"
            style={{ backgroundColor: accent.dot }}
          >
            <CalendarPlus className="h-4 w-4" /> Randevu ekle
          </button>
        </div>
      </div>
    </>
  );
}
