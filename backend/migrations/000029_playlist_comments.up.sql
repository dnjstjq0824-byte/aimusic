-- 2026-11-27 playlist rebuild + track comments.
-- 1) Playlist detail reads entries in (position, added_at) order in one
--    joined query; the composite index serves that scan.
-- 2) track_comments: listener comments on READY tracks. Authors (or admins)
--    delete their own; rows go with the track or user (cascade).
CREATE INDEX IF NOT EXISTS idx_pt_playlist_position ON playlist_tracks (playlist_id, position);

CREATE TABLE IF NOT EXISTS track_comments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  track_id UUID NOT NULL REFERENCES tracks (id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  body TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 1000),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_track_comments_track ON track_comments (track_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_track_comments_user ON track_comments (user_id);
