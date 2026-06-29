import React, { useEffect, useState, useCallback } from 'react'
import { api } from '../api'
import { Icon } from '../icons.jsx'
import { Card, Bar, Stat, Badge, Toolbar, Avatar, money } from './ui.jsx'
import { clinicById } from '../data/conversations.js'

// ---- Klinikler (clinic management: guarantee health per clinic) ----
export function Clinics() {
  const [rows, setRows] = useState([]); const [err, setErr] = useState('')
  const load = useCallback(() => api.sla().then(setRows).catch((e) => setErr(e.message)), [])
  useEffect(() => { load() }, [load])
  return (
    <div>
      <Toolbar onRefresh={load} err={err} />
      <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {rows.map((r) => {
          const ratio = r.Guaranteed ? r.Delivered / r.Guaranteed : 0
          const name = clinicById(r.ClinicID)?.name || r.ClinicID
          return (
            <Card key={r.ClinicID}>
              <div className="flex items-center gap-3 mb-3">
                <Avatar name={name} size={34} />
                <div className="min-w-0">
                  <div className="text-[13.5px] font-medium truncate">{name}</div>
                  <div className="text-[11px] text-zinc-400">aylık garanti</div>
                </div>
                <div className="ml-auto"><Badge status={r.OnTrack ? 'ok' : 'behind'} /></div>
              </div>
              <div className="text-[26px] font-semibold tracking-tight">{r.Delivered}<span className="text-zinc-300 text-lg font-normal"> / {r.Guaranteed}</span></div>
              <div className="text-[11px] text-zinc-400 mb-3">nitelikli randevu (bu ay)</div>
              <Bar value={ratio} color={ratio >= 1 ? 'bg-emerald-500' : 'bg-zinc-900'} />
              <div className="flex justify-between mt-3 text-[12.5px]"><span className="text-zinc-400">gölge fiyat λ</span><span className="text-zinc-800 font-medium tabular-nums">{(r.ShadowPrice ?? 1).toFixed(2)}×</span></div>
            </Card>
          )
        })}
      </div>
    </div>
  )
}

// ---- Reklam Kolları ----
export function Arms() {
  const [rows, setRows] = useState([]); const [err, setErr] = useState('')
  const load = useCallback(() => api.arms().then(setRows).catch((e) => setErr(e.message)), [])
  useEffect(() => { load() }, [load])
  const maxTheta = Math.max(0.01, ...rows.map((r) => r.thetaHat || 0))
  return (
    <div>
      <Toolbar onRefresh={load} err={err} />
      <Card>
        <table className="w-full text-[13px]">
          <thead className="text-zinc-400 text-left text-[10.5px] uppercase tracking-wider">
            <tr>{['Kol', 'Segment', 'θ̂ kalite', 'CPL', 'Lead', 'Randevu', 'Harcama'].map((h, i) => <th key={i} className={`pb-2.5 font-medium ${h.startsWith('θ') ? 'w-44' : ''}`}>{h}</th>)}</tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.armId} className="border-t border-zinc-100">
                <td className="py-2.5 font-mono text-[11px] text-zinc-600">{r.armId}</td>
                <td className="text-zinc-500">{r.segment}</td>
                <td><div className="flex items-center gap-2"><div className="flex-1"><Bar value={r.thetaHat} max={maxTheta} /></div><span className="tabular-nums w-12 text-right text-zinc-700">{(r.thetaHat || 0).toFixed(3)}</span></div></td>
                <td className="tabular-nums text-zinc-700">{money(r.cpl)}</td>
                <td className="tabular-nums text-zinc-700">{r.leads}</td>
                <td className="tabular-nums text-zinc-700">{r.appts}</td>
                <td className="tabular-nums text-zinc-700">{money(r.spend)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <div className="text-[11px] text-zinc-400 mt-3">θ̂ = beynin öğrendiği "kaliteli lead getirme" olasılığı (Thompson sampling).</div>
      </Card>
    </div>
  )
}

// ---- Bütçe ----
export function Budget() {
  const [data, setData] = useState(null); const [err, setErr] = useState('')
  const load = useCallback(() => api.budget(30).then(setData).catch((e) => setErr(e.message)), [])
  useEffect(() => { load() }, [load])
  const allocs = (data?.allocations || []).filter((a) => a.DailyBudget > 0).sort((a, b) => b.DailyBudget - a.DailyBudget)
  const maxB = Math.max(1, ...allocs.map((a) => a.DailyBudget))
  return (
    <div>
      <Toolbar onRefresh={load} err={err} />
      <div className="grid sm:grid-cols-3 gap-4 mb-4">
        <Stat k="Ağ günlük bütçe" v={`${money(data?.networkDaily)} ₺`} />
        <Stat k="Gölge fiyat λ" v={(data?.lambda ?? 0).toFixed(3)} sub="değer / ₺" />
        <Stat k="Fonlanan kol" v={allocs.length} />
      </div>
      <Card>
        <div className="space-y-2.5">
          {allocs.map((a) => (
            <div key={a.ArmID} className="flex items-center gap-3 text-[13px]">
              <div className="w-56 font-mono text-[11px] truncate text-zinc-500">{a.ArmID}</div>
              <div className="flex-1"><Bar value={a.DailyBudget} max={maxB} /></div>
              <div className="w-24 text-right tabular-nums text-zinc-700">{money(a.DailyBudget)} ₺</div>
            </div>
          ))}
        </div>
        <div className="text-[11px] text-zinc-400 mt-3">Her klinik kendi bütçesini kendi platformlarına böler (passthrough).</div>
      </Card>
    </div>
  )
}

// ---- Takvim (stub) ----
export function Calendar() {
  return (
    <Card className="max-w-2xl p-6">
      <div className="flex items-center gap-2 text-zinc-900 font-medium mb-2"><Icon.calendar size={18} /> Takvim modülü — yakında</div>
      <p className="text-[13.5px] text-zinc-500 leading-relaxed">
        Gün/slot uygunluğu bu modülden gelecek. Randevu günü şu an beyindeki <code className="text-zinc-700 bg-zinc-100 px-1 rounded">reserveSlot</code> ile
        seçiliyor; takvim bağlandığında klinik müsaitliği buradan beslenecek. Bağlantı noktası hazır.
      </p>
    </Card>
  )
}
