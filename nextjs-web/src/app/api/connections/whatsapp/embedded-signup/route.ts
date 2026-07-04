import { NextRequest, NextResponse } from "next/server";
import { auth } from "@/auth";
import { env } from "@/lib/env";

// Receives the WhatsApp Embedded Signup result from EmbeddedSignupButton: the
// authorization `code` (from FB.login's callback) plus the waba_id/phone_number_id
// captured from the SDK's WA_EMBEDDED_SIGNUP postMessage event. Exchanges the code
// server-side (App Secret never touches the client) and hands the resulting token
// + WhatsApp identifiers to the Go backend's write-only oauth-token endpoint,
// which is what lets inbound webhooks resolve to this clinic (see
// internal/api/webhooks.go's handleWAInbound + store.ResolveClinicByPhoneNumberID).
//
// NOTE: unlike the generic OAuth dialog (oauth/callback), Embedded Signup's code
// exchange must NOT include redirect_uri — there is no server redirect in this
// flow, only a JS SDK popup. Per Meta's documented Embedded Signup contract.
export async function POST(req: NextRequest) {
  const session = await auth();
  if (!session?.brainToken) {
    return NextResponse.json({ error: "unauthorized" }, { status: 401 });
  }

  const body = (await req.json().catch(() => null)) as {
    clinicId?: string;
    code?: string;
    phoneNumberId?: string;
    wabaId?: string;
  } | null;
  if (!body?.clinicId || !body.code) {
    return NextResponse.json({ error: "clinicId and code required" }, { status: 400 });
  }

  const appId = process.env.NEXT_PUBLIC_META_APP_ID || "";
  const appSecret = process.env.META_APP_SECRET || "";
  if (!appId || !appSecret) {
    return NextResponse.json({ error: "embedded signup not configured" }, { status: 503 });
  }

  const tokenURL = new URL("https://graph.facebook.com/v21.0/oauth/access_token");
  tokenURL.searchParams.set("client_id", appId);
  tokenURL.searchParams.set("client_secret", appSecret);
  tokenURL.searchParams.set("code", body.code);
  // No redirect_uri — Embedded Signup is a JS SDK popup, not a redirect flow.

  let accessToken: string | undefined;
  try {
    const tokRes = await fetch(tokenURL.toString());
    const tok = (await tokRes.json().catch(() => null)) as { access_token?: string } | null;
    accessToken = tok?.access_token;
  } catch {
    return NextResponse.json({ error: "token exchange failed (network)" }, { status: 502 });
  }
  if (!accessToken) {
    return NextResponse.json({ error: "token exchange failed" }, { status: 502 });
  }

  const ok = await fetch(`${env.BRAIN_API_URL}/v1/connections/oauth-token`, {
    method: "POST",
    headers: {
      "content-type": "application/json",
      Authorization: `Bearer ${session.brainToken}`,
    },
    body: JSON.stringify({
      clinicId: body.clinicId,
      provider: "meta",
      type: "whatsapp",
      refreshToken: accessToken,
      phoneNumberId: body.phoneNumberId ?? "",
      wabaId: body.wabaId ?? "",
      detail: "WhatsApp Embedded Signup ile bağlandı",
    }),
  })
    .then((r) => r.ok)
    .catch(() => false);

  if (!ok) {
    return NextResponse.json({ error: "backend persist failed" }, { status: 502 });
  }
  return NextResponse.json({ ok: true });
}
