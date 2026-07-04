export interface Message {
  id: string;
  sender: 'patient' | 'agent';
  text: string;
  timestamp: string;
}

export interface Qualification {
  segment: string;
  intentPct: number; // e.g. 85%
  urgency: 'Yüksek' | 'Orta' | 'Düşük';
  budget: string; // e.g. "45.000 ₺"
  language: string; // e.g. "TR"
  booked: boolean;
  appointmentTime?: string;
}

export interface Conversation {
  id: string;
  name: string;
  phoneNumber?: string;
  countryCode?: string;
  clinicId: string;
  lastMessage: string;
  lastMessageTime: string;
  status: 'Randevu' | 'Niteleniyor' | 'Düştü' | 'Gelmedi';
  messages: Message[];
  qualification: Qualification;
}

export interface Clinic {
  id: string;
  name: string;
  delivered: number;
  guarantee: number;
  shadowPrice: number; // λ shadow price
  status: 'on-track' | 'behind';
}

export interface AdArm {
  armId: string;
  segment: string;
  thetaHat: number; // [0, 1] quality probability
  cpl: number; // cost per lead
  leads: number;
  appointments: number;
  spend: number;
}

export interface BudgetStat {
  networkDailyBudget: number; // Ağ günlük bütçesi (e.g. 45000)
  shadowPrice: number; // Gölge fiyat λ (e.g. 1.24)
  fundedArmsCount: number; // Fonlanan kol sayısı
}

export interface Template {
  id: string;
  name: string;
  category: 'UTILITY' | 'MARKETING';
  language: string;
  status: 'APPROVED';
  body: string;
}

export interface Integration {
  id: string;
  name: string;
  description: string;
  connected: boolean;
  type: 'whatsapp' | 'meta_ads' | 'google_ads' | 'web_form';
}
