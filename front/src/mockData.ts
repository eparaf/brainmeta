import { Conversation, Clinic, AdArm, Template, Integration } from './types';

export const mockClinics: Clinic[] = [
  {
    id: 'dentplus-nisantasi',
    name: 'DentPlus Nişantaşı',
    delivered: 42,
    guarantee: 50,
    shadowPrice: 1.45,
    status: 'on-track'
  },
  {
    id: 'acibadem-agiz-cene',
    name: 'Acıbadem Ağız & Çene',
    delivered: 28,
    guarantee: 30,
    shadowPrice: 1.80,
    status: 'on-track'
  },
  {
    id: 'atakoy-dental-studio',
    name: 'Ataköy Dental Studio',
    delivered: 15,
    guarantee: 25,
    shadowPrice: 0.95,
    status: 'behind'
  }
];

export const mockAdArms: Record<string, AdArm[]> = {
  'dentplus-nisantasi': [
    { armId: 'ARM-DP-01', segment: 'İmplant - Premium', thetaHat: 0.88, cpl: 185, leads: 340, appointments: 42, spend: 62900 },
    { armId: 'ARM-DP-02', segment: 'Estetik Gülüş Tasarımı', thetaHat: 0.72, cpl: 210, leads: 180, appointments: 21, spend: 37800 },
    { armId: 'ARM-DP-03', segment: 'Şeffaf Plak (Invisalign)', thetaHat: 0.61, cpl: 340, leads: 112, appointments: 12, spend: 38080 },
    { armId: 'ARM-DP-04', segment: 'Zirkonyum Kaplama', thetaHat: 0.52, cpl: 160, leads: 220, appointments: 18, spend: 35200 },
    { armId: 'ARM-DP-05', segment: 'Diş Beyazlatma (Zoom)', thetaHat: 0.35, cpl: 95, leads: 150, appointments: 9, spend: 14250 }
  ],
  'acibadem-agiz-cene': [
    { armId: 'ARM-AC-01', segment: 'All-on-4 Çene Cerrahisi', thetaHat: 0.94, cpl: 450, leads: 95, appointments: 18, spend: 42750 },
    { armId: 'ARM-AC-02', segment: 'Gömülü Yirmi Yaş Dişi', thetaHat: 0.79, cpl: 120, leads: 310, appointments: 48, spend: 37200 },
    { armId: 'ARM-AC-03', segment: 'Kanal Tedavisi (Endodonti)', thetaHat: 0.68, cpl: 85, leads: 420, appointments: 52, spend: 35700 },
    { armId: 'ARM-AC-04', segment: 'Periodontoloji (Diş Eti)', thetaHat: 0.45, cpl: 110, leads: 150, appointments: 12, spend: 16500 }
  ],
  'atakoy-dental-studio': [
    { armId: 'ARM-AS-01', segment: 'Zirkonyum Kaplama', thetaHat: 0.76, cpl: 150, leads: 190, appointments: 15, spend: 28500 },
    { armId: 'ARM-AS-02', segment: 'Pedodonti (Çocuk Diş)', thetaHat: 0.58, cpl: 90, leads: 140, appointments: 8, spend: 12600 },
    { armId: 'ARM-AS-03', segment: 'Genel Muayene & Dolgu', thetaHat: 0.42, cpl: 70, leads: 280, appointments: 14, spend: 19600 },
    { armId: 'ARM-AS-04', segment: 'Lamina Porselen', thetaHat: 0.39, cpl: 310, leads: 65, appointments: 3, spend: 20150 }
  ]
};

