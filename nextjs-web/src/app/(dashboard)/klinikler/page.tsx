"use client";

import { Building2, Radio } from "lucide-react";
import { useClinics } from "@/lib/queries";
import { Card } from "@/components/ui/Card";
import { Badge } from "@/components/ui/Badge";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { QueryBoundary } from "@/components/ui/QueryBoundary";
import { formatTRY } from "@/lib/utils";

export default function KliniklerPage() {
  const q = useClinics();
  const clinics = q.data ?? [];

  return (
    <QueryBoundary
      isLoading={q.isLoading}
      error={q.error}
      isEmpty={!q.isLoading && clinics.length === 0}
    >
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
        {clinics.map((c) => {
          const delivered = c.delivered ?? 0;
          const guarantee = c.guarantee ?? 0;
          const pct = guarantee > 0 ? (delivered / guarantee) * 100 : 0;
          const onTrack = c.status === "on-track";
          return (
            <Card key={c.id} className="p-5">
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-2.5">
                  <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-zinc-100 text-zinc-600">
                    <Building2 className="h-4 w-4" />
                  </div>
                  <div>
                    <div className="text-sm font-bold text-zinc-900">{c.name}</div>
                    <div className="text-[11px] text-zinc-500">
                      {c.district} · {c.segment}
                    </div>
                  </div>
                </div>
                <Badge tone={onTrack ? "emerald" : "amber"}>
                  <Radio className="h-2.5 w-2.5" />
                  {onTrack ? "Hedefte" : "Geride"}
                </Badge>
              </div>
              <div className="mt-4 flex items-end justify-between text-xs">
                <div>
                  <span className="font-mono text-xl font-bold text-zinc-900">{delivered}</span>
                  <span className="text-zinc-400"> / {guarantee} randevu</span>
                </div>
                <div className="text-zinc-500">%{pct.toFixed(0)}</div>
              </div>
              <div className="mt-2">
                <ProgressBar value={pct} tone={onTrack ? "emerald" : "amber"} />
              </div>
              <div className="mt-4 grid grid-cols-2 gap-2 text-[11px]">
                <div className="rounded-lg bg-zinc-50 px-2.5 py-2">
                  <div className="text-zinc-400">Gölge fiyat λ</div>
                  <div className="font-mono text-sm font-bold text-zinc-900">
                    {(c.shadowPrice ?? 0).toFixed(2)}
                  </div>
                </div>
                <div className="rounded-lg bg-zinc-50 px-2.5 py-2">
                  <div className="text-zinc-400">Aylık bütçe</div>
                  <div className="font-mono text-sm font-bold text-zinc-900">
                    {formatTRY(c.monthlyAdBudget)}
                  </div>
                </div>
              </div>
            </Card>
          );
        })}
      </div>
    </QueryBoundary>
  );
}
