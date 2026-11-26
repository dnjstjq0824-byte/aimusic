#!/usr/bin/env bash
# Daily Postgres dump + off-site copy (WI-N3).
#
# 1. pg_dump from the postgres container, gzip -> $BACKUP_DIR/avibe-<UTC>.sql.gz
# 2. when R2_DB_BACKUP_BUCKET is set, upload the dump to that (private) bucket
# 3. prune local dumps older than 14 days (R2 keeps 30 days via lifecycle rule)
#
# Runs on the VPS as /opt/avibe/backup-db.sh from root crontab
# (30 18 * * *). This repo copy is the source of truth; prod install is
# a G-2 (PO-approved) copy. Works against the local stack too:
#
#   BACKUP_DIR=$PWD/.local-backups PG_CONTAINER=avibe-postgres \
#   R2_ENDPOINT=http://localhost:14566 R2_ACCESS_KEY_ID=x R2_SECRET_ACCESS_KEY=y \
#   R2_DB_BACKUP_BUCKET=test-db-backups bash deploy/backup-db.sh
#
# Exit code: 0 on success. Non-zero when the dump fails or when the R2
# upload was configured but failed (cron mail/log catches it). When the
# R2 variables are unset the upload step is skipped with a warning.
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-/opt/avibe/backups}"
PG_CONTAINER="${PG_CONTAINER:-avibe-prod-postgres}"
PG_USER="${PG_USER:-avibe}"
PG_DB="${PG_DB:-avibe}"
KEEP_DAYS="${KEEP_DAYS:-14}"

R2_ENDPOINT="${R2_ENDPOINT:-}"
R2_ACCESS_KEY_ID="${R2_ACCESS_KEY_ID:-}"
R2_SECRET_ACCESS_KEY="${R2_SECRET_ACCESS_KEY:-}"
R2_DB_BACKUP_BUCKET="${R2_DB_BACKUP_BUCKET:-}"
AWS_CLI_IMAGE="${AWS_CLI_IMAGE:-amazon/aws-cli:2.0.9}"
# Extra `docker run` network for the aws-cli container (local verify only;
# the VPS reaches R2 over the internet, so this stays empty there).
AWS_DOCKER_NETWORK="${AWS_DOCKER_NETWORK:-}"

log() { printf '[backup-db] %s\n' "$*"; }
die() { log "ERROR: $*"; exit 1; }

mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR"

TS=$(date -u +%Y%m%d-%H%M%S)
OUT="$BACKUP_DIR/avibe-$TS.sql.gz"

log "dumping $PG_DB from $PG_CONTAINER -> $OUT"
if ! docker exec "$PG_CONTAINER" pg_dump -U "$PG_USER" -d "$PG_DB" --no-owner 2>"$OUT.dump.err" | gzip >"$OUT.tmp"; then
  rm -f "$OUT.tmp"
  cat "$OUT.dump.err" >&2 || true
  rm -f "$OUT.dump.err"
  die "pg_dump failed"
fi
rm -f "$OUT.dump.err"
mv "$OUT.tmp" "$OUT"
SIZE=$(wc -c <"$OUT" | tr -d ' ')
[ "$SIZE" -gt 0 ] || die "dump is empty: $OUT"
log "dump ok ($SIZE bytes)"

if [ -z "$R2_DB_BACKUP_BUCKET" ]; then
  log "WARN: R2_DB_BACKUP_BUCKET unset, skipping off-site upload (local-only backup)"
else
  [ -n "$R2_ENDPOINT" ] || die "R2_DB_BACKUP_BUCKET is set but R2_ENDPOINT is empty"
  [ -n "$R2_ACCESS_KEY_ID" ] || die "R2_DB_BACKUP_BUCKET is set but R2_ACCESS_KEY_ID is empty"
  [ -n "$R2_SECRET_ACCESS_KEY" ] || die "R2_DB_BACKUP_BUCKET is set but R2_SECRET_ACCESS_KEY is empty"
  KEY="avibe-$TS.sql.gz"
  log "uploading to $R2_DB_BACKUP_BUCKET/$KEY"
  WORK=$(mktemp -d "${TMPDIR:-/tmp}/avibe-dbbackup-XXXXXXXX")
  chmod 700 "$WORK"
  cleanup() { rm -rf "$WORK"; }
  trap cleanup EXIT
  # Credentials via a mounted file (mode 600), never on a command line.
  mkdir -p "$WORK/aws"
  printf '[default]\nregion = auto\noutput = text\n' >"$WORK/aws/config"
  printf '[default]\naws_access_key_id = %s\naws_secret_access_key = %s\n' \
    "$R2_ACCESS_KEY_ID" "$R2_SECRET_ACCESS_KEY" >"$WORK/aws/credentials"
  chmod 600 "$WORK/aws/credentials"
  NET_ARGS=()
  if [ -n "$AWS_DOCKER_NETWORK" ]; then NET_ARGS=(--network "$AWS_DOCKER_NETWORK"); fi
  if ! docker run --rm "${NET_ARGS[@]}" -v "$WORK/aws:/root/.aws:ro" -v "$BACKUP_DIR:/data:ro" "$AWS_CLI_IMAGE" \
    --endpoint-url "$R2_ENDPOINT" s3 cp "/data/avibe-$TS.sql.gz" "s3://$R2_DB_BACKUP_BUCKET/$KEY" --only-show-errors; then
    die "R2 upload failed for $KEY"
  fi
  REMOTE_SIZE=$(docker run --rm "${NET_ARGS[@]}" -v "$WORK/aws:/root/.aws:ro" "$AWS_CLI_IMAGE" \
    --endpoint-url "$R2_ENDPOINT" s3api head-object --bucket "$R2_DB_BACKUP_BUCKET" --key "$KEY" \
    --query ContentLength --output text 2>/dev/null | tr -d ' \r\n' || echo "")
  if [ "$REMOTE_SIZE" != "$SIZE" ]; then
    die "R2 size mismatch for $KEY (local=$SIZE remote=${REMOTE_SIZE:-unknown})"
  fi
  log "upload ok ($KEY, $REMOTE_SIZE bytes, size verified)"
  trap - EXIT
  cleanup
fi

find "$BACKUP_DIR" -name "avibe-*.sql.gz" -mtime "+$KEEP_DAYS" -delete
log "pruned local dumps older than $KEEP_DAYS days"
log "done: $OUT"
