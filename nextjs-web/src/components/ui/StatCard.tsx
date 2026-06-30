import { Card } from "./Card";

export function StatCard({
  icon,
  label,
  value,
  sub,
}: {
  icon?: React.ReactNode;
  label: string;
  value: React.ReactNode;
  sub?: React.ReactNode;
}) {
  return (
    <Card className="p-4">
      <div className="flex items-center gap-2 text-zinc-400">
        {icon}
        <span className="text-[11px] font-semibold uppercase tracking-wide text-zinc-500">
          {label}
        </span>
      </div>
      <div className="mt-2 font-mono text-2xl font-bold text-zinc-900">{value}</div>
      {sub ? <div className="mt-0.5 text-[11px] text-zinc-400">{sub}</div> : null}
    </Card>
  );
}