export const mockConversations: Record<string, Conversation[]> = {
  'dentplus-nisantasi': [
    {
      id: 'c-dp-1',
      name: 'Ahmet Yılmaz',
      phoneNumber: '5321112233',
      countryCode: '+90',
      clinicId: 'dentplus-nisantasi',
      lastMessage: 'Yarın saat 14:00 için randevuyu onaylıyorum.',
      lastMessageTime: '10:42',
      status: 'Randevu',
      messages: [
        { id: 'm1', sender: 'patient', text: 'Merhabalar, implant tedavisi hakkında bilgi almak istiyorum.', timestamp: '10:15' },
        { id: 'm2', sender: 'agent', text: 'Merhabalar Ahmet Bey! DentPlus Nişantaşı kliniğimizde implant tedavisi alanında uzman hekimlerimiz görev yapmaktadır. Size yardımcı olmaktan memnuniyet duyarım. Öncelikle kaç dişiniz için implant düşünüyorsunuz ve daha önce çekilmiş bir röntgeniniz var mı?', timestamp: '10:17' },
        { id: 'm3', sender: 'patient', text: 'Sol alt tarafta iki eksik dişim var. Röntgenciye henüz gitmedim ama implant fiyatlarınız ortalama ne kadar?', timestamp: '10:22' },
        { id: 'm4', sender: 'agent', text: 'İki eksik dişiniz için kemik durumunuzun hekimimiz tarafından incelenmesi en doğrusu olacaktır. Fiyatlarımız kullandığımız premium marka seçeneğine göre (İsviçre veya Alman menşeili) 15.000 ₺ ile 28.000 ₺ arasında değişmektedir. Randevu planlamak ister misiniz?', timestamp: '10:25' },
        { id: 'm5', sender: 'patient', text: 'Anladım, bütçem bu aralığa uygun. Yarın öğleden sonra gelebilirim.', timestamp: '10:38' },
        { id: 'm6', sender: 'agent', text: 'Harika! Yarın saat 14:00 kliniğimiz ve implant koordinatörümüz Dr. Burak Bey için uygundur. Adınıza bu saati rezerve edeyim mi?', timestamp: '10:40' },
        { id: 'm7', sender: 'patient', text: 'Yarın saat 14:00 için randevuyu onaylıyorum.', timestamp: '10:42' }
      ],
      qualification: {
        segment: 'İmplant - Premium',
        intentPct: 95,
        urgency: 'Yüksek',
        budget: '30.000 - 50.000 ₺',
        language: 'TR',
        booked: true,
        appointmentTime: 'Yarın 14:00'
      }
    },
    {
      id: 'c-dp-2',
      name: 'Zeynep Kaya',
      phoneNumber: '5554443322',
      countryCode: '+90',
      clinicId: 'dentplus-nisantasi',
      lastMessage: 'Beyazlatma sonrasında hassasiyet kalıcı oluyor mu?',
      lastMessageTime: 'Dün',
      status: 'Niteleniyor',
      messages: [
        { id: 'm1', sender: 'patient', text: 'Diş beyazlatma işlemi yaptırmak istiyorum. Kampanyanız var mı?', timestamp: 'Dün 15:20' },
        { id: 'm2', sender: 'agent', text: 'Zeynep Hanım merhaba, kliniğimizde Amerikan Zoom Philips beyazlatma teknolojisi uygulanmaktadır. Şu an yaz dönemine özel muayene + tek seans beyazlatma paketimiz 6.500 ₺ yerine 4.800 ₺\'dir.', timestamp: 'Dün 15:24' },
        { id: 'm3', sender: 'patient', text: 'Beyazlatma sonrasında hassasiyet kalıcı oluyor mu?', timestamp: 'Dün 15:30' }
      ],
      qualification: {
        segment: 'Estetik Diş Beyazlatma',
        intentPct: 75,
        urgency: 'Orta',
        budget: '4.800 ₺',
        language: 'TR',
        booked: false
      }
    },
    {
      id: 'c-dp-3',
      name: 'Mert Demir',
      phoneNumber: '5078889900',
      countryCode: '+90',
      clinicId: 'dentplus-nisantasi',
      lastMessage: 'Şu an inanılmaz ağrım var, acil doktor görebilir miyim?',
      lastMessageTime: '08:15',
      status: 'Niteleniyor',
      messages: [
        { id: 'm1', sender: 'patient', text: 'Şu an inanılmaz ağrım var, acil doktor görebilir miyim?', timestamp: '08:15' }
      ],
      qualification: {
        segment: 'Acil Çene Cerrahisi',
        intentPct: 90,
        urgency: 'Yüksek',
        budget: 'Belirsiz',
        language: 'TR',
        booked: false
      }
    },
    {
      id: 'c-dp-4',
      name: 'Sarah Jenkins',
      phoneNumber: '7412345678',
      countryCode: '+44',
      clinicId: 'dentplus-nisantasi',
      lastMessage: 'Do you offer airport pick up for international patients?',
      lastMessageTime: '3 Gün Önce',
      status: 'Düştü',
      messages: [
        { id: 'm1', sender: 'patient', text: 'Hi, I want to get full mouth veneers. Do you have package offers including hotel?', timestamp: '3 Gün Önce' },
        { id: 'm2', sender: 'agent', text: 'Hello Sarah! Yes, we have health tourism packages (VIP Transfer + 5 Star Hotel + Veneers). Let us connect you to our medical travel consultant.', timestamp: '3 Gün Önce' },
        { id: 'm3', sender: 'patient', text: 'Do you offer airport pick up for international patients?', timestamp: '3 Gün Önce' }
      ],
      qualification: {
        segment: 'Sağlık Turizmi - Gülüş',
        intentPct: 40,
        urgency: 'Düşük',
        budget: '350.000 ₺',
        language: 'EN',
        booked: false
      }
    }
  ],
  'acibadem-agiz-cene': [
    {
      id: 'c-ac-1',
      name: 'Caner Özkan',
      phoneNumber: '5329998877',
      countryCode: '+90',
      clinicId: 'acibadem-agiz-cene',
      lastMessage: 'Tamamdır, kanal tedavisini haftaya pazartesi yapalım.',
      lastMessageTime: '12:00',
      status: 'Randevu',
      messages: [
        { id: 'm1', sender: 'patient', text: 'Kanal tedavisi fiyatını öğrenebilir miyim?', timestamp: '11:15' },
        { id: 'm2', sender: 'agent', text: 'Merhaba Caner Bey, tek kanallı tedavi 3.500 ₺, çok kanallı tedavi ise 5.000 ₺\'dir.', timestamp: '11:20' },
        { id: 'm3', sender: 'patient', text: 'Tamamdır, kanal tedavisini haftaya pazartesi yapalım.', timestamp: '12:00' }
      ],
      qualification: {
        segment: 'Kanal Tedavisi',
        intentPct: 99,
        urgency: 'Yüksek',
        budget: '5.000 ₺',
        language: 'TR',
        booked: true,
        appointmentTime: 'Pazartesi 10:00'
      }
    },
    {
      id: 'c-ac-2',
      name: 'Buse Yurt',
      clinicId: 'acibadem-agiz-cene',
      lastMessage: 'Randevu vermeyin şimdilik, fiyat bana yüksek geldi.',
      lastMessageTime: '1 Gün Önce',
      status: 'Gelmedi',
      messages: [
        { id: 'm1', sender: 'patient', text: 'Zirkonyum kaplama adet fiyatı nedir?', timestamp: '1 Gün Önce' },
        { id: 'm2', sender: 'agent', text: 'Buse Hanım, yüksek kaliteli zirkonyum kaplama adetimiz 8.500 ₺\'dir.', timestamp: '1 Gün Önce' },
        { id: 'm3', sender: 'patient', text: 'Randevu vermeyin şimdilik, fiyat bana yüksek geldi.', timestamp: '1 Gün Önce' }
      ],
      qualification: {
        segment: 'Zirkonyum Kaplama',
        intentPct: 20,
        urgency: 'Düşük',
        budget: 'Yetersiz',
        language: 'TR',
        booked: false
      }
    }
  ],
  'atakoy-dental-studio': [
    {
      id: 'c-as-1',
      name: 'Kadir Bulut',
      clinicId: 'atakoy-dental-studio',
      lastMessage: 'All-on-4 sistemini implant hocası mı yapıyor?',
      lastMessageTime: '09:00',
      status: 'Niteleniyor',
      messages: [
        { id: 'm1', sender: 'patient', text: 'All-on-4 tekniği hakkında bilgi almak istiyorum, yaşım 62.', timestamp: '08:45' },
        { id: 'm2', sender: 'agent', text: 'Kadir Bey merhaba. All-on-4 tekniği tamamen dişsiz çenelerde 4 adet özel açılı implant üzerine sabit protez yapılması işlemidir. Kliniğimiz kurucusu Çene Cerrahı Doç. Dr. Ahmet Bey bu operasyonları gerçekleştirmektedir.', timestamp: '08:52' },
        { id: 'm3', sender: 'patient', text: 'All-on-4 sistemini implant hocası mı yapıyor?', timestamp: '09:00' }
      ],
      qualification: {
        segment: 'İmplant - All-on-4',
        intentPct: 85,
        urgency: 'Orta',
        budget: '180.000 ₺',
        language: 'TR',
        booked: false
      }
    }
  ]
};

