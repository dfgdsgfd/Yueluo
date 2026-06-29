"use client";
import {
  type MobilePublishVisibility
} from "../mobile-drafts";
import { type UploadProgress } from "@/lib/api";
import { type UploadAsset } from "@/lib/types";

export const titleLimit = 30;

export const bodyLimit = 100000;

export const defaultImageLimit = 100;

export const acceptedMediaTypes = "image/*,video/*";


export const emojiGroups = [
  {
    id: "recent",
    labelKey: "emoji.recent",
    emojis: ["😀", "😄", "😂", "🥰", "😍", "😘", "😎", "😭", "😡", "👍", "👏", "🙏", "🔥", "✨", "💖", "🎉"],
  },
  {
    id: "faces",
    labelKey: "emoji.faces",
    emojis: ["🙂", "😉", "😊", "😋", "🤔", "😳", "🥺", "😴", "🤤", "😇", "🤩", "😈", "😤", "😱", "🤯", "🥹"],
  },
  {
    id: "gestures",
    labelKey: "emoji.gestures",
    emojis: ["👌", "✌️", "🤞", "🤟", "🤘", "👋", "🤲", "🫶", "💪", "👀", "💋", "💅", "🙌", "🤝", "🫰", "🫵"],
  },
  {
    id: "objects",
    labelKey: "emoji.symbols",
    emojis: ["❤️", "🧡", "💛", "💚", "💙", "💜", "🖤", "🤍", "⭐", "🌙", "☀️", "🌈", "🍓", "🍑", "🍬", "🎀"],
  },
] as const;


export type EmojiGroupId = (typeof emojiGroups)[number]["id"];


export type MediaKind = "image" | "video";

export type MobileMarkdownEditorMode = "live" | "source";


export type MobileMediaAsset = {
  file: File | null;
  id: string;
  isFreePreview: boolean;
  isProtected: boolean;
  kind: MediaKind;
  name: string;
  previewUrl: string;
  remoteAsset?: UploadAsset;
  uploadError?: string | null;
  uploadProgress?: number;
  uploadStatus?: "queued" | "uploading" | "succeeded" | "failed";
};


export type Visibility = MobilePublishVisibility;

export type MobilePaymentMethod = "balance" | "points";

export const defaultMobilePaymentMaxPrices: Record<MobilePaymentMethod, number> = {
  balance: 2000,
  points: 50000,
};


export type MediaDragState = {
  currentX: number;
  currentY: number;
  id: string;
  isDragging: boolean;
  pointerId: number;
  startX: number;
  startY: number;
  targetId: string | null;
};


export type MobileUploadProgressState = {
  detail: UploadProgress | null;
  percent: number;
  phase: "uploading" | "publishing";
} | null;


export function isMobilePublishVisibility(value: unknown): value is Visibility {
  return value === "public" || value === "followers" || value === "private";
}

export function mapBackendMobileVisibility(value: unknown): Visibility {
  if (value === "friends_only" || value === "followers") return "followers";
  if (value === "private") return "private";
  return "public";
}

export function parseMobileTopics(value: string) {
  return Array.from(new Set(
    value
      .split(/[,，]/)
      .map((topic) => topic.trim().replace(/^#+/, ""))
      .filter(Boolean),
  )).slice(0, 12);
}

export function formatMobileTopics(topics: string[]) {
  return topics.join(", ");
}

export function appendMobileCustomTopic(currentTopic: string, input: string, limit = 12) {
  const nextTopic = input.trim().replace(/^#+/, "").trim();
  if (!nextTopic) {
    return { reason: "required" as const };
  }
  const nextTopics = parseMobileTopics(`${currentTopic},${nextTopic}`);
  if (nextTopics.length >= limit && !nextTopics.includes(nextTopic)) {
    return { reason: "limit" as const };
  }
  return { value: formatMobileTopics(nextTopics) };
}
