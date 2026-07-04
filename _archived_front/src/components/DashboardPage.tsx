import React from 'react';
import { 
  CheckCircle2, 
  Clock, 
  AlertTriangle, 
  TrendingUp, 
  Users, 
  CalendarDays,
  ArrowRight,
  BrainCircuit,
  Phone,
  Banknote,
  Calculator,
  Wallet,
  PieChart,
  BarChart3
} from 'lucide-react';
import { Clinic, Conversation, AdArm } from '../types';

interface DashboardPageProps {
  activeClinic: Clinic;
  conversations: Conversation[];
  adArms: AdArm[];
  onNavigateToChat: (id: string) => void;
}

export default function DashboardPage({
  activeClinic,
  conversations,
  adArms,
  onNavigateToChat
}: DashboardPageProps) {

  // Derive stats
  const totalLeads = adArms.reduce((sum, arm) => sum + arm.leads, 0);
  const totalAppointments = adArms.reduce((sum, arm) => sum + arm.appointments, 0);
  const totalSpend = adArms.reduce((sum, arm) => sum + arm.spend, 0);
  
  const avgCpl = totalLeads > 0 ? (totalSpend / totalLeads) : 0;
  const avgCpa = totalAppointments > 0 ? (totalSpend / totalAppointments) : 0;
  
  const estTreatmentValue = 40000; // Mock average ticket size in TRY
  const estRevenue = totalAppointments * estTreatmentValue;
  const roi = totalSpend > 0 ? ((estRevenue - totalSpend) / totalSpend) * 100 : 0;
  
  // Get patients needing action
  const pendingConversations = conversations.filter(c => c.status === 'Niteleniyor' || c.status === 'Düştü');
  const bookedConversations = conversations.filter(c => c.status === 'Randevu');

  const getInitials = (name: string) => {
    return name.split(' ').map(n => n[0]).join('').toUpperCase().substring(0, 2);
  };

  const maskPhoneNumber = (phone?: string, code?: string) => {
    if (!phone) return 'Bilinmeyen Numara';
    const cleanPhone = phone.replace(/\D/g, '');
    if (cleanPhone.length >= 10) {
      const p1 = cleanPhone.substring(0, 3);
      const last2 = cleanPhone.substring(cleanPhone.length - 2);
      const flag = code === '+44' ? '🇬🇧' : code === '+1' ? '🇺🇸' : code === '+49' ? '🇩🇪' : '🇹🇷';
      return `${flag} ${code || '+90'} (${p1}) *** ** ${last2}`;
    }
    return phone;
  };

  return (
    <div className="space-y-6">
      
      {/* Financial & ROI Overview */}
      <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-6 gap-4">
        {/* Core Conversion Metrics */}
        <div className="bg-white border border-zinc-200/60 rounded-xl p-5 shadow-sm flex flex-col justify-between col-span-1 md:col-span-1 lg:col-span-2">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 bg-blue-50 text-blue-600 rounded-lg">
              <PieChart className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-xs font-bold text-zinc-900 uppercase tracking-wide">Dönüşüm Hunisi</h3>
              <p className="text-[10px] text-zinc-500 font-medium">Toplam lead ve randevular</p>
            </div>
          </div>
          <div className="flex items-end justify-between mt-auto">
            <div>
              <div className="text-[10px] text-zinc-400 font-semibold mb-1 uppercase tracking-wider">Lead</div>
              <div className="text-2xl font-bold font-mono text-zinc-900">{totalLeads.toLocaleString('tr-TR')}</div>
            </div>
            <div className="w-px h-8 bg-zinc-200"></div>
            <div className="text-right">
              <div className="text-[10px] text-zinc-400 font-semibold mb-1 uppercase tracking-wider">Randevu</div>
              <div className="text-2xl font-bold font-mono text-zinc-900">{totalAppointments.toLocaleString('tr-TR')}</div>
            </div>
          </div>
        </div>

        {/* Spend & Cost Metrics */}
        <div className="bg-white border border-zinc-200/60 rounded-xl p-5 shadow-sm flex flex-col justify-between col-span-1 md:col-span-2 lg:col-span-2">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 bg-rose-50 text-rose-600 rounded-lg">
              <Wallet className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-xs font-bold text-zinc-900 uppercase tracking-wide">Maliyet (Cost)</h3>
              <p className="text-[10px] text-zinc-500 font-medium">Harcama ve edinim bedelleri</p>
            </div>
          </div>
          <div className="grid grid-cols-3 gap-2 mt-auto">
            <div>
              <div className="text-[10px] text-zinc-400 font-semibold mb-1 uppercase tracking-wider">Harcama</div>
              <div className="text-lg font-bold font-mono text-zinc-900">₺{totalSpend.toLocaleString('tr-TR')}</div>
            </div>
            <div>
              <div className="text-[10px] text-zinc-400 font-semibold mb-1 uppercase tracking-wider">Ort. CPL</div>
              <div className="text-lg font-bold font-mono text-zinc-900">₺{Math.round(avgCpl).toLocaleString('tr-TR')}</div>
            </div>
            <div className="text-right">
              <div className="text-[10px] text-zinc-400 font-semibold mb-1 uppercase tracking-wider">Ort. CPA</div>
              <div className="text-lg font-bold font-mono text-zinc-900">₺{Math.round(avgCpa).toLocaleString('tr-TR')}</div>
            </div>
          </div>
        </div>

        {/* ROI / Revenue */}
        <div className="bg-zinc-950 border border-zinc-900 rounded-xl p-5 shadow-md flex flex-col justify-between col-span-1 md:col-span-3 lg:col-span-2 text-white relative overflow-hidden">
          {/* Decorative background element */}
          <div className="absolute -right-6 -top-6 w-24 h-24 bg-emerald-500/20 rounded-full blur-2xl"></div>
          
          <div className="flex items-center gap-3 mb-4 relative z-10">
            <div className="p-2 bg-emerald-500/20 text-emerald-400 rounded-lg border border-emerald-500/30">
              <TrendingUp className="w-5 h-5" />
            </div>
            <div>
              <h3 className="text-xs font-bold text-zinc-100 uppercase tracking-wide">Tahmini ROI</h3>
              <p className="text-[10px] text-zinc-400 font-medium">Beklenen ciro geri dönüşü</p>
            </div>
          </div>
          <div className="flex items-end justify-between mt-auto relative z-10">
            <div>
              <div className="text-[10px] text-zinc-400 font-semibold mb-1 uppercase tracking-wider">Beklenen Ciro</div>
              <div className="text-xl font-bold font-mono text-emerald-400">₺{estRevenue.toLocaleString('tr-TR')}</div>
            </div>
            <div className="text-right">
              <div className="text-[10px] text-zinc-400 font-semibold mb-1 uppercase tracking-wider">ROI</div>
              <div className="text-3xl font-bold font-mono text-white">%{Math.round(roi).toLocaleString('tr-TR')}</div>
            </div>
          </div>
        </div>
      </div>

      {/* Secondary Metrics Row */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
         <div className="bg-white border border-zinc-200/60 rounded-xl p-5 shadow-sm flex items-center justify-between">
          <div>
            <div className="flex items-center gap-2 mb-1">
              <Users className="w-4 h-4 text-indigo-500" />
              <span className="text-[11px] text-zinc-500 font-semibold uppercase tracking-wider">Aktif Performans Kotası</span>
            </div>
            <div className="text-sm font-medium text-zinc-400 mt-2">
              <span className="text-2xl font-bold font-mono text-zinc-900">{activeClinic.delivered}</span> / {activeClinic.guarantee}
            </div>
          </div>
          <div className="text-right">
            {activeClinic.delivered >= activeClinic.guarantee ? (
              <span className="inline-flex items-center gap-1 bg-emerald-50 text-emerald-700 border border-emerald-200 text-[10px] font-bold px-2 py-1 rounded-md uppercase">
                <CheckCircle2 className="w-3 h-3" /> Hedefe Ulaşıldı
              </span>
            ) : (
              <span className="inline-flex items-center gap-1 bg-indigo-50 text-indigo-700 border border-indigo-200 text-[10px] font-bold px-2 py-1 rounded-md uppercase">
                <BarChart3 className="w-3 h-3" /> İlerliyor
              </span>
            )}
          </div>
         </div>

         <div className="bg-white border border-zinc-200/60 rounded-xl p-5 shadow-sm flex items-center justify-between">
          <div>
            <div className="flex items-center gap-2 mb-1">
              <BrainCircuit className="w-4 h-4 text-purple-500" />
              <span className="text-[11px] text-zinc-500 font-semibold uppercase tracking-wider">AI Optimizasyon Sınırı (Gölge Fiyat)</span>
            </div>
            <div className="text-[11px] font-medium text-zinc-400 mt-2 max-w-[200px] leading-relaxed">
               Sistem tarafından belirlenen maksimum marjinal edinim katsayısı
            </div>
          </div>
          <div className="text-3xl font-bold font-mono text-zinc-900 bg-zinc-50 px-4 py-2 rounded-lg border border-zinc-100">
            λ = {activeClinic.shadowPrice.toFixed(2)}
          </div>
         </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 text-left">
        {/* Kisi Bazli Hatırlatmalar */}
        <div className="bg-white border border-zinc-200/60 rounded-xl shadow-sm flex flex-col overflow-hidden">
          <div className="px-5 py-4 border-b border-zinc-100 flex items-center justify-between bg-zinc-50/50">
            <h3 className="text-sm font-bold text-zinc-900 flex items-center gap-2">
              <AlertTriangle className="w-4 h-4 text-amber-500" />
              Kişi Bazlı Bekleyen İşlemler
            </h3>
            <span className="text-[10px] font-bold bg-amber-100 text-amber-800 px-2 py-0.5 rounded-md border border-amber-200/50 shadow-sm">
              {pendingConversations.length} Aksiyon Bekliyor
            </span>
          </div>
          <div className="p-0 flex-1 overflow-y-auto max-h-[400px]">
            <div className="flex flex-col divide-y divide-zinc-100">
              {pendingConversations.length > 0 ? pendingConversations.map(conv => (
                <div key={conv.id} className="p-4 hover:bg-zinc-50/80 transition-all flex flex-col sm:flex-row sm:items-center gap-4 group">
                  {/* Patient Profile Snapshot */}
                  <div className="flex items-center gap-3 w-full sm:w-[45%]">
                    <div className="w-9 h-9 rounded-full bg-zinc-100 border border-zinc-200 shadow-sm flex items-center justify-center text-zinc-600 text-xs font-bold shrink-0">
                      {getInitials(conv.name)}
                    </div>
                    <div className="min-w-0">
                      <p className="text-sm font-bold text-zinc-900 truncate">{conv.name}</p>
                      <div className="flex items-center gap-1.5 mt-0.5">
                         <div className={`w-1.5 h-1.5 rounded-full ${conv.status === 'Niteleniyor' ? 'bg-amber-400' : 'bg-red-400'}`} />
                         <span className="text-[10px] text-zinc-500 font-medium truncate">{conv.qualification.segment || 'Segment Yok'}</span>
                      </div>
                    </div>
                  </div>
                  
                  {/* Action Info */}
                  <div className="flex-1 flex items-center justify-between min-w-0">
                    <div className="min-w-0 pr-4">
                      <p className="text-[10px] font-bold text-zinc-400 uppercase tracking-wider mb-0.5">
                        {conv.status === 'Niteleniyor' ? 'Nitelendirme Bekliyor' : 'İletişim Koptu'}
                      </p>
                      <p className="text-xs text-zinc-600 truncate bg-zinc-100/50 px-2 py-1 rounded-md border border-zinc-100 inline-block max-w-full">
                        "{conv.lastMessage}"
                      </p>
                    </div>
                    <button 
                      onClick={() => onNavigateToChat(conv.id)}
                      className="shrink-0 w-8 h-8 flex items-center justify-center bg-white border border-zinc-200 text-zinc-500 hover:text-zinc-900 hover:bg-zinc-100 rounded-lg shadow-sm transition-all opacity-100 sm:opacity-0 sm:group-hover:opacity-100"
                      title="Sohbete Git"
                    >
                      <ArrowRight className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              )) : (
                <div className="p-8 text-center flex flex-col items-center justify-center text-zinc-400">
                  <CheckCircle2 className="w-8 h-8 text-zinc-200 mb-2" />
                  <span className="text-xs font-medium">Bekleyen kişi bazlı işlem bulunmuyor.</span>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Yaklaşan Randevular / Onaylı Kişiler */}
        <div className="bg-white border border-zinc-200/60 rounded-xl shadow-sm flex flex-col overflow-hidden">
          <div className="px-5 py-4 border-b border-zinc-100 flex items-center justify-between bg-zinc-50/50">
            <h3 className="text-sm font-bold text-zinc-900 flex items-center gap-2">
              <CalendarDays className="w-4 h-4 text-emerald-500" />
              Onaylanmış Hasta Profilleri
            </h3>
            <span className="text-[10px] font-bold bg-emerald-50 text-emerald-700 px-2 py-0.5 rounded-md border border-emerald-200/50">
              {bookedConversations.length} Yaklaşan
            </span>
          </div>
          <div className="p-0 flex-1 overflow-y-auto max-h-[400px]">
            <div className="flex flex-col divide-y divide-zinc-100">
              
              {bookedConversations.map(conv => (
                <div key={conv.id} className="p-4 hover:bg-zinc-50 transition-colors flex flex-col sm:flex-row sm:items-center justify-between gap-4 group">
                  
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-full bg-emerald-50 border border-emerald-100 flex items-center justify-center text-emerald-700 text-xs font-bold shrink-0">
                      {getInitials(conv.name)}
                    </div>
                    <div>
                      <p className="text-xs font-bold text-zinc-900 flex items-center gap-1.5">
                        {conv.name}
                        {conv.qualification.intentPct >= 90 && (
                          <span className="bg-zinc-900 text-white text-[8px] px-1.5 py-0.5 rounded-sm uppercase tracking-wider">Yüksek Niyet</span>
                        )}
                      </p>
                      <div className="text-[10px] text-zinc-500 flex items-center gap-2 mt-1">
                        <span className="flex items-center gap-1 font-mono text-zinc-600"><Clock className="w-3 h-3 text-emerald-500" /> {conv.qualification.appointmentTime}</span>
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center gap-4">
                     <div className="text-right hidden sm:block">
                       <div className="text-[9px] font-semibold text-zinc-400 uppercase">Tedavi</div>
                       <div className="text-xs font-bold text-zinc-700">{conv.qualification.segment}</div>
                     </div>
                     <button 
                       onClick={() => onNavigateToChat(conv.id)}
                       className="w-8 h-8 flex items-center justify-center rounded-md border border-zinc-200 text-zinc-500 hover:text-zinc-900 hover:bg-zinc-100 transition-colors shadow-sm opacity-0 group-hover:opacity-100"
                       title="Sohbete Git"
                     >
                       <ArrowRight className="w-3.5 h-3.5" />
                     </button>
                  </div>

                </div>
              ))}

              {bookedConversations.length === 0 && (
                <div className="p-8 text-center flex flex-col items-center justify-center text-zinc-400">
                  <CalendarDays className="w-8 h-8 text-zinc-200 mb-2" />
                  <span className="text-xs font-medium">Yakın zamanda planlanmış randevu yok.</span>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
