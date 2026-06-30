import { redirect } from "next/navigation";
import { auth } from "@/auth";
import { DashboardShell } from "@/components/layout/DashboardShell";

export default async function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const session = await auth();
  if (!session) redirect("/login");

  return (
    <DashboardShell
      clinics={session.clinics}
      user={{ name: session.user.name, role: session.user.role }}
    >
      {children}
    </DashboardShell>
  );
}
