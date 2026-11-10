-- Batch 3 rollback: drop launch-compliance columns.
ALTER TABLE partner_settlements DROP COLUMN IF EXISTS income_type;
ALTER TABLE users DROP COLUMN IF EXISTS age_confirmed_at;
ALTER TABLE users DROP COLUMN IF EXISTS birth_year;
ALTER TABLE billing_orders DROP COLUMN IF EXISTS refund_terms_version;
ALTER TABLE billing_orders DROP COLUMN IF EXISTS refund_consent_at;
ALTER TABLE copyright_disputes DROP COLUMN IF EXISTS resumed_at;
ALTER TABLE copyright_disputes DROP COLUMN IF EXISTS resume_scheduled_at;
ALTER TABLE copyright_disputes DROP COLUMN IF EXISTS resume_requested_at;
ALTER TABLE copyright_disputes DROP COLUMN IF EXISTS partner_notified_at;
ALTER TABLE copyright_disputes DROP COLUMN IF EXISTS takedown_requested_at;
ALTER TABLE track_rights_declarations DROP COLUMN IF EXISTS collective_name;
ALTER TABLE track_rights_declarations DROP COLUMN IF EXISTS collective_trust_status;
ALTER TABLE ai_provider_policies DROP COLUMN IF EXISTS terms_checked_at;
ALTER TABLE ai_provider_policies DROP COLUMN IF EXISTS terms_snapshot_url;
ALTER TABLE ai_provider_policies DROP COLUMN IF EXISTS max_generated_at;
