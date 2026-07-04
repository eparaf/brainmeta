"use client";

import { z } from "zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { jdelete, jget, jpost } from "@/lib/api";
import {
  appointmentListSchema,
  armListSchema,
  budgetPlanSchema,
  clinicListSchema,
  connectionListSchema,
  scenarioResultSchema,
  connectionSchema,
  conversationListSchema,
  conversationSchema,
  doctorListSchema,
  doctorSchema,
  serviceListSchema,
  serviceSchema,
  templateListSchema,
  templateSchema,
  widgetConfigSchema,
  type Doctor,
  type Service,
  type WidgetConfig,
} from "@/lib/brain/schemas";

// All hooks key by clinic where relevant, so switching the active clinic refetches.

export function useClinics() {
  return useQuery({
    queryKey: ["clinics"],
    queryFn: () => jget("/api/brain/v1/clinics", clinicListSchema),
  });
}

export function useArms(clinicId: string | null) {
  return useQuery({
    queryKey: ["arms", clinicId],
    enabled: !!clinicId,
    queryFn: async () => {
      const all = await jget("/api/brain/v1/arms", armListSchema);
      return all.filter((a) => a.clinicId === clinicId);
    },
  });
}

export function useBudget(days = 30) {
  return useQuery({
    queryKey: ["budget", days],
    queryFn: () =>
      jpost("/api/brain/v1/budget/plan", { daysInMonth: days }, budgetPlanSchema),
  });
}

export function useLeads(clinicId: string | null) {
  return useQuery({
    queryKey: ["leads", clinicId],
    enabled: !!clinicId,
    queryFn: () =>
      jget(
        `/api/brain/v1/leads?clinicId=${encodeURIComponent(clinicId ?? "")}`,
        conversationListSchema,
      ),
  });
}

export function useConversations(clinicId: string | null) {
  return useQuery({
    queryKey: ["conversations", clinicId],
    enabled: !!clinicId,
    queryFn: () =>
      jget(
        `/api/brain/v1/conversations?clinicId=${encodeURIComponent(clinicId ?? "")}`,
        conversationListSchema,
      ),
  });
}

export function useConversation(id: string | null) {
  return useQuery({
    queryKey: ["conversation", id],
    enabled: !!id,
    queryFn: () =>
      jget(`/api/brain/v1/conversations/${encodeURIComponent(id ?? "")}`, conversationSchema),
  });
}

export function useAppointments(clinicId: string | null) {
  return useQuery({
    queryKey: ["appointments", clinicId],
    enabled: !!clinicId,
    queryFn: () =>
      jget(
        `/api/brain/v1/appointments?clinicId=${encodeURIComponent(clinicId ?? "")}`,
        appointmentListSchema,
      ),
  });
}

export function useTemplates() {
  return useQuery({
    queryKey: ["templates"],
    queryFn: () => jget("/api/brain/v1/templates", templateListSchema),
  });
}

export function useCreateTemplate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: {
      clinicId: string;
      name: string;
      category: string;
      language: string;
      body: string;
    }) => jpost("/api/brain/v1/templates", body, templateSchema),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["templates"] }),
  });
}

export function useConnections(clinicId: string | null) {
  return useQuery({
    queryKey: ["connections", clinicId],
    enabled: !!clinicId,
    queryFn: () =>
      jget(
        `/api/brain/v1/connections?clinicId=${encodeURIComponent(clinicId ?? "")}`,
        connectionListSchema,
      ),
  });
}

export function useUpsertConnection(clinicId: string | null) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { clinicId: string; type: string; connected: boolean }) =>
      jpost("/api/brain/v1/connections", body, connectionSchema),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["connections", clinicId] }),
  });
}

// z.object strips unknown keys by default, so the extra fields (qualification,
// apptTime, reason) parse fine without erroring.
const whatsappReplySchema = z.object({
  reply: z.string().default(""),
  booked: z.boolean().optional(),
});

export function useWidget(clinicId: string | null) {
  return useQuery({
    queryKey: ["widget", clinicId],
    enabled: !!clinicId,
    queryFn: () =>
      jget(
        `/api/brain/v1/widget?clinicId=${encodeURIComponent(clinicId ?? "")}`,
        widgetConfigSchema,
      ),
  });
}

export function useSaveWidget(clinicId: string | null) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (cfg: WidgetConfig) =>
      jpost("/api/brain/v1/widget", cfg, widgetConfigSchema),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["widget", clinicId] }),
  });
}

export function useRotateWidgetKey(clinicId: string | null) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      jpost(
        `/api/brain/v1/widget/rotate-key?clinicId=${encodeURIComponent(clinicId ?? "")}`,
        {},
        widgetConfigSchema,
      ),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["widget", clinicId] }),
  });
}

export function useDoctors(clinicId: string | null) {
  return useQuery({
    queryKey: ["doctors", clinicId],
    enabled: !!clinicId,
    queryFn: () =>
      jget(
        `/api/brain/v1/doctors?clinicId=${encodeURIComponent(clinicId ?? "")}`,
        doctorListSchema,
      ),
  });
}

export function useSaveDoctor(clinicId: string | null) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (d: Doctor) => jpost("/api/brain/v1/doctors", d, doctorSchema),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["doctors", clinicId] }),
  });
}

export function useDeleteDoctor(clinicId: string | null) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => jdelete(`/api/brain/v1/doctors/${encodeURIComponent(id)}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["doctors", clinicId] }),
  });
}

export function useServices(clinicId: string | null) {
  return useQuery({
    queryKey: ["services", clinicId],
    enabled: !!clinicId,
    queryFn: () =>
      jget(
        `/api/brain/v1/services?clinicId=${encodeURIComponent(clinicId ?? "")}`,
        serviceListSchema,
      ),
  });
}

export function useSaveService(clinicId: string | null) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (svc: Service) => jpost("/api/brain/v1/services", svc, serviceSchema),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["services", clinicId] }),
  });
}

export function useDeleteService(clinicId: string | null) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => jdelete(`/api/brain/v1/services/${encodeURIComponent(id)}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["services", clinicId] }),
  });
}

export function useSendMessage() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { phone: string; clinicId: string; armId: string; message: string }) =>
      jpost("/api/brain/v1/whatsapp", body, whatsappReplySchema),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ["conversations", vars.clinicId] });
      qc.invalidateQueries({ queryKey: ["conversation"] });
    },
  });
}

// Scenario engine: offline "with this budget, how many appointments?" forecast.
// A mutation (not a query) because it's a what-if the user triggers explicitly
// with a chosen budget, not data to auto-fetch on mount.
export function useScenario() {
  return useMutation({
    mutationFn: (body: {
      clinicId?: string;
      segment?: string;
      platform?: string;
      audience?: string;
      monthlyBudget: number;
    }) => jpost("/api/brain/v1/scenario", body, scenarioResultSchema),
  });
}
