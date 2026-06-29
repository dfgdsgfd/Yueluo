BEGIN;

ALTER TABLE user_wallet
    DROP COLUMN IF EXISTS moon_coin;

COMMIT;
