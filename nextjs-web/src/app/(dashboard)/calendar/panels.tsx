"use client";

import { useState } from "react";
import { CalendarPlus, Clock, Stethoscope, Trash2, X } from "lucide-react";
import type { Appointment } from "@/lib/brain/schemas";
import { maskPhone } from "@/lib/utils";
import type { Accent, CalendarSettings } from "./settings";
import { formatTime, TR_MONTHS, TR_WEEKDAYS_LONG } from "./dates";

// Shared event model + side drawer used by every calendar view. Drafts are
// browser-only sketches (the brain owns real bookings); appointments are read-only.

export const DEFAULT_DURATION_MIN = 30;
export const DURATION_OPTIONS = [15, 30, 45, 60, 90, 120];

export type Draft = {
  id: string;
  name: string;
  when: string;
  durationMin: number;
  service: string;
  note: string;
};
export type Selection =
  | { kind: "appt"; data: Appointment }
  | { kind: "draft"; id: string }
  | null;

type Resolved =
  | { kind: "appt"; data: Appointment }
  | { kind: "draft"; draft: Draft }
  | null;

export type NewEvent = { name: string; when: string; durationMin: number; service: string; note: string };
export type Compose = { day: Date; minutes: number; durationMin: number } | null;

export function DetailPanel({
  selection,
  composing,
  settings,
  accent,
  onClose,
  onUpdateDraft,
  onRemoveDraft,
  onCommitNew,
}: {
  selection: Resolved;
  composing: Compose;
  settings: CalendarSettings;
  accent: Accent;
  onClose: () => void;
  onUpdateDraft: (id: string, patch: Partial<Draft>) => void;
  onRemoveDraft: (id: string) => void;
  onCommitNew: (data: NewEvent) => void;
}) {
  const open = selection !== null || composing !== null;
  return (
    <>
      <div
        onClick={onClose}
        className={`fixed inset-0 z-40 bg-zinc-900/20 backdrop-blur-[2px] transition-opacity duration-300 ${
          open ? "opacity-100" : "pointer-events-none opacity-0"
        }`}
      />
      <aside
        className={`fixed inset-y-0 right-0 z-50 flex w-full max-w-sm flex-col bg-white shadow-2xl ring-1 ring-zinc-900/5 transition-transform duration-300 ease-out ${
          open ? "translate-x-0" : "translate-x-full"
        }`}
      >
        {composing && (
          <NewEventForm
            key={composing.day.toISOString() + composing.minutes}
            compose={composing}
            settings={settings}
            accent={accent}
            onClose={onClose}
            onCommit={onCommitNew}
          />
        )}
        {!composing && selection?.kind === "appt" && (
          <ApptDetail appt={selection.data} settings={settings} accent={accent} onClose={onClose} />
        )}
        {!composing && selection?.kind === "draft" && (
          <DraftEditor
            draft={selection.draft}
            settings={settings}
            accent={accent}
            onClose={onClose}
            onUpdate={(patch) => onUpdateDraft(selection.draft.id, patch)}
            onRemove={() => onRemoveDraft(selection.draft.id)}
          />
        )}
      </aside>
    </>
  );
}

