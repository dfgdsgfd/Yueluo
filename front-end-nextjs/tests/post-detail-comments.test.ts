import { describe, expect, it } from "vitest";
import {
  mapDetailComment,
  mergeCommentReplies
} from "../src/components/feed/post-detail/comments";
import type { DetailComment } from "../src/components/feed/post-detail/post-detail-types";

describe("post detail comments", () => {
  it("maps backend reply_count so nested replies can be expanded", () => {
    const comment = mapDetailComment(
      {
        id: 21,
        content: "child reply",
        parent_id: 20,
        reply_count: 1,
      },
      "Unknown",
      "recent",
      "en",
    );

    expect(comment.parentId).toBe(20);
    expect(comment.replyCount).toBe(1);
    expect(comment.repliesPage).toBe(0);
  });

  it("merges paged replies without duplicating existing comments", () => {
    const first = detailComment(1);
    const second = detailComment(2);
    const third = detailComment(3);

    expect(mergeCommentReplies([first, second], [second, third], true).map((reply) => reply.id))
      .toEqual([1, 2, 3]);
    expect(mergeCommentReplies([first], [second, first], false).map((reply) => reply.id))
      .toEqual([2, 1]);
  });
});

function detailComment(id: number): DetailComment {
  return {
    id,
    author: `User ${id}`,
    body: `Comment ${id}`,
    likes: 0,
    meta: "recent",
    ownerIds: [String(id)],
    replies: [],
    repliesExpanded: false,
    repliesPage: 0,
    repliesStatus: "idle",
    replyCount: 0,
    userId: String(id),
  };
}
