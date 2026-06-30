import { createHmac, timingSafeEqual } from "node:crypto";

// Server-only OAuth2 helper for connecting clinics to Meta (WhatsApp + Ads) and
// Google (Ads). App credentials come from env; the flow stays dormant (and the UI
// shows "not configured") until they're set. Never import this from a client file.

export const APP_URL = process.env.NEXT_PUBLIC_APP_URL || "http://localhost:3002";
export const REDIRECT_URI = `${APP_URL}/api/connections/oauth/callback`;

export type Provider = "meta" | "google";

interface ProviderConfig {
  clientId: string;
  clientSecret: string;
  authorizeUrl: string;
  tokenUrl: string;
  scopes: Record<string, string>; // connection type → scope string
  extraAuthorizeParams?: Record<string, string>;
}

export const PROVIDERS: Record<Provider, ProviderConfig> = {
  meta: {
    clientId: process.env.META_APP_ID || "",
    clientSecret: process.env.META_APP_SECRET || "",
    authorizeUrl: "https://www.facebook.com/v21.0/dialog/oauth",
    tokenUrl: "https://graph.facebook.com/v21.0/oauth/access_token",
    scopes: {
      whatsapp:
        "whatsapp_business_management,whatsapp_business_messaging,business_management",
      meta_ads: "ads_management,ads_read,business_management",
    },
  },
  google: {
    clientId: process.env.GOOGLE_CLIENT_ID || "",
    clientSecret: process.env.GOOGLE_CLIENT_SECRET || "",
    authorizeUrl: "https://accounts.google.com/o/oauth2/v2/auth",
    tokenUrl: "https://oauth2.googleapis.com/token",
    scopes: { google_ads: "https://www.googleapis.com/auth/adwords" },
    extraAuthorizeParams: { access_type: "offline", prompt: "consent" },
  },
};

export function providerForType(type: string): Provider {
  return type === "google_ads" ? "google" : "meta";
}

export function isConfigured(p: Provider): boolean {
  const c = PROVIDERS[p];
  return !!(c.clientId && c.clientSecret);
}

const STATE_SECRET = process.env.AUTH_SECRET || "dev-secret";

export interface OAuthState {
  clinicId: string;
  type: string;
  provider: Provider;
  ts: number;
}

/** signState encodes + HMAC-signs the OAuth state to survive the round trip safely. */
export function signState(payload: OAuthState): string {
  const data = Buffer.from(JSON.stringify(payload)).toString("base64url");
  const sig = createHmac("sha256", STATE_SECRET).update(data).digest("base64url");
  return `${data}.${sig}`;
}

/** verifyState validates the signature and returns the decoded state, or null. */
export function verifyState(state: string): OAuthState | null {
  const [data, sig] = state.split(".");
  if (!data || !sig) return null;
  const expected = createHmac("sha256", STATE_SECRET).update(data).digest("base64url");
  const a = Buffer.from(sig);
  const b = Buffer.from(expected);
  if (a.length !== b.length || !timingSafeEqual(a, b)) return null;
  try {
    return JSON.parse(Buffer.from(data, "base64url").toString()) as OAuthState;
  } catch {
    return null;
  }
}

/** authorizeUrl builds the provider's consent URL for a connection type. */
export function buildAuthorizeUrl(provider: Provider, type: string, state: string): string {
  const cfg = PROVIDERS[provider];
  const scope = cfg.scopes[type] ?? Object.values(cfg.scopes)[0];
  const u = new URL(cfg.authorizeUrl);
  u.searchParams.set("client_id", cfg.clientId);
  u.searchParams.set("redirect_uri", REDIRECT_URI);
  u.searchParams.set("response_type", "code");
  u.searchParams.set("scope", scope);
  u.searchParams.set("state", state);
  for (const [k, v] of Object.entries(cfg.extraAuthorizeParams ?? {})) {
    u.searchParams.set(k, v);
  }
  return u.toString();
}

/** exchangeCode swaps an authorization code for an access token + masked detail. */
export async function exchangeCode(
  provider: Provider,
  code: string,
): Promise<{ accessToken: string; refreshToken?: string; detail: string } | null> {
  const cfg = PROVIDERS[provider];
  try {
    if (provider === "meta") {
      const u = new URL(cfg.tokenUrl);
      u.searchParams.set("client_id", cfg.clientId);
      u.searchParams.set("client_secret", cfg.clientSecret);
      u.searchParams.set("redirect_uri", REDIRECT_URI);
      u.searchParams.set("code", code);
      const tok = (await (await fetch(u.toString())).json()) as { access_token?: string };
      if (!tok.access_token) return null;
      const me = (await (
        await fetch(
          `https://graph.facebook.com/v21.0/me?fields=id,name&access_token=${tok.access_token}`,
        )
      ).json()) as { name?: string };
      return {
        accessToken: tok.access_token,
        detail: me.name ? `Meta: ${me.name}` : "Meta hesabı bağlı",
      };
    }
    const body = new URLSearchParams({
      code,
      client_id: cfg.clientId,
      client_secret: cfg.clientSecret,
      redirect_uri: REDIRECT_URI,
      grant_type: "authorization_code",
    });
    const tok = (await (
      await fetch(cfg.tokenUrl, {
        method: "POST",
        headers: { "content-type": "application/x-www-form-urlencoded" },
        body,
      })
    ).json()) as { access_token?: string; refresh_token?: string };
    if (!tok.access_token) return null;
    return {
      accessToken: tok.access_token,
      refreshToken: tok.refresh_token, // present because access_type=offline+prompt=consent
      detail: "Google Ads bağlı",
    };
  } catch {
    return null;
  }
}
