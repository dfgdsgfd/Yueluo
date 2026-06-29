import type { CreatorListPagination } from "./creator";

export type PointsTaskConfig = {
  id?: number | string;
  task_type: string;
  name: string;
  description?: string | null;
  points: number;
  daily_limit: number;
  is_daily_task: boolean;
  is_active: boolean;
  sort_order?: number;
  completed?: number;
  awarded_points?: number;
  remaining_count?: number;
  reached_limit?: boolean;
  created_at?: string;
  updated_at?: string | null;
};

export type PointsCreatorBonus = {
  id?: number | string;
  rule_id?: number | string | null;
  bonus_percent: number;
  is_active?: boolean;
  starts_at?: string;
  expires_at?: string | null;
};

export type PointsGiftCardProduct = {
  id: number | string;
  name: string;
  description?: string | null;
  face_value?: string | null;
  points_required: number;
  is_active: boolean;
  sort_order?: number;
  available_stock: number;
  redeemed_stock?: number;
  created_at?: string;
  updated_at?: string | null;
};

export type PointsOverviewPayload = {
  points: number;
  today_earned: number;
  daily_cap: number;
  tasks: PointsTaskConfig[];
  gift_cards: PointsGiftCardProduct[];
  active_bonus?: PointsCreatorBonus | null;
  generated_at?: string;
};

export type PointsLogItem = {
  id: number | string;
  amount: number;
  balance_after: number;
  type: string;
  reason?: string | null;
  created_at?: string;
};

export type PointsLogsPayload = {
  list: PointsLogItem[];
  pagination: CreatorListPagination;
};

export type PointsGiftCardRedemption = {
  id: number | string;
  user_id?: number | string;
  product_id: number | string;
  code_id?: number | string;
  code?: string;
  points_spent: number;
  balance_after: number;
  status: string;
  product?: PointsGiftCardProduct;
  created_at?: string;
};

export type PointsRedemptionsPayload = {
  list: PointsGiftCardRedemption[];
  pagination: CreatorListPagination;
};

export type PointsAchievementRule = {
  id?: number | string;
  name: string;
  trigger_type: string;
  threshold_value: number;
  points_reward: number;
  creator_bonus_percent: number;
  bonus_days: number;
  description?: string | null;
  is_active: boolean;
  created_at?: string;
  updated_at?: string | null;
};

export type PointsAdminStatsPayload = {
  total_users: number;
  total_points: number;
  today_awarded: number;
  total_redeemed: number;
  available_cards: number;
  active_tasks: number;
  active_bonus_users: number;
};

export type PointsAdminSettingsPayload = {
  daily_cap: number;
};

export type PointsMaintenancePayload = {
  affected_users?: number;
  deleted_events?: number;
  deleted_stats?: number;
  deleted_achievements?: number;
  deleted_bonuses?: number;
};

export type PointsGiftCardCode = {
  id: number | string;
  product_id: number | string;
  code: string;
  status: string;
  import_batch?: string | null;
  redemption_id?: number | string | null;
  user_id?: number | string | null;
  redeemed_at?: string | null;
  created_at?: string;
  updated_at?: string | null;
};

export type PointsGiftCardCodesPayload = {
  list: PointsGiftCardCode[];
  pagination: CreatorListPagination;
};

export type PointsGiftCardImportPayload = {
  imported: number;
  skipped: number;
  batch: string;
};

export type InviteCodePayload = {
  invite_code: string;
  invite_url: string;
  click_count: number;
  register_count: number;
  total_earnings: number;
  is_active: boolean;
  created_at?: string;
};

export type InviteeItem = {
  user_id: string;
  nickname: string;
  avatar?: string | null;
  joined_at?: string;
};

export type InviteEarningsLogItem = {
  id: number | string;
  amount: number;
  type: string;
  reason?: string | null;
  created_at?: string;
};

export type InviteStatsPayload = InviteCodePayload & {
  invitees: InviteeItem[];
  invitees_total: number;
  earnings_logs: InviteEarningsLogItem[];
  pagination: CreatorListPagination;
};

export type InviteInfoPayload = {
  nickname: string;
  avatar?: string | null;
};

export type InviteClickPayload = {
  recorded: boolean;
};
