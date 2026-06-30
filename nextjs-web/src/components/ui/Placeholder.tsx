import { Card } from "./Card";

/** Placeholder marks a screen whose design/data lands in a later delivery piece. */
export function Placeholder({
  piece,
  children,
}: {
  piece?: string;
  children?: React.ReactNode;
}) {
  return (
    <Card className="flex flex-col items-center justify-center gap-2 p-12 text-center">
      <div className="text-sm font-semibold text-zinc-700">Bu ekran yakında bağlanacak</div>
      <p className="max-w-md text-xs text-zinc-500">
        {children ??
          "Tasarım bu sayfaya birebir taşınacak ve gerçek backend verisine bağlanacak."}
      </p>
      {piece ? (
        <span className="mt-1 rounded-full bg-zinc-100 px-2.5 py-1 text-[10px] font-bold uppercase tracking-wider text-zinc-500">
          {piece}
        </span>
      ) : null}
    </Card>
  );
}
