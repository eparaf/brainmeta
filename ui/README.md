# Dişçi Brain — UI (Vite + React + Tailwind)

A no-auth demo console that talks to the **real Go backend**. Four panels:

- **Sohbet** — WhatsApp tester: type as a patient, the agent qualifies and the
  brain decides (booking + real slot). Shows the structured qualification.
- **Garantiler** — per-clinic guarantee health (delivered / guaranteed, shadow price λ).
- **Reklam Kolları** — what the bandit learned per ad arm (θ̂, CPL, leads, appts).
- **Bütçe** — current budget allocation across arms + the shadow price λ.

## Run

**If `:8080` is free** (default):

```bash
# 1) backend, from the Go module root (../)
go run ./cmd/brain serve            # :8080
# 2) UI
cd ui && npm install && npm run dev # http://localhost:5173
```

**If you get `404` on `/v1/*`** — something else already owns `:8080` (very
common on WSL2, where Windows and Linux share `localhost`). Run the backend on a
free port and point the proxy at it:

```bash
BRAIN_ADDR=:8090 go run ./cmd/brain serve         # backend on :8090
BACKEND_URL=http://localhost:8090 npm run dev     # UI proxies to :8090
```

The Vite dev server proxies `/v1` and `/healthz` to `BACKEND_URL`
(default `http://localhost:8080`) — see `vite.config.js`. No CORS issues; the
backend also sends permissive CORS headers for direct calls.

> Sanity check the backend directly:
> `curl -s -XPOST localhost:8090/v1/whatsapp -d '{"clinicId":"umraniye","armId":"umraniye:meta:implant","message":"implant istiyorum"}'`
> should return JSON with `booked` and `reply`. If `curl localhost:8090/healthz`
> works but `/v1/whatsapp` 404s, you're hitting an OLD/other server on that port.

## Notes

- **No auth** — dev/demo only. Lock down CORS + add auth before exposing publicly.
- The agent uses the **mock LLM** unless the backend has `ANTHROPIC_API_KEY` set
  (then it uses Claude). Either way the booking decision is the deterministic
  brain's — the LLM only phrases the reply.
- Backend seeds 4 demo Istanbul clinics on startup; learned state persists to a
  snapshot file, so numbers grow as you chat and survive restarts.
