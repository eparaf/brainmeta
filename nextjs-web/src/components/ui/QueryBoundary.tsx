"use client";

import { Loader2 } from "lucide-react";
import { Card } from "./Card";

/** QueryBoundary renders consistent loading / error / empty states for a query. */
export function QueryBoundary({
  isLoading,
  error,
  isEmpty,
  emptyText,
  children,
}: {
  isLoading: boolean;
  error?: unknown;
  isEmpty?: boolean;
  emptyText?: string;
  children: React.ReactNode;
}) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center gap-2 py-16 text-xs text-zinc-400">
        <Loader2 className="h-4 w-4 animate-spin" /> Yükleniyor…
      </div>
    );
  }
  if (error) {
    return (
      <Card className="p-6 text-center text-xs text-red-500">
        Veri alınamadı: {error instanceof Error ? error.message : String(error)}
      </Card>
    );
  }
  if (isEmpty) {
    return (
      <Card className="p-12 text-center text-xs text-zinc-400">
        {emptyText ?? "Kayıt bulunamadı."}
      </Card>
    );
  }
  return <>{children}</>;
}
