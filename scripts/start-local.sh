#!/usr/bin/env bash
# Start the local stack (PostgreSQL 16 + Redis 7 + LocalStack S3 + API + worker).
#
# Hardened Phase 5 version:
#   - `compose up` with health-gated dependencies (see docker-compose.local.yml)
#   - active health waits (postgres/redis/localstack/api/worker) with timeouts
#   - idempotent migrate + seed (up.sql once, seed.sql always safe via ON CONFLICT)
#
# Usage:
#   ./scripts/start-local.sh [--no-build] [--seed-only] [--help]
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

COMPOSE_FILE="docker-compose.local.yml"
NO_BUILD=0
SEED_ONLY=0
for arg in "$@"; do
  case "$arg" in
    --no-build) NO_BUILD=1 ;;
    --seed-only) SEED_ONLY=1 ;;
    -h|--help)
      echo "Usage: ./scripts/start-local.sh [--no-build] [--seed-only]"
      echo "  --no-build   skip --build (reuse existing api/worker images)"
      echo "  --seed-only  skip compose up, only run migrate+seed against running postgres"
      exit 0
      ;;
    *) echo "[start-local] unknown arg: $arg" >&2; exit 1 ;;
  esac
done

if [ ! -f .env ]; then
  echo "[start-local] .env not found; copying from .env.example"
  cp .env.example .env
  echo "[start-local] Edit .env secrets before running in any shared environment."
fi

set -a
# shellcheck disable=SC1091
. ./.env
set +a

DB_NAME="${DB_NAME:-avibe}"
DB_USER="${DB_USER:-avibe}"
DB_PASSWORD="${DB_PASSWORD:-avibe_local_pw}"
API_PORT="${API_PORT:-8080}"
S3_PORT="${S3_PORT:-4566}"
REDIS_PORT="${REDIS_PORT:-6379}"

if ! command -v docker >/dev/null 2>&1; then
  echo "[start-local] docker is required" >&2
  exit 1
fi
if ! docker compose version >/dev/null 2>&1; then
  echo "[start-local] 'docker compose' is required (Docker Compose v2)" >&2
  exit 1
fi

compose() { docker compose -f "$COMPOSE_FILE" "$@"; }

wait_http() {
  local url="$1"; local timeout="${2:-120}"; local start now code
  start="$(date +%s)"
  while true; do
    code="$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$url" 2>/dev/null || echo "000")"
    if [ "$code" = "200" ]; then return 0; fi
    now="$(date +%s)"
    if [ $((now - start)) -ge "$timeout" ]; then
      echo "[start-local] timeout waiting for $url (last=$code)" >&2
      return 1
    fi
    sleep 3
  done
}

wait_exec() {
  local desc="$1"; local timeout="$2"; shift 2
  local start now
  start="$(date +%s)"
  while true; do
    if "$@" >/dev/null 2>&1; then return 0; fi
    now="$(date +%s)"
    if [ $((now - start)) -ge "$timeout" ]; then
      echo "[start-local] timeout waiting for $desc" >&2
      return 1
    fi
    sleep 3
  done
}

if [ "$SEED_ONLY" -eq 0 ]; then
  echo "[start-local] Bringing up docker compose stack..."
  if [ "$NO_BUILD" -eq 1 ]; then
    compose up -d
  else
    compose up --build -d
  fi
fi

echo "[start-local] Waiting for healthy services (timeout 180s each)..."
wait_exec "postgres pg_isready" 180 \
  compose exec -T postgres pg_isready -U "$DB_USER" -d "$DB_NAME"
echo "[start-local] postgres ready"
wait_exec "redis ping" 120 \
  compose exec -T redis redis-cli ping
echo "[start-local] redis ready"
wait_http "http://localhost:$S3_PORT/_localstack/health" 240
echo "[start-local] localstack ready"
# s3-init is a one-shot bucket bootstrap (staging + public-hls + read policy); wait for exit 0.
s3_init_start="$(date +%s)"
while true; do
  init_state="$(compose ps --format json 2>/dev/null | python3 -c "
import json,sys
try:
  rows=json.load(sys.stdin)
  if isinstance(rows,dict): rows=[rows]
  for r in rows:
    name=r.get('Service') or r.get('Name') or ''
    if 's3-init' in name:
      print(r.get('State','')+'/'+r.get('ExitCode',''))
      raise SystemExit(0)
