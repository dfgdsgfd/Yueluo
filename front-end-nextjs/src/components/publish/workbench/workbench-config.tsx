"use client";
import dynamic from "next/dynamic";
import {
  AlignLeft,
  Banknote,
  CreditCard,
  FileText,
  Home,
  ImagePlus,
  Play,
  Video,
  Wallet,
  type LucideIcon
} from "lucide-react";
import type {
  RichTextEditorProps
} from "@/components/publish/rich-text-editor";
import {
  type UploadProgress
} from "@/lib/api";
import type {
  WithdrawType
} from "@/lib/types";
import {
  RichTextEditorSkeleton
} from "./article-composer";

export const RichTextEditor = dynamic<RichTextEditorProps>(
  () => import("@/components/publish/rich-text-editor").then((mod) => mod.RichTextEditor),
  {
    loading: () => <RichTextEditorSkeleton />,
    ssr: false,
  },
);


export type PublishMode = "video" | "image" | "article" | "podcast";

export type WorkspaceSection = "home" | "publish" | "withdraw";

export type Visibility = "public" | "private" | "followers";

export type UploadMode = Exclude<PublishMode, "article">;
export type PaymentMethod = "balance" | "points";

export const defaultPaymentMaxPrices: Record<PaymentMethod, number> = {
  balance: 2000,
  points: 50000,
};

export type ImagePaymentSettings = {
  enabled: boolean;
  paymentMethod: PaymentMethod;
  price: string;
};

export type UploadFailure = {
  file: File;
  id: string;
  message: string;
  name: string;
  size: number;
  thumbnailDataUrl?: string | null;
};

export type PendingUploadFile = {
  blobUrl: string;
  file: File;
  thumbnailDataUrl?: string | null;
};

export type UploadProgressState = Record<UploadMode, number | null>;

export type UploadProgressDetailState = Record<UploadMode, UploadProgress | null>;

export type MetricsView = "note" | "live";

export type NoteRange = "recent7" | "recent30";

export type LiveRange = "today" | "yesterday" | "recent7" | "recent30" | "custom";


export type CreatorListPanel = "earnings" | "paid" | "rewards";

export type CreatorTrendSeriesKey = "views" | "likes" | "collects" | "comments" | "followers";


export const sideNavItems = [
  { key: "home", icon: Home },
  { key: "noteManagement", icon: FileText },
  { key: "withdrawManagement", icon: Wallet },
] as const;


export const publishModes = [
  { key: "video", icon: Video },
  { key: "image", icon: ImagePlus },
  { key: "article", icon: AlignLeft },
] as const;


export const visibilityOptions = ["public", "followers", "private"] as const;


export const uploadGuides = {
  video: ["size", "format", "resolution"],
  image: ["imageSize", "imageFormat", "imageResolution"],
  podcast: ["audioFormat", "audioDuration", "rss"],
} as const;


export const uploadLimits = {
  image: 10 * 1024 * 1024,
  podcast: 1024 * 1024 * 1024,
  video: 100 * 1024 * 1024,
} as const;

export const defaultImagePostLimit = 100;


export const uploadAccept = {
  image: "image/*",
  video: "video/*",
  podcast: "audio/*",
} as const satisfies Record<UploadMode, string>;


export const homeCreateActions = [
  {
    key: "image",
    icon: ImagePlus,
    mode: "image",
    cardClassName: "bg-[#fff7ef] hover:bg-[#fff2e4]",
    className: "bg-[#fff4ea] text-[#ff7a1a]",
  },
  {
    key: "video",
    icon: Play,
    mode: "video",
    cardClassName: "bg-[#f0f7ff] hover:bg-[#e9f3ff]",
    className: "bg-[#edf6ff] text-[#2688df]",
  },
] as const satisfies ReadonlyArray<{
  key: string;
  icon: LucideIcon;
  mode: PublishMode;
  cardClassName: string;
  className: string;
}>;


export const noteMetricKeys = [
  "exposure",
  "views",
  "likes",
  "comments",
  "netFollowers",
  "newFollows",
  "coverClicks",
  "completionRate",
  "collections",
  "shares",
  "unfollows",
  "profileVisitors",
] as const;


export const liveMetricKeys = [
  "liveSessions",
  "liveDuration",
  "averageViewers",
  "averageInteractions",
  "liveFollowers",
  "diamonds",
  "paidOrders",
  "merchantRevenue",
] as const;


export const noteRanges = ["recent7", "recent30"] as const;

export const liveRanges = ["today", "yesterday", "recent7", "recent30", "custom"] as const;

export const metricsViews = ["note"] as const;

export const creatorEarningsLogLimit = 5;

export const creatorPaidContentLimit = 4;

export const creatorQualityRewardsLimit = 4;

export const withdrawOrderLimit = 8;

export const withdrawTypes = [
  { value: "cash", labelKey: "withdrawType.cash", icon: Banknote },
  { value: "moon_coin", labelKey: "withdrawType.moonCoin", icon: CreditCard },
] as const satisfies ReadonlyArray<{ value: WithdrawType; labelKey: string; icon: LucideIcon }>;

export const creatorTrendSeries = [
  { key: "views", labelKey: "trend.views", className: "bg-primary" },
  { key: "likes", labelKey: "trend.likes", className: "bg-[#79d8d0]" },
  { key: "collects", labelKey: "trend.collects", className: "bg-[#f2b35d]" },
  { key: "comments", labelKey: "trend.comments", className: "bg-[#7c8cf8]" },
  { key: "followers", labelKey: "trend.followers", className: "bg-[#5aa7ee]" },
] as const satisfies ReadonlyArray<{
  key: CreatorTrendSeriesKey;
  labelKey: string;
  className: string;
}>;


export const notePeriodKeys = {
  recent7: "notePeriodRecent7",
  recent30: "notePeriodRecent30",
} as const satisfies Record<NoteRange, string>;


export const livePeriodKeys = {
  today: "livePeriodToday",
  yesterday: "livePeriodYesterday",
  recent7: "livePeriodRecent7",
  recent30: "livePeriodRecent30",
  custom: "livePeriodCustom",
} as const satisfies Record<LiveRange, string>;
