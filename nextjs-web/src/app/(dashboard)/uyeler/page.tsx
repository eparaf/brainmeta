"use client";

import { useActiveClinic } from "@/stores/ui-store";
import { useLeads } from "@/lib/queries";
import { Card } from "@/components/ui/Card";
import { Badge, type Tone } from "@/components/ui/Badge";
import { QueryBoundary } from "@/components/ui/QueryBoundary";
import { maskPhone } from "@/lib/utils";

const COLUMNS: { key: string; title: string; tone: Tone }[] = [
  { key: "Niteleniyor", title: "Nitelendirme / Takip", tone: "amber" },
  { key: "Randevu", title: "Onaylı Randevular", tone: "emerald" },
  { key: "lost", title: "Kayıp / Ulaşılamadı", tone: "red" },
];

function columnOf(status: string): string {
  if (status === "Randevu") return "Randevu";
  if (status === "Düştü" || status === "Gelmedi") return "lost";
  return "Niteleniyor";
}

export default function UyelerPage() {
  const active = useActiveClinic();
  const q = useLeads(active?.id ?? null);
  const leads = q.data ?? [];

  return (
    <QueryBoundary isLoading={q.isLoading} error={q.error}>
      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        {COLUMNS.map((col) => {
          const items = leads.filter((l) => columnOf(l.status) === col.key);
          return (
            <div key={col.key}>
              <div className="mb-3 flex items-center justify-between">
                <h3 className="text-xs font-bold uppercase tracking-wider text-zinc-500">
                  {col.title}
                </h3>
                <Badge tone={col.tone}>{items.length}</Badge>
              </div>
              <div className="space-y-2">
                {items.length === 0 ? (
                  <div className="rounded-xl border border-dashed border-zinc-200 p-6 text-center text-[11px] text-zinc-400">
                    Kayıt yok
                  </div>
                ) : (
                  items.map((l) => (
                    <Card key={l.id} className="p-3">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          {l.qualification.urgency === "Yüksek" && (
                            <span className="h-2 w-2 rounded-full bg-red-500" />
                          )}
                          <span className="text-xs font-bold text-zinc-900">
                            {l.name || "İsimsiz"}
                          </span>
                        </div>
                        <span className="font-mono text-[10px] text-zinc-400">
                          %{l.qualification.intentPct}
                        </span>
                      </div>
                      <div className="mt-1.5 flex items-center justify-between text-[11px] text-zinc-500">
                        <span>{l.qualification.segment || "—"}</span>
                        <span className="font-mono">{maskPhone(l.phoneNumber)}</span>
                      </div>
                      {l.qualification.appointmentTime && (
                        <div className="mt-1.5 text-[10px] font-semibold text-emerald-600">
                          {l.qualification.appointmentTime}
                        </div>
                      )}
                    </Card>
                  ))
                )}
              </div>
            </div>
          );
        })}
      </div>
    </QueryBoundary>
  );
}
