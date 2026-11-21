# AVIBE Production Runbook

How to deploy, operate, and recover the AVIBE stack. Code references are
authoritative; this file only indexes them.

## 1. Topology (prod)

| Piece | Code | Notes |
|---|---|---|
| API | `backend/cmd/api` | Stateless; scale horizontally. Boot refuses prod without secrets (`Config.Validate`). |
| Worker | `backend/cmd/worker` | Transcode queue + generation loop (WI-03) + dispute-SLA sweeper + stuck-job reaper. Run ≥1; sweepers are idempotent. |
| Postgres | managed (RDS/Cloud SQL) | Migrations via `cmd/migrate` journal. |
| Redis | managed (ElastiCache/Memorystore) | Transcode queue, login throttle, token-version reads. |
| Object storage | Cloudflare R2 (`STORAGE_DRIVER=r2`; default `s3local` is LocalStack) | Private staging + HLS; Worker verifies HLS signatures. |
| Edge | Cloudflare Worker (`worker.js`) | HLS signature verify, cache, CSP/HSTS. |
| Web | `web/` SPA behind nginx | `web/nginx.conf.template` (CSP/HSTS parity with Worker). |
| Mobile | `mobile/` Flutter | Release signing via `key.properties` / CI secrets. |

Local mirror: `docker-compose.local.yml` + `./scripts/start-local.sh --build`.

## 2. Pre-deploy gates (must all pass)

```bash
cd backend && go build ./... && go vet ./... && go test ./...
cd ../web && npm run typecheck && npm run test -- --run && npm run build
cd ../mobile && flutter analyze && flutter test
bash -n scripts/test-e2e.sh && bash -n scripts/start-local.sh
./scripts/start-local.sh --build && ./scripts/test-e2e.sh   # needs Docker
```

## 3. Environment (prod `APP_ENV=prod`)

Fail-closed at boot (`backend/internal/config/config.go` `Validate`):

| Var | Required in prod | Purpose |
|---|---|---|
| `APP_ENV` | `prod` | Enables guards |
| `DB_DSN` | yes | Postgres DSN |
| `REDIS_ADDR` | yes | `host:port` |
| `JWT_ACCESS_SECRET` / `JWT_REFRESH_SECRET` | yes (≥32 chars) | Token signing |
| `HLS_SIGNING_KEY` | yes | Stream/Worker HMAC |
| `SMTP_HOST` (+`SMTP_USER`/`SMTP_PASS`) | yes | Password-reset mail; without it resets silently never send |
| `TOSS_SECRET_KEY` | for billing | Empty → billing 503s cleanly (`ERR_PAYMENT_UNAVAILABLE`) |
| `AES256_KEY_BASE64` | for settlements | 32-byte payout-account encryption |
| `R2_*` / `S3_*` | yes | Object storage (see `loadStorage`) |
| `CORS_ALLOWED_ORIGINS` | yes | SPA origin(s) |
| `TRUSTED_PROXIES` | yes if behind LB | Else client IPs spoof rate limits |
| `FRONTEND_BASE_URL` | yes | OAuth/reset links |
| `OAUTH_*` | for social | Google/Kakao; disabled providers hide buttons |
| `REGISTRATION_INVITE_CODE` | optional | Pre-launch gate (C8) |
| `GENERATION_BACKEND_URL` | for generation | GPU server base URL (WI-03, e.g. home-PC `generator/server.py`); empty → generation jobs FAIL with a clear message, never fake audio |
| `GENERATION_TIMEOUT_SECONDS` | no (600) | Per-generation backend call timeout |
| `GENERATION_DAILY_CAP` | no (5) | Per-partner daily submit quota (server-enforced) |
| `GENERATION_BANNED_KEYWORDS` | no | Extra comma-separated prompt block phrases (built-ins cover voice-mimic/impersonation) |

Worker-only knobs: `DISPUTE_SLA_INTERVAL_SECONDS` (900),
`DISPUTE_SLA_HOURS` (72), `JOB_REAP_INTERVAL_SECONDS` (300),
`JOB_STUCK_AFTER_SECONDS` (1800).

## 4. Deploy order

1. `DB_DSN=... go run ./backend/cmd/migrate up` (embedded `*.up.sql` 000001→000012; journal `schema_migrations`). Verify: `migrate status`.
2. Deploy API + worker images. Health: `GET /healthz` (liveness), `GET /readyz` (db+redis+queue depth; 503 when degraded).
3. Bootstrap admin: `DB_DSN=... ADMINCTL_PASSWORD=... adminctl create-admin --email ops@...` (audited as `ADMIN_CREATE`). Role changes: `adminctl set-role/set-active` (audited).
4. Seed legal terms / plans via SQL inserts (never `seed.sql`: it is guarded by `app.allow_seed` and is local-only).
5. Publish Cloudflare Worker (`wrangler deploy`) with `HLS_SIGNING_KEY` secret; verify an HLS URL 403s without `sig`.

