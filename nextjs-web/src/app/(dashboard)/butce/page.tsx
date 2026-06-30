"use client";

import { Gauge, Layers, Wallet } from "lucide-react";
import { useActiveClinic } from "@/stores/ui-store";
import { useBudget } from "@/lib/queries";
import { Card } from "@/components/ui/Card";
import { StatCard } from "@/components/ui/StatCard";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { QueryBoundary } from "@/components/ui/QueryBoundary";
import { formatTRY } from "@/lib/utils";

export default function ButcePage() {
  const active = useActiveClinic();
  const q = useBudget(30);
  const plan = q.data;
  const allocations = (plan?.allocations ?? []).filter(
    (a) => !active || a.ClinicID === active.id,
  );
  const clinicDaily = allocations.reduce((s, a) => s + a.DailyBudget, 0);
  const maxDaily = Math.max(1, ...allocations.map((a) => a.DailyBudget));
  const funded = allocations.filter((a) => a.DailyBudget > 0).length;

  return (
    <QueryBoundary isLoading={q.isLoading} error={q.error}>
      <div className="space-y-6">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <StatCard
            icon={<Wallet className="h-4 w-4" />}
            label="Klinik Günlük Bütçe"
            value={formatTRY(clinicDaily)}
          />
          <StatCard
            icon={<Gauge className="h-4 w-4" />}
            label="Gölge Fiyat λ"
            value={(plan?.lambda ?? 0).toFixed(2)}
          />
          <StatCard icon={<Layers className="h-4 w-4" />} label="Fonlanan Kol" value={funded} />
        </div>

        <Card className="p-5">
          <h3 className="mb-4 text-xs font-bold uppercase tracking-wider text-zinc-500">
            Kol bazlı günlük dağılım
          </h3>
          {allocations.length === 0 ? (
            <p className="py-6 text-center text-xs text-zinc-400">
              Bu klinik için tahsis bulunamadı.
            </p>
          ) : (
            <div className="space-y-4">
              {allocations.map((a) => (
                <div key={a.ArmID}>
                  <div className="mb-1 flex items-center justify-between text-xs">
                    <span className="font-mono text-[11px] text-zinc-600">{a.ArmID}</span>
                    <span className="font-mono font-bold text-zinc-900">
                      {formatTRY(a.DailyBudget)}/gün
                    </span>
                  </div>
                  <ProgressBar value={(a.DailyBudget / maxDaily) * 100} tone="zinc" />
                  <div className="mt-1 text-[10px] text-zinc-400">
                    Beklenen randevu/gün: {a.ExpectedAppts.toFixed(1)} · θ≈
                    {a.SampledTheta.toFixed(2)}
                  </div>
                </div>
              ))}
            </div>
          )}
        </Card>
      </div>
    </QueryBoundary>
  );
}
