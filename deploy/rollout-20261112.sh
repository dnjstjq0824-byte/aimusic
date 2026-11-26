#!/usr/bin/env bash
# Rollout of branch audit-fixes-20261112 (2026-11-12 audit fixes) to production.
# HISTORICAL: executed 2026-11-12; kept as a record. Current rollouts use
# rollout-web.sh (web-only) and rollout-redesign.sh (backend). Do not re-run
# this unless you understand the legacy-grant bridge sequence below.
# Run from the repo root on the operator PC (Git Bash), with that branch checked out.
#
#   bash deploy/rollout-20261112.sh            # full rollout
#   bash deploy/rollout-20261112.sh --rollback # restore the previous backend + configs
#   bash deploy/rollout-20261112.sh --dry-run [--rollback|--resume-at-5]  # print the plan only
#
# Order matters: the Worker that accepts both grant formats goes first, then the
# backend (new SQ/HQ grants), then the Worker bridge is closed.
#
# NEVER runs against prod without PO approval (G-2): the run asks for a typed
# YES, and --dry-run prints every remote command without executing it.
set -euo pipefail

# shellcheck source=common.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"
TS=$(date -u +%Y%m%d-%H%M%S)
DRY=0

if [ "${1:-}" = "--dry-run" ]; then DRY=1; shift; fi

RESUME5=0
[[ "${1:-}" == "--resume-at-5" ]] && RESUME5=1

if [[ "${1:-}" == "--rollback" ]]; then
  say "Rollback: restore previous backend dir, .env, tunnel config, then rebuild"
  confirm "20261112 rollback"
  ssh_run 'set -e; cd /opt/avibe
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
git diff --quiet || precheck "working tree has uncommitted changes"
[[ "$(git rev-parse --abbrev-ref HEAD)" == "audit-fixes-20261112" ]] || precheck "checkout audit-fixes-20261112 first"
npx wrangler whoami 2>/dev/null | grep -q "$CF_ACCOUNT" || precheck "run: npx wrangler login  (avibemusic.com account)"
ssh_run 'docker ps --format "{{.Names}}" | grep -q avibe-prod-api'
confirm "20261112 full rollout"

if [[ $RESUME5 == 0 ]]; then
say "1. Worker + web (bridge ON: accepts old and new grants)"
run npx wrangler deploy

say "2. DB backup now + daily cron (D-01)"
ssh_run 'set -e; mkdir -p /opt/avibe/backups && chmod 700 /opt/avibe/backups
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
( crontab -l 2>/dev/null | grep -v backup-db.sh || true; echo "30 18 * * * /opt/avibe/backup-db.sh >> /opt/avibe/backups/backup.log 2>&1" ) | crontab -
ls -lt /opt/avibe/backups | head -3'

say "3. Ship backend source (previous kept as backend.prev-$TS)"
if [ "$DRY" -eq 1 ]; then
  echo "[dry-run] git archive HEAD backend | ssh $VPS '... tar -x ... mv backend backend.prev-$TS ...'"
else
  git -c core.autocrlf=false archive HEAD backend | "${SSH[@]}" "set -e; cd /opt/avibe; rm -rf backend.new; mkdir backend.new; tar -x -C backend.new --strip-components=1; mv backend backend.prev-$TS; mv backend.new backend"
fi

say "4. Env + tunnel config (backups *.bak-rollout-$TS)"
ssh_run "set -e; cd /opt/avibe; cp -p .env .env.bak-rollout-$TS
  grep -q '^S3_PUBLIC_ENDPOINT=' .env || echo 'S3_PUBLIC_ENDPOINT=https://hls.avibemusic.com' >> .env
  sed -i 's#^OAUTH_REDIRECT_BASE_URL=https://avibemusic.com\$#OAUTH_REDIRECT_BASE_URL=https://avibemusic.com/api/v1/auth/oauth#' .env
  grep -q '^OAUTH_REDIRECT_BASE_URL=https://avibemusic.com/api/v1/auth/oauth$' .env || echo 'WARN: OAUTH redirect not in expected shape (check .env manually)'
  grep -E '^(S3_ENDPOINT|S3_PUBLIC_ENDPOINT|OAUTH_REDIRECT_BASE_URL)=' .env
  cp -p cloudflared/config.yml cloudflared/config.yml.bak-rollout-$TS"
if [ "$DRY" -eq 1 ]; then
  echo "[dry-run] ship HEAD:deploy/cloudflared-config.yml to VPS, validate ingress"
else
  git -c core.autocrlf=false show HEAD:deploy/cloudflared-config.yml | "${SSH[@]}" 'cat > /opt/avibe/cloudflared/config.yml'
fi
ssh_run 'docker run --rm -v /opt/avibe/cloudflared:/etc/cloudflared:ro cloudflare/cloudflared:2026.10.7 --config /etc/cloudflared/config.yml tunnel ingress validate'

fi

say "5. Build, migrate (000018), restart"
ssh_run 'set -e; cd /opt/avibe
  docker compose -f docker-compose.prod.yml build api
  docker compose -f docker-compose.prod.yml run --rm migrate
  docker compose -f docker-compose.prod.yml up -d --no-deps --force-recreate api worker
  docker restart avibe-prod-cloudflared
  sleep 10
  docker exec avibe-prod-postgres psql -U avibe -d avibe -At -c "SELECT max(version) FROM schema_migrations"'

say "6. Smoke checks"
check() {
  local want="$1" url="$2"; local got
  if [ "$DRY" -eq 1 ]; then printf '[dry-run] want %s  %s\n' "$want" "$url"; return 0; fi
  got=$(curl -s -o /dev/null -m 15 -w '%{http_code}' "$url"); printf '%s (want %s)  %s\n' "$got" "$want" "$url"; [[ "$got" == "$want" ]]
}
check 200 https://api.avibemusic.com/readyz
check 200 "https://avibemusic.com/api/v1/tracks?limit=1"
check 404 "https://hls.avibemusic.com/local-staging?list-type=2"
check 404 "https://hls.avibemusic.com/_localstack/health"
check 200 https://avibemusic.com/
if [ "$DRY" -eq 1 ]; then
  echo "[dry-run] assert originalObjectKey absent from public API"
else
  if curl -s "https://avibemusic.com/api/v1/tracks?limit=1" | grep -q originalObjectKey; then echo "FAIL: originalObjectKey still public"; exit 1; fi
fi

say "7. Close the legacy-grant bridge"
run npx wrangler deploy --var ALLOW_LEGACY_GRANTS:false

cat <<EOM

Rollout done ($TS).
- Play one track on https://avibemusic.com while logged in (SQ), and with a pass if available (HQ).
- Admin console now requires TOTP: enroll at /account/security first.
- Then commit wrangler.toml with ALLOW_LEGACY_GRANTS = "false".
Rollback: bash deploy/rollout-20261112.sh --rollback
EOM
