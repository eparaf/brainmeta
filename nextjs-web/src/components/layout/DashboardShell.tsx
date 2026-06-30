"use client";

import { useEffect } from "react";
import { usePathname } from "next/navigation";
import { Menu } from "lucide-react";
import { Sidebar } from "./Sidebar";
import { useActiveClinic, useUIStore } from "@/stores/ui-store";
import type { Clinic } from "@/lib/brain/schemas";

// Page header copy keyed by route — ported from the design's getPageHeaderInfo().
const PAGE_META: Record<string, { title: string; subtitle: string }> = {
  "/": {
    title: "Sistem Özeti",
    subtitle:
      "Klinik performansınız, bekleyen aksiyonlar ve Thompson algoritması özet durumu.",
  },
  "/sohbetler": {
    title: "Nitelikli WhatsApp Sohbetleri",
    subtitle: "Gelen hasta başvurularının otonom analizi ve anlık müdahale arayüzü.",
  },
  "/uyeler": {
    title: "Üye / Hasta Takip Kuyruğu",
    subtitle: "Tüm başvuruların anlık durumu, randevu takibi ve segmentasyon özeti.",
  },
  "/calendar": {
    title: "Klinik Takvimi",
    subtitle: "Yapay zekâ tarafından nitelendirilip onaylanmış hasta randevuları.",
  },
  "/klinikler": {
    title: "Klinik Taahhüt Matrisi",
    subtitle:
      "Kliniklerin aylık garanti hedefleri, güncel gerçekleşenler ve gölge fiyat optimizasyonu.",
  },
  "/reklam-kollari": {
    title: "Reklam Kolları Dağılım Analizi",
    subtitle:
      "Thompson sampling algoritmasıyla öğrenen segmenter ve performans olasılıkları.",
  },
  "/butce": {
    title: "Akıllı Bütçe Yönetimi",
    subtitle:
      "Gölge fiyat (λ) katsayısına bağlı olarak günlük bütçenin otonom dağılım simülasyonu.",
  },
  "/sablonlar": {
    title: "WhatsApp Mesaj Şablonları",
    subtitle: "Meta onaylı, değişken parametreli resmî bildirim ve pazarlama şablonları.",
  },
  "/baglantilar": {
    title: "Bağlantılar ve Entegrasyonlar",
    subtitle:
      "WhatsApp Cloud API, Meta Ads, Google Ads ve Klinik Web Siteleri anlık veri akışı.",
  },
  "/baglantilar/widget": {
    title: "Web Form & Takvim Widget'ı",
    subtitle:
      "Kliniğe özel, gömülebilir başvuru formu ve randevu takvimi — JS embed ile sitenize ekleyin.",
  },
};

const FALLBACK_META = {
  title: "BrainMeta Konsolu",
  subtitle: "Klinikler için otonom yapay zekâ karar ve yönetim paneli.",
};

export function DashboardShell({
  clinics,
  user,
  children,
}: {
  clinics: Clinic[];
  user: { name?: string | null; role?: string };
  children: React.ReactNode;
}) {
  const setClinics = useUIStore((s) => s.setClinics);
  const setMobileOpen = useUIStore((s) => s.setMobileOpen);
  const activeClinic = useActiveClinic();
  const pathname = usePathname();

  // Hydrate the store from the server-provided clinic list each load.
  useEffect(() => {
    setClinics(clinics);
  }, [clinics, setClinics]);

  const meta = PAGE_META[pathname] ?? FALLBACK_META;
  // The calendar owns its own chrome and scrolling — render it edge-to-edge.
  const fullBleed = pathname === "/calendar";

  return (
    <div className="flex min-h-screen bg-zinc-50 font-sans text-zinc-900 antialiased">
      <Sidebar user={user} />

      <div className="flex h-screen min-w-0 flex-1 flex-col overflow-hidden">
        {/* Mobile header */}
        <header className="flex shrink-0 items-center justify-between border-b border-zinc-200/80 bg-white px-4 py-3 md:hidden">
          <div className="flex items-center gap-2.5">
            <button
              onClick={() => setMobileOpen(true)}
              className="rounded border border-zinc-200 bg-zinc-50 p-1.5 text-zinc-600 hover:text-zinc-900"
              aria-label="Menüyü aç"
            >
              <Menu className="h-5 w-5" />
            </button>
            <span className="text-sm font-bold tracking-tight text-zinc-900">BrainMeta</span>
          </div>
          <div className="flex items-center gap-1.5 rounded-full border border-zinc-200 bg-zinc-50 px-2.5 py-1 text-[10px] font-semibold text-zinc-700">
            <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
            <span className="max-w-[120px] truncate">{activeClinic?.name ?? "—"}</span>
          </div>
        </header>

        {/* Workspace */}
        {fullBleed ? (
          <div className="min-h-0 flex-1 overflow-hidden bg-white">{children}</div>
        ) : (
          <div className="flex flex-1 flex-col items-center overflow-y-auto bg-zinc-50/50">
            <div className="mx-auto flex h-full w-full max-w-7xl flex-col p-4 sm:p-6 lg:p-8">
              <div className="mb-6 flex w-full shrink-0 items-end justify-between border-b border-zinc-200/80 pb-5">
                <div className="text-left">
                  <h1 className="text-2xl font-bold tracking-tight text-zinc-950 sm:text-3xl">
                    {meta.title}
                  </h1>
                  <p className="mt-2 text-sm font-medium text-zinc-500">{meta.subtitle}</p>
                </div>
              </div>
              <div className="min-h-0 w-full flex-1">{children}</div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
