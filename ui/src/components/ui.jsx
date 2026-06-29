import React from 'react'
import { Icon } from '../icons.jsx'

export const money = (n) => Math.round(n || 0).toLocaleString('tr-TR')
export const pctv = (x) => (x == null ? '—' : Math.round(x * 100) + '%')

export function Card({ children, className = '', pad = 'p-5' }) {
  return <div className={`bg-white border border-zinc-200/80 rounded-xl shadow-[0_1px_2px_rgba(16,24,40,0.04)] ${pad} ${className}`}>{children}</div>
}

export function Bar({ value, max = 1, color = 'bg-zinc-900' }) {
  const pct = Math.max(0, Math.min(100, (value / (max || 1)) * 100))
  return <div className="h-1.5 rounded-full bg-zinc-100 overflow-hidden"><div className={`h-full ${color} rounded-full`} style={{ width: `${pct}%` }} /></div>
}

export function Stat({ k, v, sub }) {
  return (
    <Card pad="p-4">
      <div className="text-zinc-400 text-[11px] uppercase tracking-wide">{k}</div>
      <div className="text-[22px] font-semibold tracking-tight mt-1">{v}</div>
      {sub && <div className="text-[11px] text-zinc-400 mt-0.5">{sub}</div>}
    </Card>
  )
}

const STATUS = {
  booked: ['bg-emerald-50 text-emerald-700 border-emerald-200', 'Randevu'],
  qualifying: ['bg-amber-50 text-amber-700 border-amber-200', 'Niteleniyor'],
  lost: ['bg-zinc-100 text-zinc-500 border-zinc-200', 'Düştü'],
  noshow: ['bg-red-50 text-red-600 border-red-200', 'Gelmedi'],
  ok: ['bg-emerald-50 text-emerald-700 border-emerald-200', 'yolunda'],
  behind: ['bg-amber-50 text-amber-700 border-amber-200', 'geride'],
}
export function Badge({ status, children }) {
  const [cls, label] = STATUS[status] || ['bg-zinc-100 text-zinc-600 border-zinc-200', status]
  return <span className={`text-[11px] px-2 py-0.5 rounded-md border ${cls}`}>{children || label}</span>
}

const AV = ['bg-indigo-100 text-indigo-700', 'bg-emerald-100 text-emerald-700', 'bg-amber-100 text-amber-700', 'bg-sky-100 text-sky-700', 'bg-rose-100 text-rose-700', 'bg-violet-100 text-violet-700']
export function Avatar({ name, size = 36 }) {
  const initials = (name || '?').split(' ').map((w) => w[0]).slice(0, 2).join('').toUpperCase()
  let h = 0; for (const c of name || '') h = (h * 31 + c.charCodeAt(0)) >>> 0
  return <div className={`rounded-full flex items-center justify-center font-medium ${AV[h % AV.length]}`} style={{ width: size, height: size, fontSize: size * 0.36 }}>{initials}</div>
}

export function Toolbar({ onRefresh, err }) {
  return (
    <div className="flex items-center gap-3 mb-4">
      <button onClick={onRefresh} className="px-3 py-1.5 rounded-md border border-zinc-200 bg-white text-[12.5px] text-zinc-600 hover:bg-zinc-50 flex items-center gap-1.5"><Icon.refresh size={14} /> Yenile</button>
      {err && <span className="text-red-500 text-[12.5px]">Hata: {err} — backend çalışıyor mu?</span>}
    </div>
  )
}
