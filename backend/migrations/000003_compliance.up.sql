-- Batch 3 schema: launch-compliance columns (B2/C3/C6/C9/D1/B9).
-- All additive and idempotent; safe to run on existing volumes.
-- Conventions: TIMESTAMPTZ everywhere, app-validated enums as TEXT.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- B2/C4: per-provider generation cutoff (e.g. Udio 2025-10-30) + ToS snapshot pointer.
ALTER TABLE ai_provider_policies ADD COLUMN IF NOT EXISTS max_generated_at TIMESTAMPTZ;
ALTER TABLE ai_provider_policies ADD COLUMN IF NOT EXISTS terms_snapshot_url TEXT;
ALTER TABLE ai_provider_policies ADD COLUMN IF NOT EXISTS terms_checked_at TIMESTAMPTZ;

-- C3: KOMCA/collective-society trust declaration on rights rows.
ALTER TABLE track_rights_declarations ADD COLUMN IF NOT EXISTS collective_trust_status TEXT NOT NULL DEFAULT 'NONE';
ALTER TABLE track_rights_declarations ADD COLUMN IF NOT EXISTS collective_name TEXT;

-- D1: Copyright Act §103 interruption timeline on disputes.
ALTER TABLE copyright_disputes ADD COLUMN IF NOT EXISTS takedown_requested_at TIMESTAMPTZ;
ALTER TABLE copyright_disputes ADD COLUMN IF NOT EXISTS partner_notified_at TIMESTAMPTZ;
ALTER TABLE copyright_disputes ADD COLUMN IF NOT EXISTS resume_requested_at TIMESTAMPTZ;
ALTER TABLE copyright_disputes ADD COLUMN IF NOT EXISTS resume_scheduled_at TIMESTAMPTZ;
ALTER TABLE copyright_disputes ADD COLUMN IF NOT EXISTS resumed_at TIMESTAMPTZ;

-- C6: refund/withdrawal consent evidence on billing orders.
ALTER TABLE billing_orders ADD COLUMN IF NOT EXISTS refund_consent_at TIMESTAMPTZ;
ALTER TABLE billing_orders ADD COLUMN IF NOT EXISTS refund_terms_version TEXT;

-- C9: 14+ age gate evidence on users.
ALTER TABLE users ADD COLUMN IF NOT EXISTS birth_year INT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS age_confirmed_at TIMESTAMPTZ;

-- B9: withholding income type on settlements (BUSINESS_33 default, OTHER_88).
ALTER TABLE partner_settlements ADD COLUMN IF NOT EXISTS income_type TEXT NOT NULL DEFAULT 'BUSINESS_33';

-- B2: Udio cutoff 2025-10-30T00:00:00Z (UMG deal download halt; LAUNCH_COMPLIANCE §2.3).
UPDATE ai_provider_policies
SET max_generated_at = '2025-10-30T00:00:00Z'::timestamptz
WHERE provider = 'UDIO' AND max_generated_at IS NULL;