function NewEventForm({
  compose,
  settings,
  accent,
  onClose,
  onCommit,
}: {
  compose: { day: Date; minutes: number; durationMin: number };
  settings: CalendarSettings;
  accent: Accent;
  onClose: () => void;
  onCommit: (data: NewEvent) => void;
}) {
  const [name, setName] = useState("");
  const [hour, setHour] = useState(Math.floor(compose.minutes / 60));
  const [minute, setMinute] = useState(compose.minutes % 60);
  const [durationMin, setDurationMin] = useState(compose.durationMin);
  const [service, setService] = useState("");
  const [note, setNote] = useState("");

  const dayLabel = `${compose.day.getDate()} ${TR_MONTHS[compose.day.getMonth()]} ${
    TR_WEEKDAYS_LONG[compose.day.getDay()]
  }`;

  const commit = () => {
    const when = new Date(compose.day);
    when.setHours(hour, minute, 0, 0);
    onCommit({
      name: name.trim() || "Yeni randevu",
      when: when.toISOString(),
      durationMin,
      service: service.trim(),
      note: note.trim(),
    });
  };

  return (
    <>
      <PanelHeader title="Yeni randevu" subtitle={dayLabel} onClose={onClose} />
      <div className="flex-1 space-y-5 overflow-y-auto px-6 py-6">
        <Field label="Hasta / başlık">
          <input
            autoFocus
            value={name}
            onChange={(e) => setName(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && commit()}
            placeholder="Ad Soyad"
            className="w-full rounded-xl border border-zinc-200 px-3 py-2 text-sm text-zinc-900 outline-none transition-colors placeholder:text-zinc-300 focus:border-zinc-400 focus:ring-2 focus:ring-zinc-900/5"
          />
        </Field>

        <Field label="Saat">
          <div className="flex items-center gap-2">
            <Select
              value={hour}
              options={hourOptions(settings.dayStartHour, settings.dayEndHour)}
              onChange={setHour}
            />
            <span className="text-sm font-semibold text-zinc-400">:</span>
            <Select value={minute} options={[0, 15, 30, 45]} onChange={setMinute} pad />
          </div>
        </Field>

        <Field label="Süre">
          <DurationSelect value={durationMin} onChange={setDurationMin} />
        </Field>

        <Field label="İşlem">
          <input
            value={service}
            onChange={(e) => setService(e.target.value)}
            placeholder="örn. İmplant kontrol"
            className="w-full rounded-xl border border-zinc-200 px-3 py-2 text-sm text-zinc-900 outline-none transition-colors placeholder:text-zinc-300 focus:border-zinc-400 focus:ring-2 focus:ring-zinc-900/5"
          />
        </Field>

        <Field label="Not">
          <textarea
            value={note}
            onChange={(e) => setNote(e.target.value)}
            rows={3}
            placeholder="Kısa not…"
            className="w-full resize-none rounded-xl border border-zinc-200 px-3 py-2 text-sm text-zinc-900 outline-none transition-colors placeholder:text-zinc-300 focus:border-zinc-400 focus:ring-2 focus:ring-zinc-900/5"
          />
        </Field>

        <div
          className="rounded-xl px-3 py-2 text-[11px] leading-relaxed"
          style={{ backgroundColor: accent.soft, color: accent.text }}
        >
          Taslak olarak eklenir (yalnızca bu cihazda). Gerçek randevu beyin onayından geçer.
        </div>
      </div>

      <footer className="flex items-center justify-end gap-2 border-t border-zinc-100 px-6 py-4">
        <button
          onClick={onClose}
          className="rounded-xl px-3 py-2 text-xs font-medium text-zinc-500 transition-colors hover:bg-zinc-100"
        >
          İptal
        </button>
        <button
          onClick={commit}
          className="flex items-center gap-1.5 rounded-xl bg-zinc-900 px-4 py-2 text-xs font-semibold text-white transition-colors hover:bg-zinc-800"
        >
          <CalendarPlus className="h-3.5 w-3.5" /> Taslak ekle
        </button>
      </footer>
    </>
  );
}

function PanelHeader({
  title,
  subtitle,
  onClose,
}: {
  title: string;
  subtitle: string;
  onClose: () => void;
}) {
  return (
    <header className="flex items-start justify-between border-b border-zinc-100 px-6 py-5">
      <div className="min-w-0">
        <h2 className="truncate text-base font-semibold tracking-tight text-zinc-900">{title}</h2>
        <p className="mt-0.5 text-xs text-zinc-500">{subtitle}</p>
      </div>
      <button
        onClick={onClose}
        className="grid h-9 w-9 shrink-0 place-items-center rounded-full text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-700"
        aria-label="Kapat"
      >
        <X className="h-4 w-4" />
      </button>
    </header>
  );
}

function ApptDetail({
  appt,
  settings,
  accent,
  onClose,
}: {
  appt: Appointment;
  settings: CalendarSettings;
  accent: Accent;
  onClose: () => void;
}) {
  const date = new Date(appt.when);
  const dayLabel = `${date.getDate()} ${TR_MONTHS[date.getMonth()]} ${TR_WEEKDAYS_LONG[date.getDay()]}`;
  const over = settings.showOverbook && appt.overbook;
  return (
    <>
      <PanelHeader title={appt.name || "İsimsiz"} subtitle="Onaylı randevu" onClose={onClose} />
      <div className="flex-1 space-y-5 overflow-y-auto px-6 py-6">
        <div
          className="flex items-center gap-3 rounded-2xl px-4 py-3"
          style={{ backgroundColor: accent.soft, color: accent.text }}
        >
          <Clock className="h-5 w-5" />
          <div>
            <div className="font-mono text-lg font-bold tabular-nums">
              {formatTime(appt.when, settings.time24h)}
            </div>
            <div className="text-xs opacity-80">{dayLabel}</div>
          </div>
        </div>

        <dl className="space-y-3">
          {(appt.service || appt.segment) && (
            <PanelRow label="İşlem" value={appt.service || appt.segment} />
          )}
          {appt.doctor && (
            <PanelRow
              label="Hekim"
              value={
                <span className="inline-flex items-center gap-1.5">
                  <Stethoscope className="h-3.5 w-3.5 text-zinc-400" /> {appt.doctor}
                </span>
              }
            />
          )}
          <PanelRow label="Telefon" value={maskPhone(appt.phone)} />
          <PanelRow label="Gösterim olasılığı" value={`%${Math.round(appt.pShow * 100)}`} />
          <PanelRow
            label="Durum"
            value={
              over ? (
                <span className="rounded-full border border-amber-200 bg-amber-50 px-2 py-0.5 text-[11px] font-semibold text-amber-700">
                  Overbook
                </span>
              ) : (
                <span className="rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-[11px] font-semibold text-emerald-700">
                  Onaylı
                </span>
              )
            }
          />
        </dl>

        <p className="rounded-xl bg-zinc-50 px-4 py-3 text-[11px] leading-relaxed text-zinc-500">
          Bu randevu beyin tarafından nitelendirilip onaylandı. Durum değişiklikleri WhatsApp ajanı
          ve beyin üzerinden yürür.
        </p>
      </div>
    </>
  );
}

function DraftEditor({
  draft,
  settings,
  accent,
  onClose,
  onUpdate,
  onRemove,
}: {
  draft: Draft;
  settings: CalendarSettings;
  accent: Accent;
  onClose: () => void;
  onUpdate: (patch: Partial<Draft>) => void;
  onRemove: () => void;
}) {
  const date = new Date(draft.when);
  const dayLabel = `${date.getDate()} ${TR_MONTHS[date.getMonth()]} ${TR_WEEKDAYS_LONG[date.getDay()]}`;
  const setTime = (h: number, m: number) => {
    const next = new Date(draft.when);
    next.setHours(h, m, 0, 0);
    onUpdate({ when: next.toISOString() });
  };

  return (
    <>
      <PanelHeader title="Taslak randevu" subtitle={dayLabel} onClose={onClose} />
      <div className="flex-1 space-y-5 overflow-y-auto px-6 py-6">
        <Field label="Hasta / başlık">
          <input
            value={draft.name}
            onChange={(e) => onUpdate({ name: e.target.value })}
            placeholder="Ad Soyad"
            className="w-full rounded-xl border border-zinc-200 px-3 py-2 text-sm text-zinc-900 outline-none transition-colors placeholder:text-zinc-300 focus:border-zinc-400 focus:ring-2 focus:ring-zinc-900/5"
          />
        </Field>

        <Field label="Saat">
          <div className="flex items-center gap-2">
            <Select
              value={date.getHours()}
              options={hourOptions(settings.dayStartHour, settings.dayEndHour)}
              onChange={(h) => setTime(h, date.getMinutes())}
            />
            <span className="text-sm font-semibold text-zinc-400">:</span>
            <Select
              value={date.getMinutes()}
              options={[0, 15, 30, 45]}
              onChange={(m) => setTime(date.getHours(), m)}
              pad
            />
          </div>
        </Field>

        <Field label="Süre">
          <DurationSelect
            value={draft.durationMin}
            onChange={(d) => onUpdate({ durationMin: d })}
          />
        </Field>

        <Field label="İşlem">
          <input
            value={draft.service}
            onChange={(e) => onUpdate({ service: e.target.value })}
            placeholder="örn. İmplant kontrol"
            className="w-full rounded-xl border border-zinc-200 px-3 py-2 text-sm text-zinc-900 outline-none transition-colors placeholder:text-zinc-300 focus:border-zinc-400 focus:ring-2 focus:ring-zinc-900/5"
          />
        </Field>

        <Field label="Not">
          <textarea
            value={draft.note}
            onChange={(e) => onUpdate({ note: e.target.value })}
            rows={3}
            placeholder="Kısa not…"
            className="w-full resize-none rounded-xl border border-zinc-200 px-3 py-2 text-sm text-zinc-900 outline-none transition-colors placeholder:text-zinc-300 focus:border-zinc-400 focus:ring-2 focus:ring-zinc-900/5"
          />
        </Field>

        <div
          className="flex items-center gap-2 rounded-xl px-3 py-2 text-[11px] font-medium"
          style={{ backgroundColor: accent.soft, color: accent.text }}
        >
          <span className="rounded-full bg-white/70 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide">
            Taslak
          </span>
          Yalnızca bu cihazda saklanır — beyin onayı gerektirir.
        </div>
      </div>

      <footer className="border-t border-zinc-100 px-6 py-4">
        <button
          onClick={onRemove}
          className="flex items-center gap-2 text-xs font-medium text-red-500 transition-colors hover:text-red-600"
        >
          <Trash2 className="h-3.5 w-3.5" /> Taslağı sil
        </button>
      </footer>
    </>
  );
}

function PanelRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-3 border-b border-zinc-50 pb-3">
      <dt className="text-xs font-medium text-zinc-400">{label}</dt>
      <dd className="text-right text-sm font-medium text-zinc-800">{value}</dd>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="mb-1.5 block text-[11px] font-semibold uppercase tracking-wide text-zinc-400">
        {label}
      </label>
      {children}
    </div>
  );
}

