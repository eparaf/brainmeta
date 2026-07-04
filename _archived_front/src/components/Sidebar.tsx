import React, { useState } from 'react';
import { 
  MessageSquare, 
  Building2, 
  GitBranch, 
  Wallet, 
  FileText, 
  Link2, 
  Calendar, 
  ChevronDown, 
  X,
  Radio,
  LayoutDashboard,
  Users
} from 'lucide-react';
import { Clinic, Integration } from '../types';
import { WhatsAppIcon, MetaIcon, GoogleAdsIcon } from './Icons';

interface SidebarProps {
  clinics: Clinic[];
  activeClinic: Clinic;
  setActiveClinic: (clinic: Clinic) => void;
  activeTab: string;
  setActiveTab: (tab: string) => void;
  mobileOpen: boolean;
  setMobileOpen: (open: boolean) => void;
  integrations: Integration[];
}

export default function Sidebar({
  clinics,
  activeClinic,
  setActiveClinic,
  activeTab,
  setActiveTab,
  mobileOpen,
  setMobileOpen,
  integrations
}: SidebarProps) {
  const [dropdownOpen, setDropdownOpen] = useState(false);

  const navGroups = [
    {
      title: 'Platform',
      items: [
        { id: 'dashboard', label: 'Genel Bakış', icon: LayoutDashboard },
        { id: 'sohbetler', label: 'Sohbetler', icon: MessageSquare },
        { id: 'uyeler', label: 'Üyeler', icon: Users },
        { id: 'takvim', label: 'Takvim', icon: Calendar },
      ]
    },
    {
      title: 'Yönetim',
      items: [
        { id: 'klinikler', label: 'Klinikler', icon: Building2 },
        { id: 'reklam-kollari', label: 'Reklam Kolları', icon: GitBranch },
        { id: 'butce', label: 'Bütçe', icon: Wallet },
      ]
    },
    {
      title: 'Sistem',
      items: [
        { id: 'sablonlar', label: 'Şablonlar', icon: FileText },
        { id: 'baglantilar', label: 'Bağlantılar', icon: Link2 },
      ]
    }
  ];

  return (
    <>
      {/* Sidebar Container */}
      <aside className={`
        fixed inset-y-0 left-0 z-50 flex flex-col w-64 bg-zinc-950 border-r border-zinc-900/50 transition-transform duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]
        md:translate-x-0 md:static md:h-screen text-zinc-300 shadow-2xl md:shadow-none
        ${mobileOpen ? 'translate-x-0' : '-translate-x-full'}
      `}>
        {/* Logo Section */}
        <div className="flex items-center justify-between p-5 border-b border-zinc-900/50">
          <div className="flex items-center gap-3">
            {/* Modern gradient logo icon */}
            <div className="flex items-center justify-center w-9 h-9 rounded-xl bg-gradient-to-br from-zinc-100 to-zinc-400 text-zinc-950 shadow-md">
              <svg 
                className="w-5 h-5" 
                viewBox="0 0 24 24" 
                fill="none" 
                stroke="currentColor" 
                strokeWidth="2.5" 
                strokeLinecap="round" 
                strokeLinejoin="round"
              >
                <circle cx="12" cy="5" r="2.5" />
                <circle cx="6" cy="18" r="2.5" />
                <circle cx="18" cy="18" r="2.5" />
                <line x1="12" y1="7.5" x2="6" y2="15.5" />
                <line x1="12" y1="7.5" x2="18" y2="15.5" />
                <line x1="6" y1="18" x2="18" y2="18" />
              </svg>
            </div>
            <div>
              <div className="font-sans font-bold text-[18px] tracking-tight text-white leading-none">
                BrainMeta
              </div>
              <div className="text-[10px] text-zinc-500 font-bold tracking-widest uppercase mt-1">
                Karar Konsolu
              </div>
            </div>
          </div>
          
          {/* Close button on mobile */}
          <button 
            className="p-1.5 text-zinc-500 hover:text-white bg-zinc-900 hover:bg-zinc-800 rounded-md transition-colors md:hidden"
            onClick={() => setMobileOpen(false)}
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Clinic Switcher */}
        <div className="px-4 py-5 relative z-20">
          <button 
            onClick={() => setDropdownOpen(!dropdownOpen)}
            className="w-full flex items-center justify-between px-3 py-2.5 rounded-xl bg-zinc-900/60 border border-zinc-800/80 hover:bg-zinc-800 hover:border-zinc-700 transition-all duration-200 text-left group shadow-sm"
          >
            <div>
              <div className="text-[9px] font-bold text-zinc-500 uppercase tracking-widest mb-1 group-hover:text-zinc-400 transition-colors">Seçili Klinik</div>
              <div className="text-xs font-bold text-zinc-100 flex items-center gap-2">
                <Radio className={`w-3.5 h-3.5 ${activeClinic.status === 'on-track' ? 'text-emerald-400' : 'text-amber-400'}`} />
                {activeClinic.name}
              </div>
            </div>
            <div className="w-6 h-6 rounded-md bg-zinc-950 border border-zinc-800 flex items-center justify-center group-hover:bg-zinc-800 transition-colors">
              <ChevronDown className={`w-3.5 h-3.5 text-zinc-400 transition-transform duration-200 ${dropdownOpen ? 'rotate-180' : ''}`} />
            </div>
          </button>

          {/* Dropdown Menu */}
          {dropdownOpen && (
            <>
              <div 
                className="fixed inset-0 z-10" 
                onClick={() => setDropdownOpen(false)} 
              />
              <div className="absolute top-[calc(100%-8px)] left-4 right-4 bg-zinc-900 border border-zinc-800 rounded-xl shadow-2xl overflow-hidden z-20 py-1.5 backdrop-blur-xl bg-zinc-900/95 transform origin-top animate-in fade-in slide-in-from-top-2 duration-150">
                {clinics.map(clinic => (
                  <button
                    key={clinic.id}
                    onClick={() => {
                      setActiveClinic(clinic);
                      setDropdownOpen(false);
                    }}
                    className={`w-full text-left px-4 py-2.5 text-xs flex items-center justify-between transition-colors
                      ${activeClinic.id === clinic.id ? 'bg-zinc-800/80 text-white font-bold' : 'text-zinc-400 font-medium hover:bg-zinc-800/50 hover:text-zinc-200'}
                    `}
                  >
                    <div className="flex items-center gap-2.5">
                      <Radio className={`w-3.5 h-3.5 ${clinic.status === 'on-track' ? 'text-emerald-400' : 'text-amber-400'}`} />
                      {clinic.name}
                    </div>
                  </button>
                ))}
              </div>
            </>
          )}
        </div>

        {/* Navigation List */}
        <div className="flex-1 overflow-y-auto px-3 space-y-6 scrollbar-none pb-4">
          {navGroups.map((group, idx) => (
            <div key={idx}>
              <h4 className="px-3 mb-2.5 text-[10px] font-bold uppercase tracking-[0.2em] text-zinc-600">
                {group.title}
              </h4>
              <div className="space-y-1">
                {group.items.map(item => {
                  const isActive = activeTab === item.id;
                  const IconComponent = item.icon;
                  return (
                    <button
                      key={item.id}
                      onClick={() => {
                        setActiveTab(item.id);
                        setMobileOpen(false);
                      }}
                      className={`
                        w-full flex items-center gap-3 px-3 py-2 rounded-lg text-xs font-semibold transition-all duration-200 group
                        ${isActive 
                          ? 'bg-zinc-800/80 text-white shadow-sm ring-1 ring-zinc-700/50' 
                          : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200'
                        }
                      `}
                    >
                      <IconComponent className={`w-4 h-4 transition-colors ${isActive ? 'text-white' : 'text-zinc-500 group-hover:text-zinc-400'}`} />
                      {item.label}
                    </button>
                  );
                })}
              </div>
            </div>
          ))}
        </div>

        {/* Integrations Status */}
        <div className="px-4 py-3 mx-3 mb-3 bg-zinc-900/40 border border-zinc-800/50 rounded-xl">
           <h4 className="text-[9px] font-bold uppercase tracking-widest text-zinc-500 mb-2">Bağlantılar</h4>
           <div className="flex items-center gap-2.5">
             <div className="relative group">
                <div className={`w-6 h-6 rounded-md flex items-center justify-center transition-colors ${integrations.find(i => i.type === 'whatsapp')?.connected ? 'bg-[#25D366]/10 text-[#25D366] border border-[#25D366]/20' : 'bg-zinc-800 text-zinc-600'}`}>
                  <WhatsAppIcon className="w-3.5 h-3.5" />
                </div>
             </div>
             <div className="relative group">
                <div className={`w-6 h-6 rounded-md flex items-center justify-center transition-colors ${integrations.find(i => i.type === 'meta_ads')?.connected ? 'bg-[#0668E1]/10 text-[#0668E1] border border-[#0668E1]/20' : 'bg-zinc-800 text-zinc-600'}`}>
                  <MetaIcon className="w-3.5 h-3.5" />
                </div>
             </div>
             <div className="relative group">
                <div className={`w-6 h-6 rounded-md flex items-center justify-center transition-colors ${integrations.find(i => i.type === 'google_ads')?.connected ? 'bg-amber-500/10 text-amber-500 border border-amber-500/20' : 'bg-zinc-800 text-zinc-600'}`}>
                  <GoogleAdsIcon className="w-3.5 h-3.5" />
                </div>
             </div>
           </div>
        </div>

        {/* User / Settings Footer */}
        <div className="p-4 border-t border-zinc-900/50 bg-zinc-950 mt-auto">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-zinc-800 to-zinc-700 border border-zinc-600/30 flex items-center justify-center text-xs font-bold text-zinc-200 shadow-sm">
              AD
            </div>
            <div>
              <div className="text-xs font-bold text-zinc-200">Admin User</div>
              <div className="text-[10px] text-zinc-500 font-medium">Sistem Yöneticisi</div>
            </div>
          </div>
        </div>
      </aside>
      
      {/* Mobile Drawer Overlay */}
      {mobileOpen && (
        <div 
          className="fixed inset-0 bg-black/60 backdrop-blur-sm z-40 md:hidden transition-opacity" 
          onClick={() => setMobileOpen(false)}
        />
      )}
    </>
  );
}
