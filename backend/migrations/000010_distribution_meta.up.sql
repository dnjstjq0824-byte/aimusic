-- 000010: distribution metadata (WI-05 / gap #31). ISRC is assigned by ops
-- (national agency block); BPM and musical key are worker-detected
-- (best-effort, NULL when analysis fails or audio is unsuitable).
ALTER TABLE tracks ADD COLUMN IF NOT EXISTS isrc TEXT;
ALTER TABLE tracks ADD COLUMN IF NOT EXISTS bpm REAL CHECK (bpm IS NULL OR (bpm >= 30 AND bpm <= 300));
ALTER TABLE tracks ADD COLUMN IF NOT EXISTS musical_key TEXT;
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'uq_tracks_isrc') THEN
    ALTER TABLE tracks ADD CONSTRAINT uq_tracks_isrc UNIQUE (isrc);
  END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_tracks_isrc ON tracks (isrc);
