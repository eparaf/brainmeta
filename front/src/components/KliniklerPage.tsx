import React, { useState } from 'react';
import { 
  Building2, 
  Target, 
  TrendingUp, 
  Plus, 
  Minus,
  Percent,
  Activity,
  Award
} from 'lucide-react';
import { Clinic } from '../types';

interface KliniklerPageProps {
  clinics: Clinic[];
  setClinics: React.Dispatch<React.SetStateAction<Clinic[]>>;
}

export default function KliniklerPage({
  clinics,
  setClinics
}: KliniklerPageProps) {

  // Action to simulate a lead delivery
  const handleSimulateDelivery = (id: string) => {
    setClinics(prev => prev.map(c => {
      if (c.id === id) {
        const nextDelivered = c.delivered + 1;
        const nextStatus = nextDelivered >= c.guarantee ? 'on-track' : c.status;
        return {
          ...c,
          delivered: nextDelivered,
          status: nextStatus
        };
      }
      return c;
    }));
  };

  // Action to change shadow price
  const handleUpdateShadowPrice = (id: string, amount: number) => {
    setClinics(prev => prev.map(c => {
      if (c.id === id) {
        return {
          ...c,
          shadowPrice: Math.max(0.1, parseFloat((c.shadowPrice + amount).toFixed(2)))
        };
      }
      return c;
    }));
  };

  return (
    <div className="space-y-6">
      {/* Intro Header */}
      <div>
        <h2 className="text-xs font-bold text-zinc-900 tracking-tight">Klinik Taahhüt ve Büyüme Matrisi</h2>
        <p className="text-[11px] text-zinc-500 mt-1">
          Klinik bazlı garanti edilen aylık dönüşüm hedefleri, gerçekleşen teslimatlar ve gölge fiyat ($\lambda$) seviyeleri.
        </p>
      </div>

      {/* Grid List */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
        {clinics.map((clinic) => {
          const percent = Math.min(100, Math.round((clinic.delivered / clinic.guarantee) * 100));
          const isMet = clinic.delivered >= clinic.guarantee;

          return (
            <div 
              key={clinic.id}
              className="bg-white border border-zinc-200/80 rounded-lg p-5 shadow-sm hover:border-zinc-300 transition-all flex flex-col justify-between"
            >
              {/* Top Bar: Icon + Status Pill */}
              <div className="flex items-center justify-between mb-4">
                <div className="p-2 bg-zinc-50 rounded-md border border-zinc-100">
                  <Building2 className="w-5 h-5 text-zinc-800" />
                </div>
                
                {isMet ? (
                  <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[10px] font-semibold bg-emerald-50 text-emerald-700 border border-emerald-200">
                    <Award className="w-3 h-3 text-emerald-500" /> Hedef Yakalandı
                  </span>
                ) : clinic.status === 'on-track' ? (
                  <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[10px] font-semibold bg-emerald-50/50 text-emerald-600 border border-emerald-100">
                    Yolda
                  </span>
                ) : (
                  <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[10px] font-semibold bg-amber-50 text-amber-700 border border-amber-200">
                    Hedef Gerisinde
                  </span>
                )}
              </div>

              {/* Clinic Name */}
              <div className="mb-4 text-left">
                <h3 className="text-sm font-bold text-zinc-900 leading-snug">
                  {clinic.name}
                </h3>
                <p className="text-[10px] text-zinc-400 font-medium font-mono uppercase tracking-wider mt-0.5">
                  ID: {clinic.id}
                </p>
              </div>

              {/* Progress Metric */}
              <div className="space-y-2 mb-5">
                <div className="flex justify-between items-baseline text-xs">
                  <span className="text-zinc-500 font-medium">Teslimat İlerlemesi</span>
                  <span className="font-mono font-bold text-zinc-900">
                    {clinic.delivered} / <span className="text-zinc-400 font-normal">{clinic.guarantee}</span>
                  </span>
                </div>

                {/* Progress Bar */}
                <div className="w-full h-2 bg-zinc-100 rounded-full overflow-hidden border border-zinc-200/10">
                  <div 
                    className={`h-full rounded-full transition-all duration-300 ${isMet ? 'bg-emerald-500' : 'bg-zinc-800'}`}
                    style={{ width: `${percent}%` }}
                  />
                </div>

                <div className="flex justify-between text-[10px] text-zinc-400 font-mono">
                  <span>Tamamlanma Oranı:</span>
                  <span className={isMet ? 'text-emerald-600 font-bold' : 'font-semibold text-zinc-600'}>
                    %{percent}
                  </span>
                </div>
              </div>

              {/* Stats & Optimization coefficients */}
              <div className="grid grid-cols-2 gap-3.5 bg-zinc-50 p-3 rounded border border-zinc-200/50 mb-5">
                <div className="text-left">
                  <span className="block text-[9px] text-zinc-400 font-semibold uppercase tracking-wider">
                    GÖLGE FİYAT λ
                  </span>
                  <span className="text-sm font-bold font-mono text-zinc-900">
                    {clinic.shadowPrice.toFixed(2)} ₺
                  </span>
                </div>
                <div className="text-left border-l border-zinc-200/80 pl-3">
                  <span className="block text-[9px] text-zinc-400 font-semibold uppercase tracking-wider">
                    DURUM
                  </span>
                  <span className={`text-xs font-bold ${isMet ? 'text-emerald-600' : clinic.status === 'on-track' ? 'text-zinc-700' : 'text-amber-600'}`}>
                    {isMet ? 'Hedef Üstü' : clinic.status === 'on-track' ? 'Plan Dahilinde' : 'Risk Sınıfı'}
                  </span>
                </div>
              </div>

              {/* Simulation Controls (SaaS interactive feel!) */}
              <div className="border-t border-zinc-100 pt-3.5 flex flex-col gap-2">
                <span className="text-[9px] text-zinc-400 font-semibold tracking-wide uppercase text-left">
                  MOCK DENEYSEL KONTROLLER
                </span>
                <div className="flex items-center justify-between gap-1.5">
                  <button
                    onClick={() => handleSimulateDelivery(clinic.id)}
                    className="flex-1 inline-flex items-center justify-center gap-1 py-1.5 px-2 bg-zinc-900 text-white hover:bg-zinc-800 text-[10px] font-bold rounded shadow-sm transition-colors"
                  >
                    <Plus className="w-3 h-3" /> Teslimat Ekle
                  </button>

                  <div className="flex items-center border border-zinc-200 rounded-md bg-white">
                    <button
                      onClick={() => handleUpdateShadowPrice(clinic.id, -0.05)}
                      className="p-1 text-zinc-500 hover:text-zinc-900 border-r border-zinc-200"
                      title="Gölge fiyatı azalt"
                    >
                      <Minus className="w-3 h-3" />
                    </button>
                    <span className="px-1.5 text-[9px] font-bold font-mono text-zinc-600">λ</span>
                    <button
                      onClick={() => handleUpdateShadowPrice(clinic.id, 0.05)}
                      className="p-1 text-zinc-500 hover:text-zinc-900 border-l border-zinc-200"
                      title="Gölge fiyatı arttır"
                    >
                      <Plus className="w-3 h-3" />
                    </button>
                  </div>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
