-- 000016: P2-1 (#22) download log (quota + audit).
CREATE TABLE IF NOT EXISTS download_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  track_id UUID NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
  bytes BIGINT NOT NULL DEFAULT 0 CHECK (bytes >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_download_logs_user_created ON download_logs (user_id, created_at DESC);
