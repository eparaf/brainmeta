"use client";

import { useEffect, useState } from "react";
import { Check, ExternalLink, Loader2 } from "lucide-react";
import Link from "next/link";
import { useSaveWidget, useWidget } from "@/lib/queries";
import type { WidgetConfig } from "@/lib/brain/schemas";

// Widget (public booking) calendar — configured here from the main calendar.
// Bookings made through the embedded widget flow through the brain and land in
// this same calendar automatically (they are appointments), so this section only
// owns the widget calendar's *appearance + copy*.

const SWATCHES = ["#30d158", "#2563eb", "#0f766e", "#7c3aed", "#e11d48", "#d97706", "#111827"];

export function WidgetCalendarSection({ clinicId }: { clinicId: string | null }) {
  const q = useWidget(clinicId);
  const save = useSaveWidget(clinicId);
  const [draft, setDraft] = useState<WidgetConfig | null>(null);
  const [savedAt, setSavedAt] = useState(false);

  // Seed the editable draft once the config arrives / clinic changes.
  /* eslint-disable react-hooks/set-state-in-effect -- mirror server config into an editable draft */
  useEffect(() => {
    if (q.data) setDraft(q.data);
  }, [q.data]);
  /* eslint-enable react-hooks/set-state-in-effect */

  if (!clinicId) {
    return (
      <Wrapper>
        <p className="text-xs text-zinc-400">Önce bir klinik seçin.</p>
      </Wrapper>
    );
  }

  if (q.isLoading || !draft) {
    return (
      <Wrapper>
        <div className="flex items-center gap-2 py-2 text-xs text-zinc-400">
          <Loader2 className="h-3.5 w-3.5 animate-spin" /> Widget ayarları yükleniyor…
        </div>
      </Wrapper>
    );
  }

  const set = <K extends keyof WidgetConfig>(key: K, value: WidgetConfig[K]) =>
    setDraft((d) => (d ? { ...d, [key]: value } : d));

  const onSave = () => {
    if (!draft) return;
    save.mutate(draft, {
      onSuccess: () => {
        setSavedAt(true);
        window.setTimeout(() => setSavedAt(false), 1800);
      },
    });
  };

  return (
    <Wrapper>
      <div className="space-y-4">
        <Field label="Takvim rengi">
          <div className="flex flex-wrap items-center gap-2">
            {SWATCHES.map((c) => {
              const selected = draft.calendarColor?.toLowerCase() === c.toLowerCase();
              return (
                <button
                  key={c}
                  onClick={() => set("calendarColor", c)}
                  className="grid h-8 w-8 place-items-center rounded-full transition-transform hover:scale-105"
                  style={{
                    backgroundColor: c,
                    boxShadow: selected ? `0 0 0 2px white, 0 0 0 4px ${c}` : "none",
                  }}
                  aria-label={c}
                >
                  {selected && <Check className="h-4 w-4 text-white" strokeWidth={3} />}
                </button>
              );
            })}
          </div>
        </Field>

        <Field label="Takvim başlığı">
          <TextInput
            value={draft.calendarTitle}
            placeholder="Randevu seçin"
            onChange={(v) => set("calendarTitle", v)}
          />
        </Field>

        <Field label="Takvim alt başlığı">
          <TextInput
            value={draft.calendarSubtitle}
            placeholder="Size uygun günü ve saati seçin"
            onChange={(v) => set("calendarSubtitle", v)}
          />
        </Field>

        <Field label="Onay metni">
          <TextInput
            value={draft.confirmText}
            placeholder="Randevunuzu onaylıyor musunuz?"
            onChange={(v) => set("confirmText", v)}
          />
        </Field>

        <button
          onClick={() => set("recommend", !draft.recommend)}
          className="flex w-full items-center justify-between rounded-lg px-1 py-1.5 text-left transition-colors hover:bg-zinc-50"
        >
          <span className="text-sm font-medium text-zinc-700">Akıllı saat önerisi</span>
          <span
            className={`relative h-5 w-9 shrink-0 rounded-full transition-colors ${
              draft.recommend ? "bg-zinc-900" : "bg-zinc-200"
            }`}
          >
            <span
              className={`absolute top-0.5 h-4 w-4 rounded-full bg-white shadow-sm transition-transform ${
                draft.recommend ? "translate-x-4" : "translate-x-0.5"
              }`}
            />
          </span>
        </button>

        <div className="flex items-center gap-2 pt-1">
          <button
            onClick={onSave}
            disabled={save.isPending}
            className="flex items-center gap-1.5 rounded-xl bg-zinc-900 px-4 py-2 text-xs font-semibold text-white transition-colors hover:bg-zinc-800 disabled:opacity-50"
          >
            {save.isPending ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : savedAt ? (
              <Check className="h-3.5 w-3.5" />
            ) : null}
            {savedAt ? "Kaydedildi" : "Widget'ı kaydet"}
          </button>
          <Link
            href="/baglantilar/widget"
            className="flex items-center gap-1 text-xs font-medium text-zinc-500 transition-colors hover:text-zinc-900"
          >
            Tüm widget ayarları <ExternalLink className="h-3 w-3" />
          </Link>
        </div>

        <p className="text-[11px] leading-relaxed text-zinc-400">
          Widget üzerinden alınan randevular beyin onayından geçip bu takvime otomatik düşer.
        </p>
      </div>
    </Wrapper>
  );
}

function Wrapper({ children }: { children: React.ReactNode }) {
  return (
    <section>
      <h3 className="mb-3 text-[11px] font-semibold uppercase tracking-[0.12em] text-zinc-400">
        Widget Takvimi
      </h3>
      {children}
    </section>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="mb-1.5 block text-xs font-medium text-zinc-600">{label}</label>
      {children}
    </div>
  );
}

function TextInput({
  value,
  placeholder,
  onChange,
}: {
  value: string;
  placeholder?: string;
  onChange: (v: string) => void;
}) {
  return (
    <input
      value={value}
      placeholder={placeholder}
      onChange={(e) => onChange(e.target.value)}
      className="w-full rounded-xl border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-900 outline-none transition-colors placeholder:text-zinc-300 focus:border-zinc-400 focus:ring-2 focus:ring-zinc-900/5"
    />
  );
}
