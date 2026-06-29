import type { FeedPagination } from "./content";

export type CreatorOverviewPayload = {
  balance: number;
  total_earnings: number;
  withdrawn_amount: number;
  today_earnings: number;
  month_earnings: number;
};

export type CreatorStatsPayload = {
  window_days: number;
  generated_at: string;
  fans: Record<string, number>;
  post_totals: Record<string, number>;
  interactions: Record<string, Record<string, number>>;
};

export type CreatorConfigPayload = {
  platformFeeRate: number;
  creatorShareRate: number;
  withdrawEnabled: boolean;
  minWithdrawAmount: number;
  extendedEarnings?: {
    enabled: boolean;
    dailyCap: number;
    rates: Record<string, number>;
  };
};

export type CreatorTrendsPayload = {
  days: number;
  labels: string[];
  views: number[];
  likes: number[];
  collects: number[];
  comments: number[];
  followers: number[];
};

export type CreatorListPagination = FeedPagination & {
  totalPages?: number;
};

export type CreatorSource = {
  id: number | string;
  title?: string | null;
};

export type CreatorBuyer = {
  id: number | string;
  nickname?: string | null;
  avatar?: string | null;
  user_id?: string | null;
};

export type CreatorEarningsLogItem = {
  id: number | string;
  amount: number;
  gross_amount?: number;
  balance_after?: number;
  type?: string | null;
  platform_fee?: number;
  reason?: string | null;
  source?: CreatorSource | null;
  buyer?: CreatorBuyer | null;
  created_at?: string;
};

export type CreatorEarningsLogPayload = {
  list: CreatorEarningsLogItem[];
  pagination: CreatorListPagination;
};

export type CreatorPaidContentItem = {
  id: number | string;
  title: string;
  type?: number;
  cover?: string | null;
  price: number;
  view_count?: number;
  like_count?: number;
  collect_count?: number;
  sales_count: number;
  total_revenue: number;
  created_at?: string;
};

export type CreatorPaidContentPayload = {
  list: CreatorPaidContentItem[];
  pagination: CreatorListPagination;
};

export type CreatorQualityRewardPost = {
  id: number | string;
  title?: string | null;
  type?: number;
  quality_level?: string | null;
  cover?: string | null;
  created_at?: string;
};

export type CreatorQualityRewardItem = {
  id: number | string;
  amount: number;
  reason?: string | null;
  post?: CreatorQualityRewardPost | null;
  created_at?: string;
};

export type CreatorQualityRewardStatsItem = {
  quality_label: string;
  count: number;
  total_amount: number;
};

export type CreatorQualityRewardsPayload = {
  list: CreatorQualityRewardItem[];
  total_earnings: number;
  stats: CreatorQualityRewardStatsItem[];
  pagination: CreatorListPagination;
};
