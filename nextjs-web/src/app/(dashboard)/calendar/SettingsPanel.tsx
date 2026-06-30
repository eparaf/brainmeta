"use client";

import { Check, RotateCcw, X } from "lucide-react";
import { ACCENTS, type AccentKey, type CalendarSettings } from "./settings";

type Update = <K extends keyof CalendarSettings>(key: K, value: CalendarSettings[K]) => void;

export function SettingsPanel({
  open,
  onClose,
  settings,
  update,
  reset,
  children,
}: {
  open: boolean;
  onClose: () => void;
  settings: CalendarSettings;
  update: Update;
  reset: () => void;
  children?: React.ReactNode;
}) {
  return (
    <>
      {/* Scrim */}
      <div
        onClick={onClose}
        className={`fixed inset-0 z-40 bg-zinc-900/20 backdrop-blur-[2px] transition-opacity duration-300 ${
          open ? "opacity-100" : "pointer-events-none opacity-0"
        }`}
      />

      {/* Drawer */}
      <aside
        className={`fixed inset-y-0 right-0 z-50 flex w-full max-w-sm flex-col bg-white shadow-2xl ring-1 ring-zinc-900/5 transition-transform duration-300 ease-out ${
          open ? "translate-x-0" : "translate-x-full"
        }`}
      >
        <header className="flex items-center justify-between border-b border-zinc-100 px-6 py-5">
          <div>
            <h2 className="text-base font-semibold tracking-tight text-zinc-900">
              Takvim Özelleştirme
            </h2>
            <p className="mt-0.5 text-xs text-zinc-500">Görünüm tercihleri bu cihazda saklanır.</p>
          </div>
          <button
            onClick={onClose}
            className="grid h-9 w-9 place-items-center rounded-full text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-700"
            aria-label="Kapat"
          >
            <X className="h-4.5 w-4.5" />
          </button>
        </header>

        <div className="flex-1 space-y-7 overflow-y-auto px-6 py-6">
          <Section title="Vurgu rengi">
            <div className="flex flex-wrap gap-2.5">
              {(Object.keys(ACCENTS) as AccentKey[]).map((key) => {
                const a = ACCENTS[key];
                const selected = settings.accent === key;
                return (
                  <button
                    key={key}
                    onClick={() => update("accent", key)}
                    title={a.label}
                    aria-label={a.label}
                    className="grid h-9 w-9 place-items-center rounded-full transition-transform hover:scale-105"
                    style={{
                      backgroundColor: a.dot,
                      boxShadow: selected ? `0 0 0 2px white, 0 0 0 4px ${a.dot}` : "none",
                    }}
                  >
                    {selected && <Check className="h-4 w-4 text-white" strokeWidth={3} />}
                  </button>
                );
              })}
            </div>
          </Section>

          <Section title="Varsayılan görünüm">
            <Segmented
              value={settings.defaultView}
              onChange={(v) => update("defaultView", v)}
              options={[
                { value: "day", label: "Gün" },
                { value: "week", label: "Hafta" },
                { value: "month", label: "Ay" },
              ]}
            />
          </Section>

          <Section title="Yoğunluk">
            <Segmented
              value={settings.density}
              onChange={(v) => update("density", v)}
              options={[
                { value: "comfortable", label: "Geniş" },
                { value: "compact", label: "Sıkışık" },
              ]}
            />
          </Section>

          <Section title="Hafta başlangıcı">
            <Segmented
              value={String(settings.weekStart)}
              onChange={(v) => update("weekStart", Number(v) as 0 | 1)}
              options={[
                { value: "1", label: "Pazartesi" },
                { value: "0", label: "Pazar" },
              ]}
            />
          </Section>

          <Section title="Saat biçimi">
            <Segmented
              value={settings.time24h ? "24" : "12"}
              onChange={(v) => update("time24h", v === "24")}
              options={[
                { value: "24", label: "24 saat" },
                { value: "12", label: "12 saat" },
              ]}
            />
          </Section>

          <Section title="Çalışma saat aralığı">
            <div className="flex items-center gap-3">
              <TimeStepper
                value={settings.dayStartHour}
                min={0}
                max={settings.dayEndHour - 1}
                onChange={(v) => update("dayStartHour", v)}
              />
              <span className="text-xs font-medium text-zinc-400">—</span>
              <TimeStepper
                value={settings.dayEndHour}
                min={settings.dayStartHour + 1}
                max={24}
                onChange={(v) => update("dayEndHour", v)}
              />
            </div>
          </Section>

          <Section title="Görüntülenecekler">
            <div className="space-y-1">
              <Toggle
                label="Hafta sonları"
                checked={settings.showWeekends}
                onChange={(v) => update("showWeekends", v)}
              />
              <Toggle
                label="Gösterim olasılığı"
                checked={settings.showProbability}
                onChange={(v) => update("showProbability", v)}
              />
              <Toggle
                label="Overbook rozeti"
                checked={settings.showOverbook}
                onChange={(v) => update("showOverbook", v)}
              />
              <Toggle
                label="Hekim adı"
                checked={settings.showDoctor}
                onChange={(v) => update("showDoctor", v)}
              />
              <Toggle
                label="İşlem / hizmet"
                checked={settings.showService}
                onChange={(v) => update("showService", v)}
              />
            </div>
          </Section>

          {children}
        </div>

        <footer className="border-t border-zinc-100 px-6 py-4">
          <button
            onClick={reset}
            className="flex items-center gap-2 text-xs font-medium text-zinc-500 transition-colors hover:text-zinc-900"
          >
            <RotateCcw className="h-3.5 w-3.5" />
            Varsayılanlara dön
          </button>
        </footer>
      </aside>
    </>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section>
      <h3 className="mb-3 text-[11px] font-semibold uppercase tracking-[0.12em] text-zinc-400">
        {title}
      </h3>
      {children}
    </section>
  );
}

