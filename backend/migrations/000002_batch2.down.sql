-- Batch 2 down migration: drop batch-2 tables in reverse dependency order.

DROP TABLE IF EXISTS billing_webhook_events;
DROP TABLE IF EXISTS billing_subscriptions;
DROP TABLE IF EXISTS billing_orders;
DROP TABLE IF EXISTS billing_plans;
DROP TABLE IF EXISTS oauth_accounts;
DROP TABLE IF EXISTS password_reset_tokens;
