import type { DefaultSession } from "next-auth";
import type { Clinic } from "@/lib/brain/schemas";

// Augment Auth.js types so the Go JWT, role, clinic membership, and clinic list we
// carry through the session are fully typed (no `any` in the callbacks/consumers).

declare module "next-auth" {
  interface Session {
    brainToken: string;
    clinics: Clinic[];
    user: {
      id: string;
      role: string;
      clinicIds: string[];
    } & DefaultSession["user"];
  }

  interface User {
    role?: string;
    clinicIds?: string[];
    brainToken?: string;
    clinics?: Clinic[];
  }
}

declare module "next-auth/jwt" {
  interface JWT {
    brainToken?: string;
    role?: string;
    clinicIds?: string[];
    clinics?: Clinic[];
  }
}
