import { z } from "zod";

// Zod schemas mirroring the Go backend's JSON responses. Parsing responses through
// these gives us runtime safety AND the TS types (via z.infer) from one source.

// Clinic — from /v1/auth/login (base) and /v1/clinics (SLA-enriched: the last
// fields are present only on the enriched endpoint).
export const clinicSchema = z.object({
  id: z.string(),
  name: z.string(),
  district: z.string().default(""),
  side: z.string().default(""),
  segment: z.string().default(""),
  guarantee: z.number().default(0),
  dailyCapacity: z.number().default(0),
  monthlyAdBudget: z.number().default(0),
  delivered: z.number().optional(),
  shadowPrice: z.number().optional(),
  deficit: z.number().optional(),
  targetNow: z.number().optional(),
  status: z.enum(["on-track", "behind"]).optional(),
});
export type Clinic = z.infer<typeof clinicSchema>;
export const clinicListSchema = z.array(clinicSchema);

export const brainUserSchema = z.object({
  id: z.string(),
  email: z.string(),
  name: z.string().default(""),
  role: z.string(),
  clinicIds: z.array(z.string()).default([]),
  createdAt: z.string().optional(),
});
export type BrainUser = z.infer<typeof brainUserSchema>;

export const loginResponseSchema = z.object({
  token: z.string(),
  user: brainUserSchema,
  clinics: z.array(clinicSchema).default([]),
});
export type LoginResponse = z.infer<typeof loginResponseSchema>;

// Ad arm — /v1/arms (Budget.Snapshot maps).
export const armSchema = z.object({
  armId: z.string(),
  clinicId: z.string(),
  segment: z.string(),
  thetaHat: z.number(),
  cpl: z.number(),
  leads: z.number(),
  appts: z.number(),
  spend: z.number(),
});
export type Arm = z.infer<typeof armSchema>;
export const armListSchema = z.array(armSchema);

// Budget plan — /v1/budget/plan (allocations are PascalCase from the Go struct).
export const allocationSchema = z.object({
  ArmID: z.string(),
  ClinicID: z.string(),
  DailyBudget: z.number(),
  SampledTheta: z.number(),
  ExpectedAppts: z.number(),
  SLABias: z.number(),
});
export const budgetPlanSchema = z.object({
  daysInMonth: z.number(),
  networkDaily: z.number(),
  lambda: z.number(),
  allocations: z.array(allocationSchema).default([]),
});
export type BudgetPlan = z.infer<typeof budgetPlanSchema>;

// Qualification + conversation/lead — /v1/leads, /v1/conversations[/:id].
export const qualificationSchema = z.object({
  segment: z.string().default(""),
  intentPct: z.number().default(0),
  urgency: z.string().default("Düşük"),
  budgetTry: z.number().default(0),
  language: z.string().default("TR"),
  booked: z.boolean().default(false),
  appointmentTime: z.string().default(""),
});
export const messageSchema = z.object({
  id: z.string(),
  sender: z.enum(["patient", "agent"]).catch("agent"),
  text: z.string(),
  timestamp: z.string().default(""),
});
export const conversationSchema = z.object({
  id: z.string(),
  name: z.string().default(""),
  phoneNumber: z.string().default(""),
  clinicId: z.string(),
  status: z.string().default("Niteleniyor"),
  createdAt: z.string().default(""),
  lastMessage: z.string().default(""),
  lastMessageTime: z.string().default(""),
  messages: z.array(messageSchema).default([]),
  qualification: qualificationSchema,
});
export type Conversation = z.infer<typeof conversationSchema>;
export const conversationListSchema = z.array(conversationSchema);

// Appointment — /v1/appointments.
export const appointmentSchema = z.object({
  id: z.string(),
  clinicId: z.string(),
  leadId: z.string().default(""),
  name: z.string().default(""),
  phone: z.string().default(""),
  when: z.string(),
  segment: z.string().default(""),
  pShow: z.number().default(0),
  overbook: z.boolean().default(false),
  doctorId: z.string().default(""),
  doctor: z.string().optional(),
  service: z.string().default(""),
});
export type Appointment = z.infer<typeof appointmentSchema>;
export const appointmentListSchema = z.array(appointmentSchema);

