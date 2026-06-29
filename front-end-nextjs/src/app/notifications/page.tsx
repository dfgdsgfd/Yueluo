import { headers } from "next/headers";
import { NotificationsPage } from "@/components/notifications/notifications-page";
import { apiRequestContextFromHeaders } from "@/lib/api";
import { getNotificationsInitialData } from "@/lib/server/notifications-page-data";

export const metadata = {
  title: "Notifications",
};

export default async function NotificationsRoute() {
  const headerStore = await headers();
  const initialData = await getNotificationsInitialData(
    apiRequestContextFromHeaders(headerStore),
  ).catch(() => null);

  return <NotificationsPage initialData={initialData} />;
}
