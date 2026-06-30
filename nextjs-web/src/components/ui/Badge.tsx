import { cn } from "@/lib/utils";

export type Tone = "emerald" | "amber" | "red" | "blue" | "zinc";

const tones: Record<Tone, string> = {
  emerald: "bg-emerald-50 text-emerald-700 border-emerald-200",
  amber: "bg-amber-50 text-amber-700 border-amber-200",
  red: "bg-red-50 text-red-600 border-red-200",
  blue: "bg-blue-50 text-blue-700 border-blue-200",
  zinc: "bg-zinc-100 text-zinc-600 border-zinc-200",
};

export function Badge({
  tone = "zinc",
  className,
  ...props
}: React.HTMLAttributes<HTMLSpanElement> & { tone?: Tone }) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[10px] font-bold",
        tones[tone],
        className,
      )}
      {...props}
    />
  );
}
