import React, { useState, useMemo } from 'react'
import { api } from '../api'
import { Icon } from '../icons.jsx'
import { Avatar, Badge, money, pctv } from './ui.jsx'
import Intake from './Intake.jsx'
import { SEED, CLINICS, armOf, clinicById } from '../data/conversations.js'

let seq = 1000
const now = () => new Date().toLocaleTimeString('tr-TR', { hour: '2-digit', minute: '2-digit' })

export default function Conversations() {
  const [convs, setConvs] = useState(SEED)
  const [selId, setSelId] = useState(SEED[0].id)
  const [q, setQ] = useState('')
  const [input, setInput] = useState('')
  const [busy, setBusy] = useState(false)

  const sel = convs.find((c) => c.id === selId)
  const filtered = useMemo(() => {
    const s = q.toLowerCase()
    return convs.filter((c) => !s || c.name.toLowerCase().includes(s) || clinicById(c.clinicId)?.name.toLowerCase().includes(s))
  }, [convs, q])

  const patch = (id, fn) => setConvs((cs) => cs.map((c) => (c.id === id ? fn(c) : c)))

  const startNew = () => {
    const id = 'live-' + ++seq
    const phone = '+90 5' + Math.floor(10 + Math.random() * 89) + ' ' + Math.floor(100 + Math.random() * 899) + ' ' + Math.floor(10 + Math.random() * 89) + ' ' + Math.floor(10 + Math.random() * 89)
    setConvs((cs) => [{ id, name: 'Yeni Randevu', phone, clinicId: 'umraniye', channel: 'WhatsApp', time: 'şimdi', status: 'qualifying', live: true, kind: 'intake', messages: [], decision: null }, ...cs])
    setSelId(id)
  }

  const send = async () => {
    if (!input.trim() || !sel) return
    const text = input; setInput(''); setBusy(true)
    patch(sel.id, (c) => ({ ...c, messages: [...c.messages, { role: 'patient', text, t: now() }] }))
    try {
      const res = await api.whatsapp({ phone: sel.phone, clinicId: sel.clinicId, armId: armOf(clinicById(sel.clinicId)), message: text, hourOfDay: 14, distanceKm: 6 })
      patch(sel.id, (c) => ({
        ...c, time: 'şimdi',
        name: c.name === 'Yeni Sohbet' && res.qualification?.segment ? guessName(res.qualification.locale) : c.name,
        messages: [...c.messages, { role: 'agent', text: res.reply, t: now() }],
        decision: res, status: res.booked ? 'booked' : 'qualifying',
      }))
    } catch (e) {
      patch(sel.id, (c) => ({ ...c, messages: [...c.messages, { role: 'agent', text: 'Hata: ' + e.message, t: now() }] }))
    } finally { setBusy(false) }
  }

  return (
    <div className="flex h-[74vh] bg-white border border-zinc-200/80 rounded-xl shadow-[0_1px_2px_rgba(16,24,40,0.04)] overflow-hidden">
      {/* List */}
      <div className="w-[320px] shrink-0 border-r border-zinc-200 flex flex-col">
        <div className="p-3 border-b border-zinc-100 flex items-center gap-2">
          <div className="flex-1 flex items-center gap-2 bg-zinc-50 border border-zinc-200 rounded-md px-2.5 py-1.5">
            <Icon.search size={14} className="text-zinc-400" />
            <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Ara…" className="bg-transparent text-[13px] outline-none w-full" />
          </div>
          <button onClick={startNew} title="Yeni sohbet" className="h-8 w-8 grid place-items-center rounded-md bg-zinc-900 text-white hover:bg-zinc-800"><Icon.plus size={16} /></button>
        </div>
        <div className="flex-1 overflow-y-auto">
          {filtered.map((c) => {
            const last = c.messages[c.messages.length - 1]
            return (
              <button key={c.id} onClick={() => setSelId(c.id)}
                className={`w-full text-left px-3 py-2.5 flex gap-3 border-b border-zinc-50 ${selId === c.id ? 'bg-zinc-50' : 'hover:bg-zinc-50/60'}`}>
                <Avatar name={c.name} />
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <div className="text-[13px] font-medium truncate">{c.name}</div>
                    <span className="ml-auto text-[10.5px] text-zinc-400 shrink-0">{c.time}</span>
                  </div>
                  <div className="text-[12px] text-zinc-500 truncate">{last ? last.text : 'Yeni…'}</div>
                  <div className="flex items-center gap-1.5 mt-1">
                    <span className="text-[10.5px] text-zinc-400">{clinicById(c.clinicId)?.name}</span>
                    <span className="text-zinc-300">·</span>
                    <Badge status={c.status} />
                  </div>
                </div>
              </button>
            )
          })}
        </div>
      </div>

      {/* Thread */}
      {!sel ? <div className="flex-1 grid place-items-center text-zinc-400 text-sm">Bir sohbet seç</div>
        : sel.kind === 'intake' ? (
          <Intake key={sel.id} phone={sel.phone} onUpdate={(p) => patch(sel.id, (c) => ({ ...c, ...p }))} />
        ) : (
        <div className="flex-1 flex flex-col min-w-0">
          <div className="px-4 h-14 border-b border-zinc-100 flex items-center gap-3">
            <Avatar name={sel.name} size={32} />
            <div className="min-w-0">
              <div className="text-[13.5px] font-medium truncate">{sel.name}</div>
              <div className="text-[11px] text-zinc-400 truncate">{sel.phone} · {sel.channel}</div>
            </div>
            <div className="ml-auto"><Badge status={sel.status} /></div>
          </div>

          <div className="flex-1 overflow-y-auto p-4 space-y-3 bg-[#fbfbfc]">
            {sel.messages.length === 0 && <div className="text-zinc-400 text-[13px] text-center mt-12">Hasta gibi yaz — ajan niteler, beyin karar verir.</div>}
            {sel.messages.map((mm, i) => (
              <div key={i} className={`flex ${mm.role === 'patient' ? 'justify-end' : 'justify-start'}`}>
                <div className={`max-w-[76%] px-3.5 py-2 text-[13.5px] leading-snug rounded-2xl ${mm.role === 'patient' ? 'bg-zinc-900 text-white rounded-br-md' : 'bg-white border border-zinc-200 text-zinc-800 rounded-bl-md'}`}>
                  {mm.text}
                  {mm.t && <span className={`ml-2 text-[10px] align-baseline ${mm.role === 'patient' ? 'text-zinc-400' : 'text-zinc-300'}`}>{mm.t}</span>}
                </div>
              </div>
            ))}
          </div>

          {sel.decision && <Decision d={sel.decision} />}

          <div className="p-3 border-t border-zinc-100 flex gap-2">
            <input value={input} onChange={(e) => setInput(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && send()}
              placeholder="Mesaj yaz…" className="flex-1 bg-white border border-zinc-300 rounded-md px-3.5 py-2 text-[13.5px] outline-none focus:border-zinc-900" />
            <button onClick={send} disabled={busy} className="px-4 rounded-md bg-zinc-900 text-white text-[13px] font-medium hover:bg-zinc-800 disabled:opacity-40 flex items-center gap-1.5"><Icon.send size={15} />{busy ? '' : 'Gönder'}</button>
          </div>
        </div>
      )}
    </div>
  )
}

function Decision({ d }) {
  const q = d.qualification || {}
  return (
    <div className="px-4 py-2.5 border-t border-zinc-100 bg-white flex items-center gap-4 text-[12px] flex-wrap">
      <span className={`px-2 py-0.5 rounded-md border text-[11px] ${d.booked ? 'bg-emerald-50 text-emerald-700 border-emerald-200' : 'bg-amber-50 text-amber-700 border-amber-200'}`}>
        {d.booked ? '✓ ' + new Date(d.apptTime).toLocaleString('tr-TR', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' }) : 'niteleniyor'}
      </span>
      <Field k="segment" v={q.segment} />
      <Field k="niyet" v={pctv(q.intent)} />
      <Field k="aciliyet" v={pctv(q.urgency)} />
      <Field k="bütçe" v={q.budgetTry ? money(q.budgetTry) + ' ₺' : '—'} />
      <span className="ml-auto text-[10.5px] text-zinc-400">LLM önerir · beyin karar verir</span>
    </div>
  )
}
const Field = ({ k, v }) => <span className="text-zinc-400">{k} <span className="text-zinc-700 font-medium">{String(v ?? '—')}</span></span>

function guessName(locale) {
  const tr = ['Ahmet Yıldız', 'Fatma Çelik', 'Can Öztürk', 'Derya Kaya', 'Emre Şen']
  const en = ['Oliver Smith', 'Emma Brown', 'Liam Jones']
  const arr = locale === 'en' ? en : tr
  return arr[Math.floor(Math.random() * arr.length)]
}
