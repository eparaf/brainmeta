// Minimal stroke icons (Lucide-style), inline so there's no dependency. Modern,
// monochrome, currentColor — nothing emoji / chatbot-cliché.
import React from 'react'

function S({ children, size = 18, className = '' }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" className={className}>
      {children}
    </svg>
  )
}

export const Icon = {
  chat: (p) => <S {...p}><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" /></S>,
  shield: (p) => <S {...p}><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" /><path d="m9 12 2 2 4-4" /></S>,
  activity: (p) => <S {...p}><path d="M22 12h-4l-3 9L9 3l-3 9H2" /></S>,
  wallet: (p) => <S {...p}><path d="M3 7a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v3" /><path d="M3 7v10a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-4" /><path d="M16 13h5" /></S>,
  calendar: (p) => <S {...p}><rect x="3" y="4" width="18" height="18" rx="2" /><path d="M16 2v4M8 2v4M3 10h18" /></S>,
  refresh: (p) => <S {...p}><path d="M3 12a9 9 0 0 1 15-6.7L21 8" /><path d="M21 3v5h-5" /><path d="M21 12a9 9 0 0 1-15 6.7L3 16" /><path d="M3 21v-5h5" /></S>,
  send: (p) => <S {...p}><path d="M22 2 11 13" /><path d="M22 2 15 22l-4-9-9-4z" /></S>,
  inbox: (p) => <S {...p}><path d="M22 12h-6l-2 3h-4l-2-3H2" /><path d="M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z" /></S>,
  building: (p) => <S {...p}><rect x="4" y="2" width="16" height="20" rx="2" /><path d="M9 22v-4h6v4M9 6h.01M15 6h.01M9 10h.01M15 10h.01M9 14h.01M15 14h.01" /></S>,
  plus: (p) => <S {...p}><path d="M12 5v14M5 12h14" /></S>,
  search: (p) => <S {...p}><circle cx="11" cy="11" r="7" /><path d="m21 21-4.3-4.3" /></S>,
}

// Brand mark: a small connected-node glyph in a rounded square — "meta/network",
// not a brain emoji.
export function Logo({ size = 30 }) {
  return (
    <div className="rounded-md bg-zinc-900 flex items-center justify-center" style={{ width: size, height: size }}>
      <svg width={size * 0.6} height={size * 0.6} viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round">
        <circle cx="6" cy="6" r="2" /><circle cx="18" cy="7" r="2" /><circle cx="12" cy="18" r="2" />
        <path d="M7.7 7.3 10.7 16M16.6 8.5 13.2 16.7M8 6.4h8" />
      </svg>
    </div>
  )
}
