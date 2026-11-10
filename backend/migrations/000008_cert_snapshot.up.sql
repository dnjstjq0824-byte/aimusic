-- 000008: track-level commercial certification snapshot (WI-02). Freezes the
-- exact gate inputs/outcomes at decision time so later policy-row or
-- subscription edits cannot rewrite a track's certification evidence.
ALTER TABLE track_rights_declarations ADD COLUMN IF NOT EXISTS cert_snapshot_json TEXT;
ALTER TABLE track_rights_declarations ADD COLUMN IF NOT EXISTS cert_snapshot_at TIMESTAMPTZ;
