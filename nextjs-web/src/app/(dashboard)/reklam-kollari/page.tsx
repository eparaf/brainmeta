"use client";

import { useActiveClinic } from "@/stores/ui-store";
import { useArms } from "@/lib/queries";
import { Card } from "@/components/ui/Card";
import { QueryBoundary } from "@/components/ui/QueryBoundary";
import { formatTRY } from "@/lib/utils";

export default function ReklamKollariPage() {
  const active = useActiveClinic();
  const q = useArms(active?.id ?? null);
  const arms = q.data ?? [];

  return (
    <QueryBoundary
      isLoading={q.isLoading}
      error={q.error}
      isEmpty={!q.isLoading && arms.length === 0}
      emptyText="Bu klinik için reklam kolu bulunamadı."
    >
      <Card className="overflow-x-auto">
        <table className="w-full text-left text-xs">
          <thead className="border-b border-zinc-100 bg-zinc-50/50 text-[10px] uppercase tracking-wide text-zinc-400">
            <tr>
              <th className="px-4 py-3 font-bold">Segment / Kol</th>
              <th className="px-4 py-3 text-right font-bold">θ̂</th>
              <th className="px-4 py-3 text-right font-bold">CPL</th>
              <th className="px-4 py-3 text-right font-bold">Lead</th>
              <th className="px-4 py-3 text-right font-bold">Randevu</th>
              <th className="px-4 py-3 text-right font-bold">Harcama</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-100">
            {arms.map((a) => (
              <tr key={a.armId} className="hover:bg-zinc-50/50">
                <td className="px-4 py-3">
                  <div className="font-bold text-zinc-900">{a.segment}</div>
                  <div className="font-mono text-[10px] text-zinc-400">{a.armId}</div>
                </td>
                <td className="px-4 py-3 text-right font-mono">
                  <div className="inline-flex items-center gap-2">
                    <div className="h-1.5 w-12 overflow-hidden rounded-full bg-zinc-100">
                      <div
                        className="h-full bg-emerald-500"
                        style={{ width: `${Math.min(100, a.thetaHat * 100)}%` }}
                      />
                    </div>
                    {a.thetaHat.toFixed(2)}
                  </div>
                </td>
                <td className="px-4 py-3 text-right font-mono">{formatTRY(a.cpl)}</td>
                <td className="px-4 py-3 text-right font-mono">{a.leads}</td>
                <td className="px-4 py-3 text-right font-mono">{a.appts}</td>
                <td className="px-4 py-3 text-right font-mono">{formatTRY(a.spend)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>
    </QueryBoundary>
  );
}
