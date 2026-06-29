import { AdminPage } from "@/components/admin/admin-page";
import { getPrivateEntryPaths } from "@/lib/private-entry-paths";

export const metadata = {
  title: "Admin",
};

export default function AdminRoute() {
  return <AdminPage backendApiEntryPath={getPrivateEntryPaths().backendApi} />;
}
