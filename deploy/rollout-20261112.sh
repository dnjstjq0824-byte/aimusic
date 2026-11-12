#!/usr/bin/env bash
# Rollout of branch audit-fixes-20261112 (2026-11-12 audit fixes) to production.
# Run from the repo root on the operator PC (Git Bash), with that branch checked out.
#
#   bash deploy/rollout-20261112.sh            # full rollout
#   bash deploy/rollout-20261112.sh --rollback # restore the previous backend + configs
#
# Order matters: the Worker that accepts both grant formats goes first, then the
# backend (new SQ/HQ grants), then the Worker bridge is closed.
set -euo pipefail

VPS=root@158.247.219.159
KEY="$HOME/.ssh/avibe_vultr"
SSH=(ssh -i "$KEY" -o BatchMode=yes "$VPS")
CF_ACCOUNT=f5c363135efe9b22ca66eb39f2f8e1da   # avibemusic.com account
TS=$(date -u +%Y%m%d-%H%M%S)

say() { printf '\n== %s\n' "$*"; }

if [[ "${1:-}" == "--rollback" ]]; then
  say "Rollback: restore previous backend dir, .env, tunnel config, then rebuild"
  "${SSH[@]}" 'set -e; cd /opt/avibe
    PREV=$(ls -dt backend.prev-* | head -1); test -n "$PREV"
    rm -rf backend && mv "$PREV" backend
    ENVB=$(ls -t .env.bak-rollout-* | head -1); cp -p "$ENVB" .env
    CFB=$(ls -t cloudflared/config.yml.bak-rollout-* | head -1); cp -p "$CFB" cloudflared/config.yml
    docker compose -f docker-compose.prod.yml build api
    docker compose -f docker-compose.prod.yml up -d --no-deps --force-recreate api worker
    docker restart avibe-prod-cloudflared'
  echo "Backend rolled back. Migration 000018 is additive and stays (down file: backend/migrations/000018_refund_policy.down.sql)."
  echo "Worker: npx wrangler rollback   (or redeploy main)"
  exit 0
fi

say "0. Preconditions"
git diff --quiet || { echo "working tree has uncommitted changes"; exit 1; }
[[ "$(git rev-parse --abbrev-ref HEAD)" == "audit-fixes-20261112" ]] || { echo "checkout audit-fixes-20261112 first"; exit 1; }
npx wrangler whoami 2>/dev/null | grep -q "$CF_ACCOUNT" || { echo "run: npx wrangler login  (avibemusic.com account)"; exit 1; }
"${SSH[@]}" 'docker ps --format "{{.Names}}" | grep -q avibe-prod-api'

say "1. Worker + web (bridge ON: accepts old and new grants)"
npx wrangler deploy

say "2. DB backup now + daily cron (D-01)"
"${SSH[@]}" 'set -e; mkdir -p /opt/avibe/backups && chmod 700 /opt/avibe/backups
cat > /opt/avibe/backup-db.sh <<"EOS"
#!/bin/sh
# Daily Postgres dump (D-01). Keeps 14 days in /opt/avibe/backups.
set -eu
TS=$(date -u +%Y%m%d-%H%M%S)
OUT=/opt/avibe/backups/avibe-$TS.sql.gz
docker exec avibe-prod-postgres pg_dump -U avibe -d avibe --no-owner | gzip > "$OUT.tmp"
mv "$OUT.tmp" "$OUT"
find /opt/avibe/backups -name "avibe-*.sql.gz" -mtime +14 -delete
EOS
chmod 700 /opt/avibe/backup-db.sh
/opt/avibe/backup-db.sh
( crontab -l 2>/dev/null | grep -v backup-db.sh; echo "30 18 * * * /opt/avibe/backup-db.sh >> /opt/avibe/backups/backup.log 2>&1" ) | crontab -
ls -lt /opt/avibe/backups | head -3'

say "3. Ship backend source (previous kept as backend.prev-$TS)"
git -c core.autocrlf=false archive HEAD backend | "${SSH[@]}" "set -e; cd /opt/avibe; rm -rf backend.new; mkdir backend.new; tar -x -C backend.new --strip-components=1; mv backend backend.prev-$TS; mv backend.new backend"

say "4. Env + tunnel config (backups *.bak-rollout-$TS)"
"${SSH[@]}" "set -e; cd /opt/avibe; cp -p .env .env.bak-rollout-$TS
  grep -q '^S3_PUBLIC_ENDPOINT=' .env || echo 'S3_PUBLIC_ENDPOINT=https://hls.avibemusic.com' >> .env
  sed -i 's#^OAUTH_REDIRECT_BASE_URL=https://avibemusic.com\$#OAUTH_REDIRECT_BASE_URL=https://avibemusic.com/api/v1/auth/oauth#' .env
  grep -E '^(S3_ENDPOINT|S3_PUBLIC_ENDPOINT|OAUTH_REDIRECT_BASE_URL)=' .env
  cp -p cloudflared/config.yml cloudflared/config.yml.bak-rollout-$TS"
git -c core.autocrlf=false show HEAD:deploy/cloudflared-config.yml | "${SSH[@]}" 'cat > /opt/avibe/cloudflared/config.yml'
"${SSH[@]}" 'docker run --rm -v /opt/avibe/cloudflared:/etc/cloudflared:ro cloudflare/cloudflared:latest --config /etc/cloudflared/config.yml tunnel ingress validate'

say "5. Build, migrate (000018), restart"
"${SSH[@]}" 'set -e; cd /opt/avibe
  docker compose -f docker-compose.prod.yml build api
  docker compose -f docker-compose.prod.yml run --rm migrate up
  docker compose -f docker-compose.prod.yml up -d --no-deps --force-recreate api worker
  docker restart avibe-prod-cloudflared
  sleep 10
  docker exec avibe-prod-postgres psql -U avibe -d avibe -At -c "SELECT max(version) FROM schema_migrations"'

say "6. Smoke checks"
check() { local want="$1" url="$2"; local got; got=$(curl -s -o /dev/null -m 15 -w '%{http_code}' "$url"); printf '%s (want %s)  %s\n' "$got" "$want" "$url"; [[ "$got" == "$want" ]]; }
check 200 https://api.avibemusic.com/readyz
check 200 "https://avibemusic.com/api/v1/tracks?limit=1"
check 404 "https://hls.avibemusic.com/local-staging?list-type=2"
check 404 "https://hls.avibemusic.com/_localstack/health"
check 200 https://avibemusic.com/
if curl -s "https://avibemusic.com/api/v1/tracks?limit=1" | grep -q originalObjectKey; then echo "FAIL: originalObjectKey still public"; exit 1; fi

say "7. Close the legacy-grant bridge"
npx wrangler deploy --var ALLOW_LEGACY_GRANTS:false

cat <<EOM

Rollout done ($TS).
- Play one track on https://avibemusic.com while logged in (SQ), and with a pass if available (HQ).
- Admin console now requires TOTP: enroll at /account/security first.
- Then commit wrangler.toml with ALLOW_LEGACY_GRANTS = "false".
Rollback: bash deploy/rollout-20261112.sh --rollback
EOM
