-- 000013: settlement 70:30 canon inputs (방향결정서 §5-1).
-- The per-play rate is abolished: each row records the admin-attested
-- revenue inputs (store pool total + this partner's direct attribution)
-- and the time-pro-rata basis (partner vs total qualified store seconds).
ALTER TABLE partner_settlements
  ADD COLUMN IF NOT EXISTS store_pool_krw BIGINT NOT NULL DEFAULT 0 CHECK (store_pool_krw >= 0),
  ADD COLUMN IF NOT EXISTS direct_krw BIGINT NOT NULL DEFAULT 0 CHECK (direct_krw >= 0),
  ADD COLUMN IF NOT EXISTS store_play_secs BIGINT NOT NULL DEFAULT 0 CHECK (store_play_secs >= 0),
  ADD COLUMN IF NOT EXISTS total_store_play_secs BIGINT NOT NULL DEFAULT 0 CHECK (total_store_play_secs >= 0);
