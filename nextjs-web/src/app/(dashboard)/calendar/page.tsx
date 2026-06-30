"use client";

import { useState } from "react";
import { useActiveClinic } from "@/stores/ui-store";
import { useAppointments } from "@/lib/queries";
import { QueryBoundary } from "@/components/ui/QueryBoundary";
import { Calendar } from "./Calendar";
import { SettingsPanel } from "./SettingsPanel";
import { WidgetCalendarSection } from "./WidgetCalendarSection";
import { useCalendarSettings } from "./settings";

export default function CalendarPage() {
  const active = useActiveClinic();
  const q = useAppointments(active?.id ?? null);
  const { settings, update, reset, hydrated } = useCalendarSettings();
  const [panelOpen, setPanelOpen] = useState(false);

  const appointments = q.data ?? [];

  return (
    <div className="relative h-full">
      <QueryBoundary isLoading={q.isLoading || !hydrated} error={q.error}>
        <Calendar
          appointments={appointments}
          settings={settings}
          onCustomize={() => setPanelOpen(true)}
          clinicName={active?.name}
        />
      </QueryBoundary>

      <SettingsPanel
        open={panelOpen}
        onClose={() => setPanelOpen(false)}
        settings={settings}
        update={update}
        reset={reset}
      >
        <WidgetCalendarSection clinicId={active?.id ?? null} />
      </SettingsPanel>
    </div>
  );
}