// Template — /v1/templates (static approved + drafts).
export const templateSchema = z.object({
  id: z.string(),
  name: z.string(),
  category: z.string().default("UTILITY"),
  language: z.string().default("tr"),
  status: z.string().default("APPROVED"),
  body: z.string().default(""),
  vars: z.array(z.string()).optional(),
  clinicId: z.string().optional(),
});
export type Template = z.infer<typeof templateSchema>;
export const templateListSchema = z.array(templateSchema);

// Connection — /v1/connections.
export const connectionSchema = z.object({
  id: z.string(),
  clinicId: z.string(),
  type: z.enum(["whatsapp", "meta_ads", "google_ads", "web_form"]).catch("web_form"),
  connected: z.boolean().default(false),
  detail: z.string().default(""),
  updatedAt: z.string().default(""),
});
export type Connection = z.infer<typeof connectionSchema>;
export const connectionListSchema = z.array(connectionSchema);

// Embeddable widget config — /v1/widget.
export const widgetFieldSchema = z.object({
  key: z.string(),
  label: z.string(),
  required: z.boolean().default(false),
  enabled: z.boolean().default(true),
});
export type WidgetField = z.infer<typeof widgetFieldSchema>;

export const widgetConfigSchema = z.object({
  clinicId: z.string(),
  publicKey: z.string().default(""),
  primaryColor: z.string().default("#0f766e"),
  formTitle: z.string().default(""),
  formSubtitle: z.string().default(""),
  successText: z.string().default(""),
  fields: z.array(widgetFieldSchema).default([]),
  calendarColor: z.string().default("#30d158"),
  calendarTitle: z.string().default(""),
  calendarSubtitle: z.string().default(""),
  confirmText: z.string().default(""),
  theme: z.string().default("dark"),
  recommend: z.boolean().default(true),
  updatedAt: z.string().optional(),
});
export type WidgetConfig = z.infer<typeof widgetConfigSchema>;

// Doctors & services (clinic calendar) — /v1/doctors, /v1/services.
export const doctorSchema = z.object({
  id: z.string().default(""),
  clinicId: z.string(),
  name: z.string(),
  title: z.string().default(""),
  specialty: z.string().default(""),
  active: z.boolean().default(true),
  days: z.array(z.number()).default([1, 2, 3, 4, 5]),
  startHour: z.number().default(9),
  endHour: z.number().default(17),
  slotMins: z.number().default(30),
});
export type Doctor = z.infer<typeof doctorSchema>;
export const doctorListSchema = z.array(doctorSchema);

export const serviceSchema = z.object({
  id: z.string().default(""),
  clinicId: z.string(),
  name: z.string(),
  durationMins: z.number().default(30),
  doctorIds: z.array(z.string()).default([]),
  active: z.boolean().default(true),
});
export type Service = z.infer<typeof serviceSchema>;
export const serviceListSchema = z.array(serviceSchema);

// Scenario engine — POST /v1/scenario. Offline Monte-Carlo "with this budget, how
// many appointments per month?" forecast. Costs nothing, calls no LLM.
export const scenarioBandSchema = z.object({
  p10: z.number(),
  p50: z.number(),
  p90: z.number(),
  mean: z.number(),
});
export type ScenarioBand = z.infer<typeof scenarioBandSchema>;

export const scenarioFunnelSchema = z.object({
  Qualify: z.number(),
  Book: z.number(),
  Show: z.number(),
  Close: z.number(),
});

export const scenarioResultSchema = z.object({
  runs: z.number(),
  budget: z.number(),
  bookedAppointments: scenarioBandSchema,
  keptAppointments: scenarioBandSchema,
  qualifiedLeads: scenarioBandSchema,
  clicks: scenarioBandSchema,
  costPerAppointmentTRY: scenarioBandSchema,
  costPerLeadTRY: scenarioBandSchema,
  assumptions: z.object({
    funnel: scenarioFunnelSchema,
    clickToLead: z.number(),
    avgCpcTRY: z.number(),
    searchVolume: z.number(),
    maxImpressionShare: z.number(),
  }),
});
export type ScenarioResult = z.infer<typeof scenarioResultSchema>;
