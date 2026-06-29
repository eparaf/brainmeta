import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// The UI calls the Go brain. We proxy /v1 and /healthz to the backend so there
// are no CORS issues in dev. The backend URL is configurable:
//
//   BACKEND_URL=http://localhost:8090 npm run dev
//
// Default is :8080. If you get 404s on /v1/* it means something ELSE is on that
// port (common on WSL2, where Windows and Linux share localhost) — run the
// backend on a free port (BRAIN_ADDR=:8090 go run ./cmd/brain serve) and point
// BACKEND_URL here.
const target = process.env.BACKEND_URL || 'http://localhost:8080'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/v1': { target, changeOrigin: true },
      '/healthz': { target, changeOrigin: true },
    },
  },
})
