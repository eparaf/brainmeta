// Thin client over the Go brain's HTTP API. Paths are relative so the Vite proxy
// (dev) and same-origin (prod) both work without CORS gymnastics.

async function jget(path) {
  const r = await fetch(path)
  if (!r.ok) throw new Error(`${path}: ${r.status}`)
  return r.json()
}
async function jpost(path, body) {
  const r = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body || {}),
  })
  if (!r.ok) throw new Error(`${path}: ${r.status}`)
  return r.json()
}

export const api = {
  health: () => jget('/healthz'),
  sla: () => jget('/v1/sla'),
  arms: () => jget('/v1/arms'),
  budget: (daysInMonth = 30) => jpost('/v1/budget/plan', { daysInMonth }),
  whatsapp: (msg) => jpost('/v1/whatsapp', msg),
  intake: (answers) => jpost('/v1/intake', answers),
}
