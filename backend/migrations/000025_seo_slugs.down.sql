-- S1/S3/S5 rollback: drop the SEO event/log tables and slug columns.
DROP TABLE IF EXISTS indexnow_log;
DROP TABLE IF EXISTS analytics_events;
DROP INDEX IF EXISTS users_handle_unique;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_handle_check;
ALTER TABLE users DROP COLUMN IF EXISTS handle;
ALTER TABLE playlists DROP CONSTRAINT IF EXISTS playlists_short_id_unique;
DROP INDEX IF EXISTS idx_playlists_slug;
ALTER TABLE playlists DROP COLUMN IF EXISTS slug;
ALTER TABLE playlists DROP COLUMN IF EXISTS short_id;
ALTER TABLE tracks DROP CONSTRAINT IF EXISTS tracks_short_id_unique;
DROP INDEX IF EXISTS idx_tracks_slug;
ALTER TABLE tracks DROP COLUMN IF EXISTS slug;
ALTER TABLE tracks DROP COLUMN IF EXISTS short_id;
