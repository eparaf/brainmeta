import { env } from "@/lib/env";
import { loginResponseSchema, type LoginResponse } from "./schemas";

/**
 * brainLogin calls the Go backend's POST /v1/auth/login. Runs server-side only
 * (from the Auth.js Credentials provider). Returns null on any failure so the
 * provider can reject the sign-in without leaking why.
 */
export async function brainLogin(
  email: string,
  password: string,
): Promise<LoginResponse | null> {
  try {
    const res = await fetch(`${env.BRAIN_API_URL}/v1/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
      cache: "no-store",
    });
    if (!res.ok) return null;
    const parsed = loginResponseSchema.safeParse(await res.json());
    return parsed.success ? parsed.data : null;
  } catch {
    return null;
  }
}
