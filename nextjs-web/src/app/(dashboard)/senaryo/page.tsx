"use client";

import { useState } from "react";
import { Calculator, TrendingDown, TrendingUp } from "lucide-react";
import { useActiveClinic } from "@/stores/ui-store";
import { useScenario } from "@/lib/queries";
import { Card } from "@/components/ui/Card";
import { StatCard } from "@/components/ui/StatCard";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { formatTRY } from "@/lib/utils";
import type { ScenarioBand } from "@/lib/brain/schemas";

// Senaryo/fizibilite kartı: "bu bütçeyle ayda kaç randevu beklenir?" — çevrimdışı
// Monte-Carlo tahmini (POST /v1/scenario). Sıfır harcama, LLM yok; sadece bir
// what-if hesaplayıcı, satış konuşması öncesi hızlı fizibilite kontrolü.
export default function SenaryoPage() {
  const active = useActiveClinic();
  const [budget, setBudget] = useState(() => String(active?.monthlyAdBudget || 100_000));
  const scenario = useScenario();

  const run = () => {
    const monthlyBudget = Number(budget);
    if (!monthlyBudget || monthlyBudget <= 0) return;
    scenario.mutate({ clinicId: active?.id, monthlyBudget });
  };

  const r = scenario.data;

  return (
    <div className="space-y-6">
      <Card className="p-5">
        <h3 className="mb-1 text-xs font-bold uppercase tracking-wider text-zinc-500">
          Senaryo / Fizibilite
        </h3>
        <p className="mb-4 text-xs text-zinc-400">
          Bu bütçeyle ayda kaç randevu beklenir? Çevrimdışı Monte-Carlo tahmini — sıfır harcama,
          LLM yok.
        </p>
        <div className="flex flex-wrap items-end gap-3">
          <div className="w-48">
            <label className="mb-1 block text-[10px] font-bold uppercase tracking-wider text-zinc-500">
              Aylık bütçe (TRY)
            </label>
            <Input
              type="number"
              min={0}
              value={budget}
              onChange={(e) => setBudget(e.target.value)}
            />
          </div>
          <Button onClick={run} disabled={scenario.isPending}>
            <Calculator className="mr-1.5 h-4 w-4" />
            {scenario.isPending ? "Hesaplanıyor…" : "Hesapla"}
          </Button>
          {active && (
            <span className="pb-2 text-[11px] text-zinc-400">
              Klinik: <span className="font-semibold text-zinc-600">{active.name}</span>
            </span>
          )}
        </div>
        {scenario.error && (
          <p className="mt-3 text-xs font-medium text-red-600">
            {(scenario.error as Error).message}
          </p>
        )}
      </Card>

      {r && (
        <>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <StatCard
              icon={<TrendingDown className="h-4 w-4" />}
              label="Pesimist (P10)"
              value={r.bookedAppointments.p10.toFixed(1)}
            />
            <StatCard
              icon={<Calculator className="h-4 w-4" />}
              label="Gerçekçi (P50)"
              value={r.bookedAppointments.p50.toFixed(1)}
            />
            <StatCard
              icon={<TrendingUp className="h-4 w-4" />}
              label="Optimist (P90)"
              value={r.bookedAppointments.p90.toFixed(1)}
            />
          </div>

          <Card className="p-5">
            <h3 className="mb-4 text-xs font-bold uppercase tracking-wider text-zinc-500">
              Aylık randevu bandı: {r.bookedAppointments.p10.toFixed(0)}–
              {r.bookedAppointments.p90.toFixed(0)} (gerçekçi:{" "}
              {r.bookedAppointments.p50.toFixed(0)})
            </h3>
            <div className="space-y-3 text-xs">
              <BandRow label="Tıklama / ay" band={r.clicks} />
              <BandRow label="Nitelikli lead / ay" band={r.qualifiedLeads} />
              <BandRow label="Randevu (booked) / ay" band={r.bookedAppointments} />
              <BandRow label="Gelen randevu (kept) / ay" band={r.keptAppointments} />
            </div>
            <div className="mt-5 grid grid-cols-1 gap-3 border-t border-zinc-100 pt-4 sm:grid-cols-2">
              <CostRow label="Lead başı maliyet" band={r.costPerLeadTRY} />
              <CostRow label="Randevu başı maliyet" band={r.costPerAppointmentTRY} />
            </div>
            <div className="mt-4 border-t border-zinc-100 pt-3 text-[10px] text-zinc-400">
              Varsayımlar: qualify {(r.assumptions.funnel.Qualify * 100).toFixed(0)}% · book{" "}
              {(r.assumptions.funnel.Book * 100).toFixed(0)}% · show{" "}
              {(r.assumptions.funnel.Show * 100).toFixed(0)}% · CPC≈
              {formatTRY(r.assumptions.avgCpcTRY)} · click→lead{" "}
              {(r.assumptions.clickToLead * 100).toFixed(1)}% · {r.runs} koşu
            </div>
          </Card>
        </>
      )}
    </div>
  );
}

function BandRow({ label, band }: { label: string; band: ScenarioBand }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-zinc-500">{label}</span>
      <span className="font-mono font-semibold text-zinc-900">
        {band.p10.toFixed(1)} – {band.p50.toFixed(1)} – {band.p90.toFixed(1)}
      </span>
    </div>
  );
}

function CostRow({ label, band }: { label: string; band: ScenarioBand }) {
  return (
    <div>
      <div className="text-[10px] font-bold uppercase tracking-wider text-zinc-500">{label}</div>
      <div className="mt-0.5 font-mono text-xs text-zinc-700">
        {formatTRY(band.p10)} – {formatTRY(band.p50)} – {formatTRY(band.p90)}
      </div>
    </div>
  );
}
