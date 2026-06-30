import { z } from "zod";

// Server-side environment, validated once at import. Defaults keep local dev
// working with zero config. AUTH_SECRET is read by Auth.js directly; we surface it
// here only for completeness. Never import this from a Client Component.
const schema = z.object({
  BRAIN_API_URL: z.string().min(1).default("http://localhost:8091"),
  AUTH_SECRET: z.string().min(1).optional(),
});

export const env = schema.parse({
  BRAIN_API_URL: process.env.BRAIN_API_URL,
  AUTH_SECRET: process.env.AUTH_SECRET,
});