except Exception:
  pass
print('unknown')
" 2>/dev/null || echo "unknown")"
  if [[ "$init_state" == *"exited/0"* ]] || [[ "$init_state" == *"completed"* ]]; then
    echo "[start-local] s3-init buckets ready ($init_state)"
    break
  fi
  now="$(date +%s)"
  if [ $((now - s3_init_start)) -ge 180 ]; then
    echo "[start-local] WARN: s3-init state=$init_state (continuing; check 'docker compose logs s3-init')" >&2
    break
  fi
  sleep 5
done
wait_http "http://localhost:$API_PORT/healthz" 180
echo "[start-local] api ready"
wait_exec "worker process" 180 \
  compose exec -T worker pgrep -f ./worker
echo "[start-local] worker ready"

# ---- idempotent migrate + seed ----
echo "[start-local] migrate + seed (idempotent)..."
TABLE_EXISTS="$(compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A \
  -c "SELECT to_regclass('public.users') IS NOT NULL;" 2>/dev/null | tr -d ' \r\n' || echo "f")"
if [ "$TABLE_EXISTS" != "t" ]; then
  echo "[start-local] applying backend/migrations/000001_init.up.sql..."
  compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" \
    < backend/migrations/000001_init.up.sql
  echo "[start-local] init schema applied"
else
  echo "[start-local] schema already present; skipping 000001_init.up.sql"
fi
BATCH2_EXISTS="$(compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A \
  -c "SELECT to_regclass('public.password_reset_tokens') IS NOT NULL;" 2>/dev/null | tr -d ' \r\n' || echo "f")"
if [ "$BATCH2_EXISTS" != "t" ]; then
  echo "[start-local] applying backend/migrations/000002_batch2.up.sql..."
  compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" \
    < backend/migrations/000002_batch2.up.sql
  echo "[start-local] batch2 schema applied"
else
  echo "[start-local] batch2 schema already present; skipping 000002_batch2.up.sql"
fi
BATCH3_EXISTS="$(compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A \
  -c "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='ai_provider_policies' AND column_name='max_generated_at';" 2>/dev/null | tr -d ' \r\n' || echo "0")"
if [ "$BATCH3_EXISTS" != "1" ]; then
  echo "[start-local] applying backend/migrations/000003_compliance.up.sql..."
  compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" \
    < backend/migrations/000003_compliance.up.sql
  echo "[start-local] batch3 compliance schema applied"
else
  echo "[start-local] batch3 schema already present; skipping 000003_compliance.up.sql"
fi
BATCH4_EXISTS="$(compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A \
  -c "SELECT to_regclass('public.notifications') IS NOT NULL;" 2>/dev/null | tr -d ' \r\n' || echo "f")"
if [ "$BATCH4_EXISTS" != "t" ]; then
  echo "[start-local] applying backend/migrations/000004_operations.up.sql..."
  compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" \
    < backend/migrations/000004_operations.up.sql
  echo "[start-local] batch4 operations schema applied"
else
  echo "[start-local] batch4 schema already present; skipping 000004_operations.up.sql"
fi
BATCH5_EXISTS="$(compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A \
  -c "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='users' AND column_name='token_version';" 2>/dev/null | tr -d ' \r\n' || echo "0")"
if [ "$BATCH5_EXISTS" != "1" ]; then
  echo "[start-local] applying backend/migrations/000005_auth_hardening.up.sql..."
  compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" \
    < backend/migrations/000005_auth_hardening.up.sql
  echo "[start-local] batch5 auth schema applied"
else
  echo "[start-local] batch5 schema already present; skipping 000005_auth_hardening.up.sql"
fi
BATCH6_EXISTS="$(compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A \
  -c "SELECT to_regclass('public.audit_events') IS NOT NULL;" 2>/dev/null | tr -d ' \r\n' || echo "f")"
if [ "$BATCH6_EXISTS" != "t" ]; then
  echo "[start-local] applying backend/migrations/000006_audit.up.sql..."
  compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" \
    < backend/migrations/000006_audit.up.sql
  echo "[start-local] batch6 audit schema applied"
