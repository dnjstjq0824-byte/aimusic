-- 000006: admin audit trail. Every mutating admin action records who did
-- what to which entity, with request context. Append-only by convention
-- (no UPDATE/DELETE paths in the API).
CREATE TABLE IF NOT EXISTS audit_events (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  actor_user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  action        TEXT NOT NULL,
  entity_type   TEXT NOT NULL DEFAULT '',
  entity_id     TEXT NOT NULL DEFAULT '',
  metadata      JSONB NOT NULL DEFAULT '{}'::jsonb,
  ip            TEXT NOT NULL DEFAULT '',
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audit_events_created ON audit_events (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_entity ON audit_events (entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_action ON audit_events (action);
