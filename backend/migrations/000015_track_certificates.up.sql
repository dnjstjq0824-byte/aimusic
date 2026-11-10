-- 000015: P2-6 (#24) per-track certificates (number + verify URL).
-- One certificate per track; issuance is an explicit admin act recorded
-- with the issuing admin. Verification is public by number.
CREATE TABLE IF NOT EXISTS track_certificates (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  track_id UUID NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
  cert_number TEXT NOT NULL,
  issued_by UUID REFERENCES users(id) ON DELETE SET NULL,
  issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_track_certificates_track UNIQUE (track_id),
  CONSTRAINT uq_track_certificates_number UNIQUE (cert_number)
);
CREATE INDEX IF NOT EXISTS idx_track_certificates_number ON track_certificates (cert_number);
