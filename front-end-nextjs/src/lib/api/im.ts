import type {
  ImConversation,
  ImMessagesPayload,
  ImMessage,
  ImSyncPayload,
} from "../types";
import {
  ApiRequestContext,
  type ApiRequestOptions,
  DEFAULT_SERVER_API_ORIGIN,
  apiGet,
  apiPost,
  apiRequestEnvelope,
  isRecord,
  numberFromUnknown,
} from "./core";
import type { ImRequestOptions } from "../im-background-request";

export function getImConversations(): Promise<ImConversation[]>;
export function getImConversations(
  context: ApiRequestContext,
  options?: ImRequestOptions,
): Promise<ImConversation[]>;
export function getImConversations(
  context: ApiRequestContext = {},
  options: ImRequestOptions = {},
) {
  return apiGet<ImConversation[]>("/api/im/conversations", undefined, {
    context,
    ...options,
  });
}

export function createImConversation(memberIds: Array<number | string>) {
  return apiPost<ImConversation>("/api/im/conversations", {
    member_ids: memberIds,
  });
}

export async function getImMessages(
  conversationId: string | number,
  query: { before?: string | number | null; limit?: number } = {},
  context: ApiRequestContext = {},
  options: ImRequestOptions = {},
): Promise<ImMessagesPayload> {
  const limit = query.limit ?? 50;
  const envelope = await apiRequestEnvelope<ImMessage[]>(
    `/api/im/conversations/${conversationId}/messages`,
    {
      method: "GET",
      query: {
        before: query.before,
        limit,
      },
      context,
      ...options,
    },
  );
  const dataRecord = isRecord(envelope.data) ? envelope.data : null;
  const topPagination = isRecord(envelope.pagination)
    ? envelope.pagination
    : null;
  const nestedPagination = isRecord(dataRecord?.pagination)
    ? dataRecord.pagination
    : null;
  const messages = Array.isArray(envelope.data)
    ? envelope.data
    : Array.isArray(dataRecord?.messages)
      ? (dataRecord.messages as ImMessage[])
      : [];

  return {
    messages,
    pagination: {
      limit: numberFromUnknown(
        topPagination?.limit ?? nestedPagination?.limit,
        limit,
      ),
      has_more: Boolean(
        topPagination?.has_more ??
        nestedPagination?.has_more ??
        messages.length >= limit,
      ),
      next_before:
        (topPagination?.next_before as string | number | null | undefined) ??
        (nestedPagination?.next_before as string | number | null | undefined) ??
        messages[0]?.id ??
        null,
    },
  };
}

export function sendImMessage(
  conversationId: string | number,
  content: string,
) {
  return apiPost<ImMessage>(
    `/api/im/conversations/${conversationId}/messages`,
    {
      content,
      client_msg_id: `${Date.now()}-${Math.random().toString(36).slice(2)}`,
    },
  );
}

export function markImMessageRead(
  messageId: string | number,
  options: ImRequestOptions = {},
) {
  if (Object.keys(options).length === 0) {
    return apiPost(`/api/im/messages/${messageId}/read`);
  }

  return apiPost(`/api/im/messages/${messageId}/read`, undefined, options);
}

export function syncImMessages(
  since: string | number = 0,
  options: ImRequestOptions = {},
) {
  return apiGet<ImSyncPayload>("/api/im/sync", { since, limit: 200 }, options);
}

type ImSessionPayload = {
  uid: number;
  ts: string;
  sig: string;
  api_base: string;
  ws_url: string;
  mode: string;
  use_proxy: boolean;
};

export function getImWebSocketUrl(token: string): Promise<string>;
export function getImWebSocketUrl(
  token: string,
  options: Omit<ApiRequestOptions, "body" | "method" | "query">,
): Promise<string>;
export async function getImWebSocketUrl(
  token: string,
  options: Omit<ApiRequestOptions, "body" | "method" | "query"> = {},
): Promise<string> {
  const mode = process.env.NEXT_PUBLIC_IM_MODE ?? "direct";

  if (mode === "proxy") {
    const base =
      process.env.NEXT_PUBLIC_API_BASE_URL ??
      process.env.NEXT_PUBLIC_BACKEND_ORIGIN ??
      DEFAULT_SERVER_API_ORIGIN;
    const url = new URL("/api/im/ws", base);
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
    url.searchParams.set("token", token);
    return url.toString();
  }

  const { context, ...requestOptions } = options;
  const session = await apiPost<ImSessionPayload>(
    "/api/im/session",
    undefined,
    {
      context: { ...context, token },
      ...requestOptions,
    },
  );
  return session.ws_url;
}
