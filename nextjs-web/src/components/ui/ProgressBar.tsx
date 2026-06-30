import { cn } from "@/lib/utils";

const colors = {
  emerald: "bg-emerald-500",
  amber: "bg-amber-500",
  red: "bg-red-500",
  zinc: "bg-zinc-900",
} as const;

export function ProgressBar({
  value,
  tone = "zinc",
}: {
  value: number;
  tone?: keyof typeof colors;
}) {
  const pct = Math.max(0, Math.min(100, value));
  return (
    <div className="h-2 w-full overflow-hidden rounded-full bg-zinc-100">
      <div
        className={cn("h-full rounded-full transition-all", colors[tone])}
        style={{ width: `${pct}%` }}
      />
    </div>
  );
}