else
  echo "[start-local] batch6 schema already present; skipping 000006_audit.up.sql"
fi
BATCH7_EXISTS="$(compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A \
  -c "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='copyright_disputes' AND column_name='sla_notified_at';" 2>/dev/null | tr -d ' \r\n' || echo "0")"
if [ "$BATCH7_EXISTS" != "1" ]; then
  echo "[start-local] applying backend/migrations/000007_dispute_sla.up.sql..."
  compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" \
    < backend/migrations/000007_dispute_sla.up.sql
  echo "[start-local] batch7 dispute SLA schema applied"
else
  echo "[start-local] batch7 schema already present; skipping 000007_dispute_sla.up.sql"
fi
BATCH8_EXISTS="$(compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A \
  -c "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='track_rights_declarations' AND column_name='cert_snapshot_json';" 2>/dev/null | tr -d ' \r\n' || echo "0")"
if [ "$BATCH8_EXISTS" != "1" ]; then
  echo "[start-local] applying backend/migrations/000008_cert_snapshot.up.sql..."
  compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" \
    < backend/migrations/000008_cert_snapshot.up.sql
  echo "[start-local] batch8 cert snapshot schema applied"
else
  echo "[start-local] batch8 schema already present; skipping 000008_cert_snapshot.up.sql"
fi
BATCH9_EXISTS="$(compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A \
  -c "SELECT to_regclass('public.generation_jobs') IS NOT NULL;" 2>/dev/null | tr -d ' \r\n' || echo "f")"
if [ "$BATCH9_EXISTS" != "t" ]; then
  echo "[start-local] applying backend/migrations/000009_generation_jobs.up.sql..."
  compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" \
    < backend/migrations/000009_generation_jobs.up.sql
  echo "[start-local] batch9 generation jobs schema applied"
else
  echo "[start-local] batch9 schema already present; skipping 000009_generation_jobs.up.sql"
fi
BATCH10_EXISTS="$(compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A \
  -c "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='tracks' AND column_name='isrc';" 2>/dev/null | tr -d ' \r\n' || echo "0")"
if [ "$BATCH10_EXISTS" != "1" ]; then
  echo "[start-local] applying backend/migrations/000010_distribution_meta.up.sql..."
  compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" \
    < backend/migrations/000010_distribution_meta.up.sql
  echo "[start-local] batch10 distribution meta schema applied"
else
  echo "[start-local] batch10 schema already present; skipping 000010_distribution_meta.up.sql"
fi
BATCH11_EXISTS="$(compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A \
  -c "SELECT to_regclass('public.store_businesses') IS NOT NULL;" 2>/dev/null | tr -d ' \r\n' || echo "f")"
if [ "$BATCH11_EXISTS" != "t" ]; then
  echo "[start-local] applying backend/migrations/000011_store_player.up.sql..."
  compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" \
    < backend/migrations/000011_store_player.up.sql
  echo "[start-local] batch11 store player schema applied"
else
  echo "[start-local] batch11 schema already present; skipping 000011_store_player.up.sql"
fi
BATCH12_EXISTS="$(compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A \
  -c "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='generation_jobs' AND column_name='license_ref';" 2>/dev/null | tr -d ' \r\n' || echo "0")"
if [ "$BATCH12_EXISTS" != "1" ]; then
  echo "[start-local] applying backend/migrations/000012_generation_guards.up.sql..."
  compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" \
    < backend/migrations/000012_generation_guards.up.sql
  echo "[start-local] batch12 generation guards schema applied"
else
  echo "[start-local] batch12 schema already present; skipping 000012_generation_guards.up.sql"
fi
BATCH13_EXISTS="$(compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A \
  -c "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='partner_settlements' AND column_name='store_pool_krw';" 2>/dev/null | tr -d ' \r\n' || echo "0")"
if [ "$BATCH13_EXISTS" != "1" ]; then
  echo "[start-local] applying backend/migrations/000013_settlement_7030.up.sql..."
  compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" \
    < backend/migrations/000013_settlement_7030.up.sql
  echo "[start-local] batch13 settlement 70:30 schema applied"
else
  echo "[start-local] batch13 schema already present; skipping 000013_settlement_7030.up.sql"
