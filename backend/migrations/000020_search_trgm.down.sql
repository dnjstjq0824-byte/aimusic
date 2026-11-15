DROP INDEX IF EXISTS idx_tracks_mood_tags_gin;
DROP INDEX IF EXISTS idx_users_display_name_trgm;
DROP INDEX IF EXISTS idx_tracks_title_trgm;
-- pg_trgm extension stays installed (shared, harmless).
