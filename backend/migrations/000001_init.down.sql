-- Phase 1 down migration: drop all tables in reverse dependency order.

DROP TABLE IF EXISTS user_blocks;
DROP TABLE IF EXISTS partner_settlements;
DROP TABLE IF EXISTS streaming_logs;
DROP TABLE IF EXISTS user_playback_states;
DROP TABLE IF EXISTS playlist_tracks;
DROP TABLE IF EXISTS playlists;
DROP TABLE IF EXISTS copyright_disputes;
DROP TABLE IF EXISTS transcoding_jobs;
DROP TABLE IF EXISTS track_rights_declarations;
DROP TABLE IF EXISTS tracks;
DROP TABLE IF EXISTS albums;
DROP TABLE IF EXISTS user_term_consents;
DROP TABLE IF EXISTS legal_terms;
DROP TABLE IF EXISTS partner_ai_subscriptions;
DROP TABLE IF EXISTS ai_provider_policies;
DROP TABLE IF EXISTS partner_profiles;
DROP TABLE IF EXISTS users;

DROP FUNCTION IF EXISTS set_updated_at();
