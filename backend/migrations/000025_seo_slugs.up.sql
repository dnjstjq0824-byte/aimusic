-- S1 URL/slug/handle + S3 IndexNow log + S5 analytics events.
-- tracks/playlists gain short_id (base62 8 chars, unique) and slug
-- (lowercased title, whitespace -> dash, unicode kept). users.handle is the
-- creator address (nullable, unique, lowercase alnum + underscore 3-30).
-- analytics_events stores cookieless product events (S5); indexnow_log
-- records IndexNow submissions with retry state (S3).
ALTER TABLE tracks ADD COLUMN short_id TEXT;
ALTER TABLE tracks ADD COLUMN slug TEXT;
-- Backfill short_id from an md5-derived hex prefix (a base62 subset).
-- The window suffix keeps colliding buckets unique.
WITH ranked AS (
  SELECT id, ROW_NUMBER() OVER (
    PARTITION BY substr(md5(id::text || ':avibe-track'), 1, 7)
    ORDER BY id) AS rn
  FROM tracks
)
UPDATE tracks t SET short_id =
  substr(md5(t.id::text || ':avibe-track'), 1, 7) ||
  substr('0123456789abcdef', ((ranked.rn - 1) % 16) + 1, 1)
FROM ranked WHERE ranked.id = t.id;
UPDATE tracks SET slug =
  substr(regexp_replace(regexp_replace(lower(trim(both from title)), '[/\\?#%&+=]', '-', 'g'), '\s+', '-', 'g'), 1, 80)
WHERE slug IS NULL;
UPDATE tracks SET slug = 'track' WHERE slug IS NULL OR slug = '';
ALTER TABLE tracks ALTER COLUMN short_id SET NOT NULL;
ALTER TABLE tracks ALTER COLUMN slug SET NOT NULL;
ALTER TABLE tracks ADD CONSTRAINT tracks_short_id_unique UNIQUE (short_id);
CREATE INDEX IF NOT EXISTS idx_tracks_slug ON tracks (slug);

ALTER TABLE playlists ADD COLUMN short_id TEXT;
ALTER TABLE playlists ADD COLUMN slug TEXT;
WITH ranked AS (
  SELECT id, ROW_NUMBER() OVER (
    PARTITION BY substr(md5(id::text || ':avibe-playlist'), 1, 7)
    ORDER BY id) AS rn
  FROM playlists
)
UPDATE playlists p SET short_id =
  substr(md5(p.id::text || ':avibe-playlist'), 1, 7) ||
  substr('0123456789abcdef', ((ranked.rn - 1) % 16) + 1, 1)
FROM ranked WHERE ranked.id = p.id;
UPDATE playlists SET slug =
  substr(regexp_replace(regexp_replace(lower(trim(both from title)), '[/\\?#%&+=]', '-', 'g'), '\s+', '-', 'g'), 1, 80)
WHERE slug IS NULL;
UPDATE playlists SET slug = 'playlist' WHERE slug IS NULL OR slug = '';
ALTER TABLE playlists ALTER COLUMN short_id SET NOT NULL;
ALTER TABLE playlists ALTER COLUMN slug SET NOT NULL;
ALTER TABLE playlists ADD CONSTRAINT playlists_short_id_unique UNIQUE (short_id);
CREATE INDEX IF NOT EXISTS idx_playlists_slug ON playlists (slug);

-- Creator handle: partners get a stable derived handle, everyone else NULL
-- until they claim one. Reserved words are enforced in Go (domain/seo.go).
ALTER TABLE users ADD COLUMN handle TEXT;
UPDATE users SET handle = 'creator_' || substr(md5(id::text || ':avibe-handle'), 1, 8)
WHERE role = 'PARTNER' AND handle IS NULL;
ALTER TABLE users ADD CONSTRAINT users_handle_check
  CHECK (handle IS NULL OR handle ~ '^[a-z0-9_]{3,30}$');
CREATE UNIQUE INDEX IF NOT EXISTS users_handle_unique ON users (handle);

CREATE TABLE IF NOT EXISTS analytics_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  locale TEXT,
  path TEXT,
  session_hash TEXT,
  props JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_analytics_events_name_created ON analytics_events (name, created_at);
CREATE INDEX IF NOT EXISTS idx_analytics_events_created ON analytics_events (created_at);

CREATE TABLE IF NOT EXISTS indexnow_log (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  url TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'queued',
  attempts INT NOT NULL DEFAULT 0,
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  sent_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_indexnow_log_created ON indexnow_log (created_at);
CREATE INDEX IF NOT EXISTS idx_indexnow_log_status ON indexnow_log (status);
