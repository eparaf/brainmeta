import React, { useState, useMemo } from 'react';
import { Conversation } from '../types';
import { Search, Phone, CalendarDays, Clock, Filter, PhoneForwarded } from 'lucide-react';

interface UyelerPageProps {
  conversations: Conversation[];
  onNavigateToChat: (id: string) => void;
}

export default function UyelerPage({ conversations, onNavigateToChat }: UyelerPageProps) {
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('All');

  const filteredConversations = useMemo(() => {
    return conversations.filter(c => {
      const matchesSearch = c.name.toLowerCase().includes(searchTerm.toLowerCase()) || 
                            c.phoneNumber?.includes(searchTerm);
      const matchesStatus = statusFilter === 'All' || c.status === statusFilter;
      return matchesSearch && matchesStatus;
    });
  }, [conversations, searchTerm, statusFilter]);

  const niteleniyor = filteredConversations.filter(c => c.status === 'Niteleniyor');
  const randevulular = filteredConversations.filter(c => c.status === 'Randevu');
  const dustu = filteredConversations.filter(c => c.status === 'Düştü' || c.status === 'Gelmedi');

  const getInitials = (name: string) => {
    return name.split(' ').map(n => n[0]).join('').substring(0, 2).toUpperCase();
  };

  const renderQueue = (title: string, items: Conversation[], isActiveQueue: boolean = false) => (
    <div className="flex-1 bg-white border border-gray-200 rounded-xl p-3.5 flex flex-col min-w-0 shadow-sm h-full max-h-full overflow-hidden">
      <div className="flex items-center justify-between mb-3 shrink-0">
        <span className="font-sans text-xs font-bold text-gray-700 uppercase tracking-wide">{title}</span>
        <span className="font-sans text-[10px] font-semibold text-gray-500 bg-gray-100 px-2 py-0.5 rounded-md border border-gray-200/60">
          {items.length} kişi
        </span>
      </div>
      <div className="flex flex-col gap-2 flex-1 overflow-y-auto pr-1 custom-scrollbar">
        {items.map(conv => {
          const isUrgent = conv.qualification.urgency === 'Yüksek';
          return (
            <div 
              key={conv.id} 
              className={`flex items-center gap-2.5 p-2 rounded-lg border transition-all ${
                isActiveQueue 
                  ? 'bg-gradient-to-br from-blue-600/10 to-blue-600/5 border-blue-600/20 shadow-sm' 
                  : 'bg-white border-gray-200 hover:border-blue-300 hover:bg-blue-50/30'
              }`}
            >
              <div className={`w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold shrink-0 shadow-sm border ${
                isActiveQueue ? 'bg-blue-100 text-blue-700 border-blue-200' : 'bg-gray-50 text-gray-600 border-gray-200'
              }`}>
                {getInitials(conv.name)}
              </div>
              
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-1.5">
                  <div className="font-sans text-xs font-semibold text-gray-900 truncate">{conv.name}</div>
                  {isUrgent && (
                    <div className="w-1.5 h-1.5 rounded-full bg-red-500 shrink-0" title="Yüksek Aciliyet" />
                  )}
                </div>
                <div className="font-sans text-[10px] text-gray-500 truncate mt-0.5 flex items-center gap-1">
                  <span className="font-medium text-gray-600">{conv.qualification.segment || 'Yeni Lead'}</span>
                  <span className="text-gray-300">•</span>
                  <span>{conv.lastMessageTime}</span>
                </div>
              </div>
              
              <button 
                onClick={() => onNavigateToChat(conv.id)}
                className={`w-7 h-7 rounded-full flex items-center justify-center shrink-0 transition-colors shadow-sm ${
                  isActiveQueue 
                    ? 'bg-blue-600 text-white hover:bg-blue-700' 
                    : 'bg-gray-100 text-gray-400 border border-gray-200 hover:bg-blue-600 hover:text-white hover:border-blue-600'
                }`}
                title="Sohbete Git"
              >
                <PhoneForwarded className="w-3.5 h-3.5" />
              </button>
            </div>
          );
        })}
        {items.length === 0 && (
          <div className="flex flex-col items-center justify-center h-24 text-gray-400">
            <span className="text-[10px] font-medium">Liste boş</span>
          </div>
        )}
      </div>
    </div>
  );

  return (
    <div className="h-full flex flex-col">
      {/* Top Bar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6 shrink-0">
        <div className="flex items-center gap-3">
          <div className="relative">
            <Search className="w-4 h-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
            <input 
              type="text" 
              placeholder="İsim veya telefon ara..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full sm:w-64 pl-9 pr-3 py-2 text-xs bg-white border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all shadow-sm"
            />
          </div>
          <div className="relative">
            <select 
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="appearance-none pl-8 pr-8 py-2 text-xs bg-white border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all shadow-sm font-medium text-gray-700"
            >
              <option value="All">Tüm Durumlar</option>
              <option value="Niteleniyor">Takip Bekleyen</option>
              <option value="Randevu">Randevulular</option>
              <option value="Düştü">Düşenler</option>
            </select>
            <Filter className="w-3.5 h-3.5 text-gray-400 absolute left-2.5 top-1/2 -translate-y-1/2" />
          </div>
        </div>
      </div>

      {/* Kanban / Queues */}
      <div className="flex-1 grid grid-cols-1 md:grid-cols-3 gap-4 min-h-0">
        {renderQueue('Nitelendirme / Takip', niteleniyor, true)}
        {renderQueue('Onaylı Randevular', randevulular, false)}
        {renderQueue('Kayıp / Ulaşılamadı', dustu, false)}
      </div>
    </div>
  );
}
