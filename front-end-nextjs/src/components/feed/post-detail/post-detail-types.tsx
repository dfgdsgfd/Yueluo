"use client";
import dynamic from "next/dynamic";
import type {
  FeedPost,
  ReportReason
} from "@/lib/types";
import {
  VideoPlayerSkeleton
} from "./media-header";

export const ShakaVideoPlayer = dynamic(
  () => import("@/components/media/shaka-video-player").then((mod) => mod.ShakaVideoPlayer),
  {
    loading: () => <VideoPlayerSkeleton />,
    ssr: false,
  },
);


export type DetailComment = {
  id: number | string;
  author: string;
  body: string;
  meta: string;
  likes: number;
  liked?: boolean;
  ownerIds: string[];
  parentId?: number | string | null;
  replies?: DetailComment[];
  repliesExpanded?: boolean;
  repliesPage?: number;
  repliesStatus?: "idle" | "loading" | "loaded" | "error";
  replyCount: number;
  status?: "pending" | "failed" | "sent";
  userId: string;
  avatar?: string | null;
};


export type CommentState = {
  comments: DetailComment[];
  countDelta: number;
  postId?: FeedPost["id"];
  status: "idle" | "loading" | "loaded" | "error";
};


export type CommentDraftState = {
  postId?: FeedPost["id"];
  replyTarget?: Pick<DetailComment, "author" | "id"> | null;
  text: string;
};


export type FollowState = {
  buttonType?: string;
  isFollowing: boolean;
  status: "idle" | "loading" | "ready" | "error";
  userId?: string;
};


export type DetailActionState = {
  disliked: boolean;
  dislikeStatus: "idle" | "loading" | "ready" | "error";
  postId?: FeedPost["id"];
  reported: boolean;
  reportStatus: "idle" | "loading" | "ready" | "error";
};


export const reportReasons = [
  "spam",
  "porn",
  "violence",
  "fake",
  "harassment",
  "copyright",
  "other",
] as const satisfies ReportReason[];
