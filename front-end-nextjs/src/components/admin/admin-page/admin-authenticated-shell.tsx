"use client";

import { lazy, Suspense, useState } from "react";
import { Loader2 } from "lucide-react";
import { usePathname, useSearchParams } from "next/navigation";
import { useTranslations } from "next-intl";
import type { AdminUser } from "@/lib/types";
import { adminViewSearchSuffix, parseAdminViewSearchParams } from "./admin-view-routing";
import { createNavSections } from "./nav";
import { AdminSidebar, AdminTopbar } from "./admin-shell-widgets";
import type { AdminView } from "./types";

const DashboardPanel = lazy(() =>
  import("./dashboard-panel").then((module) => ({ default: module.DashboardPanel })),
);
const ResourcePanel = lazy(() =>
  import("./resource-panel").then((module) => ({ default: module.ResourcePanel })),
);
const WithdrawPanel = lazy(() =>
  import("./withdraw-panel").then((module) => ({ default: module.WithdrawPanel })),
);
const InvitePanel = lazy(() =>
  import("./invite-panel").then((module) => ({ default: module.InvitePanel })),
);
const CouponPanel = lazy(() =>
  import("./coupon-panel").then((module) => ({ default: module.CouponPanel })),
);
const PointsPanel = lazy(() =>
  import("./points-panel").then((module) => ({ default: module.PointsPanel })),
);
const OnboardingSettingsPanel = lazy(() =>
  import("./onboarding-settings-panel").then((module) => ({ default: module.OnboardingSettingsPanel })),
);
const AIAgentPanel = lazy(() =>
  import("./ai-agent-panel").then((module) => ({ default: module.AIAgentPanel })),
);
const AccessBlockPanel = lazy(() =>
  import("./access-block-panel").then((module) => ({ default: module.AccessBlockPanel })),
);
const HiddenWatermarkAccessPanel = lazy(() =>
  import("./hidden-watermark-access-panel").then((module) => ({ default: module.HiddenWatermarkAccessPanel })),
);
const LogsPanel = lazy(() =>
  import("./logs-panel").then((module) => ({ default: module.LogsPanel })),
);
const SettingsPanel = lazy(() =>
  import("./settings-panel").then((module) => ({ default: module.SettingsPanel })),
);
const SystemUpdatePanel = lazy(() =>
  import("./system-update-panel").then((module) => ({ default: module.SystemUpdatePanel })),
);
const ComponentCheckPanel = lazy(() =>
  import("./component-check-panel").then((module) => ({ default: module.ComponentCheckPanel })),
);
const MaintenancePanel = lazy(() =>
  import("./maintenance-panel").then((module) => ({ default: module.MaintenancePanel })),
);
const DatabasePanel = lazy(() =>
  import("./database-panel").then((module) => ({ default: module.DatabasePanel })),
);
const OperationsPanel = lazy(() =>
  import("./operations-panel").then((module) => ({ default: module.OperationsPanel })),
);
const QueuesPanel = lazy(() =>
  import("@/components/admin/runtime-panels").then((module) => ({ default: module.QueuesPanel })),
);
const ObservabilityPanel = lazy(() =>
  import("@/components/admin/runtime-panels").then((module) => ({
    default: module.ObservabilityPanel,
  })),
);

export type AdminAuthenticatedShellProps = {
  admin: AdminUser;
  backendApiEntryPath: string;
  onLogout: () => void;
  token: string;
};