function Select({
  value,
  options,
  onChange,
  pad,
}: {
  value: number;
  options: number[];
  onChange: (v: number) => void;
  pad?: boolean;
}) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(Number(e.target.value))}
      className="rounded-xl border border-zinc-200 bg-white px-3 py-2 font-mono text-sm font-semibold tabular-nums text-zinc-900 outline-none focus:border-zinc-400"
    >
      {options.map((o) => (
        <option key={o} value={o}>
          {pad ? String(o).padStart(2, "0") : o}
        </option>
      ))}
    </select>
  );
}

function hourOptions(start: number, end: number) {
  return Array.from({ length: end - start }, (_, i) => start + i);
}

function DurationSelect({ value, onChange }: { value: number; onChange: (v: number) => void }) {
  const label = (m: number) => (m < 60 ? `${m} dk` : m % 60 === 0 ? `${m / 60} saat` : `${Math.floor(m / 60)} sa ${m % 60} dk`);
  const opts = DURATION_OPTIONS.includes(value) ? DURATION_OPTIONS : [...DURATION_OPTIONS, value].sort((a, b) => a - b);
  return (
    <select
      value={value}
      onChange={(e) => onChange(Number(e.target.value))}
      className="w-full rounded-xl border border-zinc-200 bg-white px-3 py-2 text-sm font-semibold text-zinc-900 outline-none focus:border-zinc-400"
    >
      {opts.map((o) => (
        <option key={o} value={o}>
          {label(o)}
        </option>
      ))}
    </select>
  );
}
