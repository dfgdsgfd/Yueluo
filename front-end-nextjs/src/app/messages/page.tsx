import { headers } from "next/headers";
import { MessagesPage } from "@/components/messages/messages-page";
import { apiRequestContextFromHeaders } from "@/lib/api";
import { getMessagesInitialData } from "@/lib/server/messages-page-data";

export const metadata = {
  title: "Messages",
};

export default async function MessagesRoute() {
  const headerStore = await headers();
  const initialData = await getMessagesInitialData(
    apiRequestContextFromHeaders(headerStore),
  ).catch(() => null);

  return <MessagesPage initialData={initialData} />;
}
