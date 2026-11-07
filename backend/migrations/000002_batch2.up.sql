-- Batch 2 schema: password recovery, social login links, billing/subscriptions.
-- Conventions: UUID PKs via gen_random_uuid(), TIMESTAMPTZ everywhere, CHECK-based enums.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- 1. password_reset_tokens (single-use, hashed at rest)
CREATE TABLE IF NOT EXISTS password_reset_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT password_reset_tokens_user_fk FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT password_reset_tokens_hash_unique UNIQUE (token_hash)
);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user ON password_reset_tokens (user_id);

-- 2. oauth_accounts (social login links)
CREATE TABLE IF NOT EXISTS oauth_accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  provider TEXT NOT NULL
    CHECK (provider IN ('GOOGLE', 'KAKAO')),
  provider_user_id TEXT NOT NULL,
  email TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT oauth_accounts_user_fk FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT oauth_accounts_provider_uid_unique UNIQUE (provider, provider_user_id),
  CONSTRAINT oauth_accounts_user_provider_unique UNIQUE (user_id, provider)
);
DROP TRIGGER IF EXISTS trg_oauth_accounts_updated_at ON oauth_accounts;
CREATE TRIGGER trg_oauth_accounts_updated_at
  BEFORE UPDATE ON oauth_accounts FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 3. billing_plans (plan catalog)
CREATE TABLE IF NOT EXISTS billing_plans (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code TEXT NOT NULL,
  name_ko TEXT NOT NULL,
  amount_krw INT NOT NULL CHECK (amount_krw > 0),
  interval TEXT NOT NULL DEFAULT 'MONTHLY'
    CHECK (interval IN ('MONTHLY')),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT billing_plans_code_unique UNIQUE (code)
);
DROP TRIGGER IF EXISTS trg_billing_plans_updated_at ON billing_plans;
CREATE TRIGGER trg_billing_plans_updated_at
  BEFORE UPDATE ON billing_plans FOR EACH ROW EXECUTE FUNCTION set_updated_at();

INSERT INTO billing_plans (code, name_ko, amount_krw, interval, is_active)
VALUES
  ('LISTENER_PLUS_MONTHLY', '리스너 플러스 (월간)', 5900, 'MONTHLY', TRUE),
  ('STUDIO_PRO_MONTHLY', '스튜디오 프로 (월간)', 12900, 'MONTHLY', TRUE)
ON CONFLICT (code) DO UPDATE SET
  name_ko = EXCLUDED.name_ko,
  amount_krw = EXCLUDED.amount_krw,
  is_active = EXCLUDED.is_active;

-- 4. billing_orders (merchant orders for Toss approval)
CREATE TABLE IF NOT EXISTS billing_orders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  plan_id UUID NOT NULL,
  order_id TEXT NOT NULL,
  amount_krw INT NOT NULL CHECK (amount_krw > 0),
  status TEXT NOT NULL DEFAULT 'PENDING'
    CHECK (status IN ('PENDING', 'PAID', 'FAILED', 'CANCELLED', 'REFUNDED')),
  toss_payment_key TEXT,
  approved_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT billing_orders_user_fk FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT billing_orders_plan_fk FOREIGN KEY (plan_id) REFERENCES billing_plans (id),
  CONSTRAINT billing_orders_order_unique UNIQUE (order_id)
);
CREATE INDEX IF NOT EXISTS idx_billing_orders_user ON billing_orders (user_id);
DROP TRIGGER IF EXISTS trg_billing_orders_updated_at ON billing_orders;
CREATE TRIGGER trg_billing_orders_updated_at
  BEFORE UPDATE ON billing_orders FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 5. billing_subscriptions (one ACTIVE row per user)
CREATE TABLE IF NOT EXISTS billing_subscriptions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  plan_id UUID NOT NULL,
  order_id UUID,
  status TEXT NOT NULL DEFAULT 'ACTIVE'
    CHECK (status IN ('ACTIVE', 'CANCELLED', 'EXPIRED')),
  current_period_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  current_period_end TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT billing_subscriptions_user_fk FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT billing_subscriptions_plan_fk FOREIGN KEY (plan_id) REFERENCES billing_plans (id),
  CONSTRAINT billing_subscriptions_order_fk FOREIGN KEY (order_id) REFERENCES billing_orders (id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_billing_subscriptions_active_user
  ON billing_subscriptions (user_id) WHERE status = 'ACTIVE';
DROP TRIGGER IF EXISTS trg_billing_subscriptions_updated_at ON billing_subscriptions;
CREATE TRIGGER trg_billing_subscriptions_updated_at
  BEFORE UPDATE ON billing_subscriptions FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 6. billing_webhook_events (idempotent webhook journal)
CREATE TABLE IF NOT EXISTS billing_webhook_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  provider TEXT NOT NULL,
  event_id TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  processed BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT billing_webhook_events_provider_event_unique UNIQUE (provider, event_id)
);