## 4b. Web-only rollout (WI-N4)

When only `web/` changed, `deploy/rollout-web.sh` swaps the web container
without touching the backend (no API restart). Order: union old+new
`/app/public` → `wrangler deploy` → ship image → recreate `web` →
smoke (incl. every live `/_nuxt/*` asset 200). G-2: PO approval first.

```sh
bash deploy/rollout-web.sh --dry-run web   # print the plan, run nothing
bash deploy/rollout-web.sh web             # approved runs only
bash deploy/rollout-web.sh smoke           # re-verify any time
bash deploy/rollout-web.sh rollback        # previous image + wrangler rollback
```

Preconditions (checked, not assumed): clean tree, docker daemon,
`wrangler whoami` on the avibemusic.com account, local
`avibe-prod-web:$REV` (`REV` = last commit touching `web/`), and the
prod `avibe-prod-web` container reachable. The previous prod image ID
is recorded in `deploy/.web-prev-image` (git-ignored, per operator)
before the swap; rollback re-tags it `:latest` and recreates `web`.

## 5. Daily operations

- Admin console `/admin`: reviews → disputes (SLA badges) → curation/settlement (`calculate`→`approve`→`mark-paid`, double-pay 409s) → audit trail → users.
- Pending legal re-consents gate commerce automatically; publish new required versions via `legal_terms` inserts.
- Refunds: user self-service (`POST /billing/payments/refund`) voids at Toss in full; already-cancelled converges to `REFUNDED`.
- Withdrawals purge PII immediately; settlements/consents/logs stay pseudonymous (tax retention).
- ISRC (WI-05): no agency API is wired — ops owns a national-agency registrant block offline.
  Assign from the block via `PUT /admin/tracks/:id/isrc` (canonical 12-char, UNIQUE, audited
  `TRACK_ISRC_ASSIGN`); DSP-kit exports only ISRC-bearing READY tracks. Never invent codes:
  duplicates across distributors corrupt royalty matching.
- Generation (WI-03/WI-08): `GENERATION_BACKEND_URL` points at the GPU host; daily cap
  (`GENERATION_DAILY_CAP`) and prompt strikes (`generation_violations`) are automatic.
  Third strike in 24h suspends that partner's generation for 24h; account termination stays
  a manual admin suspension.

## 6. Incidents

| Symptom | Check | Fix |
|---|---|---|
| `/readyz` 503 db down | managed DB status | failover; API stays up, requests 500 with request IDs |
| `/readyz` 503 redis down | managed Redis | queue/throttle degrade; transcode stalls until back |
| `queueDepth` growing | worker logs, `jobsFailed24h` | restart worker; reaper re-queues stuck RUNNING jobs ≤3 attempts |
| HLS 403 everywhere | `HLS_SIGNING_KEY` mismatch API↔Worker | rotate secret both sides |
| Login 429 flood | `LOGIN_LOCKED` throttle (10/15m per email) | legit users wait; abuse → WAF rule |
| Stale OPEN disputes | admin disputes SLA badges; worker `dispute SLA sweep` logs | resolve; alerts fan out to all active admins once |
| Audit gap suspected | `audit_events` table vs admin actions | audit writes log `audit record failed` but never block |

## 7. Rollback

- API/worker: redeploy previous image (migrations are additive; `*.down.sql` exist per batch but prod rollback = forward-fix, not down-migrate).
- Bad migration: fix with a new `00000N` file; never edit a shipped migration (journal checksums by version row).
- Worker (edge): `wrangler rollback`.

## 8. Backups

- Postgres: VPS-local dumps + off-site R2 copy (WI-N3), not managed PITR.
  `/opt/avibe/backup-db.sh` (source of truth: `deploy/backup-db.sh`;
  prod install is G-2) runs daily at 03:30 KST via root crontab
  (`30 18 * * *`): `pg_dump` + gzip → `/opt/avibe/backups/avibe-<UTC>.sql.gz`
  (14-day local retention), then upload to the private bucket
  `R2_DB_BACKUP_BUCKET` (`avibe-db-backups`) with a size check. Upload
  failure exits non-zero (cron log catches it). RPO ≤24h. Every deploy
  takes one more dump first (G-2).
- R2 retention: 30-day lifecycle rule on the backup bucket (PO sets once
  in the Cloudflare dashboard: R2 → bucket → Lifecycle → delete objects
  older than 30 days). Local keeps 14 days as before.
