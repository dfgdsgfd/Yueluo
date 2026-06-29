export type ImUser = {
  id: number | string;
  username?: string | null;
  nickname?: string | null;
  avatar?: string | null;
};

export type ImConversation = {
  id: number | string;
  external_id?: number | string | null;
  type: string;
  name?: string | null;
  creator_id?: number | string;
  member_ids?: Array<number | string>;
  members?: ImUser[];
  last_message?: ImMessage | null;
  unread_count?: number;
  created_at?: string;
  updated_at?: string | null;
};

export type ImMessage = {
  id: number | string;
  conversation_id: number | string;
  sender_id: number | string;
  content: string;
  client_msg_id?: string | null;
  created_at?: string;
  status?: "sent" | "delivered" | "read" | string;
};

export type ImMessagesPayload = {
  messages: ImMessage[];
  pagination: {
    limit?: number;
    has_more?: boolean;
    next_before?: number | string | null;
  };
};

export type ImSyncPayload = {
  messages: ImMessage[];
  cursor: number | string;
  has_more?: boolean;
};
