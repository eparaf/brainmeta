"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { CheckCircle2, Link2, Settings, Sliders, XCircle } from "lucide-react";
import { useActiveClinic } from "@/stores/ui-store";
import { useConnections, useUpsertConnection } from "@/lib/queries";
import { Card } from "@/components/ui/Card";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Modal } from "@/components/ui/Modal";
import { QueryBoundary } from "@/components/ui/QueryBoundary";
import { GoogleAdsIcon, MetaIcon, WhatsAppIcon } from "@/components/layout/Icons";
import type { Connection } from "@/lib/brain/schemas";

const META: Record<
  string,
  { label: string; desc: string; icon: React.ComponentType<{ className?: string }>; oauth: boolean }
> = {
  whatsapp: {
    label: "WhatsApp Cloud API",
    desc: "Meta OAuth — gelen mesaj + şablon gönderimi",
    icon: WhatsAppIcon,
    oauth: true,
  },
  meta_ads: {
    label: "Meta Ads",
    desc: "Meta OAuth — Facebook & Instagram reklam senkronu",
    icon: MetaIcon,
    oauth: true,
  },
  google_ads: { label: "Google Ads", desc: "Google OAuth — Arama & Haritalar reklamları", icon: GoogleAdsIcon, oauth: true },
  web_form: {
    label: "Klinik Web Formu",
    desc: "Gömülebilir form + randevu takvimi widget'ı",
    icon: Link2,
    oauth: false,
  },
};

function noticeFromQuery(params: URLSearchParams): { ok: boolean; text: string } | null {
  if (params.get("connected")) {
    return { ok: true, text: `${params.get("connected")} bağlantısı kuruldu.` };
  }
  const err = params.get("error");
  if (!err) return null;
  const provider = params.get("provider");
  const map: Record<string, string> = {
    not_configured:
      provider === "google"
        ? "Google OAuth yapılandırılmamış — GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET gerekli."
        : "Meta OAuth yapılandırılmamış — META_APP_ID / META_APP_SECRET gerekli.",
    denied: "İzin reddedildi.",
    exchange_failed: "Token alışverişi başarısız oldu.",
    bad_state: "Geçersiz veya süresi geçmiş istek.",
    bad_request: "Eksik parametre.",
  };
  return { ok: false, text: map[err] ?? "Bağlantı kurulamadı." };
}

export default function BaglantilarPage() {
  const router = useRouter();
  const active = useActiveClinic();
  const clinicId = active?.id ?? null;
  const q = useConnections(clinicId);
  const upsert = useUpsertConnection(clinicId);
  const conns = q.data ?? [];

  const [notice, setNotice] = useState<{ ok: boolean; text: string } | null>(null);
  const [settings, setSettings] = useState<Connection | null>(null);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const n = noticeFromQuery(params);
    if (n) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setNotice(n);
      window.history.replaceState({}, "", window.location.pathname);
    }
  }, []);

  function connect(type: string, oauth: boolean) {
    if (!clinicId) return;
    if (oauth) {
      window.location.assign(
        `/api/connections/oauth/start?clinicId=${encodeURIComponent(clinicId)}&type=${encodeURIComponent(type)}`,
      );
    } else {
      upsert.mutate({ clinicId, type, connected: true });
    }
  }

  function openSettings(c: Connection) {
    if (c.type === "web_form") {
      router.push("/baglantilar/widget");
    } else {
      setSettings(c);
    }
  }

  return (
    <QueryBoundary isLoading={q.isLoading} error={q.error}>
      {notice && (
        <div
          className={
            "mb-4 flex items-center gap-2 rounded-lg border px-3 py-2 text-xs font-medium " +
            (notice.ok
              ? "border-emerald-200 bg-emerald-50 text-emerald-700"
              : "border-amber-200 bg-amber-50 text-amber-700")
          }
        >
          {notice.ok ? <CheckCircle2 className="h-4 w-4" /> : <XCircle className="h-4 w-4" />}
          {notice.text}
        </div>
      )}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        {conns.map((c) => {
          const meta = META[c.type];
          const Icon = meta?.icon ?? Link2;
          const isForm = c.type === "web_form";
          return (
            <Card key={c.id} className="flex items-center justify-between gap-2 p-4">
              <div className="flex min-w-0 items-center gap-3">
                <div
                  className={
                    "flex h-10 w-10 shrink-0 items-center justify-center rounded-lg " +
                    (c.connected ? "bg-emerald-50 text-emerald-600" : "bg-zinc-100 text-zinc-400")
                  }
                >
                  <Icon className="h-5 w-5" />
                </div>
                <div className="min-w-0">
                  <div className="truncate text-sm font-bold text-zinc-900">
                    {meta?.label ?? c.type}
                  </div>
                  <div className="truncate text-[11px] text-zinc-500">{c.detail || meta?.desc}</div>
                </div>
              </div>
              <div className="flex shrink-0 items-center gap-2">
                <Badge tone={c.connected ? "emerald" : "zinc"}>
                  {c.connected ? "Bağlı" : isForm ? "Hazır" : "Bağlı değil"}
                </Badge>
                <button
                  onClick={() => openSettings(c)}
                  className="rounded-lg border border-zinc-200 p-2 text-zinc-500 hover:bg-zinc-50 hover:text-zinc-900"
                  title={isForm ? "Widget'ı özelleştir" : "Ayarlar"}
                >
                  {isForm ? <Sliders className="h-4 w-4" /> : <Settings className="h-4 w-4" />}
                </button>
                {!isForm &&
                  (c.connected ? (
                    <Button
                      variant="secondary"
                      disabled={upsert.isPending}
                      onClick={() =>
                        clinicId && upsert.mutate({ clinicId, type: c.type, connected: false })
                      }
                    >
                      Kes
                    </Button>
                  ) : (
                    <Button disabled={upsert.isPending} onClick={() => connect(c.type, meta?.oauth ?? false)}>
                      Bağla
                    </Button>
                  ))}
                {isForm && (
                  <Button onClick={() => router.push("/baglantilar/widget")}>Özelleştir</Button>
                )}
              </div>
            </Card>
          );
        })}
      </div>

      <Modal
        open={!!settings}
        onOpenChange={(o) => !o && setSettings(null)}
        title={settings ? `${META[settings.type]?.label ?? settings.type} ayarları` : "Ayarlar"}
      >
        {settings && (
          <div className="space-y-3 text-xs">
            <div className="flex items-center justify-between">
              <span className="text-zinc-500">Durum</span>
              <Badge tone={settings.connected ? "emerald" : "zinc"}>
                {settings.connected ? "Bağlı" : "Bağlı değil"}
              </Badge>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-zinc-500">Hesap</span>
              <span className="font-medium text-zinc-800">{settings.detail || "—"}</span>
            </div>
            <p className="text-[11px] text-zinc-400">
              Sağlayıcı izinleri OAuth ile yönetilir. Kapsamları güncellemek için bağlantıyı kesip
              yeniden bağlayın.
            </p>
            {settings.connected && (
              <Button
                variant="secondary"
                className="w-full"
                onClick={() => {
                  if (clinicId)
                    upsert.mutate({ clinicId, type: settings.type, connected: false });
                  setSettings(null);
                }}
              >
                Bağlantıyı kes
              </Button>
            )}
          </div>
        )}
      </Modal>
    </QueryBoundary>
  );
}
