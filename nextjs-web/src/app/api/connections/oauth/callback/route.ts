import { NextRequest } from "next/server";
import { auth } from "@/auth";
import { env } from "@/lib/env";
import { APP_URL, exchangeCode, verifyState } from "@/lib/oauth";

// OAuth2 redirect target. Verifies the signed state, exchanges the code for a
// token, then records the connection (connected=true + masked detail) in the Go
// backend using the user's session token. The raw token is NOT persisted here —
// wiring it into the Meta/Google API clients is a follow-up.
export async function GET(req: NextRequest) {
  const back = `${APP_URL}/baglantilar`;
  const session = await auth();
  if (!session?.brainToken) return Response.redirect(`${APP_URL}/login`, 302);

  const url = new URL(req.url);
  if (url.searchParams.get("error")) {
    return Response.redirect(`${back}?error=denied`, 302);
  }
  const code = url.searchParams.get("code");
  const state = verifyState(url.searchParams.get("state") ?? "");
  if (!code || !state) return Response.redirect(`${back}?error=bad_state`, 302);

  const result = await exchangeCode(state.provider, code);
  if (!result) return Response.redirect(`${back}?error=exchange_failed`, 302);

  // If the provider returned a refresh token, hand it to the backend's secret
  // store via the write-only token endpoint — that ALSO flips the connection to
  // connected=true and is what enables live ad-platform sync. Without a refresh
  // token (e.g. re-consent that omitted it) fall back to a status-only upsert.
  if (result.refreshToken) {
    await fetch(`${env.BRAIN_API_URL}/v1/connections/oauth-token`, {
      method: "POST",
      headers: {
        "content-type": "application/json",
        Authorization: `Bearer ${session.brainToken}`,
      },
      body: JSON.stringify({
        clinicId: state.clinicId,
        provider: state.provider,
        refreshToken: result.refreshToken,
        type: state.type,
        detail: result.detail,
      }),
    }).catch(() => {});
  } else {
    await fetch(`${env.BRAIN_API_URL}/v1/connections`, {
      method: "POST",
      headers: {
        "content-type": "application/json",
        Authorization: `Bearer ${session.brainToken}`,
      },
      body: JSON.stringify({
        clinicId: state.clinicId,
        type: state.type,
        connected: true,
        detail: result.detail,
      }),
    }).catch(() => {});
  }

  return Response.redirect(`${back}?connected=${encodeURIComponent(state.type)}`, 302);
}
