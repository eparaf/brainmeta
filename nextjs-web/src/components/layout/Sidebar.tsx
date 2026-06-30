"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import {
  Building2,
  Calendar,
  ChevronDown,
  FileText,
  GitBranch,
  LayoutDashboard,
  Link2,
  MessageSquare,
  Radio,
  Users,
  Wallet,
  X,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useUIStore } from "@/stores/ui-store";
import { GoogleAdsIcon, MetaIcon, WhatsAppIcon } from "./Icons";

const NAV_GROUPS = [
  {
    title: "Platform",
    items: [
      { href: "/", label: "Genel Bakış", icon: LayoutDashboard },
      { href: "/sohbetler", label: "Sohbetler", icon: MessageSquare },
      { href: "/uyeler", label: "Üyeler", icon: Users },
      { href: "/calendar", label: "Takvim", icon: Calendar },
    ],
  },
  {
    title: "Yönetim",
    items: [
      { href: "/klinikler", label: "Klinikler", icon: Building2 },
      { href: "/reklam-kollari", label: "Reklam Kolları", icon: GitBranch },
      { href: "/butce", label: "Bütçe", icon: Wallet },
    ],
  },
  {
    title: "Sistem",
    items: [
      { href: "/sablonlar", label: "Şablonlar", icon: FileText },
      { href: "/baglantilar", label: "Bağlantılar", icon: Link2 },
    ],
  },
] as const;

