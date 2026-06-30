import type { ZodType } from "zod";

// Client-side fetch helpers that hit the same-origin /api/brain proxy and validate
// the response through a zod schema. Used by the TanStack Query hooks.

export async function jget<T>(path: string, schema: ZodType<T>): Promise<T> {
  const res = await fetch(path, { cache: "no-store" });
  const json = await res.json().catch(() => null);
  if (!res.ok) {
    throw new Error((json as { error?: string })?.error ?? `HTTP ${res.status}`);
  }
  return schema.parse(json);
}

export async function jpost<T>(
  path: string,
  body: unknown,
  schema: ZodType<T>,
): Promise<T> {
  const res = await fetch(path, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(body),
  });
  const json = await res.json().catch(() => null);
  if (!res.ok) {
    throw new Error((json as { error?: string })?.error ?? `HTTP ${res.status}`);
  }
  return schema.parse(json);
}

export async function jdelete(path: string): Promise<void> {
  const res = await fetch(path, { method: "DELETE" });
  if (!res.ok) {
    const json = await res.json().catch(() => null);
    throw new Error((json as { error?: string })?.error ?? `HTTP ${res.status}`);
  }
}
