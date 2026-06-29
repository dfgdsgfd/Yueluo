import { Suspense } from "react";
import { AdminConsolePage } from "@/components/admin/admin-page/admin-console-page";
import { getPrivateEntryPaths } from "@/lib/private-entry-paths";

export default function AdminConsoleRoute() {
  return (
    <Suspense fallback={null}>
      <AdminConsolePage backendApiEntryPath={getPrivateEntryPaths().backendApi} />
    </Suspense>
  );
}
