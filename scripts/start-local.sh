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

DB_NAME="${DB_NAME:-ai_music}"
DB_USER="${DB_USER:-ai_music}"
DB_PASSWORD="${DB_PASSWORD:-ai_music_local_pw}"
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
echo "[start-local] applying backend/migrations/seed.sql (ON CONFLICT-safe)..."
compose exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" \
  < backend/migrations/seed.sql
echo "[start-local] seed applied"

echo "[start-local] stack status:"
compose ps
echo "[start-local] Done. API: http://localhost:$API_PORT  S3: http://localhost:$S3_PORT"
echo "[start-local] Verify: curl http://localhost:$API_PORT/healthz && ./scripts/test-e2e.sh"
