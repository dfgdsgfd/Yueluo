BEGIN;

CREATE TABLE IF NOT EXISTS points_task_configs (
  id BIGSERIAL PRIMARY KEY,
  task_type VARCHAR(64) NOT NULL,
  name VARCHAR(128) NOT NULL,
  description TEXT,
  points NUMERIC(12, 2) NOT NULL DEFAULT 0,
  daily_limit INTEGER NOT NULL DEFAULT 0,
  is_daily_task BOOLEAN NOT NULL DEFAULT TRUE,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_points_task_configs_task_type
  ON points_task_configs (task_type);

CREATE TABLE IF NOT EXISTS points_task_events (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  task_type VARCHAR(64) NOT NULL,
  target_key VARCHAR(191) NOT NULL,
  event_date DATE NOT NULL,
  points NUMERIC(12, 2) NOT NULL DEFAULT 0,
  reason TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_points_task_events_daily_target
  ON points_task_events (user_id, task_type, target_key, event_date);
CREATE INDEX IF NOT EXISTS idx_points_task_events_date
  ON points_task_events (event_date);

CREATE TABLE IF NOT EXISTS points_daily_stats (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  task_type VARCHAR(64) NOT NULL,
  stat_date DATE NOT NULL,
  completed_count INTEGER NOT NULL DEFAULT 0,
  awarded_points NUMERIC(12, 2) NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_points_daily_stats_user_task_date
  ON points_daily_stats (user_id, task_type, stat_date);
CREATE INDEX IF NOT EXISTS idx_points_daily_stats_user_date
  ON points_daily_stats (user_id, stat_date);

CREATE TABLE IF NOT EXISTS points_achievement_rules (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(128) NOT NULL,
  trigger_type VARCHAR(64) NOT NULL,
  threshold_value INTEGER NOT NULL DEFAULT 0,
  points_reward NUMERIC(12, 2) NOT NULL DEFAULT 0,
  creator_bonus_percent NUMERIC(8, 2) NOT NULL DEFAULT 0,
  bonus_days INTEGER NOT NULL DEFAULT 0,
  description TEXT,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_points_achievement_rules_active
  ON points_achievement_rules (is_active);

CREATE TABLE IF NOT EXISTS user_achievement_rewards (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  rule_id BIGINT NOT NULL,
  points_awarded NUMERIC(12, 2) NOT NULL DEFAULT 0,
  creator_bonus_percent NUMERIC(8, 2) NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_achievement_rewards_user_rule
  ON user_achievement_rewards (user_id, rule_id);

CREATE TABLE IF NOT EXISTS user_creator_bonus (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  rule_id BIGINT,
  bonus_percent NUMERIC(8, 2) NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  starts_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_creator_bonus_user_rule
  ON user_creator_bonus (user_id, rule_id);
CREATE INDEX IF NOT EXISTS idx_user_creator_bonus_active
  ON user_creator_bonus (user_id, is_active, starts_at, expires_at);

CREATE TABLE IF NOT EXISTS gift_card_products (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(128) NOT NULL,
  description TEXT,
  face_value VARCHAR(64),
  points_required NUMERIC(12, 2) NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_gift_card_products_active_sort
  ON gift_card_products (is_active, sort_order, id);

CREATE TABLE IF NOT EXISTS gift_card_codes (
  id BIGSERIAL PRIMARY KEY,
  product_id BIGINT NOT NULL,
  code TEXT NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'available',
  import_batch VARCHAR(64),
  redemption_id BIGINT,
  user_id BIGINT,
  redeemed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_gift_card_codes_product_code
  ON gift_card_codes (product_id, code);
CREATE INDEX IF NOT EXISTS idx_gift_card_codes_stock
  ON gift_card_codes (product_id, status, id);

CREATE TABLE IF NOT EXISTS gift_card_redemptions (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  product_id BIGINT NOT NULL,
  code_id BIGINT NOT NULL,
  code_snapshot TEXT NOT NULL,
  points_spent NUMERIC(12, 2) NOT NULL DEFAULT 0,
  balance_after NUMERIC(12, 2) NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL DEFAULT 'completed',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gift_card_redemptions_user_created
  ON gift_card_redemptions (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_gift_card_redemptions_product_created
  ON gift_card_redemptions (product_id, created_at DESC);

INSERT INTO points_task_configs
  (task_type, name, description, points, daily_limit, is_daily_task, is_active, sort_order, created_at, updated_at)
VALUES
  ('comment', '评论', '发布评论自动获得积分', 2, 10, TRUE, TRUE, 10, NOW(), NOW()),
  ('click', '点击', '点击进入内容详情自动获得积分', 1, 30, TRUE, TRUE, 20, NOW(), NOW()),
  ('like', '点赞', '点赞内容自动获得积分', 1, 20, TRUE, TRUE, 30, NOW(), NOW()),
  ('collect', '收藏', '收藏内容自动获得积分', 2, 10, TRUE, TRUE, 40, NOW(), NOW()),
  ('view', '浏览', '浏览内容自动获得积分', 1, 30, TRUE, TRUE, 50, NOW(), NOW()),
  ('post', '发帖', '发布公开内容自动获得积分', 5, 5, TRUE, TRUE, 60, NOW(), NOW()),
  ('set_avatar', '设置头像', '设置头像获得一次性积分', 2, 1, FALSE, TRUE, 70, NOW(), NOW()),
  ('set_background', '设置背景', '设置个人背景获得一次性积分', 2, 1, FALSE, TRUE, 80, NOW(), NOW()),
  ('set_signature', '设置签名', '设置个人签名获得一次性积分', 2, 1, FALSE, TRUE, 90, NOW(), NOW()),
  ('set_name', '设置名称', '设置名称获得一次性积分', 2, 1, FALSE, TRUE, 100, NOW(), NOW())
ON CONFLICT (task_type) DO NOTHING;

INSERT INTO system_settings (setting_key, setting_value, setting_group, created_at, updated_at)
SELECT 'points_daily_cap', '50', 'points', NOW(), NOW()
WHERE NOT EXISTS (
  SELECT 1 FROM system_settings WHERE setting_key = 'points_daily_cap'
);

COMMIT;
