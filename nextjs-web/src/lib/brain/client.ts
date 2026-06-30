import { auth } from "@/auth";
import { env } from "@/lib/env";

/** BrainError carries the HTTP status from a failed backend call. */
export class BrainError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
    this.name = "BrainError";
  }
}

type BrainInit = RequestInit & { token?: string };

/**
 * brainFetch calls the Go backend from the SERVER, attaching the current user's
 * JWT (from the Auth.js session) as a bearer token. The token never reaches the
 * browser. Pass an explicit `token` to bypass the session lookup. Data endpoints
 * are wired in later pieces; this is the shared client they build on.
 */
export async function brainFetch<T>(path: string, init: BrainInit = {}): Promise<T> {
  const { token: explicit, headers, ...rest } = init;
  let token = explicit;
  if (!token) {
    const session = await auth();
    token = session?.brainToken;
  }
  const res = await fetch(`${env.BRAIN_API_URL}${path}`, {
    ...rest,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...headers,
    },
    cache: "no-store",
  });
  if (!res.ok) {
    let message = res.statusText;
    try {
      const body = (await res.json()) as { error?: string };
      if (body?.error) message = body.error;
    } catch {
      /* non-JSON error body — keep statusText */
    }
    throw new BrainError(res.status, message);
  }
  return (await res.json()) as T;
}
