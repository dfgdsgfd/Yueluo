import { headers } from "next/headers";
import { MessagesPage } from "@/components/messages/messages-page";
import { apiRequestContextFromHeaders } from "@/lib/api";
import { getMessagesInitialData } from "@/lib/server/messages-page-data";

export const metadata = {
  title: "Conversation",
};

type MessageDetailRouteProps = {
  params: Promise<{
    conversationId: string;
  }>;
};

export default async function MessageDetailRoute({ params }: MessageDetailRouteProps) {
  const { conversationId } = await params;
  const headerStore = await headers();
  const initialData = await getMessagesInitialData(
    apiRequestContextFromHeaders(headerStore),
    conversationId,
  ).catch(() => null);

  return (
    <MessagesPage
      conversationId={conversationId}
      initialData={initialData}
    />
  );
}
