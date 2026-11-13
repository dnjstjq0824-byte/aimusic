-- 000019: streaming_logs heartbeat lookup indexes (WI-D4 D-02).
-- Additive only. Speeds per-user fraud windows, play_ms accumulation, and
-- last-heartbeat lookups keyed by (user_id[, track_id], played_at).
CREATE INDEX IF NOT EXISTS idx_streaming_logs_user_played
  ON streaming_logs (user_id, played_at);
CREATE INDEX IF NOT EXISTS idx_streaming_logs_user_track_played
  ON streaming_logs (user_id, track_id, played_at);
