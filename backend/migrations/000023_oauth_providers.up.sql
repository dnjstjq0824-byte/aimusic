-- A1 social providers: widen oauth_accounts provider CHECK to 5 providers.
ALTER TABLE oauth_accounts DROP CONSTRAINT IF EXISTS oauth_accounts_provider_check;
ALTER TABLE oauth_accounts ADD CONSTRAINT oauth_accounts_provider_check
  CHECK (provider IN ('GOOGLE', 'KAKAO', 'APPLE', 'NAVER', 'LINE'));
