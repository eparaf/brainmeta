"use client";

import { useState } from "react";
import { Loader2, MessageSquare, Search, Send } from "lucide-react";
import { useActiveClinic } from "@/stores/ui-store";
import { useConversation, useConversations, useSendMessage } from "@/lib/queries";
import { Card } from "@/components/ui/Card";
import { Badge, type Tone } from "@/components/ui/Badge";
import { Input } from "@/components/ui/Input";
import { Button } from "@/components/ui/Button";
import { QueryBoundary } from "@/components/ui/QueryBoundary";
import { cn, formatTRY, maskPhone } from "@/lib/utils";

function statusTone(s: string): Tone {
  if (s === "Randevu") return "emerald";
  if (s === "Düştü" || s === "Gelmedi") return "red";
  return "amber";
}

export default function SohbetlerPage() {
  const active = useActiveClinic();
  const clinicId = active?.id ?? null;
  const list = useConversations(clinicId);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [draft, setDraft] = useState("");
  const detail = useConversation(selectedId);
  const send = useSendMessage();

  const conversations = (list.data ?? []).filter(
    (c) =>
      c.name.toLowerCase().includes(search.toLowerCase()) ||
      c.qualification.segment.toLowerCase().includes(search.toLowerCase()),
  );
  const conv = detail.data;

  function onSend(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!draft.trim() || !conv || !clinicId) return;
    send.mutate(
      { phone: conv.phoneNumber, clinicId, armId: "", message: draft.trim() },
      {
        onSuccess: () => {
          setDraft("");
          detail.refetch();
        },
      },
    );
  }

  return (
    <div className="grid h-full grid-cols-1 gap-4 lg:grid-cols-[320px_1fr]">
      {/* Conversation list */}
      <Card className="flex min-h-0 flex-col">
        <div className="border-b border-zinc-100 p-3">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-zinc-400" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Sohbet ara…"
              className="pl-8"
            />
          </div>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto">
          <QueryBoundary
            isLoading={list.isLoading}
            error={list.error}
            isEmpty={!list.isLoading && conversations.length === 0}
            emptyText="Sohbet bulunamadı."
          >
            <div className="divide-y divide-zinc-100">
              {conversations.map((c) => (
                <button
                  key={c.id}
                  onClick={() => setSelectedId(c.id)}
                  className={cn(
                    "flex w-full flex-col items-start gap-1 px-3 py-2.5 text-left transition-colors hover:bg-zinc-50",
                    selectedId === c.id && "bg-zinc-50",
                  )}
                >
                  <div className="flex w-full items-center justify-between">
                    <span className="text-xs font-bold text-zinc-900">
                      {c.name || "İsimsiz"}
                    </span>
                    <Badge tone={statusTone(c.status)}>{c.status}</Badge>
                  </div>
                  <span className="text-[11px] text-zinc-500">
                    {c.qualification.segment || "—"} · {maskPhone(c.phoneNumber)}
                  </span>
                </button>
              ))}
            </div>
          </QueryBoundary>
        </div>
      </Card>

      {/* Detail */}
      <Card className="flex min-h-0 flex-col">
        {!conv ? (
          <div className="flex flex-1 flex-col items-center justify-center gap-2 text-zinc-400">
            <MessageSquare className="h-8 w-8" />
            <p className="text-xs">Soldan bir sohbet seç.</p>
          </div>
        ) : (
          <>
            <div className="flex items-center justify-between border-b border-zinc-100 p-4">
              <div>
                <div className="text-sm font-bold text-zinc-900">{conv.name || "İsimsiz"}</div>
                <div className="text-[11px] text-zinc-500">{maskPhone(conv.phoneNumber)}</div>
              </div>
              <div className="flex flex-wrap items-center justify-end gap-1.5">
                <Badge tone="blue">{conv.qualification.segment || "—"}</Badge>
                <Badge tone="zinc">Niyet %{conv.qualification.intentPct}</Badge>
                {conv.qualification.budgetTry > 0 && (
                  <Badge tone="zinc">{formatTRY(conv.qualification.budgetTry)}</Badge>
                )}
                {conv.qualification.booked && conv.qualification.appointmentTime && (
                  <Badge tone="emerald">{conv.qualification.appointmentTime}</Badge>
                )}
              </div>
            </div>
            <div className="min-h-0 flex-1 space-y-2 overflow-y-auto bg-zinc-50/50 p-4">
              {conv.messages.length === 0 ? (
                <p className="py-8 text-center text-[11px] text-zinc-400">
                  Mesaj geçmişi yok. Aşağıdan bir hasta mesajı gönder; ajan yanıtlasın.
                </p>
              ) : (
                conv.messages.map((m) => (
                  <div
                    key={m.id}
                    className={cn("flex", m.sender === "agent" ? "justify-end" : "justify-start")}
                  >
                    <div
                      className={cn(
                        "max-w-[75%] rounded-2xl px-3 py-2 text-xs",
                        m.sender === "agent"
                          ? "bg-zinc-900 text-white"
                          : "border border-zinc-200 bg-white text-zinc-800",
                      )}
                    >
                      {m.text}
                    </div>
                  </div>
                ))
              )}
            </div>
            <form onSubmit={onSend} className="flex items-center gap-2 border-t border-zinc-100 p-3">
              <Input
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                placeholder="Hasta mesajı yaz (ajan yanıtlar)…"
              />
              <Button type="submit" disabled={send.isPending || !draft.trim()}>
                {send.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Send className="h-4 w-4" />
                )}
              </Button>
            </form>
          </>
        )}
      </Card>
    </div>
  );
}
