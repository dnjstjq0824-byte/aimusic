-- R5 site search: trigram similarity over title + creator display name.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS idx_tracks_title_trgm ON tracks USING gin (title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_users_display_name_trgm ON users USING gin (display_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_tracks_mood_tags_gin ON tracks USING gin (mood_tags);