function Segmented<T extends string>({
  value,
  onChange,
  options,
}: {
  value: T;
  onChange: (v: T) => void;
  options: { value: T; label: string }[];
}) {
  return (
    <div className="inline-flex w-full rounded-xl bg-zinc-100 p-1">
      {options.map((o) => {
        const active = o.value === value;
        return (
          <button
            key={o.value}
            onClick={() => onChange(o.value)}
            className={`flex-1 rounded-lg px-3 py-1.5 text-xs font-semibold transition-all ${
              active
                ? "bg-white text-zinc-900 shadow-sm ring-1 ring-zinc-900/5"
                : "text-zinc-500 hover:text-zinc-800"
            }`}
          >
            {o.label}
          </button>
        );
      })}
    </div>
  );
}

function Toggle({
  label,
  checked,
  onChange,
}: {
  label: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <button
      onClick={() => onChange(!checked)}
      className="flex w-full items-center justify-between rounded-lg px-1 py-2 text-left transition-colors hover:bg-zinc-50"
    >
      <span className="text-sm font-medium text-zinc-700">{label}</span>
      <span
        className={`relative h-5 w-9 shrink-0 rounded-full transition-colors ${
          checked ? "bg-zinc-900" : "bg-zinc-200"
        }`}
      >
        <span
          className={`absolute top-0.5 h-4 w-4 rounded-full bg-white shadow-sm transition-transform ${
            checked ? "translate-x-4" : "translate-x-0.5"
          }`}
        />
      </span>
    </button>
  );
}

function TimeStepper({
  value,
  min,
  max,
  onChange,
}: {
  value: number;
  min: number;
  max: number;
  onChange: (v: number) => void;
}) {
  const label = `${String(value).padStart(2, "0")}:00`;
  return (
    <div className="flex flex-1 items-center justify-between rounded-xl border border-zinc-200 bg-white px-1 py-1">
      <button
        onClick={() => onChange(Math.max(min, value - 1))}
        disabled={value <= min}
        className="grid h-7 w-7 place-items-center rounded-lg text-zinc-500 transition-colors hover:bg-zinc-100 disabled:opacity-30"
      >
        −
      </button>
      <span className="font-mono text-sm font-semibold tabular-nums text-zinc-900">{label}</span>
      <button
        onClick={() => onChange(Math.min(max, value + 1))}
        disabled={value >= max}
        className="grid h-7 w-7 place-items-center rounded-lg text-zinc-500 transition-colors hover:bg-zinc-100 disabled:opacity-30"
      >
        +
      </button>
    </div>
  );
}
