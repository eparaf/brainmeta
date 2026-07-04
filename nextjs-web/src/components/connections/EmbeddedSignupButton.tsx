"use client";

import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/Button";

// Real WhatsApp Embedded Signup (Meta's dedicated onboarding flow — NOT the
// generic OAuth dialog used for meta_ads/google_ads). This is the guided
// Facebook-hosted popup that lets a clinic pick/create a WhatsApp Business
// Account (WABA) and register a phone number in one flow, per Meta's documented
// pattern: https://developers.facebook.com/docs/whatsapp/embedded-signup
//
// Needs NEXT_PUBLIC_META_APP_ID + NEXT_PUBLIC_META_CONFIG_ID (from Meta App
// Dashboard > WhatsApp > Embedded Signup) — shows "not configured" without
// them, same convention as the generic OAuth flow's isConfigured() gate.
//
// IMPORTANT — this needs a real Meta Business/App with Embedded Signup
// configured and App-Reviewed to actually complete; it cannot be end-to-end
// tested without one. The code follows Meta's documented JS SDK contract
// faithfully; verify against a real Meta App before relying on it in production.

declare global {
  interface Window {
    FB?: {
      init: (opts: Record<string, unknown>) => void;
      login: (
        cb: (res: { authResponse?: { code?: string } }) => void,
        opts: Record<string, unknown>,
      ) => void;
    };
    fbAsyncInit?: () => void;
  }
}

const SDK_SRC = "https://connect.facebook.net/en_US/sdk.js";

function loadFacebookSDK(appId: string): Promise<void> {
  return new Promise((resolve) => {
    if (window.FB) return resolve();
    window.fbAsyncInit = () => {
      window.FB!.init({ appId, autoLogAppEvents: true, xfbml: false, version: "v21.0" });
      resolve();
    };
    if (document.getElementById("facebook-jssdk")) return;
    const script = document.createElement("script");
    script.id = "facebook-jssdk";
    script.src = SDK_SRC;
    script.async = true;
    document.body.appendChild(script);
  });
}

// WA_EMBEDDED_SIGNUP posts { type, event, data: { phone_number_id, waba_id } } via
// window.postMessage — this is the ONLY way to get the ids; FB.login's callback
// only carries the authorization `code`.
interface EmbeddedSignupMessage {
  type?: string;
  event?: string;
  data?: { phone_number_id?: string; waba_id?: string };
}

export function EmbeddedSignupButton({
  clinicId,
  onDone,
}: {
  clinicId: string;
  onDone: (result: { ok: boolean; error?: string }) => void;
}) {
  const [loading, setLoading] = useState(false);
  const captured = useRef<{ phoneNumberId?: string; wabaId?: string }>({});

  const appId = process.env.NEXT_PUBLIC_META_APP_ID;
  const configId = process.env.NEXT_PUBLIC_META_CONFIG_ID;
  const configured = !!(appId && configId);

  useEffect(() => {
    function onMessage(event: MessageEvent) {
      if (event.origin !== "https://www.facebook.com") return;
      let data: EmbeddedSignupMessage;
      try {
        data = JSON.parse(event.data);
      } catch {
        return;
      }
      if (data.type === "WA_EMBEDDED_SIGNUP" && data.event === "FINISH" && data.data) {
        captured.current = {
          phoneNumberId: data.data.phone_number_id,
          wabaId: data.data.waba_id,
        };
      }
    }
    window.addEventListener("message", onMessage);
    return () => window.removeEventListener("message", onMessage);
  }, []);

  async function start() {
    if (!appId || !configId) return;
    setLoading(true);
    captured.current = {};
    try {
      await loadFacebookSDK(appId);
      window.FB!.login(
        (response) => {
          void (async () => {
            const code = response.authResponse?.code;
            if (!code) {
              setLoading(false);
              onDone({ ok: false, error: "İzin verilmedi." });
              return;
            }
            // The postMessage listener fires before FB.login's callback in Meta's
            // documented flow, but give it a beat in case of ordering jitter.
            await new Promise((r) => setTimeout(r, 300));
            try {
              const res = await fetch("/api/connections/whatsapp/embedded-signup", {
                method: "POST",
                headers: { "content-type": "application/json" },
                body: JSON.stringify({
                  clinicId,
                  code,
                  phoneNumberId: captured.current.phoneNumberId,
                  wabaId: captured.current.wabaId,
                }),
              });
              const json = (await res.json().catch(() => null)) as { error?: string } | null;
              onDone(res.ok ? { ok: true } : { ok: false, error: json?.error ?? "Bağlantı kurulamadı." });
            } catch {
              onDone({ ok: false, error: "Sunucu hatası." });
            } finally {
              setLoading(false);
            }
          })();
        },
        {
          config_id: configId,
          response_type: "code",
          override_default_response_type: true,
          extras: { setup: {}, featureType: "", sessionInfoVersion: "3" },
        },
      );
    } catch {
      setLoading(false);
      onDone({ ok: false, error: "Facebook SDK yüklenemedi." });
    }
  }

  if (!configured) {
    return (
      <Button variant="secondary" disabled className="text-[11px]">
        WhatsApp Embedded Signup yapılandırılmamış
      </Button>
    );
  }

  return (
    <Button onClick={start} disabled={loading}>
      {loading ? "Bağlanıyor…" : "WhatsApp'ı Bağla (Embedded Signup)"}
    </Button>
  );
}
