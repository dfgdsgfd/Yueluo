import { expect, test } from "@playwright/test";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import {
  getMessageBadgeCount,
  markLatestImMessageRead,
} from "../../src/lib/im-unread";
import type { ImMessage } from "../../src/lib/types";

test("marks the latest message as read to advance the conversation cursor", async () => {
  const messages = [
    { id: 11, sender_id: 2 },
    { id: 12, sender_id: 1 },
    { id: 13, sender_id: 2 },
    { id: 14, sender_id: 1 },
  ] as ImMessage[];
  const markedMessageIds: Array<string | number> = [];

  const result = await markLatestImMessageRead(messages, async (messageId) => {
    markedMessageIds.push(messageId);
  });

  expect(result).toBe(14);
  expect(markedMessageIds).toEqual([14]);
});

test("marks the latest message when the loaded page only contains the viewer's messages", async () => {
  const messages = [{ id: 21, sender_id: 1 }] as ImMessage[];
  const markedMessageIds: Array<string | number> = [];

  const result = await markLatestImMessageRead(messages, async (messageId) => {
    markedMessageIds.push(messageId);
  });

  expect(result).toBe(21);
  expect(markedMessageIds).toEqual([21]);
});

test("background badge count degrades to zero when IM is unavailable", async () => {
  const originalFetch = globalThis.fetch;
  globalThis.fetch = (async () =>
    new Response(JSON.stringify({ message: "unavailable" }), {
      status: 503,
      headers: { "content-type": "application/json" },
    })) as typeof fetch;

  try {
    const result = await getMessageBadgeCount({ background: true });

    expect(result).toBe(0);
  } finally {
    globalThis.fetch = originalFetch;
  }
});

test("global message badge sync stays HTTP-only without opening a websocket", async () => {
  const source = await readFile(
    join(
      process.cwd(),
      "src/components/feed/explore/use-message-badge-sync.ts",
    ),
    "utf8",
  );

  expect(source).not.toContain("new WebSocket");
  expect(source).not.toContain("getImWebSocketUrl");
  expect(source).not.toContain("/api/im/session");
  expect(source).toContain("getMessageBadgeCount");
});
