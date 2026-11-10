ALTER TABLE partner_settlements
  DROP COLUMN IF EXISTS store_pool_krw,
  DROP COLUMN IF EXISTS direct_krw,
  DROP COLUMN IF EXISTS store_play_secs,
  DROP COLUMN IF EXISTS total_store_play_secs;
