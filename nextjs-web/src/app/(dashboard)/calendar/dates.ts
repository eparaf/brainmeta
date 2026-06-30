// Self-contained date math for the calendar grid — no external date library.

export const TR_MONTHS = [
  "Ocak", "Şubat", "Mart", "Nisan", "Mayıs", "Haziran",
  "Temmuz", "Ağustos", "Eylül", "Ekim", "Kasım", "Aralık",
];

// Weekday short labels indexed by JS getDay() (0 = Pazar).
export const TR_WEEKDAYS_SHORT = ["Paz", "Pzt", "Sal", "Çar", "Per", "Cum", "Cmt"];
export const TR_WEEKDAYS_LONG = [
  "Pazar", "Pazartesi", "Salı", "Çarşamba", "Perşembe", "Cuma", "Cumartesi",
];

export function startOfDay(d: Date): Date {
  const n = new Date(d);
  n.setHours(0, 0, 0, 0);
  return n;
}

export function addDays(d: Date, n: number): Date {
  const r = new Date(d);
  r.setDate(r.getDate() + n);
  return r;
}

export function addMonths(d: Date, n: number): Date {
  const r = new Date(d);
  r.setDate(1);
  r.setMonth(r.getMonth() + n);
  return r;
}

export function isSameDay(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}

export function dayKey(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

// Start of the visible grid week given a week-start preference (0 = Pazar, 1 = Pazartesi).
export function startOfWeek(d: Date, weekStart: 0 | 1): Date {
  const r = startOfDay(d);
  const diff = (r.getDay() - weekStart + 7) % 7;
  return addDays(r, -diff);
}

// 6-week (42 day) month grid covering the month containing `anchor`.
export function monthGrid(anchor: Date, weekStart: 0 | 1): Date[] {
  const first = new Date(anchor.getFullYear(), anchor.getMonth(), 1);
  const gridStart = startOfWeek(first, weekStart);
  return Array.from({ length: 42 }, (_, i) => addDays(gridStart, i));
}

// 7 days of the week containing `anchor`.
export function weekGrid(anchor: Date, weekStart: 0 | 1): Date[] {
  const start = startOfWeek(anchor, weekStart);
  return Array.from({ length: 7 }, (_, i) => addDays(start, i));
}

// Ordered weekday indices (into the *_WEEKDAYS arrays) for the chosen week start.
export function weekdayOrder(weekStart: 0 | 1): number[] {
  return Array.from({ length: 7 }, (_, i) => (i + weekStart) % 7);
}

export function formatTime(iso: string, time24h: boolean): string {
  return new Date(iso).toLocaleTimeString("tr-TR", {
    hour: "2-digit",
    minute: "2-digit",
    hour12: !time24h,
  });
}

export function isWeekend(d: Date): boolean {
  const g = d.getDay();
  return g === 0 || g === 6;
}
