# AVIBE Launch Checklist (code gates)

Check each box with the exact command. Anything unchecked blocks launch.

## A. Automated gates (all green in this repo state)

- [ ] Backend: `cd backend && go build ./... && go vet ./... && go test ./...`
- [ ] Web: `cd web && npm run typecheck && npm run test -- --run && npm run build`
- [ ] Mobile: `cd mobile && flutter analyze && flutter test`
- [ ] Scripts: `bash -n scripts/test-e2e.sh && bash -n scripts/start-local.sh`
- [ ] Live E2E: `./scripts/start-local.sh --build && ./scripts/test-e2e.sh` → `E2E PASSED`
      (covers: auth/TOTP/age/consents, pre-check gates, dedupe, worker, HLS
      signed stream, disputes/takedown+purge, strikes, settlement+mark-paid,
      refund-surface fail-closed, audit trail, withdraw purge, export, admin users)

## B. Prod environment

- [ ] `APP_ENV=prod`, `DB_DSN`, `REDIS_ADDR` point at managed services
- [ ] `JWT_ACCESS_SECRET` / `JWT_REFRESH_SECRET` ≥32 random chars
- [ ] `HLS_SIGNING_KEY` set identically in API env + Worker secret
- [ ] `SMTP_HOST/USER/PASS` deliver (send a real reset mail, click it)
- [ ] `TOSS_SECRET_KEY` is the **live** key; run a real ₩100 order → confirm → refund
- [ ] `AES256_KEY_BASE64` is 32 bytes; settlement preview shows masked account
- [ ] `STORAGE_DRIVER=r2` + `R2_*` buckets exist (`staging-private`, `public-hls`)
- [ ] `CORS_ALLOWED_ORIGINS` = prod SPA origin(s) only
- [ ] `TRUSTED_PROXIES` = LB/CDN egress IPs
- [ ] `FRONTEND_BASE_URL` / `OAUTH_REDIRECT_BASE_URL` = prod URLs
- [ ] API boot log shows no `invalid config` fatal

## C. Data & access

- [ ] `migrate up` applied 000001→000012; `migrate status` clean
- [ ] `adminctl create-admin` done; password in vault, not chat
- [ ] Seed parity: `legal_terms` (4 codes, required v-current), `billing_plans`,
      `ai_provider_policies` match `seed.sql` intent (insert via SQL, never seed file)
- [ ] Withdrawal test on staging: PII purged, email reusable, settlements kept
- [ ] Export test: bundle downloads, contains no secret columns

## D. Edge & clients

- [ ] `wrangler deploy` live; unsigned HLS URL → 403, signed → 200
- [ ] CSP/HSTS headers present on SPA + Worker responses
- [ ] Android release signed (`key.properties` in CI, never in repo); iOS IPA built
- [ ] Store listings: 14+ age rating, UGC block/report paths work (artist block + dispute report)

## E. Ops readiness

- [ ] `/readyz` wired to load-balancer health checks; alerts on 503
- [ ] Log aggregation captures API/worker JSON logs; alert on `audit record failed`
- [ ] Postgres PITR + monthly restore drill scheduled
- [ ] Worker `DISPUTE_SLA_*` / `JOB_*` intervals reviewed (defaults 15m/72h, 5m/30m)
- [ ] On-call has `docs/PROD_RUNBOOK.md` §6 incident table

## F. Legal sign-off (see `docs/legal/LAUNCH_COMPLIANCE.md`)

- [ ] Suno/Udio standalone-streaming terms re-verified against current provider ToS
- [ ] OSP recipient designation + §103 takedown/resume path walkthrough done
- [ ] Business footer values (company, registration, address, phone) filled in prod config
- [ ] Counsel review of TOS/Privacy/Partner-contract/Copyright-policy seed texts
