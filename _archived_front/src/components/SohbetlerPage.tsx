import React, { useState, useMemo, useEffect, useRef } from 'react';
import { 
  Search, 
  Plus, 
  Send, 
  BrainCircuit, 
  CalendarDays, 
  CheckCircle2, 
  X, 
  Clock, 
  HeartPulse, 
  AlertTriangle,
  ArrowRight,
  ChevronLeft,
  MessageSquare,
  MoreVertical,
  PhoneCall,
  UserCircle2,
  DollarSign
} from 'lucide-react';
import { Conversation, Message, Qualification } from '../types';

interface SohbetlerPageProps {
  conversations: Conversation[];
  setConversations: React.Dispatch<React.SetStateAction<Record<string, Conversation[]>>>;
  activeClinicId: string;
  selectedChatId: string | null;
  setSelectedChatId: (id: string | null) => void;
}

export default function SohbetlerPage({
  conversations,
  setConversations,
  activeClinicId,
  selectedChatId,
  setSelectedChatId
}: SohbetlerPageProps) {
  const [searchTerm, setSearchTerm] = useState('');
  const [messageText, setMessageText] = useState('');
  
  // Guided Intake Wizard States
  const [showNewModal, setShowNewModal] = useState(false);
  const [intakeStep, setIntakeStep] = useState(1);
  const [newPatientName, setNewPatientName] = useState('');
  const [selectedTedavi, setSelectedTedavi] = useState('');
  const [selectedAgri, setSelectedAgri] = useState('');
  const [selectedButce, setSelectedButce] = useState('');
  const [selectedZaman, setSelectedZaman] = useState('');

  // Mobile navigation
  const [mobileShowThread, setMobileShowThread] = useState(false);

  // Auto-scroll
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Derive conversations for this clinic (already filtered by parent)
  const clinicConversations = conversations || [];

  // Filter conversations based on search
  const filteredConversations = useMemo(() => {
    if (!searchTerm.trim()) return clinicConversations;
    const term = searchTerm.toLowerCase();
    return clinicConversations.filter(c => 
      c.name.toLowerCase().includes(term) || 
      c.lastMessage.toLowerCase().includes(term) ||
      (c.qualification?.segment || '').toLowerCase().includes(term)
    );
  }, [clinicConversations, searchTerm]);

  // Get active conversation
  const activeConversation = useMemo(() => {
    if (selectedChatId) {
      const found = clinicConversations.find(c => c.id === selectedChatId);
      if (found) return found;
    }
    if (clinicConversations.length > 0) {
      return clinicConversations[0];
    }
    return null;
  }, [clinicConversations, selectedChatId]);

  // Initialize selection
  useEffect(() => {
    if (clinicConversations.length > 0) {
      const stillExists = clinicConversations.some(c => c.id === selectedChatId);
      if (!stillExists) {
        setSelectedChatId(clinicConversations[0].id);
      }
    } else {
      setSelectedChatId(null);
    }
  }, [activeClinicId, clinicConversations, selectedChatId]);

  // Scroll to bottom when active conversation changes or new messages arrive
  useEffect(() => {
    if (messagesEndRef.current) {
      messagesEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [activeConversation?.messages]);

  const handleSendMessage = (e: React.FormEvent) => {
    e.preventDefault();
    if (!messageText.trim() || !activeConversation) return;

    const newMsg: Message = {
      id: 'm-new-' + Date.now(),
      sender: 'agent',
      text: messageText,
      timestamp: new Date().toLocaleTimeString('tr-TR', { hour: '2-digit', minute: '2-digit' })
    };

    setConversations(prev => {
      const clinicList = prev[activeClinicId] || [];
      const updatedList = clinicList.map(c => {
        if (c.id === activeConversation.id) {
          return {
            ...c,
            lastMessage: messageText,
            lastMessageTime: newMsg.timestamp,
            messages: [...c.messages, newMsg]
          };
        }
        return c;
      });
      return {
        ...prev,
        [activeClinicId]: updatedList
      };
    });

    setMessageText('');
  };

  const resetIntake = () => {
    setIntakeStep(1);
    setNewPatientName('');
    setSelectedTedavi('');
    setSelectedAgri('');
    setSelectedButce('');
    setSelectedZaman('');
  };

  const handleConfirmIntake = () => {
    if (!newPatientName.trim()) return;

    let urgencyValue: Qualification['urgency'] = 'Orta';
    if (selectedAgri.includes('Çok şiddetli')) urgencyValue = 'Yüksek';
    else if (selectedAgri.includes('Ağrı yok')) urgencyValue = 'Düşük';

    const calculatedIntent = 80 + Math.floor(Math.random() * 20);
    
    const newConv: Conversation = {
      id: 'c-new-' + Date.now(),
      name: newPatientName,
      clinicId: activeClinicId,
      lastMessage: 'Kayıt ve randevu talebi oluşturuldu.',
      lastMessageTime: new Date().toLocaleTimeString('tr-TR', { hour: '2-digit', minute: '2-digit' }),
      status: 'Randevu',
      messages: [
        { id: 'm1', sender: 'patient', text: `Yeni Kayıt Başvurusu: ${selectedTedavi} için bilgi almak istiyorum.`, timestamp: 'Şimdi' },
        { id: 'm2', sender: 'agent', text: `Sistem Kaydı: Tedavi=${selectedTedavi}, Ağrı=${selectedAgri}, Bütçe=${selectedButce}, Zaman=${selectedZaman}. Temsilcilerimiz sizinle en kısa sürede iletişime geçecektir.`, timestamp: 'Şimdi' }
      ],
      qualification: {
        segment: selectedTedavi,
        intentPct: calculatedIntent,
        urgency: urgencyValue,
        budget: selectedButce,
        language: 'TR',
        booked: true,
        appointmentTime: `Seçilen zaman: ${selectedZaman}`
      }
    };

    setConversations(prev => ({
      ...prev,
      [activeClinicId]: [newConv, ...(prev[activeClinicId] || [])]
    }));

    setSelectedChatId(newConv.id);
    setShowNewModal(false);
    resetIntake();
  };

  const getInitials = (name: string) => {
    return name.split(' ').map(n => n[0]).join('').toUpperCase().substring(0, 2);
  };

  const getStatusColor = (status: Conversation['status']) => {
    switch(status) {
      case 'Randevu': return 'bg-emerald-500';
      case 'Niteleniyor': return 'bg-amber-400';
      case 'Düştü': return 'bg-red-500';
      case 'Gelmedi': return 'bg-rose-600';
      default: return 'bg-zinc-400';
    }
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
    <div className="flex flex-col w-full h-[calc(100vh-14rem)] md:h-[calc(100vh-12rem)] bg-white border border-zinc-200/80 rounded-xl overflow-hidden shadow-sm shadow-black/5">
      <div className="flex flex-1 h-full overflow-hidden">
        
        {/* LEFT PANEL: CONVERSATIONS LIST */}
        <div className={`w-full md:w-80 border-r border-zinc-200/80 bg-zinc-50/30 flex flex-col shrink-0 ${mobileShowThread ? 'hidden md:flex' : 'flex'}`}>
          {/* Header & Search */}
          <div className="p-4 border-b border-zinc-100 bg-white">
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-sm font-bold text-zinc-900">Sohbetler</h2>
              <button 
                onClick={() => setShowNewModal(true)}
                className="w-7 h-7 flex items-center justify-center bg-zinc-900 text-white rounded-md hover:bg-zinc-800 transition-colors shadow-sm"
              >
                <Plus className="w-4 h-4" />
              </button>
            </div>
            <div className="relative">
              <Search className="w-4 h-4 text-zinc-400 absolute left-2.5 top-1/2 -translate-y-1/2" />
              <input 
                type="text" 
                placeholder="Hasta ara..." 
                value={searchTerm}
                onChange={e => setSearchTerm(e.target.value)}
                className="w-full bg-zinc-100 border-none rounded-md pl-9 pr-3 py-1.5 text-xs font-medium focus:outline-none focus:ring-1 focus:ring-zinc-300 placeholder:text-zinc-400"
              />
            </div>
          </div>

          {/* List */}
          <div className="flex-1 overflow-y-auto">
            {filteredConversations.length > 0 ? (
              <div className="flex flex-col">
                {filteredConversations.map(conv => {
                  const isActive = conv.id === selectedChatId;
                  return (
                    <button
                      key={conv.id}
                      onClick={() => {
                        setSelectedChatId(conv.id);
                        setMobileShowThread(true);
                      }}
                      className={`w-full text-left p-4 border-b border-zinc-100 transition-all ${
                        isActive 
                        ? 'bg-white shadow-[inset_3px_0_0_0_#18181b]' 
                        : 'hover:bg-white/60'
                      }`}
                    >
                      <div className="flex items-center gap-3">
                        {/* Avatar */}
                        <div className="relative shrink-0">
                          <div className={`w-11 h-11 rounded-full flex items-center justify-center text-xs font-bold border
                            ${isActive ? 'bg-zinc-900 text-white border-zinc-900' : 'bg-zinc-100 text-zinc-600 border-zinc-200'}
                          `}>
                            {getInitials(conv.name)}
                          </div>
                          <div className={`absolute bottom-0 right-0 w-3 h-3 rounded-full border-2 border-white ${getStatusColor(conv.status)}`} />
                        </div>
                        
                        {/* Info */}
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center justify-between mb-0.5">
                            <span className="text-xs font-bold text-zinc-900 truncate pr-2">{conv.name}</span>
                            <span className="text-[10px] text-zinc-400 font-mono whitespace-nowrap">{conv.lastMessageTime}</span>
                          </div>
                          <p className="text-[10px] text-zinc-400 font-mono mb-1 truncate flex items-center gap-1">
                            <PhoneCall className="w-2.5 h-2.5" />
                            {maskPhoneNumber(conv.phoneNumber, conv.countryCode)}
                          </p>
                          <p className={`text-[11px] truncate mb-1 ${isActive ? 'text-zinc-700 font-medium' : 'text-zinc-500'}`}>
                            {conv.lastMessage}
                          </p>
                          <div className="flex items-center gap-1">
                            <span className="text-[9px] font-bold uppercase tracking-wider text-zinc-400 border border-zinc-200 rounded px-1.5 py-0.5 bg-zinc-50">
                              {conv.qualification.segment}
                            </span>
                            {conv.status === 'Randevu' && (
                              <CheckCircle2 className="w-3 h-3 text-emerald-500" />
                            )}
                          </div>
                        </div>
                      </div>
                    </button>
                  );
                })}
              </div>
            ) : (
              <div className="p-8 text-center text-zinc-400 text-xs flex flex-col gap-2">
                <span>Sonuç bulunamadı.</span>
              </div>
            )}
          </div>
        </div>

        {/* RIGHT PANEL: CHAT VIEW */}
        <div className={`flex-1 flex flex-col bg-zinc-50/50 relative ${!mobileShowThread ? 'hidden md:flex' : 'flex'}`}>
          {activeConversation ? (
            <>
              {/* Chat Header */}
              <div className="h-16 px-4 bg-white border-b border-zinc-200/80 flex items-center justify-between shrink-0 shadow-sm z-10">
                <div className="flex items-center gap-3">
                  <button 
                    onClick={() => setMobileShowThread(false)}
                    className="md:hidden p-1 -ml-1 mr-1 text-zinc-500 hover:text-zinc-900"
                  >
                    <ChevronLeft className="w-5 h-5" />
                  </button>
                  
                  <div className="w-10 h-10 rounded-full bg-zinc-100 border border-zinc-200 flex items-center justify-center text-zinc-700 text-xs font-bold shrink-0">
                    {getInitials(activeConversation.name)}
                  </div>
                  <div>
                    <h3 className="text-sm font-bold text-zinc-900 flex items-center gap-2">
                      {activeConversation.name}
                      {activeConversation.status === 'Randevu' && (
                        <span className="bg-emerald-50 text-emerald-700 border border-emerald-200/50 px-1.5 py-0.5 rounded text-[9px] font-bold uppercase tracking-widest flex items-center gap-1">
                          <CheckCircle2 className="w-2.5 h-2.5" /> Randevu Alındı
                        </span>
                      )}
                    </h3>
                    <p className="text-[11px] font-medium text-zinc-500 flex items-center gap-1.5 mt-0.5">
                      <PhoneCall className="w-3 h-3 text-emerald-500" /> 
                      {activeConversation.phoneNumber 
                        ? <span className="font-mono text-zinc-700">{activeConversation.countryCode} {activeConversation.phoneNumber}</span>
                        : "WhatsApp'tan bağlı"
                      }
                    </p>
                  </div>
                </div>
                
                <div className="flex items-center gap-2">
                  <button className="w-8 h-8 flex items-center justify-center text-zinc-500 hover:bg-zinc-100 hover:text-zinc-900 rounded-md transition-colors">
                    <UserCircle2 className="w-5 h-5" />
                  </button>
                  <button className="w-8 h-8 flex items-center justify-center text-zinc-500 hover:bg-zinc-100 hover:text-zinc-900 rounded-md transition-colors">
                    <MoreVertical className="w-5 h-5" />
                  </button>
                </div>
              </div>

              {/* Patient CRM Strip (Brain's Perspective) */}
              <div className="bg-white px-4 py-2.5 border-b border-zinc-200/80 flex items-center justify-between text-xs overflow-x-auto whitespace-nowrap scrollbar-hide">
                <div className="flex items-center gap-4 text-zinc-600">
                  <div className="flex items-center gap-1.5 font-medium">
                    <BrainCircuit className="w-3.5 h-3.5 text-indigo-500" />
                    <span>Niyet Skoru:</span>
                    <span className="font-bold text-zinc-900">%{activeConversation.qualification.intentPct}</span>
                  </div>
                  <div className="w-px h-3 bg-zinc-200" />
                  <div className="flex items-center gap-1.5 font-medium">
                    <HeartPulse className="w-3.5 h-3.5 text-rose-400" />
                    <span>Aciliyet:</span>
                    <span className={`font-bold ${
                      activeConversation.qualification.urgency === 'Yüksek' ? 'text-rose-600' : 
                      activeConversation.qualification.urgency === 'Orta' ? 'text-amber-600' : 'text-emerald-600'
                    }`}>
                      {activeConversation.qualification.urgency}
                    </span>
                  </div>
                  <div className="w-px h-3 bg-zinc-200" />
                  <div className="flex items-center gap-1.5 font-medium">
                    <DollarSign className="w-3.5 h-3.5 text-zinc-400" />
                    <span>Bütçe:</span>
                    <span className="font-bold text-zinc-900">{activeConversation.qualification.budget}</span>
                  </div>
                </div>
                
                {activeConversation.status === 'Niteleniyor' && (
                  <button className="ml-4 bg-amber-100 text-amber-800 border border-amber-200 px-3 py-1 rounded text-[10px] font-bold uppercase tracking-wider flex items-center gap-1 shrink-0 hover:bg-amber-200 transition-colors">
                    <AlertTriangle className="w-3 h-3" /> Aksiyon Bekleniyor
                  </button>
                )}
              </div>

              {/* Messages Area */}
              <div className="flex-1 overflow-y-auto p-4 space-y-4">
                <div className="text-center pb-2">
                  <span className="bg-zinc-200/50 text-zinc-500 text-[10px] font-bold uppercase tracking-widest px-3 py-1 rounded-full">
                    Sohbet Başlangıcı
                  </span>
                </div>
                
                {activeConversation.messages.map(msg => {
                  const isAgent = msg.sender === 'agent';
                  return (
                    <div key={msg.id} className={`flex flex-col ${isAgent ? 'items-end' : 'items-start'}`}>
                      <div className={`max-w-[85%] md:max-w-[75%] px-4 py-2.5 text-[13px] shadow-sm leading-relaxed
                        ${isAgent 
                          ? 'bg-zinc-900 text-white rounded-2xl rounded-tr-sm' 
                          : 'bg-white text-zinc-900 border border-zinc-200 rounded-2xl rounded-tl-sm'
                        }
                      `}>
                        {msg.text}
                      </div>
                      <span className="text-[10px] text-zinc-400 mt-1.5 font-mono px-1 flex items-center gap-1">
                        {msg.timestamp}
                        {isAgent && <CheckCircle2 className="w-3 h-3 text-emerald-500 ml-0.5" />}
                      </span>
                    </div>
                  );
                })}
                <div ref={messagesEndRef} />
              </div>

              {/* Composer */}
              <div className="p-4 bg-white border-t border-zinc-200/80">
                <form 
                  onSubmit={handleSendMessage} 
                  className="flex items-center gap-2 bg-zinc-50 border border-zinc-200 rounded-lg p-1.5 focus-within:ring-1 focus-within:ring-zinc-400 focus-within:border-zinc-400 transition-all shadow-inner"
                >
                  <input
                    type="text"
                    placeholder="Hastaya cevap yazın..."
                    value={messageText}
                    onChange={(e) => setMessageText(e.target.value)}
                    className="flex-1 bg-transparent border-none px-3 py-2 text-xs font-medium focus:outline-none text-zinc-900 placeholder:text-zinc-400"
                  />
                  <button
                    type="submit"
                    disabled={!messageText.trim()}
                    className="bg-zinc-900 hover:bg-zinc-800 disabled:opacity-50 disabled:hover:bg-zinc-900 text-white rounded-md p-2.5 transition-colors flex items-center justify-center shrink-0 shadow-sm"
                  >
                    <Send className="w-4 h-4" />
                  </button>
                </form>
              </div>
            </>
          ) : (
            <div className="flex-1 flex flex-col items-center justify-center text-center p-8 bg-zinc-50/50">
              <div className="w-16 h-16 bg-white border border-zinc-200 rounded-full flex items-center justify-center text-zinc-300 mb-4 shadow-sm">
                <MessageSquare className="w-8 h-8" />
              </div>
              <h3 className="text-sm font-bold text-zinc-900 mb-1.5">Sohbet Seçilmedi</h3>
              <p className="text-xs text-zinc-500 max-w-xs leading-relaxed">
                Hasta profillerini görmek ve mesajlaşmak için sol taraftaki listeden bir konuşma seçin.
              </p>
            </div>
          )}
        </div>
      </div>

      {/* Guided Intake Modal */}
      {showNewModal && (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/40 backdrop-blur-sm p-4 animate-in fade-in duration-200">
          <div className="bg-white rounded-xl shadow-2xl w-full max-w-md overflow-hidden animate-in zoom-in-95 duration-200">
            {/* Header */}
            <div className="px-5 py-4 border-b border-zinc-100 flex items-center justify-between bg-zinc-50/80">
              <div className="flex items-center gap-2.5">
                <div className="p-1.5 bg-zinc-900 text-white rounded-md">
                  <BrainCircuit className="w-4 h-4" />
                </div>
                <div>
                  <h3 className="text-sm font-bold text-zinc-900">Kayıt Sihirbazı</h3>
                  <p className="text-[10px] text-zinc-500 font-medium">Manuel hasta tanımlama ve nitelendirme</p>
                </div>
              </div>
              <button 
                onClick={() => setShowNewModal(false)}
                className="w-8 h-8 flex items-center justify-center text-zinc-400 hover:bg-zinc-200 hover:text-zinc-900 rounded-full transition-colors"
              >
                <X className="w-4.5 h-4.5" />
              </button>
            </div>

            {/* Progress */}
            <div className="px-5 py-2.5 border-b border-zinc-100 bg-white flex items-center justify-between">
              <div className="flex gap-1">
                {[1,2,3,4,5].map(step => (
                  <div key={step} className={`h-1.5 w-6 rounded-full ${step <= intakeStep ? 'bg-zinc-900' : 'bg-zinc-200'}`} />
                ))}
              </div>
              <span className="text-[10px] font-bold text-zinc-400 uppercase tracking-widest">
                AŞAMA {intakeStep} / 5
              </span>
            </div>

            {/* Content */}
            <div className="p-6">
              {intakeStep === 1 && (
                <div className="space-y-5 animate-in slide-in-from-right-4 duration-300">
                  <div>
                    <label className="block text-xs font-bold text-zinc-900 mb-2">Hasta Adı Soyadı</label>
                    <input
                      type="text"
                      placeholder="Örn: Ahmet Demir"
                      value={newPatientName}
                      onChange={(e) => setNewPatientName(e.target.value)}
                      className="w-full bg-white border border-zinc-300 rounded-lg px-3.5 py-2.5 text-xs font-semibold focus:outline-none focus:border-zinc-900 focus:ring-1 focus:ring-zinc-900 shadow-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-bold text-zinc-900 mb-2.5">Talep Edilen Tedavi</label>
                    <div className="flex flex-wrap gap-2">
                      {['İmplant - Premium', 'Estetik Gülüş Tasarımı', 'Şeffaf Plak (Invisalign)', 'Kanal Tedavisi', 'Zirkonyum Kaplama'].map((t) => (
                        <button
                          key={t}
                          onClick={() => setSelectedTedavi(t)}
                          className={`px-3 py-1.5 rounded-full text-[11px] font-bold border transition-colors ${
                            selectedTedavi === t 
                            ? 'bg-zinc-900 text-white border-zinc-900' 
                            : 'bg-zinc-50 text-zinc-600 border-zinc-200 hover:bg-zinc-100 hover:border-zinc-300'
                          }`}
                        >
                          {t}
                        </button>
                      ))}
                    </div>
                  </div>
                </div>
              )}

              {intakeStep === 2 && (
                <div className="space-y-4 animate-in slide-in-from-right-4 duration-300">
                  <label className="block text-xs font-bold text-zinc-900 mb-3">Hastanın Şikayet Seviyesi Nedir?</label>
                  <div className="space-y-2">
                    {['Çok şiddetli ağrım var, acil destek!', 'Soğuk/Sıcak hassasiyeti var, rahatsız ediyor.', 'Ağrı yok, sadece estetik görünüm için.', 'Düzenli kontrol/ufak bir dolgu ihtiyacı.'].map((a) => (
                      <button
                        key={a}
                        onClick={() => setSelectedAgri(a)}
                        className={`w-full text-left px-4 py-3 rounded-lg text-xs font-semibold border transition-all ${
                          selectedAgri === a 
                          ? 'bg-zinc-900 text-white border-zinc-900 shadow-md' 
                          : 'bg-white text-zinc-600 border-zinc-200 hover:border-zinc-400'
                        }`}
                      >
                        {a}
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {intakeStep === 3 && (
                <div className="space-y-4 animate-in slide-in-from-right-4 duration-300">
                  <label className="block text-xs font-bold text-zinc-900 mb-3">Bütçe / Fiyat Algısı Seviyesi</label>
                  <div className="space-y-2">
                    {[
                      { label: 'Kalite odaklı, fiyat önemli değil (Premium)', val: 'Premium' },
                      { label: 'Fiyat/Performans dengesi arıyor', val: 'F/P' },
                      { label: 'Kampanya/indirim bekliyor (Düşük)', val: 'İndirim' }
                    ].map((b) => (
                      <button
                        key={b.val}
                        onClick={() => setSelectedButce(b.val)}
                        className={`w-full flex items-center justify-between px-4 py-3 rounded-lg text-xs font-semibold border transition-all ${
                          selectedButce === b.val 
                          ? 'bg-zinc-900 text-white border-zinc-900 shadow-md' 
                          : 'bg-white text-zinc-600 border-zinc-200 hover:border-zinc-400'
                        }`}
                      >
                        {b.label}
                        <DollarSign className={`w-4 h-4 ${selectedButce === b.val ? 'text-zinc-300' : 'text-zinc-400'}`} />
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {intakeStep === 4 && (
                <div className="space-y-4 animate-in slide-in-from-right-4 duration-300">
                  <label className="block text-xs font-bold text-zinc-900 mb-3">Zaman Tercihi</label>
                  <div className="space-y-2">
                    {['Bugün içerisinde (Acil)', 'Yarın öğleden sonra', 'Bu hafta sonu', 'Önümüzdeki hafta içi', 'Henüz emin değilim, bilgi alıyorum'].map((z) => (
                      <button
                        key={z}
                        onClick={() => setSelectedZaman(z)}
                        className={`w-full text-left px-4 py-3 rounded-lg text-xs font-semibold border transition-all ${
                          selectedZaman === z 
                          ? 'bg-zinc-900 text-white border-zinc-900 shadow-md' 
                          : 'bg-white text-zinc-600 border-zinc-200 hover:border-zinc-400'
                        }`}
                      >
                        {z}
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {intakeStep === 5 && (
                <div className="space-y-5 animate-in slide-in-from-right-4 duration-300">
                  <div className="text-center">
                    <div className="w-12 h-12 bg-emerald-100 text-emerald-600 rounded-full flex items-center justify-center mx-auto mb-3 shadow-sm border border-emerald-200">
                      <CheckCircle2 className="w-6 h-6" />
                    </div>
                    <h4 className="text-sm font-bold text-zinc-900 mb-1">Analiz Tamamlandı</h4>
                    <p className="text-xs text-zinc-500">
                      Hastanın kayıt bilgileri AI tarafından derlendi.
                    </p>
                  </div>
                  <div className="bg-zinc-50 rounded-lg border border-zinc-200 p-4 space-y-3">
                    <div className="flex justify-between text-xs border-b border-zinc-200/60 pb-2">
                      <span className="text-zinc-500 font-medium">Hasta:</span>
                      <span className="font-bold text-zinc-900">{newPatientName}</span>
                    </div>
                    <div className="flex justify-between text-xs border-b border-zinc-200/60 pb-2">
                      <span className="text-zinc-500 font-medium">Tedavi:</span>
                      <span className="font-bold text-zinc-900">{selectedTedavi}</span>
                    </div>
                    <div className="flex justify-between text-xs">
                      <span className="text-zinc-500 font-medium">Tahmini Niyet:</span>
                      <span className="font-bold text-emerald-600">Yüksek (Kayda Hazır)</span>
                    </div>
                  </div>
                </div>
              )}
            </div>

            {/* Footer */}
            <div className="p-4 bg-zinc-50 border-t border-zinc-200 flex items-center justify-between">
              {intakeStep > 1 ? (
                <button
                  onClick={() => setIntakeStep(s => s - 1)}
                  className="px-4 py-2 text-xs font-bold text-zinc-600 hover:text-zinc-900 bg-white border border-zinc-200 rounded-lg shadow-sm"
                >
                  Geri
                </button>
              ) : <div />}
              
              {intakeStep < 5 ? (
                <button
                  onClick={() => setIntakeStep(s => s + 1)}
                  disabled={
                    (intakeStep === 1 && (!newPatientName || !selectedTedavi)) ||
                    (intakeStep === 2 && !selectedAgri) ||
                    (intakeStep === 3 && !selectedButce) ||
                    (intakeStep === 4 && !selectedZaman)
                  }
                  className="px-4 py-2 text-xs font-bold bg-zinc-900 text-white rounded-lg hover:bg-zinc-800 transition-colors shadow-sm disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5"
                >
                  Devam Et <ArrowRight className="w-3.5 h-3.5" />
                </button>
              ) : (
                <button
                  onClick={handleConfirmIntake}
                  className="px-4 py-2 text-xs font-bold bg-emerald-600 text-white rounded-lg hover:bg-emerald-500 transition-colors shadow-sm flex items-center gap-1.5 w-full justify-center"
                >
                  Kaydı Onayla ve Randevu Oluştur
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
