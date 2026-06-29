import React, { useState } from 'react'
import { api } from '../api'
import { Icon } from '../icons.jsx'
import { money } from './ui.jsx'
import { clinicById, armOf } from '../data/conversations.js'

// Guided, OPTION-BASED intake — consistent and hallucination-proof. Fixed
// questions + tappable answers; the brain decides the slot deterministically via
// /v1/intake (no free-text LLM that can drift or invent appointments).
const STEPS = [
  { key: 'treatment', q: 'Hangi tedaviyle ilgileniyorsunuz?', opts: [
    { label: 'İmplant', clinic: 'umraniye', segment: 'implant' },
    { label: 'Gülüş tasarımı / beyazlatma', clinic: 'nisantasi', segment: 'aesthetic' },
    { label: 'Tel / ortodonti', clinic: 'kadikoy', segment: 'ortho' },
    { label: 'Kontrol / temizlik', clinic: 'sisli', segment: 'general' },
  ] },
  { key: 'urgency', q: 'Şu an bir ağrınız var mı?', opts: [
    { label: 'Acil, ağrım var', v: 0.9 }, { label: 'Birkaç gündür hafif', v: 0.6 }, { label: 'Ağrı yok, estetik', v: 0.2 },
  ] },
  { key: 'budget', q: 'Tedavi için düşündüğünüz bütçe?', opts: [
    { label: '20.000 ₺ altı', v: 12000 }, { label: '20–50.000 ₺', v: 35000 },
    { label: '50–150.000 ₺', v: 100000 }, { label: '150.000 ₺ üzeri', v: 200000 }, { label: 'Bilmiyorum', v: 0 },
  ] },
  { key: 'timing', q: 'Ne zaman uygun olursunuz?', opts: [
    { label: 'Bu hafta' }, { label: 'Önümüzdeki hafta' }, { label: 'Bu ay' }, { label: 'Esnek' },
  ] },
]

export default function Intake({ phone, onUpdate }) {
  const [i, setI] = useState(0)
  const [tr, setTr] = useState([{ role: 'agent', text: 'Merhaba! Size en uygun randevuyu birkaç soruda ayarlayalım. 🦷' }])
  const [ans, setAns] = useState({})
  const [result, setResult] = useState(null)
  const [busy, setBusy] = useState(false)

  const step = STEPS[i]
  const pick = async (opt) => {
    const a2 = { ...ans, [step.key]: opt }
    const tr2 = [...tr, { role: 'agent', text: step.q }, { role: 'patient', text: opt.label }]
    setAns(a2); setTr(tr2); setI(i + 1)
    if (step.key === 'treatment' && onUpdate) {
      onUpdate({ clinicId: opt.clinic, name: clinicById(opt.clinic)?.name + ' — yeni' })
    }
    if (i + 1 >= STEPS.length) await submit(a2, tr2)
  }

  const submit = async (a, trail) => {
    setBusy(true)
    const t = a.treatment
    try {
      const res = await api.intake({
        phone, clinicId: t.clinic, armId: armOf(clinicById(t.clinic)),
        segment: t.segment, urgency: a.urgency?.v ?? 0.3, budgetTry: a.budget?.v ?? 0, hourOfDay: 14,
      })
      setResult(res)
      const line = res.booked
        ? `Harika! Randevunuzu ${new Date(res.apptTime).toLocaleString('tr-TR', { day: '2-digit', month: 'long', hour: '2-digit', minute: '2-digit' })} için ayırdım. Onaylıyor musunuz?`
        : 'Bu güne uygun yer kalmadı; ekibimiz en yakın boşluk için sizi arayacak.'
      setTr([...trail, { role: 'agent', text: line }])
      if (onUpdate) onUpdate({ status: res.booked ? 'booked' : 'qualifying', decision: res })
    } catch (e) {
      setTr([...trail, { role: 'agent', text: 'Bağlantı hatası: ' + e.message }])
    } finally { setBusy(false) }
  }

  const confirm = (yes) => {
    setTr((t) => [...t, { role: 'patient', text: yes ? 'Onaylıyorum' : 'Başka saat' },
      { role: 'agent', text: yes ? 'Onaylandı ✓ Randevu gününde görüşmek üzere, hatırlatma göndereceğiz.' : 'Tamamdır, ekibimiz alternatif saatler için sizi arayacak.' }])
    setResult((r) => ({ ...r, confirmed: yes, closed: true }))
  }

  return (
    <div className="flex-1 flex flex-col min-w-0">
      <div className="px-4 h-14 border-b border-zinc-100 flex items-center gap-2">
        <span className="text-[13.5px] font-medium">Yeni randevu — yönlendirmeli akış</span>
        <span className="text-[11px] text-zinc-400 ml-1">seçenekli · tutarlı</span>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-3 bg-[#fbfbfc]">
        {tr.map((m, k) => (
          <div key={k} className={`flex ${m.role === 'patient' ? 'justify-end' : 'justify-start'}`}>
            <div className={`max-w-[76%] px-3.5 py-2 text-[13.5px] leading-snug rounded-2xl ${m.role === 'patient' ? 'bg-zinc-900 text-white rounded-br-md' : 'bg-white border border-zinc-200 text-zinc-800 rounded-bl-md'}`}>{m.text}</div>
          </div>
        ))}

        {result && result.booked && !result.closed && (
          <div className="bg-white border border-zinc-200 rounded-lg p-3 text-[13px] max-w-[80%]">
            <div className="flex items-center gap-3 text-[12px] text-zinc-500 mb-2">
              <span>segment <b className="text-zinc-700">{result.qualification?.segment}</b></span>
              <span>niyet <b className="text-zinc-700">{Math.round((result.qualification?.intent || 0) * 100)}%</b></span>
              <span>bütçe <b className="text-zinc-700">{result.qualification?.budgetTry ? money(result.qualification.budgetTry) + ' ₺' : '—'}</b></span>
            </div>
          </div>
        )}
      </div>

      {/* Options / actions */}
      <div className="p-3 border-t border-zinc-100">
        {busy ? <div className="text-[13px] text-zinc-400 px-1">Uygun slot ayarlanıyor…</div>
          : !result ? (
            <div>
              <div className="text-[12px] text-zinc-500 mb-2">{step.q}</div>
              <div className="flex flex-wrap gap-2">
                {step.opts.map((o) => (
                  <button key={o.label} onClick={() => pick(o)}
                    className="px-3 py-1.5 rounded-full border border-zinc-300 bg-white text-[13px] hover:border-zinc-900 hover:bg-zinc-50 transition">{o.label}</button>
                ))}
              </div>
            </div>
          ) : result.booked && !result.closed ? (
            <div className="flex gap-2">
              <button onClick={() => confirm(true)} className="px-4 py-2 rounded-md bg-zinc-900 text-white text-[13px] font-medium hover:bg-zinc-800 flex items-center gap-1.5"><Icon.send size={14} /> Onaylıyorum</button>
              <button onClick={() => confirm(false)} className="px-4 py-2 rounded-md border border-zinc-300 text-[13px] hover:bg-zinc-50">Başka saat</button>
            </div>
          ) : <div className="text-[13px] text-zinc-400 px-1">Akış tamamlandı.</div>}
      </div>
    </div>
  )
}
