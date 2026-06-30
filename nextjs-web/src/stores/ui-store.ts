"use client";

import { create } from "zustand";
import type { Clinic } from "@/lib/brain/schemas";

// Global UI state: the clinic list (hydrated from the session each load), the
// selected clinic, and mobile sidebar visibility. Server data lives in TanStack
// Query; this store is only client/UI concerns.
interface UIState {
  clinics: Clinic[];
  activeClinicId: string | null;
  mobileOpen: boolean;
  setClinics: (clinics: Clinic[]) => void;
  setActiveClinicId: (id: string) => void;
  setMobileOpen: (open: boolean) => void;
}

export const useUIStore = create<UIState>((set, get) => ({
  clinics: [],
  activeClinicId: null,
  mobileOpen: false,
  setClinics: (clinics) => {
    const current = get().activeClinicId;
    const stillValid = current ? clinics.some((c) => c.id === current) : false;
    set({ clinics, activeClinicId: stillValid ? current : (clinics[0]?.id ?? null) });
  },
  setActiveClinicId: (id) => set({ activeClinicId: id }),
  setMobileOpen: (open) => set({ mobileOpen: open }),
}));

/** useActiveClinic returns the currently selected clinic object (or null). */
export function useActiveClinic(): Clinic | null {
  return useUIStore((s) => s.clinics.find((c) => c.id === s.activeClinicId) ?? null);
}