export const mockTemplates: Template[] = [
  {
    id: 't-1',
    name: 'implant_qualification_followup',
    category: 'UTILITY',
    language: 'tr',
    status: 'APPROVED',
    body: 'Merhaba {{1}}, DentPlus kliniğimizden Dr. Burak Bey röntgeninizi inceledi. Kemik yoğunluğunuz implant için son derece uygun görünüyor. Size özel hazırladığımız implant tedavi planını aktarmak için bugün saat {{2}} veya {{3}} arasında 5 dakikalık bir telefon araması gerçekleştirebilir miyiz?'
  },
  {
    id: 't-2',
    name: 'gulus_tasarimi_marketing_v1',
    category: 'MARKETING',
    language: 'tr',
    status: 'APPROVED',
    body: 'Işıltılı bir gülüşe hazır mısınız {{1}}? 🌟 DentPlus estetik gülüş tasarımında bu aya özel %15 indirim fırsatı sizi bekliyor. Dijital gülüş analizi randevunuzu hemen oluşturmak için bu mesaja "RANDEVU" yazmanız yeterli.'
  },
  {
    id: 't-3',
    name: 'randevu_hatirlatma_system',
    category: 'UTILITY',
    language: 'tr',
    status: 'APPROVED',
    body: 'Sayın {{1}}, yarın saat {{2}}\'de planlanmış olan ağız ve diş sağlığı randevunuzu hatırlatmak isteriz. Kliniğimiz Nişantaşı metro çıkışına 2 dakika yürüme mesafesindedir. Katılım durumunuzu "ONAY" veya "İPTAL" yazarak bildirebilirsiniz.'
  },
  {
    id: 't-4',
    name: 'dis_beyazlatma_promo',
    category: 'MARKETING',
    language: 'tr',
    status: 'APPROVED',
    body: 'Daha beyaz dişler, daha özgüvenli gülüşler! {{1}} Beyazlatma işlemlerinde Philips Zoom teknolojisiyle tek seansta 8 tona kadar açılma sağlıyoruz. Randevunuzu {{2}} tarihine kadar oluşturarak avantajlı ücretten faydalanın.'
  }
];

export const mockIntegrations: Integration[] = [
  { id: 'whatsapp-api', name: 'WhatsApp Cloud API', description: 'Meta onaylı resmi WhatsApp numarası ve şablon entegrasyonu.', connected: true, type: 'whatsapp' },
  { id: 'meta-ads', name: 'Meta Ads (Facebook & Instagram)', description: 'Thompson bütçe optimizasyonu için reklam hesapları ve Lead formları.', connected: true, type: 'meta_ads' },
  { id: 'google-ads', name: 'Google Search & Maps Ads', description: 'Anahtar kelime reklam bütçeleri ve gölge fiyat senkronizasyonu.', connected: false, type: 'google_ads' },
  { id: 'web-form', name: 'Klinik Web Sitesi Formu', description: 'Web sitesi üzerinden gelen başvuruların anlık WhatsApp inbox aktarımı.', connected: true, type: 'web_form' }
];
