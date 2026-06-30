import { NextRequest } from "next/server";
import { auth } from "@/auth";
import {
  APP_URL,
  buildAuthorizeUrl,
  isConfigured,
  providerForType,
  signState,
} from "@/lib/oauth";

// Begins the OAuth2 consent flow for a clinic + connection type. Authenticated via
// the session cookie (this is a top-level browser navigation). Redirects to the
// provider, or back to /baglantilar with ?error=not_configured if creds are unset.
export async function GET(req: NextRequest) {
  const session = await auth();
  const back = `${APP_URL}/baglantilar`;
  if (!session) return Response.redirect(`${APP_URL}/login`, 302);

  const url = new URL(req.url);
  const clinicId = url.searchParams.get("clinicId") ?? "";
  const type = url.searchParams.get("type") ?? "";
  if (!clinicId || !type) return Response.redirect(`${back}?error=bad_request`, 302);

  const provider = providerForType(type);
  if (!isConfigured(provider)) {
    return Response.redirect(`${back}?error=not_configured&provider=${provider}`, 302);
  }

  const state = signState({ clinicId, type, provider, ts: Date.now() });
  return Response.redirect(buildAuthorizeUrl(provider, type, state), 302);
}
