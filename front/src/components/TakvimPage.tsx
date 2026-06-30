import React, { useMemo, useState } from 'react';
import { Calendar as CalendarIcon, Clock, User, CheckCircle2, ChevronRight, Phone, LayoutGrid, List } from 'lucide-react';
import { Conversation } from '../types';

interface TakvimPageProps {
  conversations: Conversation[];
  onNavigateToChat: (id: string) => void;
}

export default function TakvimPage({ conversations, onNavigateToChat }: TakvimPageProps) {
  const [viewMode, setViewMode] = useState<'list' | 'grid'>('list');

  // Filter only booked conversations and sort them roughly by time
  const appointments = useMemo(() => {
    return conversations
      .filter(c => c.status === 'Randevu' && c.qualification.booked)
      .map(c => {
        // A simple mock parser to group them (e.g. "Yarın 14:00" -> "Yarın", "14:00")
        const timeStr = c.qualification.appointmentTime || 'Belirsiz';
        let day = 'Gelecek Planlamalar';
        let time = '00:00';
        
        if (timeStr.toLowerCase().includes('yarın')) {
          day = 'Yarın';
          time = timeStr.replace(/[^0-9:]/g, '');
        } else if (timeStr.toLowerCase().includes('pazartesi') || timeStr.toLowerCase().includes('bugün')) {
          day = timeStr.split(' ')[0];
          time = timeStr.replace(/[^0-9:]/g, '');
        } else {
          day = timeStr.split(' ')[0] || day;
          time = timeStr.replace(/[^0-9:]/g, '') || time;
        }

        return {
          ...c,
          day,
          time: time || '12:00'
        };
      })
      .sort((a, b) => a.time.localeCompare(b.time));
  }, [conversations]);

  // Group by day
  const groupedAppointments = useMemo(() => {
    return appointments.reduce((acc, curr) => {
      if (!acc[curr.day]) acc[curr.day] = [];
      acc[curr.day].push(curr);
      return acc;
    }, {} as Record<string, typeof appointments>);
  }, [appointments]);

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

  const daysOfWeek = ['Pzt', 'Sal', 'Çar', 'Per', 'Cum', 'Cmt', 'Paz'];
  
  // A mock 30-day calendar array to render the grid
  const calendarDays = Array.from({ length: 30 }, (_, i) => {
    const dayName = i === 0 ? 'Bugün' : i === 1 ? 'Yarın' : 'Gelecek Planlamalar';
    const apps = groupedAppointments[dayName] || (dayName === 'Gelecek Planlamalar' && i === 5 ? groupedAppointments['Gelecek Planlamalar'] : []);
    return { date: i + 1, apps };
  });

  return (
    <div className="space-y-6">
      {/* Toolbar */}
      <div className="flex items-center justify-between bg-white border border-zinc-200/80 rounded-lg p-3 shadow-sm">
        <div className="flex items-center gap-2 bg-zinc-100 p-1 rounded-md">
          <button 
            onClick={() => setViewMode('list')}
            className={`px-3 py-1.5 text-xs font-semibold rounded-md flex items-center gap-1.5 transition-colors ${viewMode === 'list' ? 'bg-white text-zinc-900 shadow-sm' : 'text-zinc-500 hover:text-zinc-700'}`}
          >
            <List className="w-3.5 h-3.5" /> Liste
          </button>
          <button 
            onClick={() => setViewMode('grid')}
            className={`px-3 py-1.5 text-xs font-semibold rounded-md flex items-center gap-1.5 transition-colors ${viewMode === 'grid' ? 'bg-white text-zinc-900 shadow-sm' : 'text-zinc-500 hover:text-zinc-700'}`}
          >
            <LayoutGrid className="w-3.5 h-3.5" /> Takvim
          </button>
        </div>
        <button className="px-3 py-1.5 text-xs font-bold bg-zinc-900 text-white rounded-md shadow-sm">
          + Manuel Randevu
        </button>
      </div>

      <div className="bg-white border border-zinc-200/80 rounded-lg shadow-sm p-5 min-h-[500px]">
        {Object.keys(groupedAppointments).length === 0 ? (
          <div className="flex flex-col items-center justify-center h-64 text-zinc-400">
            <CalendarIcon className="w-10 h-10 mb-3 text-zinc-300" />
            <p className="text-sm font-semibold text-zinc-700">Aktif Randevu Bulunmuyor</p>
            <p className="text-xs mt-1 text-zinc-500">Sohbetler üzerinden onaylanan randevular burada listelenir.</p>
          </div>
        ) : viewMode === 'list' ? (
          <div className="space-y-8">
            {(Object.entries(groupedAppointments) as [string, typeof appointments][]).map(([day, apps]) => (
              <div key={day} className="relative">
                <div className="sticky top-0 bg-white/90 backdrop-blur py-2 mb-3 z-10 border-b border-zinc-100 flex items-center gap-2">
                  <h3 className="text-sm font-bold text-zinc-900 capitalize">{day}</h3>
                  <span className="text-[10px] font-bold bg-zinc-100 text-zinc-500 px-2 py-0.5 rounded-full">
                    {apps.length} Randevu
                  </span>
                </div>
                
                <div className="space-y-3">
                  {apps.map((app) => (
                    <div key={app.id} className="group flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-3 rounded-lg border border-zinc-200/80 bg-zinc-50/50 hover:bg-white hover:border-zinc-300 hover:shadow-sm transition-all text-left">
                      
                      {/* Left: Time & Patient Info */}
                      <div className="flex items-center gap-4">
                        <div className="text-center sm:w-16 shrink-0 border-r border-zinc-200/80 pr-4">
                          <div className="text-sm font-bold text-zinc-900 font-mono">{app.time}</div>
                          <div className="text-[9px] font-semibold text-zinc-400 uppercase">SAAT</div>
                        </div>
                        
                        <div className="flex items-center gap-3">
                          <div className="w-10 h-10 rounded-full bg-zinc-200/50 border border-zinc-200 text-zinc-700 text-xs font-bold flex items-center justify-center shrink-0">
                            {getInitials(app.name)}
                          </div>
                          <div>
                            <div className="text-sm font-bold text-zinc-900">{app.name}</div>
                            <div className="text-[11px] font-medium text-zinc-500 flex items-center gap-1 mt-0.5 font-mono">
                              <Phone className="w-3 h-3 text-emerald-500" /> {maskPhoneNumber(app.phoneNumber, app.countryCode)}
                            </div>
                          </div>
                        </div>
                      </div>

                      {/* Middle: Details */}
                      <div className="flex-1 hidden sm:flex items-center gap-6 px-4 border-l border-zinc-200/80">
                        <div>
                          <div className="text-[9px] font-semibold text-zinc-400 uppercase mb-0.5">Tedavi Segmenti</div>
                          <div className="text-xs font-bold text-zinc-800">{app.qualification.segment}</div>
                        </div>
                        <div>
                          <div className="text-[9px] font-semibold text-zinc-400 uppercase mb-0.5">Bütçe / Niyet</div>
                          <div className="text-xs font-bold text-zinc-800">{app.qualification.budget} <span className="text-emerald-600 ml-1">(%{app.qualification.intentPct})</span></div>
                        </div>
                      </div>

                      {/* Right: Actions */}
                      <div className="flex items-center gap-2 shrink-0">
                        <button 
                          onClick={() => onNavigateToChat(app.id)}
                          className="px-3 py-1.5 text-[11px] font-bold bg-white border border-zinc-200 text-zinc-700 hover:bg-zinc-50 hover:text-zinc-900 rounded-md transition-colors flex items-center gap-1"
                        >
                          Sohbete Git <ChevronRight className="w-3.5 h-3.5" />
                        </button>
                      </div>

                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="flex flex-col h-full">
            <div className="grid grid-cols-7 gap-px bg-zinc-200 border-b border-zinc-200">
              {daysOfWeek.map(d => (
                <div key={d} className="bg-zinc-50 py-2 text-center text-[10px] font-bold uppercase tracking-wider text-zinc-500">
                  {d}
                </div>
              ))}
            </div>
            <div className="grid grid-cols-7 gap-px bg-zinc-200 flex-1">
              {calendarDays.map((day, idx) => (
                <div key={idx} className="bg-white min-h-[100px] p-2 hover:bg-zinc-50/50 transition-colors">
                  <div className={`text-xs font-bold mb-2 w-6 h-6 flex items-center justify-center rounded-full ${idx === 0 ? 'bg-zinc-900 text-white' : 'text-zinc-400'}`}>
                    {day.date}
                  </div>
                  <div className="space-y-1">
                    {day.apps && day.apps.map((app: any) => (
                      <button 
                        key={app.id} 
                        onClick={() => onNavigateToChat(app.id)}
                        className="w-full text-left p-1.5 rounded bg-emerald-50 border border-emerald-100 hover:border-emerald-300 transition-colors flex items-center gap-1"
                        title={app.name}
                      >
                        <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 shrink-0" />
                        <span className="text-[10px] font-bold text-emerald-800 truncate">{app.time} {app.name}</span>
                      </button>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
