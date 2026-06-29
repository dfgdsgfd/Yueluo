BEGIN;

ALTER TABLE categories
    ADD COLUMN IF NOT EXISTS translations JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE categories
SET translations = jsonb_build_object(
    'en', COALESCE(NULLIF(name, ''), 'Category'),
    'zh-CN', COALESCE(NULLIF(category_title, ''), NULLIF(name, ''), '分类'),
    'zh-TW', COALESCE(NULLIF(category_title, ''), NULLIF(name, ''), '分類'),
    'vi', COALESCE(NULLIF(category_title, ''), NULLIF(name, ''), 'Category'),
    'ja', COALESCE(NULLIF(category_title, ''), NULLIF(name, ''), 'Category'),
    'ko', COALESCE(NULLIF(category_title, ''), NULLIF(name, ''), 'Category')
)
WHERE translations IS NULL OR translations = '{}'::jsonb;

COMMIT;
