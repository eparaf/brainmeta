import React, { useState, useMemo } from 'react';
import { 
  Coins, 
  TrendingUp, 
  Layers,
  HelpCircle,
  Plus,
  Minus
} from 'lucide-react';
import { AdArm, Clinic } from '../types';

interface ButcePageProps {
  activeClinic: Clinic;
  adArms: AdArm[];
}

export default function ButcePage({
  activeClinic,
  adArms
}: ButcePageProps) {
  
  // Local state for budget allocations per arm id to allow interactive updates
  const [allocations, setAllocations] = useState<Record<string, number>>({
    // DentPlus
    'ARM-DP-01': 14500,
    'ARM-DP-02': 11000,
    'ARM-DP-03': 8500,
    'ARM-DP-04': 7000,
    'ARM-DP-05': 4000,
    // Acıbadem
    'ARM-AC-01': 16000,
    'ARM-AC-02': 12000,
    'ARM-AC-03': 9500,
    'ARM-AC-04': 7500,
    // Ataköy
    'ARM-AS-01': 11000,
    'ARM-AS-02': 8000,
    'ARM-AS-03': 6000,
    'ARM-AS-04': 5000,
  });

  // Calculate stats based on current allocations of the active clinic's ad arms
  const activeAllocationsList = useMemo(() => {
    return adArms.map(arm => ({
      id: arm.armId,
      segment: arm.segment,
      budget: allocations[arm.armId] || 5000
    }));
  }, [adArms, allocations]);

  const totalDailyBudget = useMemo(() => {
    return activeAllocationsList.reduce((sum, item) => sum + item.budget, 0);
  }, [activeAllocationsList]);

  const fundedArmsCount = useMemo(() => {
    return activeAllocationsList.filter(item => item.budget > 0).length;
  }, [activeAllocationsList]);

  // Handle budget shift
  const handleAdjustBudget = (id: string, delta: number) => {
    setAllocations(prev => {
      const current = prev[id] || 5000;
      const next = Math.max(0, current + delta);
      return {
        ...prev,
        [id]: next
      };
    });
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h2 className="text-xs font-bold text-zinc-900 tracking-tight">Otonom Bütçe Allokasyon Paneli</h2>
        <p className="text-[11px] text-zinc-500 mt-1">
          Gölge fiyat parametresine ($\lambda$) bağlı olarak aktif reklam kollarının günlük bütçe dağılımlarını anlık düzenleyin.
        </p>
      </div>

      {/* Top Stat Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
        
        {/* Stat 1: Ağ günlük bütçe */}
        <div className="bg-white border border-zinc-200/80 rounded-lg p-4.5 shadow-sm text-left">
          <div className="flex items-center justify-between mb-3">
            <span className="text-[10px] text-zinc-400 font-semibold uppercase tracking-wider">
              Ağ Günlük Bütçesi
            </span>
            <div className="p-1.5 bg-zinc-50 border border-zinc-200/60 rounded text-zinc-800">
              <Coins className="w-4 h-4" />
            </div>
          </div>
          <div className="text-xl font-bold font-mono text-zinc-950">
            {totalDailyBudget.toLocaleString('tr-TR')} ₺
          </div>
          <p className="text-[10px] text-zinc-400 font-medium mt-1">
            Aktif kolların toplam günlük bütçe harcaması
          </p>
        </div>

        {/* Stat 2: Gölge fiyat λ */}
        <div className="bg-white border border-zinc-200/80 rounded-lg p-4.5 shadow-sm text-left">
          <div className="flex items-center justify-between mb-3">
            <span className="text-[10px] text-zinc-400 font-semibold uppercase tracking-wider">
              Gölge Fiyat (λ)
            </span>
            <div className="p-1.5 bg-zinc-50 border border-zinc-200/60 rounded text-zinc-800">
              <TrendingUp className="w-4 h-4" />
            </div>
          </div>
          <div className="text-xl font-bold font-mono text-zinc-950">
            {activeClinic.shadowPrice.toFixed(2)} ₺
          </div>
          <p className="text-[10px] text-zinc-400 font-medium mt-1">
            Maksimum teklif marjı sınır katsayısı
          </p>
        </div>

        {/* Stat 3: Fonlanan kol */}
        <div className="bg-white border border-zinc-200/80 rounded-lg p-4.5 shadow-sm text-left">
          <div className="flex items-center justify-between mb-3">
            <span className="text-[10px] text-zinc-400 font-semibold uppercase tracking-wider">
              Fonlanan Reklam Kolu
            </span>
            <div className="p-1.5 bg-zinc-50 border border-zinc-200/60 rounded text-zinc-800">
              <Layers className="w-4 h-4" />
            </div>
          </div>
          <div className="text-xl font-bold font-mono text-zinc-950">
            {fundedArmsCount} / <span className="text-zinc-400">{activeAllocationsList.length}</span>
          </div>
          <p className="text-[10px] text-zinc-400 font-medium mt-1">
            Bütçe ayrılmış aktif optimizasyon kolları
          </p>
        </div>

      </div>

      {/* Allocations list card */}
      <div className="bg-white border border-zinc-200/80 rounded-lg shadow-sm p-5 text-left">
        <h3 className="text-xs font-bold text-zinc-950 mb-1.5">Kollara Göre Günlük Bütçe Dağılımı</h3>
        <p className="text-[10.5px] text-zinc-400 mb-5">
          Her bir kolun bütçesini bütçe artırım butonları ile simüle edebilirsiniz. Toplam bütçe otomatik hesaplanır.
        </p>

        <div className="space-y-4.5">
          {activeAllocationsList.map((item) => {
            // Find max budget in list to calculate ratios beautifully
            const maxVal = Math.max(...activeAllocationsList.map(a => a.budget), 10000);
            const ratioPercent = Math.min(100, Math.round((item.budget / maxVal) * 100));

            return (
              <div key={item.id} className="group flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-3 rounded-lg border border-zinc-100 bg-zinc-50/40 hover:bg-zinc-50/80 transition-colors">
                
                {/* Information */}
                <div className="sm:w-1/3 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-[10.5px] font-bold text-zinc-900 bg-zinc-100 border border-zinc-200 px-1.5 py-0.2 rounded">
                      {item.id}
                    </span>
                    <span className="text-xs font-bold text-zinc-800 truncate">
                      {item.segment}
                    </span>
                  </div>
                </div>

                {/* Progress bar ratio list */}
                <div className="flex-1">
                  <div className="w-full h-2.5 bg-zinc-100 rounded-full overflow-hidden border border-zinc-200/30 flex items-center">
                    <div 
                      className="h-full bg-zinc-900 rounded-full transition-all duration-300" 
                      style={{ width: `${ratioPercent}%` }}
                    />
                  </div>
                </div>

                {/* Numeric Controls & Info */}
                <div className="flex items-center justify-between sm:justify-end gap-4 shrink-0">
                  <span className="text-xs font-bold font-mono text-zinc-900 w-20 text-right">
                    {item.budget.toLocaleString('tr-TR')} ₺
                  </span>

                  {/* Increment Buttons */}
                  <div className="flex items-center border border-zinc-200 rounded-md bg-white">
                    <button
                      onClick={() => handleAdjustBudget(item.id, -500)}
                      disabled={item.budget <= 0}
                      className="p-1.5 text-zinc-500 hover:text-zinc-900 disabled:opacity-40 border-r border-zinc-200"
                      title="500 ₺ azalt"
                    >
                      <Minus className="w-3.5 h-3.5" />
                    </button>
                    <button
                      onClick={() => handleAdjustBudget(item.id, 500)}
                      className="p-1.5 text-zinc-500 hover:text-zinc-900 border-l border-zinc-200"
                      title="500 ₺ arttır"
                    >
                      <Plus className="w-3.5 h-3.5" />
                    </button>
                  </div>
                </div>

              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
