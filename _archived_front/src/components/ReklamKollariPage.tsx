import React, { useState } from 'react';
import { 
  GitBranch, 
  Sparkles, 
  HelpCircle,
  TrendingUp,
  RotateCw
} from 'lucide-react';
import { AdArm } from '../types';

interface ReklamKollariPageProps {
  adArms: AdArm[];
  setAdArms: React.Dispatch<React.SetStateAction<Record<string, AdArm[]>>>;
  activeClinicId: string;
}

export default function ReklamKollariPage({
  adArms,
  setAdArms,
  activeClinicId
}: ReklamKollariPageProps) {
  const [notif, setNotif] = useState<string | null>(null);

  // Simulate Thompson Sampling algorithm step
  const handleRunThompsonSampling = () => {
    setAdArms(prev => {
      const currentList = prev[activeClinicId] || [];
      const updatedList = currentList.map(arm => {
        // Thompson Sampling updates quality parameter slightly with some beta-distribution-like noise
        const shift = (Math.random() - 0.5) * 0.08;
        const newTheta = Math.max(0.1, Math.min(0.99, parseFloat((arm.thetaHat + shift).toFixed(3))));
        // Adjust leads and appointments simulated values accordingly
        const extraLeads = Math.floor(Math.random() * 5);
        const extraAppts = Math.random() < newTheta ? 1 : 0;
        return {
          ...arm,
          thetaHat: newTheta,
          leads: arm.leads + extraLeads,
          appointments: arm.appointments + extraAppts,
          spend: arm.spend + (extraLeads * arm.cpl)
        };
      });
      return {
        ...prev,
        [activeClinicId]: updatedList
      };
    });

    setNotif('Thompson Sampling adımı simüle edildi: θ̂ olasılıkları güncellendi.');
    setTimeout(() => setNotif(null), 3500);
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-xs font-bold text-zinc-900 tracking-tight">Öğrenen Reklam Kolları (Thompson Sampling)</h2>
          <p className="text-[11px] text-zinc-500 mt-1">
            Her reklam kolunun kaliteli-lead getirme başarısı anlık olarak Thompson Sampling algoritmamız tarafından optimize edilmektedir.
          </p>
        </div>

        <button
          onClick={handleRunThompsonSampling}
          className="inline-flex items-center gap-1.5 py-1.5 px-3 bg-zinc-900 text-white hover:bg-zinc-800 text-[11px] font-bold rounded shadow-sm transition-colors self-start sm:self-auto"
        >
          <RotateCw className="w-3.5 h-3.5" /> Thompson Adımı Çalıştır
        </button>
      </div>

      {/* Info Notification */}
      {notif && (
        <div className="p-3 bg-emerald-50 border border-emerald-200 text-emerald-800 text-[11px] rounded font-semibold animate-in fade-in duration-200">
          {notif}
        </div>
      )}

      {/* Table Card */}
      <div className="bg-white border border-zinc-200/80 rounded-lg shadow-sm overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-xs text-left border-collapse">
            <thead>
              <tr className="bg-zinc-50/70 border-b border-zinc-200/80 text-[10px] text-zinc-400 font-semibold uppercase tracking-wider">
                <th className="px-4.5 py-3">KOL KODU</th>
                <th className="px-4.5 py-3">HEDEF SEGMENT</th>
                <th className="px-4.5 py-3">KALİTELİ LEAD OLASILIĞI (θ̂)</th>
                <th className="px-4.5 py-3 text-right">CPL (LEAD BAŞI MALİYET)</th>
                <th className="px-4.5 py-3 text-right">TOPLAM LEAD</th>
                <th className="px-4.5 py-3 text-right">RANDEVU</th>
                <th className="px-4.5 py-3 text-right">HARCAMA</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-100 font-medium text-zinc-700">
              {adArms.map((arm) => {
                const percent = Math.round(arm.thetaHat * 100);
                return (
                  <tr key={arm.armId} className="hover:bg-zinc-50/40 transition-colors">
                    <td className="px-4.5 py-3.5 font-mono text-[10.5px] font-bold text-zinc-900">
                      {arm.armId}
                    </td>
                    <td className="px-4.5 py-3.5">
                      {arm.segment}
                    </td>
                    <td className="px-4.5 py-3.5">
                      <div className="flex items-center gap-2.5 max-w-[160px]">
                        {/* Probability score bar */}
                        <div className="flex-1 h-2 bg-zinc-100 border border-zinc-200/50 rounded-full overflow-hidden">
                          <div 
                            className="h-full bg-zinc-900 rounded-full transition-all duration-300"
                            style={{ width: `${percent}%` }}
                          />
                        </div>
                        <span className="font-mono text-[10.5px] font-bold text-zinc-900 shrink-0 w-10">
                          {arm.thetaHat.toFixed(2)}
                        </span>
                      </div>
                    </td>
                    <td className="px-4.5 py-3.5 text-right font-mono">
                      {arm.cpl.toLocaleString('tr-TR')} ₺
                    </td>
                    <td className="px-4.5 py-3.5 text-right font-mono text-zinc-500">
                      {arm.leads.toLocaleString('tr-TR')}
                    </td>
                    <td className="px-4.5 py-3.5 text-right font-mono text-emerald-700 font-bold">
                      {arm.appointments.toLocaleString('tr-TR')}
                    </td>
                    <td className="px-4.5 py-3.5 text-right font-mono text-zinc-900 font-bold">
                      {arm.spend.toLocaleString('tr-TR')} ₺
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>

        {/* Caption */}
        <div className="px-4.5 py-3 bg-zinc-50 border-t border-zinc-100 text-left">
          <p className="text-[10px] text-zinc-400 font-semibold italic">
            * <span className="font-mono text-zinc-500">θ̂ (theta-hat)</span> = beynin öğrendiği kaliteli-lead getirme olasılığı (Thompson sampling). Modelimiz her randevuda ödül (reward=1) veya başarısızlıkta (reward=0) Beta dağılımını günceller.
          </p>
        </div>
      </div>
    </div>
  );
}
