import React, { useState } from 'react';
import { Menu } from 'lucide-react';
import Sidebar from './components/Sidebar';
import DashboardPage from './components/DashboardPage';
import SohbetlerPage from './components/SohbetlerPage';
import UyelerPage from './components/UyelerPage';
import KliniklerPage from './components/KliniklerPage';
import ReklamKollariPage from './components/ReklamKollariPage';
import ButcePage from './components/ButcePage';
import SablonlarPage from './components/SablonlarPage';
import BaglantilarPage from './components/BaglantilarPage';
import TakvimPage from './components/TakvimPage';

import { 
  mockClinics, 
  mockAdArms, 
  mockConversations, 
  mockTemplates, 
  mockIntegrations 
} from './mockData';
import { Clinic, AdArm, Conversation, Template, Integration } from './types';

export default function App() {
  // Global Lifted States to preserve data across page transitions
  const [clinics, setClinics] = useState<Clinic[]>(mockClinics);
  const [activeClinic, setActiveClinic] = useState<Clinic>(mockClinics[0]);
  
  const [conversations, setConversations] = useState<Record<string, Conversation[]>>(mockConversations);
  const [adArms, setAdArms] = useState<Record<string, AdArm[]>>(mockAdArms);
  const [templates, setTemplates] = useState<Template[]>(mockTemplates);
  const [integrations, setIntegrations] = useState<Integration[]>(mockIntegrations);

  const [activeTab, setActiveTab] = useState('dashboard');
  const [mobileOpen, setMobileOpen] = useState(false);
  const [selectedChatId, setSelectedChatId] = useState<string | null>(null);

  const navigateToChat = (convId: string) => {
    setSelectedChatId(convId);
    setActiveTab('sohbetler');
  };

  // Helper to render active page dynamically
  const renderActivePage = () => {
    switch (activeTab) {
      case 'dashboard':
        return (
          <DashboardPage
            activeClinic={activeClinic}
            conversations={conversations[activeClinic.id] || []}
            adArms={adArms[activeClinic.id] || []}
            onNavigateToChat={navigateToChat}
          />
        );
      case 'sohbetler':
        return (
          <SohbetlerPage
            conversations={conversations[activeClinic.id] || []}
            setConversations={setConversations}
            activeClinicId={activeClinic.id}
            selectedChatId={selectedChatId}
            setSelectedChatId={setSelectedChatId}
          />
        );
      case 'klinikler':
        return (
          <KliniklerPage
            clinics={clinics}
            setClinics={setClinics}
          />
        );
      case 'reklam-kollari':
        return (
          <ReklamKollariPage
            adArms={adArms[activeClinic.id] || []}
            setAdArms={setAdArms}
            activeClinicId={activeClinic.id}
          />
        );
      case 'butce':
        return (
          <ButcePage
            activeClinic={activeClinic}
            adArms={adArms[activeClinic.id] || []}
          />
        );
      case 'sablonlar':
        return (
          <SablonlarPage
            templates={templates}
            setTemplates={setTemplates}
          />
        );
      case 'baglantilar':
        return (
          <BaglantilarPage
            integrations={integrations}
            setIntegrations={setIntegrations}
          />
        );
      case 'takvim':
        return (
          <TakvimPage
            conversations={conversations[activeClinic.id] || []}
            onNavigateToChat={navigateToChat}
          />
        );
      case 'uyeler':
        return (
          <UyelerPage 
            conversations={conversations[activeClinic.id] || []}
            onNavigateToChat={(chatId) => {
              setSelectedChatId(chatId);
              setActiveTab('sohbetler');
            }}
          />
        );
      default:
        return (
          <div className="p-8 text-center text-zinc-500 text-xs">
            Geliştirme Aşamasında
          </div>
        );
    }
  };

  // Dynamic titles and subtitles based on the selected tab
  const getPageHeaderInfo = () => {
    switch (activeTab) {
      case 'dashboard':
        return {
          title: 'Sistem Özeti',
          subtitle: 'Klinik performansınız, bekleyen aksiyonlar ve Thompson algoritması özet durumu.'
        };
      case 'sohbetler':
        return {
          title: 'Nitelikli WhatsApp Sohbetleri',
          subtitle: 'Gelen hasta başvurularının otonom analizi ve anlık müdahale arayüzü.'
        };
      case 'uyeler':
        return {
          title: 'Üye / Hasta Takip Kuyruğu',
          subtitle: 'Tüm başvuruların anlık durumu, randevu takibi ve segmentasyon özeti.'
        };
      case 'klinikler':
        return {
          title: 'Klinik Taahhüt Matrisi',
          subtitle: 'Kliniklerin aylık garanti hedefleri, güncel gerçekleşenler ve gölge fiyat optimizasyonu.'
        };
      case 'reklam-kollari':
        return {
          title: 'Reklam Kolları Dağılım Analizi',
          subtitle: 'Thompson sampling algoritmasıyla öğrenen segmenter ve performans olasılıkları.'
        };
      case 'butce':
        return {
          title: 'Akıllı Bütçe Yönetimi',
          subtitle: 'Gölge fiyat (λ) katsayısına bağlı olarak günlük bütçenin otonom dağılım simülasyonu.'
        };
      case 'sablonlar':
        return {
          title: 'WhatsApp Mesaj Şablonları',
          subtitle: 'Meta onaylı, değişken parametreli resmî bildirim ve pazarlama şablonları.'
        };
      case 'baglantilar':
        return {
          title: 'Bağlantılar ve Entegrasyonlar',
          subtitle: 'WhatsApp Cloud API, Meta Ads, Google Ads ve Klinik Web Siteleri anlık veri akışı.'
        };
      case 'takvim':
        return {
          title: 'Klinik Takvimi',
          subtitle: 'Yapay zekâ tarafından nitelendirilip onaylanmış hasta randevuları.'
        };
      default:
        return {
          title: 'BrainMeta Konsolu',
          subtitle: 'Klinikler için otonom yapay zekâ karar ve yönetim paneli.'
        };
    }
  };

  const headerInfo = getPageHeaderInfo();

  return (
    <div id="app-root" className="min-h-screen flex bg-zinc-50 text-zinc-900 font-sans antialiased">
      {/* Sidebar Component */}
      <Sidebar
        clinics={clinics}
        activeClinic={activeClinic}
        setActiveClinic={setActiveClinic}
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        mobileOpen={mobileOpen}
        setMobileOpen={setMobileOpen}
        integrations={integrations}
      />

      {/* Main Content Pane */}
      <div className="flex-1 flex flex-col min-w-0 h-screen overflow-hidden">
        {/* Mobile Header Bar */}
        <header className="flex items-center justify-between px-4 py-3 bg-white border-b border-zinc-200/80 md:hidden shrink-0">
          <div className="flex items-center gap-2.5">
            <button
              onClick={() => setMobileOpen(true)}
              className="p-1.5 text-zinc-600 hover:text-zinc-900 bg-zinc-50 border border-zinc-200 rounded"
            >
              <Menu className="w-5 h-5" />
            </button>
            <span className="font-sans font-bold text-sm tracking-tight text-zinc-900">
              BrainMeta
            </span>
          </div>

          <div className="flex items-center gap-1.5 bg-zinc-50 border border-zinc-200 rounded-full px-2.5 py-1 text-[10px] font-semibold text-zinc-700">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
            <span className="truncate max-w-[120px]">{activeClinic.name}</span>
          </div>
        </header>

        {/* Workspace Scroller */}
        <div className="flex-1 overflow-y-auto bg-zinc-50/50 flex flex-col items-center">
          <div className="w-full h-full flex flex-col p-4 sm:p-6 lg:p-8 max-w-7xl mx-auto">
            {/* Section Header */}
            <div className="flex-none flex items-end justify-between w-full pb-5 border-b border-zinc-200/80 mb-6 shrink-0">
              <div className="text-left">
                <h1 className="text-2xl sm:text-3xl font-bold text-zinc-950 tracking-tight">
                  {headerInfo.title}
                </h1>
                <p className="text-sm text-zinc-500 font-medium mt-2">
                  {headerInfo.subtitle}
                </p>
              </div>
            </div>

            {/* Active Workspace */}
            <div className="flex-1 w-full min-h-0">
              {renderActivePage()}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