fi
BATCH14_EXISTS="$(compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A \
  -c "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='partner_settlements' AND column_name='hold_reason';" 2>/dev/null | tr -d ' \r\n' || echo "0")"
if [ "$BATCH14_EXISTS" != "1" ]; then
  echo "[start-local] applying backend/migrations/000014_settlement_hold.up.sql..."
  compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" \
    < backend/migrations/000014_settlement_hold.up.sql
  echo "[start-local] batch14 settlement hold schema applied"
else
  echo "[start-local] batch14 schema already present; skipping 000014_settlement_hold.up.sql"
fi
BATCH15_EXISTS="$(compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A \
  -c "SELECT to_regclass('public.track_certificates') IS NOT NULL;" 2>/dev/null | tr -d ' \r\n' || echo "f")"
if [ "$BATCH15_EXISTS" != "t" ]; then
  echo "[start-local] applying backend/migrations/000015_track_certificates.up.sql..."
  compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" \
    < backend/migrations/000015_track_certificates.up.sql
  echo "[start-local] batch15 track certificates schema applied"
else
  echo "[start-local] batch15 schema already present; skipping 000015_track_certificates.up.sql"
fi
BATCH16_EXISTS="$(compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A \
  -c "SELECT to_regclass('public.download_logs') IS NOT NULL;" 2>/dev/null | tr -d ' \r\n' || echo "f")"
if [ "$BATCH16_EXISTS" != "t" ]; then
  echo "[start-local] applying backend/migrations/000016_download_logs.up.sql..."
  compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" \
    < backend/migrations/000016_download_logs.up.sql
  echo "[start-local] batch16 download logs schema applied"
else
  echo "[start-local] batch16 schema already present; skipping 000016_download_logs.up.sql"
fi
BATCH17_EXISTS="$(compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A \
  -c "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='tracks' AND column_name='ai_credit_parts';" 2>/dev/null | tr -d ' \r\n' || echo "0")"
if [ "$BATCH17_EXISTS" != "1" ]; then
  echo "[start-local] applying backend/migrations/000017_ownership_gate.up.sql..."
  compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" \
    < backend/migrations/000017_ownership_gate.up.sql
  echo "[start-local] batch17 ownership gate schema applied"
else
  echo "[start-local] batch17 schema already present; skipping 000017_ownership_gate.up.sql"
fi
BATCH18_EXISTS="$(compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A \
  -c "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='billing_subscriptions' AND column_name='cancel_requested_at';" 2>/dev/null | tr -d ' \r\n' || echo "0")"
if [ "$BATCH18_EXISTS" != "1" ]; then
  echo "[start-local] applying backend/migrations/000018_refund_policy.up.sql..."
  compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" \
    < backend/migrations/000018_refund_policy.up.sql
  echo "[start-local] batch18 refund policy schema applied"
else
  echo "[start-local] batch18 schema already present; skipping 000018_refund_policy.up.sql"
fi
BATCH19_EXISTS="$(compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -A \
  -c "SELECT COUNT(*) FROM pg_indexes WHERE schemaname='public' AND indexname='idx_streaming_logs_user_track_played';" 2>/dev/null | tr -d ' \r\n' || echo "0")"
if [ "$BATCH19_EXISTS" != "1" ]; then
  echo "[start-local] applying backend/migrations/000019_streaming_logs_indexes.up.sql..."
  compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" \
    < backend/migrations/000019_streaming_logs_indexes.up.sql
  echo "[start-local] batch19 streaming log indexes applied"
else
  echo "[start-local] batch19 schema already present; skipping 000019_streaming_logs_indexes.up.sql"
fi
echo "[start-local] applying backend/migrations/seed.sql (ON CONFLICT-safe)..."
compose exec -T -e PGOPTIONS="-c app.allow_seed=true" postgres psql -U "$DB_USER" -d "$DB_NAME" \
  < backend/migrations/seed.sql
echo "[start-local] seed applied"

echo "[start-local] stack status:"
compose ps
echo "[start-local] Done. API: http://localhost:$API_PORT  S3: http://localhost:$S3_PORT"
echo "[start-local] Verify: curl http://localhost:$API_PORT/healthz && ./scripts/test-e2e.sh"