export function Sidebar({ user }: { user: { name?: string | null; role?: string } }) {
  const pathname = usePathname();
  const clinics = useUIStore((s) => s.clinics);
  const activeClinicId = useUIStore((s) => s.activeClinicId);
  const setActiveClinicId = useUIStore((s) => s.setActiveClinicId);
  const mobileOpen = useUIStore((s) => s.mobileOpen);
  const setMobileOpen = useUIStore((s) => s.setMobileOpen);
  const [dropdownOpen, setDropdownOpen] = useState(false);

  const activeClinic = clinics.find((c) => c.id === activeClinicId) ?? null;
  const initials = (user.name ?? "AD").slice(0, 2).toUpperCase();
  const roleLabel = user.role === "admin" ? "Sistem Yöneticisi" : "Klinik Kullanıcısı";

  return (
    <>
      <aside
        className={cn(
          "fixed inset-y-0 left-0 z-50 flex w-64 flex-col border-r border-zinc-900/50 bg-zinc-950 text-zinc-300 shadow-2xl transition-transform duration-300 md:static md:h-screen md:translate-x-0 md:shadow-none",
          mobileOpen ? "translate-x-0" : "-translate-x-full",
        )}
      >
        {/* Logo */}
        <div className="flex items-center justify-between border-b border-zinc-900/50 p-5">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-zinc-100 to-zinc-400 text-zinc-950 shadow-md">
              <svg
                className="h-5 w-5"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2.5"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <circle cx="12" cy="5" r="2.5" />
                <circle cx="6" cy="18" r="2.5" />
                <circle cx="18" cy="18" r="2.5" />
                <line x1="12" y1="7.5" x2="6" y2="15.5" />
                <line x1="12" y1="7.5" x2="18" y2="15.5" />
                <line x1="6" y1="18" x2="18" y2="18" />
              </svg>
            </div>
            <div>
              <div className="text-[18px] font-bold leading-none tracking-tight text-white">
                BrainMeta
              </div>
              <div className="mt-1 text-[10px] font-bold uppercase tracking-widest text-zinc-500">
                Karar Konsolu
              </div>
            </div>
          </div>
          <button
            onClick={() => setMobileOpen(false)}
            className="rounded-md bg-zinc-900 p-1.5 text-zinc-500 transition-colors hover:bg-zinc-800 hover:text-white md:hidden"
            aria-label="Menüyü kapat"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Clinic switcher */}
        <div className="relative z-20 px-4 py-5">
          <button
            onClick={() => setDropdownOpen((v) => !v)}
            className="group flex w-full items-center justify-between rounded-xl border border-zinc-800/80 bg-zinc-900/60 px-3 py-2.5 text-left shadow-sm transition-all hover:border-zinc-700 hover:bg-zinc-800"
          >
            <div>
              <div className="mb-1 text-[9px] font-bold uppercase tracking-widest text-zinc-500">
                Seçili Klinik
              </div>
              <div className="flex items-center gap-2 text-xs font-bold text-zinc-100">
                <Radio className="h-3.5 w-3.5 text-emerald-400" />
                {activeClinic?.name ?? "Klinik seçin"}
              </div>
            </div>
            <div className="flex h-6 w-6 items-center justify-center rounded-md border border-zinc-800 bg-zinc-950 transition-colors group-hover:bg-zinc-800">
              <ChevronDown
                className={cn(
                  "h-3.5 w-3.5 text-zinc-400 transition-transform",
                  dropdownOpen && "rotate-180",
                )}
              />
            </div>
          </button>
          {dropdownOpen && (
            <>
              <div className="fixed inset-0 z-10" onClick={() => setDropdownOpen(false)} />
              <div className="absolute left-4 right-4 top-[calc(100%-8px)] z-20 overflow-hidden rounded-xl border border-zinc-800 bg-zinc-900/95 py-1.5 shadow-2xl backdrop-blur-xl">
                {clinics.length === 0 ? (
                  <div className="px-4 py-2.5 text-xs text-zinc-500">Klinik yok</div>
                ) : (
                  clinics.map((clinic) => (
                    <button
                      key={clinic.id}
                      onClick={() => {
                        setActiveClinicId(clinic.id);
                        setDropdownOpen(false);
                      }}
                      className={cn(
                        "flex w-full items-center justify-between px-4 py-2.5 text-xs transition-colors",
                        activeClinicId === clinic.id
                          ? "bg-zinc-800/80 font-bold text-white"
                          : "font-medium text-zinc-400 hover:bg-zinc-800/50 hover:text-zinc-200",
                      )}
                    >
                      <div className="flex items-center gap-2.5">
                        <Radio className="h-3.5 w-3.5 text-emerald-400" />
                        {clinic.name}
                      </div>
                    </button>
                  ))
                )}
              </div>
            </>
          )}
        </div>

        {/* Navigation */}
        <nav className="flex-1 space-y-6 overflow-y-auto px-3 pb-4">
          {NAV_GROUPS.map((group) => (
            <div key={group.title}>
              <h4 className="mb-2.5 px-3 text-[10px] font-bold uppercase tracking-[0.2em] text-zinc-600">
                {group.title}
              </h4>
              <div className="space-y-1">
                {group.items.map((item) => {
                  const isActive =
                    item.href === "/" ? pathname === "/" : pathname.startsWith(item.href);
                  const Icon = item.icon;
                  return (
                    <Link
                      key={item.href}
                      href={item.href}
                      onClick={() => setMobileOpen(false)}
                      className={cn(
                        "group flex w-full items-center gap-3 rounded-lg px-3 py-2 text-xs font-semibold transition-all",
                        isActive
                          ? "bg-zinc-800/80 text-white ring-1 ring-zinc-700/50"
                          : "text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200",
                      )}
                    >
                      <Icon
                        className={cn(
                          "h-4 w-4 transition-colors",
                          isActive ? "text-white" : "text-zinc-500 group-hover:text-zinc-400",
                        )}
                      />
                      {item.label}
                    </Link>
                  );
                })}
              </div>
            </div>
          ))}
        </nav>

        {/* Integrations (status wired in a later piece) */}
        <div className="mx-3 mb-3 rounded-xl border border-zinc-800/50 bg-zinc-900/40 px-4 py-3">
          <h4 className="mb-2 text-[9px] font-bold uppercase tracking-widest text-zinc-500">
            Bağlantılar
          </h4>
          <div className="flex items-center gap-2.5 opacity-60">
            <div className="flex h-6 w-6 items-center justify-center rounded-md bg-zinc-800 text-zinc-600">
              <WhatsAppIcon className="h-3.5 w-3.5" />
            </div>
            <div className="flex h-6 w-6 items-center justify-center rounded-md bg-zinc-800 text-zinc-600">
              <MetaIcon className="h-3.5 w-3.5" />
            </div>
            <div className="flex h-6 w-6 items-center justify-center rounded-md bg-zinc-800 text-zinc-600">
              <GoogleAdsIcon className="h-3.5 w-3.5" />
            </div>
          </div>
        </div>

        {/* User footer */}
        <div className="mt-auto border-t border-zinc-900/50 bg-zinc-950 p-4">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl border border-zinc-600/30 bg-gradient-to-tr from-zinc-800 to-zinc-700 text-xs font-bold text-zinc-200 shadow-sm">
              {initials}
            </div>
            <div className="min-w-0">
              <div className="truncate text-xs font-bold text-zinc-200">
                {user.name || "Kullanıcı"}
              </div>
              <div className="text-[10px] font-medium text-zinc-500">{roleLabel}</div>
            </div>
          </div>
        </div>
      </aside>

      {mobileOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/60 backdrop-blur-sm md:hidden"
          onClick={() => setMobileOpen(false)}
        />
      )}
    </>
  );
}
