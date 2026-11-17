-- A1 rollback: drop links minted by the new providers, restore the 2-provider CHECK.
DELETE FROM oauth_accounts WHERE provider NOT IN ('GOOGLE', 'KAKAO');
ALTER TABLE oauth_accounts DROP CONSTRAINT IF EXISTS oauth_accounts_provider_check;
ALTER TABLE oauth_accounts ADD CONSTRAINT oauth_accounts_provider_check
  CHECK (provider IN ('GOOGLE', 'KAKAO'));
