-- R6 library engagement: track likes, playlist saves, follows.
-- Follow targets are polymorphic (creator = partner_profiles.id, playlist =
-- playlists.id), so target_id carries no FK; resolvers inner-join live rows.
CREATE TABLE IF NOT EXISTS track_likes (
  user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  track_id UUID NOT NULL REFERENCES tracks (id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT pk_track_likes PRIMARY KEY (user_id, track_id)
);
CREATE INDEX IF NOT EXISTS idx_track_likes_track ON track_likes (track_id);
CREATE TABLE IF NOT EXISTS playlist_saves (
  user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  playlist_id UUID NOT NULL REFERENCES playlists (id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT pk_playlist_saves PRIMARY KEY (user_id, playlist_id)
);
CREATE INDEX IF NOT EXISTS idx_playlist_saves_playlist ON playlist_saves (playlist_id);
CREATE TABLE IF NOT EXISTS follows (
  user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  target_type TEXT NOT NULL CHECK (target_type IN ('creator', 'playlist')),
  target_id UUID NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT pk_follows PRIMARY KEY (user_id, target_type, target_id)
);
CREATE INDEX IF NOT EXISTS idx_follows_target ON follows (target_type, target_id);
