"use client";

import { useState } from "react";
import { FileText, Loader2, Plus } from "lucide-react";
import { useActiveClinic } from "@/stores/ui-store";
import { useCreateTemplate, useTemplates } from "@/lib/queries";
import { Card } from "@/components/ui/Card";
import { Badge, type Tone } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Modal } from "@/components/ui/Modal";
import { QueryBoundary } from "@/components/ui/QueryBoundary";

function statusTone(s: string): Tone {
  if (s === "APPROVED") return "emerald";
  if (s === "REJECTED") return "red";
  return "amber";
}

const EMPTY = { name: "", category: "UTILITY", language: "tr", body: "" };

export default function SablonlarPage() {
  const active = useActiveClinic();
  const q = useTemplates();
  const create = useCreateTemplate();
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState(EMPTY);
  const templates = q.data ?? [];

  function submit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    create.mutate(
      { clinicId: active?.id ?? "", ...form },
      {
        onSuccess: () => {
          setOpen(false);
          setForm(EMPTY);
        },
      },
    );
  }

  return (
    <QueryBoundary isLoading={q.isLoading} error={q.error}>
      <div className="mb-4 flex justify-end">
        <Button onClick={() => setOpen(true)}>
          <Plus className="h-4 w-4" /> Yeni şablon
        </Button>
      </div>
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
        {templates.map((t) => (
          <Card key={t.id} className="flex flex-col p-4">
            <div className="flex items-start justify-between gap-2">
              <div className="flex items-center gap-2">
                <FileText className="h-4 w-4 shrink-0 text-zinc-400" />
                <span className="font-mono text-[11px] font-bold text-zinc-900">{t.name}</span>
              </div>
              <Badge tone={statusTone(t.status)}>{t.status}</Badge>
            </div>
            <div className="mt-2 flex gap-1.5">
              <Badge tone="zinc">{t.category}</Badge>
              <Badge tone="zinc">{t.language.toUpperCase()}</Badge>
            </div>
            <p className="mt-3 whitespace-pre-wrap rounded-lg bg-zinc-50 p-3 text-[11px] text-zinc-600">
              {t.body}
            </p>
          </Card>
        ))}
      </div>

      <Modal
        open={open}
        onOpenChange={setOpen}
        title="Yeni şablon oluştur"
        description="WhatsApp şablon taslağı — Meta onayına gider."
      >
        <form onSubmit={submit} className="space-y-3">
          <Input
            placeholder="sablon_adi"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />
          <div className="grid grid-cols-2 gap-2">
            <select
              className="rounded-lg border border-zinc-200 bg-white px-3 py-2 text-sm"
              value={form.category}
              onChange={(e) => setForm({ ...form, category: e.target.value })}
            >
              <option value="UTILITY">UTILITY</option>
              <option value="MARKETING">MARKETING</option>
            </select>
            <select
              className="rounded-lg border border-zinc-200 bg-white px-3 py-2 text-sm"
              value={form.language}
              onChange={(e) => setForm({ ...form, language: e.target.value })}
            >
              <option value="tr">TR</option>
              <option value="en">EN</option>
            </select>
          </div>
          <textarea
            className="w-full rounded-lg border border-zinc-200 px-3 py-2 text-sm focus:border-zinc-400 focus:outline-none focus:ring-2 focus:ring-zinc-200"
            rows={4}
            placeholder="Mesaj gövdesi — {{1}}, {{2}} değişkenleri"
            value={form.body}
            onChange={(e) => setForm({ ...form, body: e.target.value })}
          />
          {create.error && (
            <p className="text-[11px] text-red-500">{(create.error as Error).message}</p>
          )}
          <Button type="submit" disabled={create.isPending} className="w-full">
            {create.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : "Taslak oluştur"}
          </Button>
        </form>
      </Modal>
    </QueryBoundary>
  );
}
