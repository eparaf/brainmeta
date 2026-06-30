"use client";

import {
  AlertCircle,
  CalendarCheck,
  MessageSquare,
  ShieldCheck,
  TrendingUp,
  Wallet,
} from "lucide-react";
import { useActiveClinic } from "@/stores/ui-store";
import { useArms, useClinics, useLeads } from "@/lib/queries";
import { Card } from "@/components/ui/Card";
import { StatCard } from "@/components/ui/StatCard";
import { Badge } from "@/components/ui/Badge";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { QueryBoundary } from "@/components/ui/QueryBoundary";
import { formatTRY } from "@/lib/utils";

export default function DashboardPage() {
  const active = useActiveClinic();
  const clinicId = active?.id ?? null;
  const arms = useArms(clinicId);
  const clinics = useClinics();
  const leads = useLeads(clinicId);

  const armList = arms.data ?? [];
  const totalLeads = armList.reduce((s, a) => s + a.leads, 0);
  const totalAppts = armList.reduce((s, a) => s + a.appts, 0);
  const totalSpend = armList.reduce((s, a) => s + a.spend, 0);
  const avgCpl = totalLeads > 0 ? totalSpend / totalLeads : 0;

  const enriched = clinics.data?.find((c) => c.id === clinicId);
  const delivered = enriched?.delivered ?? 0;
  const guarantee = enriched?.guarantee ?? 0;
  const pct = guarantee > 0 ? (delivered / guarantee) * 100 : 0;
  const onTrack = enriched?.status === "on-track";

  const pending = (leads.data ?? []).filter((l) => l.status === "Niteleniyor");

  return (
    <QueryBoundary
      isLoading={arms.isLoading || clinics.isLoading}
      error={arms.error ?? clinics.error}
    >
      <div className="space-y-6">
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          <StatCard icon={<MessageSquare className="h-4 w-4" />} label="Toplam Lead" value={totalLeads} />
          <StatCard icon={<CalendarCheck className="h-4 w-4" />} label="Randevu" value={totalAppts} />
          <StatCard icon={<Wallet className="h-4 w-4" />} label="Harcama" value={formatTRY(totalSpend)} />
          <StatCard icon={<TrendingUp className="h-4 w-4" />} label="Ort. CPL" value={formatTRY(avgCpl)} />
        </div>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
          <Card className="p-5 lg:col-span-2">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <ShieldCheck className="h-4 w-4 text-emerald-600" />
                <span className="text-sm font-bold text-zinc-900">Performans Kotası</span>
              </div>
              {enriched?.status ? (
                <Badge tone={onTrack ? "emerald" : "amber"}>
                  {onTrack ? "Hedefte" : "Geride"}
                </Badge>
              ) : null}
            </div>
            <div className="mt-4 flex items-end justify-between text-xs">
              <div>
                <span className="font-mono text-2xl font-bold text-zinc-900">{delivered}</span>
                <span className="text-zinc-400"> / {guarantee} randevu</span>
              </div>
              <div className="text-zinc-500">
                Gölge fiyat λ:{" "}
                <span className="font-mono font-bold text-zinc-900">
                  {(enriched?.shadowPrice ?? 0).toFixed(2)}
                </span>
              </div>
            </div>
            <div className="mt-2">
              <ProgressBar value={pct} tone={pct >= 90 ? "emerald" : pct >= 60 ? "amber" : "red"} />
            </div>
          </Card>

          <Card className="p-5">
            <div className="flex items-center gap-2">
              <AlertCircle className="h-4 w-4 text-amber-500" />
              <span className="text-sm font-bold text-zinc-900">Bekleyen Aksiyon</span>
            </div>
            <div className="mt-3 font-mono text-3xl font-bold text-zinc-900">{pending.length}</div>
            <p className="mt-1 text-[11px] text-zinc-500">
              Niteleniyor durumundaki başvuru sırada bekliyor.
            </p>
          </Card>
        </div>
      </div>
    </QueryBoundary>
  );
}
