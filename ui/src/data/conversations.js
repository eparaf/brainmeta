// Seeded conversations — shaped like a production WhatsApp inbox so you can watch
// the qualify→decide flow. (Client-side sample history; new live chats hit the
// real backend.) Each clinic + arm matches the backend's seeded demo network.
export const CLINICS = [
  { id: 'umraniye', name: 'Ümraniye İmplant', segment: 'implant' },
  { id: 'nisantasi', name: 'Nişantaşı Estetik', segment: 'aesthetic' },
  { id: 'kadikoy', name: 'Kadıköy Diş', segment: 'ortho' },
  { id: 'sisli', name: 'Şişli Aile Diş', segment: 'general' },
]
export const armOf = (c) => `${c.id}:meta:${c.segment}`
export const clinicById = (id) => CLINICS.find((c) => c.id === id)

const m = (role, text, t) => ({ role, text, t })

export const SEED = [
  {
    id: 'c1', name: 'Ayşe Korkmaz', phone: '+90 532 114 88 02', clinicId: 'umraniye',
    channel: 'WhatsApp', time: '9 dk', status: 'booked', live: false,
    messages: [
      m('patient', 'Merhaba, alt çenede 2 dişim eksik, implant düşünüyorum. Fiyat alabilir miyim?', '14:02'),
      m('agent', 'Merhaba Ayşe Hanım! İki implant için uygun bir değerlendirme ayarlayabilirim. Bütçe aralığınız nedir, ne zaman müsaitsiniz?', '14:02'),
      m('patient', 'Bütçem 45000 TL civarı, bu hafta olabilir', '14:04'),
      m('agent', 'Harika! Randevunuzu 30 Haz 13:00 için oluşturdum. Onaylıyor musunuz? Hatırlatma göndereceğiz.', '14:04'),
      m('patient', 'evet onaylıyorum', '14:05'),
      m('agent', 'Onaylandı ✓ 30 Haz 13:00 randevunuzda görüşmek üzere. İyi günler!', '14:05'),
    ],
    decision: { booked: true, apptTime: '2026-06-30T13:00:00+03:00', qualification: { segment: 'implant', intent: 0.88, urgency: 0.5, budgetTry: 45000, locale: 'tr' }, reason: 'booked' },
  },
  {
    id: 'c2', name: 'Mehmet Demir', phone: '+90 545 220 17 39', clinicId: 'umraniye',
    channel: 'WhatsApp', time: '24 dk', status: 'booked', live: false,
    messages: [
      m('patient', 'Diş ağrım çok şiddetli acil bakabilir misiniz', '13:40'),
      m('agent', 'Geçmiş olsun! Sizi en kısa sürede alalım. Ağrı ne zamandır var, bir bölgede mi?', '13:40'),
      m('patient', 'iki gündür, sağ alt. ne gerekirse yaptırırım', '13:41'),
      m('agent', 'Anladım. Randevunuzu 30 Haz 09:30 için oluşturdum, onaylıyor musunuz?', '13:41'),
      m('patient', 'tamam', '13:42'),
      m('agent', 'Onaylandı ✓ 30 Haz 09:30. Geçmiş olsun, görüşmek üzere.', '13:42'),
    ],
    decision: { booked: true, apptTime: '2026-06-30T09:30:00+03:00', qualification: { segment: 'implant', intent: 0.9, urgency: 0.9, budgetTry: 0, locale: 'tr' }, reason: 'booked' },
  },
  {
    id: 'c3', name: 'James Walker', phone: '+44 7700 900421', clinicId: 'nisantasi',
    channel: 'Instagram', time: '1 sa', status: 'booked', live: false,
    messages: [
      m('patient', 'Hi! Interested in a full set of veneers. Flying in from London next month, budget around 5000 GBP.', '13:05'),
      m('agent', "Hello James! We'd love to help. For a full veneer set we'll plan a consultation + scan. Roughly when are your travel dates?", '13:05'),
      m('patient', 'around the 20th, staying 5 days', '13:07'),
      m('agent', "Perfect. I've reserved a consultation for 30 Jun 11:00. Shall I confirm? We'll send details and a reminder.", '13:07'),
      m('patient', 'yes please', '13:09'),
      m('agent', 'Confirmed ✓ — see you on 30 Jun 11:00. Take care!', '13:09'),
    ],
    decision: { booked: true, apptTime: '2026-06-30T11:00:00+03:00', qualification: { segment: 'aesthetic', intent: 0.82, urgency: 0.3, budgetTry: 256500, locale: 'en' }, reason: 'booked' },
  },
  {
    id: 'c4', name: 'Burak Taş', phone: '+90 538 661 09 14', clinicId: 'kadikoy',
    channel: 'WhatsApp', time: '2 sa', status: 'qualifying', live: false,
    messages: [
      m('patient', 'şeffaf plak fiyatı ne kadar', '12:10'),
      m('agent', 'Merhaba! Şeffaf plak tedavisi diş yapınıza göre değişiyor. Daha önce ortodonti muayenesi oldunuz mu, güncel bir röntgeniniz var mı?', '12:10'),
    ],
    decision: { booked: false, qualification: { segment: 'ortho', intent: 0.55, urgency: 0.2, budgetTry: 0, locale: 'tr' }, reason: 'gathering' },
  },
  {
    id: 'c5', name: 'Zeynep Aydın', phone: '+90 506 332 75 60', clinicId: 'sisli',
    channel: 'WhatsApp', time: '3 sa', status: 'booked', live: false,
    messages: [
      m('patient', 'rutin kontrol ve diş temizliği yaptırmak istiyorum', '11:20'),
      m('agent', 'Tabii ki! Kontrol + temizlik için sizi alalım. Hafta içi mi hafta sonu mu tercih edersiniz?', '11:20'),
      m('patient', 'hafta içi öğleden sonra', '11:22'),
      m('agent', 'Randevunuzu 30 Haz 15:30 için oluşturdum, onaylıyor musunuz?', '11:22'),
    ],
    decision: { booked: true, apptTime: '2026-06-30T15:30:00+03:00', qualification: { segment: 'general', intent: 0.7, urgency: 0.2, budgetTry: 0, locale: 'tr' }, reason: 'booked' },
  },
  {
    id: 'c6', name: 'Elif Şahin', phone: '+90 542 870 41 23', clinicId: 'nisantasi',
    channel: 'Instagram', time: '5 sa', status: 'lost', live: false,
    messages: [
      m('patient', 'gülüş tasarımı düşünüyorum ama henüz emin değilim, sadece bilgi alıyorum', '09:15'),
      m('agent', 'Tabii, dilediğiniz zaman buradayız. Size birkaç önceki vaka ve fiyat aralığı göndereyim, hazır olduğunuzda muayene ayarlarız.', '09:15'),
      m('patient', 'şimdilik teşekkürler', '09:18'),
    ],
    decision: { booked: false, qualification: { segment: 'aesthetic', intent: 0.25, urgency: 0.1, budgetTry: 0, locale: 'tr' }, reason: 'low_intent' },
  },
]
