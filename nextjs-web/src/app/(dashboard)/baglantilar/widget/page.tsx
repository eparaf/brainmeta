"use client";

import { useState } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  Check,
  Copy,
  Loader2,
  Pencil,
  Plus,
  RefreshCw,
  Save,
  Stethoscope,
  Trash2,
} from "lucide-react";
import { useActiveClinic } from "@/stores/ui-store";
import {
  useDeleteDoctor,
  useDeleteService,
  useDoctors,
  useRotateWidgetKey,
  useSaveDoctor,
  useSaveService,
  useSaveWidget,
  useServices,
  useWidget,
} from "@/lib/queries";
import { Card } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Badge } from "@/components/ui/Badge";
import { Modal } from "@/components/ui/Modal";
import { QueryBoundary } from "@/components/ui/QueryBoundary";
import type { Doctor, Service, WidgetConfig } from "@/lib/brain/schemas";
import { cn } from "@/lib/utils";

const PUBLIC_API = process.env.NEXT_PUBLIC_BRAIN_PUBLIC_URL || "http://localhost:8080";
const TABS = ["Form", "Takvim", "Doktorlar", "Hizmetler", "Embed"] as const;
const WEEKDAYS = [
  { n: 1, l: "Pzt" },
  { n: 2, l: "Sal" },
  { n: 3, l: "Çar" },
  { n: 4, l: "Per" },
  { n: 5, l: "Cum" },
  { n: 6, l: "Cmt" },
  { n: 7, l: "Paz" },
];

function emptyDoctor(clinicId: string): Doctor {
  return { id: "", clinicId, name: "", title: "Dt.", specialty: "", active: true, days: [1, 2, 3, 4, 5], startHour: 9, endHour: 17, slotMins: 30 };
}
function emptyService(clinicId: string): Service {
  return { id: "", clinicId, name: "", durationMins: 30, doctorIds: [], active: true };
}

function CopyButton({ text }: { text: string }) {
  const [done, setDone] = useState(false);
  return (
    <button
      type="button"
      onClick={() =>
        navigator.clipboard.writeText(text).then(() => {
          setDone(true);
          setTimeout(() => setDone(false), 1500);
        })
      }
      className="inline-flex items-center gap-1 rounded-md border border-zinc-200 bg-white px-2 py-1 text-[10px] font-semibold text-zinc-600 hover:bg-zinc-50"
    >
      {done ? <Check className="h-3 w-3 text-emerald-600" /> : <Copy className="h-3 w-3" />}
      {done ? "Kopyalandı" : "Kopyala"}
    </button>
  );
}

function FieldRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="mb-1 block text-[11px] font-semibold text-zinc-600">{label}</label>
      {children}
    </div>
  );
}

function Preview({ url }: { url: string }) {
  const [nonce, setNonce] = useState(0);
  return (
    <Card className="space-y-2 p-4">
      <div className="flex items-center justify-between">
        <h4 className="text-[11px] font-bold uppercase tracking-wider text-zinc-500">Canlı önizleme</h4>
        <button
          onClick={() => setNonce((n) => n + 1)}
          className="inline-flex items-center gap-1 text-[10px] font-semibold text-zinc-500 hover:text-zinc-900"
        >
          <RefreshCw className="h-3 w-3" /> Yenile
        </button>
      </div>
      <iframe key={nonce} title="preview" srcDoc={url} className="h-[480px] w-full rounded-lg border border-zinc-200" />
    </Card>
  );
}