export function AdminAuthenticatedShell({
  admin,
  backendApiEntryPath,
  onLogout,
  token,
}: AdminAuthenticatedShellProps) {
  const adminT = useTranslations("adminPortal");
  const pathname = usePathname() || "/admin/console";
  const searchParams = useSearchParams();
  const view = parseAdminViewSearchParams(searchParams);
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const [refreshKey, setRefreshKey] = useState(0);
  const navSections = createNavSections(
    {
      item: (id) => adminT(`navigation.items.${id}`),
      resource: (id) => adminT(`navigation.resources.${id}`),
      section: (id) => adminT(`navigation.sections.${id}`),
    },
    backendApiEntryPath,
  );
  const activeTitle =
    navSections
      .flatMap((section) => section.items)
      .find((item) => item.view && sameAdminView(view, item.view))
      ?.label ?? adminT("navigation.items.dashboard");

  return (
    <div className="min-h-dvh bg-[#f5f6fa] text-[#17171d]" translate="no">
      <div className="flex min-h-dvh">
        <AdminSidebar
          activeView={view}
          getViewHref={(nextView) => `${pathname}${adminViewSearchSuffix(nextView, searchParams)}`}
          mobileOpen={mobileNavOpen}
          onClose={() => setMobileNavOpen(false)}
          onNavigate={() => setMobileNavOpen(false)}
          sections={navSections}
        />
        <div className="flex min-w-0 flex-1 flex-col lg:pl-[280px]">
          <AdminTopbar
            admin={admin}
            title={activeTitle}
            onMenu={() => setMobileNavOpen(true)}
            onRefresh={() => setRefreshKey((value) => value + 1)}
            onLogout={onLogout}
          />
          <main className="min-w-0 flex-1 px-3 py-4 sm:px-5 lg:px-6 xl:px-8">
            <div className="mx-auto w-full max-w-[1560px] min-w-0">
              <Suspense fallback={<AdminPanelLoading label={adminT("loading")} />}>
                {view.kind === "dashboard" ? (
                  <DashboardPanel key={`dashboard-${refreshKey}`} token={token} />
                ) : view.kind === "resource" ? (
                  <ResourcePanel key={`${view.resource}-${refreshKey}`} resource={view.resource} token={token} />
                ) : view.kind === "withdraw" ? (
                  <WithdrawPanel key={`withdraw-${refreshKey}`} token={token} />
                ) : view.kind === "invite" ? (
                  <InvitePanel key={`invite-${refreshKey}`} token={token} />
                ) : view.kind === "coupon" ? (
                  <CouponPanel key={`coupon-${refreshKey}`} token={token} />
                ) : view.kind === "points" ? (
                  <PointsPanel key={`points-${refreshKey}`} token={token} />
                ) : view.kind === "onboarding-settings" ? (
                  <OnboardingSettingsPanel key={`onboarding-settings-${refreshKey}`} token={token} />
                ) : view.kind === "ai-agent" ? (
                  <AIAgentPanel key={`ai-agent-${refreshKey}`} token={token} />
                ) : view.kind === "access-block" ? (
                  <AccessBlockPanel key={`access-block-${refreshKey}`} token={token} />
                ) : view.kind === "hidden-watermark-access" ? (
                  <HiddenWatermarkAccessPanel key={`hidden-watermark-access-${refreshKey}`} token={token} />
                ) : view.kind === "settings" ? (
                  <SettingsPanel key={`settings-${refreshKey}`} token={token} />
                ) : view.kind === "system-update" ? (
                  <SystemUpdatePanel key={`system-update-${refreshKey}`} token={token} />
                ) : view.kind === "queues" ? (
                  <QueuesPanel key={`queues-${refreshKey}`} token={token} />
                ) : view.kind === "logs" ? (
                  <LogsPanel key={`logs-${refreshKey}`} token={token} />
                ) : view.kind === "observability" ? (
                  <ObservabilityPanel key={`observability-${refreshKey}`} token={token} />
                ) : view.kind === "database" ? (
                  <DatabasePanel key={`database-${refreshKey}`} token={token} />
                ) : view.kind === "component-check" ? (
                  <ComponentCheckPanel key={`component-check-${refreshKey}`} token={token} />
                ) : view.kind === "maintenance" ? (
                  <MaintenancePanel key={`maintenance-${refreshKey}`} token={token} />
                ) : (
                  <OperationsPanel key={`operations-${refreshKey}`} token={token} />
                )}
              </Suspense>
            </div>
          </main>
        </div>
      </div>
    </div>
  );
}

function sameAdminView(a: AdminView, b: AdminView) {
  if (a.kind !== b.kind) {
    return false;
  }
  return a.kind !== "resource" || (b.kind === "resource" && a.resource === b.resource);
}

function AdminPanelLoading({ label }: { label: string }) {
  return (
    <div className="flex min-h-48 items-center justify-center rounded-xl border border-black/[0.06] bg-white/80">
      <Loader2 className="size-5 animate-spin text-[#1d4ed8]" aria-hidden="true" />
      <span className="ml-2 text-sm text-[#6b7280]">{label}</span>
    </div>
  );
}
