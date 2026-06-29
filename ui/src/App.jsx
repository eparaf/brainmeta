import React, { useEffect, useState } from 'react'
import { api } from './api'
import { Icon, Logo } from './icons.jsx'
import Conversations from './components/Conversations.jsx'
import { Clinics, Arms, Budget, Calendar } from './components/Panels.jsx'

const NAV = [
  { key: 'Sohbetler', icon: Icon.inbox, sub: 'WhatsApp / Instagram inbox' },
  { key: 'Klinikler', icon: Icon.building, sub: 'Klinik yönetimi & garanti' },
  { key: 'Reklam Kolları', icon: Icon.activity, sub: 'Öğrenilen kanal kalitesi' },
  { key: 'Bütçe', icon: Icon.wallet, sub: 'Klinik bazlı dağıtım' },
  { key: 'Takvim', icon: Icon.calendar, sub: 'Yakında', soon: true },
]

export default function App() {
  const [tab, setTab] = useState('Sohbetler')
  const [health, setHealth] = useState({ status: '...', agent: '' })
  useEffect(() => {
    api.health().then((h) => setHealth({ status: 'ok', agent: h.agent || '' }))
      .catch(() => setHealth({ status: 'down', agent: '' }))
  }, [])
  const meta = NAV.find((n) => n.key === tab) || NAV[0]

  return (
    <div className="flex h-full bg-[#f6f7f9]">
      <aside className="w-64 shrink-0 bg-white border-r border-zinc-200 flex flex-col">
        <div className="px-5 h-16 flex items-center gap-3 border-b border-zinc-200">
          <Logo />
          <div className="leading-tight">
            <div className="text-[15px] font-semibold tracking-tight">BrainMeta</div>
            <div className="text-[11px] text-zinc-400">karar konsolu</div>
          </div>
        </div>
        <nav className="flex-1 p-3 space-y-0.5">
          {NAV.map((n) => {
            const active = tab === n.key
            return (
              <button key={n.key} disabled={n.soon} onClick={() => !n.soon && setTab(n.key)}
                className={`w-full flex items-center gap-3 px-3 py-2 rounded-md text-[13.5px] transition
                  ${n.soon ? 'text-zinc-300 cursor-not-allowed'
                    : active ? 'bg-zinc-100 text-zinc-900 font-medium' : 'text-zinc-500 hover:bg-zinc-50 hover:text-zinc-800'}`}>
                <n.icon size={17} />
                <span>{n.key}</span>
                {n.soon && <span className="ml-auto text-[10px] px-1.5 py-0.5 rounded bg-zinc-100 text-zinc-400">yakında</span>}
              </button>
            )
          })}
        </nav>
        <div className="px-5 py-4 border-t border-zinc-200 text-xs">
          <div className="flex items-center gap-2">
            <span className={`h-2 w-2 rounded-full ${health.status === 'ok' ? 'bg-emerald-500' : 'bg-red-500'}`} />
            <span className="text-zinc-400">backend {health.status}</span>
          </div>
          {health.agent && (
            <div className="mt-1.5 flex items-center gap-1.5">
              <span className={`text-[10px] px-1.5 py-0.5 rounded font-medium ${health.agent.startsWith('mock') ? 'bg-zinc-100 text-zinc-500' : 'bg-emerald-50 text-emerald-700 border border-emerald-200'}`}>
                {health.agent}
              </span>
            </div>
          )}
        </div>
      </aside>

      <main className="flex-1 overflow-y-auto">
        <div className="px-8 py-7 max-w-6xl mx-auto">
          <div className="mb-6">
            <h1 className="text-[19px] font-semibold tracking-tight">{meta.key}</h1>
            <p className="text-[13px] text-zinc-400 mt-0.5">{meta.sub}</p>
          </div>
          {/* All panels stay MOUNTED — switching tabs only toggles visibility, so
              chat threads and fetched data never reset. */}
          <div className={tab === 'Sohbetler' ? '' : 'hidden'}><Conversations /></div>
          <div className={tab === 'Klinikler' ? '' : 'hidden'}><Clinics /></div>
          <div className={tab === 'Reklam Kolları' ? '' : 'hidden'}><Arms /></div>
          <div className={tab === 'Bütçe' ? '' : 'hidden'}><Budget /></div>
          <div className={tab === 'Takvim' ? '' : 'hidden'}><Calendar /></div>
        </div>
      </main>
    </div>
  )
}