export default function WidgetStudioPage() {
  const active = useActiveClinic();
  const clinicId = active?.id ?? null;
  const widget = useWidget(clinicId);
  const saveWidget = useSaveWidget(clinicId);
  const rotate = useRotateWidgetKey(clinicId);
  const doctors = useDoctors(clinicId);
  const services = useServices(clinicId);
  const saveDoctor = useSaveDoctor(clinicId);
  const delDoctor = useDeleteDoctor(clinicId);
  const saveService = useSaveService(clinicId);
  const delService = useDeleteService(clinicId);

  const [tab, setTab] = useState<(typeof TABS)[number]>("Form");
  const [draft, setDraft] = useState<WidgetConfig | null>(null);
  const [docDraft, setDocDraft] = useState<Doctor | null>(null);
  const [svcDraft, setSvcDraft] = useState<Service | null>(null);
  const cfg = draft ?? widget.data ?? null;

  function patch(p: Partial<WidgetConfig>) {
    if (cfg) setDraft({ ...cfg, ...p });
  }

  const previewDoc = (m: string) =>
    cfg
      ? `<!doctype html><html><head><meta charset="utf-8"><style>body{margin:0;padding:16px;background:#f4f4f5}</style></head><body><script src="${PUBLIC_API}/embed/widget.js" data-key="${cfg.publicKey}" data-mode="${m}"></script></body></html>`
      : "";
  const snippet = (m: string) =>
    `<script src="${PUBLIC_API}/embed/widget.js"\n        data-key="${cfg?.publicKey ?? ""}" data-mode="${m}"></script>`;

  return (
    <QueryBoundary isLoading={widget.isLoading} error={widget.error}>
      <div className="mb-4 flex items-center justify-between">
        <Link href="/baglantilar" className="inline-flex items-center gap-1.5 text-xs font-semibold text-zinc-500 hover:text-zinc-900">
          <ArrowLeft className="h-3.5 w-3.5" /> Bağlantılar
        </Link>
        {(tab === "Form" || tab === "Takvim") && (
          <Button onClick={() => cfg && saveWidget.mutate(cfg)} disabled={saveWidget.isPending || !cfg}>
            {saveWidget.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
            Kaydet
          </Button>
        )}
      </div>

      {/* Tabs */}
      <div className="mb-5 flex gap-1 border-b border-zinc-200">
        {TABS.map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={cn(
              "-mb-px border-b-2 px-3.5 py-2 text-xs font-semibold transition-colors",
              tab === t
                ? "border-zinc-900 text-zinc-900"
                : "border-transparent text-zinc-400 hover:text-zinc-700",
            )}
          >
            {t}
          </button>
        ))}
      </div>

      {cfg && (
        <>
          {tab === "Form" && (
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
              <Card className="space-y-3 p-5">
                <FieldRow label="Başlık">
                  <Input value={cfg.formTitle} onChange={(e) => patch({ formTitle: e.target.value })} />
                </FieldRow>
                <FieldRow label="Alt başlık">
                  <Input value={cfg.formSubtitle} onChange={(e) => patch({ formSubtitle: e.target.value })} />
                </FieldRow>
                <FieldRow label="Başarı mesajı">
                  <Input value={cfg.successText} onChange={(e) => patch({ successText: e.target.value })} />
                </FieldRow>
                <div className="flex items-center gap-3">
                  <label className="text-[11px] font-semibold text-zinc-600">Ana renk</label>
                  <input type="color" value={cfg.primaryColor} onChange={(e) => patch({ primaryColor: e.target.value })} className="h-8 w-12 cursor-pointer rounded border border-zinc-200" />
                  <span className="font-mono text-xs text-zinc-500">{cfg.primaryColor}</span>
                </div>
                <div className="space-y-2 border-t border-zinc-100 pt-3">
                  <div className="text-[11px] font-bold uppercase tracking-wider text-zinc-500">Alanlar</div>
                  {cfg.fields.map((f, i) => (
                    <div key={f.key} className="flex items-center gap-2">
                      <Input
                        value={f.label}
                        onChange={(e) => {
                          const fields = [...cfg.fields];
                          fields[i] = { ...f, label: e.target.value };
                          patch({ fields });
                        }}
                        className="flex-1"
                      />
                      <label className="flex items-center gap-1 text-[10px] text-zinc-500">
                        <input type="checkbox" checked={f.enabled} onChange={(e) => { const fields = [...cfg.fields]; fields[i] = { ...f, enabled: e.target.checked }; patch({ fields }); }} /> Açık
                      </label>
                      <label className="flex items-center gap-1 text-[10px] text-zinc-500">
                        <input type="checkbox" checked={f.required} onChange={(e) => { const fields = [...cfg.fields]; fields[i] = { ...f, required: e.target.checked }; patch({ fields }); }} /> Zorunlu
                      </label>
                    </div>
                  ))}
                </div>
              </Card>
              <Preview url={previewDoc("form")} />
            </div>
          )}

          {tab === "Takvim" && (
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
              <Card className="space-y-3 p-5">
                <FieldRow label="Takvim başlığı">
                  <Input value={cfg.calendarTitle} onChange={(e) => patch({ calendarTitle: e.target.value })} />
                </FieldRow>
                <FieldRow label="Alt başlık / açıklama">
                  <Input value={cfg.calendarSubtitle} onChange={(e) => patch({ calendarSubtitle: e.target.value })} />
                </FieldRow>
                <FieldRow label="Onay mesajı">
                  <Input value={cfg.confirmText} onChange={(e) => patch({ confirmText: e.target.value })} />
                </FieldRow>
                <div className="flex items-center gap-3">
                  <label className="text-[11px] font-semibold text-zinc-600">Takvim rengi</label>
                  <input type="color" value={cfg.calendarColor} onChange={(e) => patch({ calendarColor: e.target.value })} className="h-8 w-12 cursor-pointer rounded border border-zinc-200" />
                  <span className="font-mono text-xs text-zinc-500">{cfg.calendarColor}</span>
                </div>
                <div className="flex items-center gap-3">
                  <label className="text-[11px] font-semibold text-zinc-600">Tema</label>
                  <div className="inline-flex rounded-lg bg-zinc-100 p-0.5">
                    {[
                      { v: "dark", l: "Koyu" },
                      { v: "light", l: "Açık" },
                    ].map((o) => (
                      <button
                        key={o.v}
                        onClick={() => patch({ theme: o.v })}
                        className={cn(
                          "rounded-md px-3 py-1 text-[11px] font-semibold transition-colors",
                          cfg.theme === o.v ? "bg-white text-zinc-900 shadow-sm" : "text-zinc-500",
                        )}
                      >
                        {o.l}
                      </button>
                    ))}
                  </div>
                </div>
                <label className="flex items-center gap-2 border-t border-zinc-100 pt-3 text-xs text-zinc-600">
                  <input type="checkbox" checked={cfg.recommend} onChange={(e) => patch({ recommend: e.target.checked })} />
                  <span>
                    <b>AI öneri</b> — boşluğa göre en uygun hekim & saati öne çıkar (brain no-show motoru)
                  </span>
                </label>
                <p className="text-[11px] text-zinc-400">
                  Takvimdeki hizmet ve hekimler <b>Hizmetler</b> ve <b>Doktorlar</b> sekmelerinden yönetilir.
                </p>
              </Card>
              <Preview url={previewDoc("calendar")} />
            </div>
          )}

          {tab === "Doktorlar" && (
            <div className="space-y-3">
              <div className="flex justify-end">
                <Button onClick={() => setDocDraft(emptyDoctor(clinicId ?? ""))}>
                  <Plus className="h-4 w-4" /> Hekim ekle
                </Button>
              </div>
              {(doctors.data ?? []).length === 0 ? (
                <Card className="p-10 text-center text-xs text-zinc-400">Henüz hekim yok.</Card>
              ) : (
                <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
                  {(doctors.data ?? []).map((d) => (
                    <Card key={d.id} className="flex items-center justify-between p-4">
                      <div className="flex items-center gap-3">
                        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-teal-50 text-teal-600">
                          <Stethoscope className="h-4 w-4" />
                        </div>
                        <div>
                          <div className="text-sm font-bold text-zinc-900">
                            {d.title} {d.name}
                          </div>
                          <div className="text-[11px] text-zinc-500">
                            {d.specialty || "—"} · {d.startHour}:00–{d.endHour}:00 · {d.slotMins}dk
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-1.5">
                        {!d.active && <Badge tone="zinc">Pasif</Badge>}
                        <button onClick={() => setDocDraft(d)} className="rounded-md border border-zinc-200 p-1.5 text-zinc-500 hover:bg-zinc-50">
                          <Pencil className="h-3.5 w-3.5" />
                        </button>
                        <button onClick={() => delDoctor.mutate(d.id)} className="rounded-md border border-zinc-200 p-1.5 text-red-500 hover:bg-red-50">
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </Card>
                  ))}
                </div>
              )}
            </div>
          )}

          {tab === "Hizmetler" && (
            <div className="space-y-3">
              <div className="flex justify-end">
                <Button onClick={() => setSvcDraft(emptyService(clinicId ?? ""))}>
                  <Plus className="h-4 w-4" /> Hizmet ekle
                </Button>
              </div>
              {(services.data ?? []).length === 0 ? (
                <Card className="p-10 text-center text-xs text-zinc-400">Henüz hizmet yok.</Card>
              ) : (
                <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
                  {(services.data ?? []).map((svc) => (
                    <Card key={svc.id} className="flex items-center justify-between p-4">
                      <div>
                        <div className="text-sm font-bold text-zinc-900">{svc.name}</div>
                        <div className="text-[11px] text-zinc-500">
                          {svc.durationMins} dk · {svc.doctorIds.length} hekim
                        </div>
                      </div>
                      <div className="flex items-center gap-1.5">
                        {!svc.active && <Badge tone="zinc">Pasif</Badge>}
                        <button onClick={() => setSvcDraft(svc)} className="rounded-md border border-zinc-200 p-1.5 text-zinc-500 hover:bg-zinc-50">
                          <Pencil className="h-3.5 w-3.5" />
                        </button>
                        <button onClick={() => delService.mutate(svc.id)} className="rounded-md border border-zinc-200 p-1.5 text-red-500 hover:bg-red-50">
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </Card>
                  ))}
                </div>
              )}
            </div>
          )}

          {tab === "Embed" && (
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
              <div className="space-y-4">
                <Card className="space-y-3 p-5">
                  <h3 className="text-xs font-bold uppercase tracking-wider text-zinc-500">Genel API anahtarı</h3>
                  <div className="flex items-center gap-2">
                    <code className="flex-1 overflow-x-auto rounded-lg bg-zinc-50 px-3 py-2 font-mono text-xs text-zinc-700">{cfg.publicKey}</code>
                    <CopyButton text={cfg.publicKey} />
                    <Button variant="secondary" disabled={rotate.isPending} onClick={() => rotate.mutate(undefined, { onSuccess: () => setDraft(null) })} title="Yenile">
                      <RefreshCw className={cn("h-3.5 w-3.5", rotate.isPending && "animate-spin")} />
                    </Button>
                  </div>
                  <p className="text-[10px] text-zinc-400">Yayınlanabilir anahtar — sitenize gömülür, yalnızca başvuru/randevu gönderebilir.</p>
                </Card>
                <Card className="space-y-3 p-5">
                  <h3 className="text-xs font-bold uppercase tracking-wider text-zinc-500">Embed kodu</h3>
                  {[
                    { l: "İletişim formu", m: "form" },
                    { l: "Randevu takvimi", m: "calendar" },
                  ].map((x) => (
                    <div key={x.m}>
                      <div className="mb-1 flex items-center justify-between">
                        <span className="text-[11px] font-semibold text-zinc-600">{x.l}</span>
                        <CopyButton text={snippet(x.m)} />
                      </div>
                      <pre className="overflow-x-auto rounded-lg bg-zinc-950 p-3 font-mono text-[10px] leading-relaxed text-zinc-100">{snippet(x.m)}</pre>
                    </div>
                  ))}
                </Card>
              </div>
              <div className="space-y-4">
                <Preview url={previewDoc("form")} />
                <Preview url={previewDoc("calendar")} />
              </div>
            </div>
          )}
        </>
      )}

      {/* Doctor editor */}
      <Modal open={!!docDraft} onOpenChange={(o) => !o && setDocDraft(null)} title={docDraft?.id ? "Hekimi düzenle" : "Yeni hekim"}>
        {docDraft && (
          <div className="space-y-3">
            <div className="grid grid-cols-3 gap-2">
              <Input placeholder="Ünvan" value={docDraft.title} onChange={(e) => setDocDraft({ ...docDraft, title: e.target.value })} />
              <Input placeholder="Ad Soyad" className="col-span-2" value={docDraft.name} onChange={(e) => setDocDraft({ ...docDraft, name: e.target.value })} />
            </div>
            <Input placeholder="Uzmanlık (ör. İmplantoloji)" value={docDraft.specialty} onChange={(e) => setDocDraft({ ...docDraft, specialty: e.target.value })} />
            <div>
              <label className="mb-1 block text-[11px] font-semibold text-zinc-600">Çalışma günleri</label>
              <div className="flex gap-1">
                {WEEKDAYS.map((d) => {
                  const on = docDraft.days.includes(d.n);
                  return (
                    <button
                      key={d.n}
                      type="button"
                      onClick={() =>
                        setDocDraft({
                          ...docDraft,
                          days: on ? docDraft.days.filter((x) => x !== d.n) : [...docDraft.days, d.n].sort(),
                        })
                      }
                      className={cn(
                        "flex-1 rounded-md border py-1.5 text-[11px] font-semibold",
                        on ? "border-zinc-900 bg-zinc-900 text-white" : "border-zinc-200 text-zinc-500",
                      )}
                    >
                      {d.l}
                    </button>
                  );
                })}
              </div>
            </div>
            <div className="grid grid-cols-3 gap-2">
              <FieldRow label="Başlangıç">
                <Input type="number" value={docDraft.startHour} onChange={(e) => setDocDraft({ ...docDraft, startHour: Number(e.target.value) })} />
              </FieldRow>
              <FieldRow label="Bitiş">
                <Input type="number" value={docDraft.endHour} onChange={(e) => setDocDraft({ ...docDraft, endHour: Number(e.target.value) })} />
              </FieldRow>
              <FieldRow label="Slot (dk)">
                <Input type="number" value={docDraft.slotMins} onChange={(e) => setDocDraft({ ...docDraft, slotMins: Number(e.target.value) })} />
              </FieldRow>
            </div>
            <label className="flex items-center gap-2 text-xs text-zinc-600">
              <input type="checkbox" checked={docDraft.active} onChange={(e) => setDocDraft({ ...docDraft, active: e.target.checked })} /> Aktif
            </label>
            <Button
              className="w-full"
              disabled={saveDoctor.isPending || !docDraft.name.trim()}
              onClick={() => saveDoctor.mutate(docDraft, { onSuccess: () => setDocDraft(null) })}
            >
              {saveDoctor.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : "Kaydet"}
            </Button>
          </div>
        )}
      </Modal>

      {/* Service editor */}
      <Modal open={!!svcDraft} onOpenChange={(o) => !o && setSvcDraft(null)} title={svcDraft?.id ? "Hizmeti düzenle" : "Yeni hizmet"}>
        {svcDraft && (
          <div className="space-y-3">
            <Input placeholder="Hizmet adı (ör. İmplant Muayenesi)" value={svcDraft.name} onChange={(e) => setSvcDraft({ ...svcDraft, name: e.target.value })} />
            <FieldRow label="Süre (dk)">
              <Input type="number" value={svcDraft.durationMins} onChange={(e) => setSvcDraft({ ...svcDraft, durationMins: Number(e.target.value) })} />
            </FieldRow>
            <div>
              <label className="mb-1 block text-[11px] font-semibold text-zinc-600">Bu hizmeti veren hekimler</label>
              <div className="space-y-1.5">
                {(doctors.data ?? []).length === 0 && <p className="text-[11px] text-zinc-400">Önce hekim ekleyin.</p>}
                {(doctors.data ?? []).map((d) => {
                  const on = svcDraft.doctorIds.includes(d.id);
                  return (
                    <label key={d.id} className="flex items-center gap-2 text-xs text-zinc-700">
                      <input
                        type="checkbox"
                        checked={on}
                        onChange={(e) =>
                          setSvcDraft({
                            ...svcDraft,
                            doctorIds: e.target.checked ? [...svcDraft.doctorIds, d.id] : svcDraft.doctorIds.filter((x) => x !== d.id),
                          })
                        }
                      />
                      {d.title} {d.name}
                    </label>
                  );
                })}
              </div>
            </div>
            <label className="flex items-center gap-2 text-xs text-zinc-600">
              <input type="checkbox" checked={svcDraft.active} onChange={(e) => setSvcDraft({ ...svcDraft, active: e.target.checked })} /> Aktif
            </label>
            <Button
              className="w-full"
              disabled={saveService.isPending || !svcDraft.name.trim()}
              onClick={() => saveService.mutate(svcDraft, { onSuccess: () => setSvcDraft(null) })}
            >
              {saveService.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : "Kaydet"}
            </Button>
          </div>
        )}
      </Modal>
    </QueryBoundary>
  );
}