- Restore (rehearsed 2026-11-12, WI-D2): dump → scratch DB → compare →
  drop. Prod is never written during rehearsal:
  ```sh
  LATEST=$(ls -t /opt/avibe/backups/avibe-*.sql.gz | head -1)
  docker exec -i avibe-prod-postgres psql -U avibe -d avibe -c 'CREATE DATABASE avibe_restore_test;'
  gunzip -c "$LATEST" | docker exec -i avibe-prod-postgres psql -U avibe -d avibe_restore_test -v ON_ERROR_STOP=1 -q
  # compare: table count + row counts + max(schema_migrations) vs prod, then:
  docker exec -i avibe-prod-postgres psql -U avibe -d avibe -c 'DROP DATABASE avibe_restore_test;'
  ```
  Rehearsal evidence: 36/36 tables, users/tracks/playlists/streaming_logs/
  settlements row counts identical, migrations max `000018` on both sides.
- Real restore (prod down): stop api/worker, `DROP DATABASE avibe` +
  `CREATE DATABASE avibe`, pipe the newest good dump into `avibe`,
  `migrate up`, start api/worker, check `/readyz`. Practice on the
  scratch flow above first.
- Restore from R2 to the local stack (rehearsed 2026-11-21, WI-N3):
  download from the backup bucket, restore into a scratch DB, compare,
  drop. Local `avibe` is never written:
  ```sh
  # Needs: local stack up, R2_* (or test-bucket) vars in the environment.
  KEY=avibe-<UTC>.sql.gz   # newest key in $R2_DB_BACKUP_BUCKET
  docker run --rm --network avibe-local_local -v ./aws:/root/.aws:ro \
    -v $PWD:/data amazon/aws-cli:2.0.9 --endpoint-url "$R2_ENDPOINT" \
    s3 cp "s3://$R2_DB_BACKUP_BUCKET/$KEY" /data/restore.sql.gz
  docker compose -f docker-compose.local.yml exec -T postgres psql \
    -U avibe -d avibe -c 'CREATE DATABASE avibe_restore_test;'
  gunzip -c restore.sql.gz | docker compose -f docker-compose.local.yml \
    exec -T postgres psql -U avibe -d avibe_restore_test -v ON_ERROR_STOP=1 -q
  # compare: table count + row counts (users/tracks/...) vs avibe, then:
  docker compose -f docker-compose.local.yml exec -T postgres psql \
    -U avibe -d avibe -c 'DROP DATABASE avibe_restore_test;'
  ```
  Rehearsal evidence (LocalStack test bucket): 50/50 tables,
  users 460/460, tracks 233/233 identical, scratch DB dropped.
- R2: private buckets, no versioning. Takedown purges are deliberate
  deletes. Pre-cutover LocalStack volume stays on the VPS until PO
  approves deletion (G-R2).
- Secrets live in `/opt/avibe/.env` (mode 600), not a manager. Rotate
  JWT/HLS/OAuth/Toss/R2 on personnel change — rotation revokes sessions
  via `token_version` bump per user or re-issue.

## 9. Uptime monitor (WI-D2, O-04)

Free-plan external monitor (UptimeRobot or equivalent), two HTTPS monitors:

| Monitor | URL | Healthy |
|---|---|---|
| api-readyz | `https://api.avibemusic.com/readyz` | 200 + `"status":"ok"` |
| home | `https://avibemusic.com/` | 200 |

- Interval 5 min, alert contact = PO email (PO-owned account).
- Setup (PO, ~5 min): sign up with the alert email → Add Monitor × 2
  (Monitor Type HTTPS, URLs above, interval 5 min) → alert contact =
  that email → use the monitor's test/suspend toggle once to confirm
  an alert email arrives → tell the engine session "alert tested".
- An alert means: open this runbook §6 Incidents; `/readyz` body names
  the failing check (`db`/`redis`/`queueDepth`).

## 10. Android signing (WI-D6, F-07)

- Release builds require `mobile/android/key.properties` (git-ignored):
  `storeFile`/`keyAlias`/`keyPassword`/`storePassword`. Without it the
  Gradle guard fails `assembleRelease`/`bundleRelease` instead of
  silently signing with debug keys.
- The production keystore is PO-held: generate once, back it up offline,
  never commit it. Losing it means a new app identity on the store.
- Every release build must pass `--dart-define=API_BASE_URL=https://...`.
  A release pointing at loopback crashes on launch by design
  (`Env.assertReleaseConfig`, fail-fast, unit-tested matcher).
- Gate script (also a CI job): after the release build, run
  `scripts/check-mobile-release.sh <apk> <expected-host>`. It fails when
  Dart sources hardcode the host or when the APK lacks it (forgotten
  `--dart-define`). CI builds with a throwaway keystore; only the host
  check matters there.
