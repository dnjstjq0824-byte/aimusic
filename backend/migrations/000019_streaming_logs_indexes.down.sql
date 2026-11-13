-- Reverts 000019: drops the heartbeat lookup indexes only.
DROP INDEX IF EXISTS idx_streaming_logs_user_track_played;
DROP INDEX IF EXISTS idx_streaming_logs_user_played;
