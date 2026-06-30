import NextAuth from "next-auth";
import Credentials from "next-auth/providers/credentials";
import { brainLogin } from "@/lib/brain/auth";
import type { Clinic } from "@/lib/brain/schemas";

// The extra fields the Credentials provider attaches to the user. We read them off
// the `user` param via this type because Auth.js types it as a User | AdapterUser
// union that doesn't surface our module augmentation on the read side.
interface BrainPrincipal {
  brainToken?: string;
  role?: string;
  clinicIds?: string[];
  clinics?: Clinic[];
}

/**
 * Auth.js v5 configuration. The Credentials provider delegates to the Go backend's
 * /v1/auth/login; the Go-issued JWT plus role/clinic membership are carried in the
 * session (JWT strategy — no DB adapter, the backend is the source of truth).
 */
export const { handlers, signIn, signOut, auth } = NextAuth({
  trustHost: true,
  session: { strategy: "jwt" },
  pages: { signIn: "/login" },
  providers: [
    Credentials({
      credentials: {
        email: { label: "E-posta", type: "email" },
        password: { label: "Parola", type: "password" },
      },
      authorize: async (credentials) => {
        const email = typeof credentials?.email === "string" ? credentials.email : "";
        const password =
          typeof credentials?.password === "string" ? credentials.password : "";
        if (!email || !password) return null;
        const res = await brainLogin(email, password);
        if (!res) return null;
        return {
          id: res.user.id,
          email: res.user.email,
          name: res.user.name,
          role: res.user.role,
          clinicIds: res.user.clinicIds,
          brainToken: res.token,
          clinics: res.clinics,
        };
      },
    }),
  ],
  callbacks: {
    jwt({ token, user }) {
      if (user) {
        const principal = user as BrainPrincipal;
        token.brainToken = principal.brainToken;
        token.role = principal.role;
        token.clinicIds = principal.clinicIds;
        token.clinics = principal.clinics;
      }
      return token;
    },
    session({ session, token }) {
      // JWT custom fields read back as the interface's index-signature type, so we
      // narrow at runtime. Session fields below are strongly typed via augmentation.
      const brainToken = token.brainToken;
      session.brainToken = typeof brainToken === "string" ? brainToken : "";
      session.user.id = typeof token.sub === "string" ? token.sub : "";
      session.user.role = typeof token.role === "string" ? token.role : "clinic";
      session.user.clinicIds = Array.isArray(token.clinicIds)
        ? (token.clinicIds as string[])
        : [];
      session.clinics = Array.isArray(token.clinics) ? (token.clinics as Clinic[]) : [];
      return session;
    },
  },
});
