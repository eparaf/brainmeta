import React, { useState } from 'react';
import { 
  Link2, 
  CheckCircle2, 
  XCircle, 
  Globe
} from 'lucide-react';
import { Integration } from '../types';
import { WhatsAppIcon, MetaIcon, GoogleAdsIcon } from './Icons';

interface BaglantilarPageProps {
  integrations: Integration[];
  setIntegrations: React.Dispatch<React.SetStateAction<Integration[]>>;
}

export default function BaglantilarPage({
  integrations,
  setIntegrations
}: BaglantilarPageProps) {
  const [toast, setToast] = useState<string | null>(null);

  // Toggle Connection Handler
  const handleToggleConnect = (id: string, name: string, isCurrentlyConnected: boolean) => {
    setIntegrations(prev => prev.map(integration => {
      if (integration.id === id) {
        return {
          ...integration,
          connected: !integration.connected
        };
      }
      return integration;
    }));

    if (isCurrentlyConnected) {
      setToast(`${name} entegrasyon bağlantısı başarıyla kesildi.`);
    } else {
      setToast(`${name} entegrasyonu başarıyla kuruldu ve senkronize edildi.`);
    }

    setTimeout(() => setToast(null), 3500);
  };

  // Helper to get integration visual icon
  const getIntegrationIcon = (type: Integration['type']) => {
    switch (type) {
      case 'whatsapp':
        return <WhatsAppIcon className="w-5 h-5 text-[#25D366]" />;
      case 'meta_ads':
        return <MetaIcon className="w-5 h-5 text-[#0668E1]" />;
      case 'google_ads':
        return <GoogleAdsIcon className="w-5 h-5 text-amber-500" />;
      case 'web_form':
        return <Globe className="w-5 h-5 text-zinc-600" />;
    }
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h2 className="text-xs font-bold text-zinc-900 tracking-tight">Klinik Veri ve Kanal Bağlantıları</h2>
        <p className="text-[11px] text-zinc-500 mt-1">
          Yapay zekâ karar konsolunun Thompson bütçe ve WhatsApp takip motorlarını çalıştırmak için entegrasyon durumlarını yönetin.
        </p>
      </div>

      {/* Toast Notification */}
      {toast && (
        <div className="p-3 bg-zinc-900 text-white text-[11px] rounded font-mono font-medium animate-in fade-in duration-200">
          ✓ {toast}
        </div>
      )}

      {/* Integrations Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
        {integrations.map((item) => (
          <div 
            key={item.id}
            className="bg-white border border-zinc-200/80 rounded-lg p-5 shadow-sm hover:border-zinc-300 transition-all flex flex-col justify-between text-left"
          >
            <div>
              {/* Header: Icon + Connected pill */}
              <div className="flex items-center justify-between mb-4">
                <div className="p-2 bg-zinc-50 border border-zinc-100 rounded-md">
                  {getIntegrationIcon(item.type)}
                </div>

                {item.connected ? (
                  <span className="inline-flex items-center gap-1 text-[10px] font-bold text-emerald-700 bg-emerald-50 border border-emerald-200/50 px-2.5 py-0.5 rounded-full">
                    <CheckCircle2 className="w-3.5 h-3.5 text-emerald-500" /> Bağlı
                  </span>
                ) : (
                  <span className="inline-flex items-center gap-1 text-[10px] font-bold text-zinc-500 bg-zinc-100 border border-zinc-200/85 px-2.5 py-0.5 rounded-full">
                    <XCircle className="w-3.5 h-3.5 text-zinc-400" /> Bağlı Değil
                  </span>
                )}
              </div>

              {/* Title & Description */}
              <div className="mb-5">
                <h3 className="text-xs font-bold text-zinc-900 mb-1">
                  {item.name}
                </h3>
                <p className="text-[11px] text-zinc-500 leading-relaxed">
                  {item.description}
                </p>
              </div>
            </div>

            {/* Connect Action Button */}
            <div className="border-t border-zinc-100 pt-4.5 flex items-center justify-between">
              <span className="text-[10px] text-zinc-400 font-mono">
                Sistem ID: {item.id}
              </span>

              <button
                onClick={() => handleToggleConnect(item.id, item.name, item.connected)}
                className={`
                  px-4.5 py-1.5 rounded text-[11px] font-bold tracking-tight transition-all duration-150
                  ${item.connected 
                    ? 'bg-white border border-zinc-200 text-zinc-700 hover:bg-zinc-50' 
                    : 'bg-zinc-900 text-white hover:bg-zinc-800'
                  }
                `}
              >
                {item.connected ? 'Bağlantıyı Kes' : 'Bağla'}
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
