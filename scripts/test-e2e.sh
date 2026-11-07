#!/usr/bin/env bash
# Phase 5 end-to-end verification for the AI music platform.
#
# Covers (grounded in backend/internal/handler/router.go + domain/errors.go):
#   T01  GET /healthz
#   T02  auth bootstrap (register/login, partner APPROVED, subscription VERIFIED)
#   T03  pre-check 403 on Suno Free      -> ERR_NON_COMMERCIAL_TIER
#   T04  pre-check 403 out-of-window date -> ERR_RETROACTIVE_GENERATION_DATE (+expired -> ERR_NON_COMMERCIAL_TIER)
#   T05  pre-check 403 SYNC-only license  -> ERR_SYNC_ONLY_LICENSE (SOUNDRAW/MUBERT seed FALSE)
#   T06  duplicate SHA-256 block          -> 409 ERR_DUPLICATE_AUDIO_FILE
#        + finalize sha-mismatch bypass    -> 403 ERR_FORBIDDEN
#   T07  worker FFprobe block on Free-Trial-tagged fixture
#        (admin adds "free-trial" to SUNO blocked_metadata_keywords, fixture tag triggers
#         transcoding_jobs FAILED + tracks LICENSE_GATE_REJECTED; see worker/main.go + audio/processor.go)
#   T08  happy path: HLS + APPROVED + playlist + stream + heartbeat
#   T09  dispute lifecycle mapping (no TAKEDOWN_TEMP enum in schema; mapped to real states):
#          TAKEDOWN_TEMP = tracks TAKEN_DOWN via POST /admin/curation/takedown (temporary, pending dispute)
#          skip        = GET /tracks/:id/stream + heartbeat + playlist-add -> 403 ERR_NOT_STREAMABLE,
#                        GET /tracks list excludes the id
#          counter     = POST /disputes/:id/counter -> UNDER_REVIEW + "[COUNTER]" in description
#          restore     = POST /admin/disputes/:id/resolve uphold=false -> DISMISSED, then
#                        POST /admin/review approve=true -> READY (stream works again)
#          permanent   = resolve uphold=true -> UPHELD_TAKEDOWN + TAKEN_DOWN + strike_applied=true
#   T10  strike_count 3 -> partner SUSPENDED (user_blocks strike_number 3, expires_at NULL)
#   T11  settlement FORFEITED_VIOLATION + 3.3% check
#        + AES-256-GCM account at rest + masked admin preview
#        (gross=plays*rate, withholding=gross*33/1000, net=gross-withholding;
#         violated -> FORFEITED + FORFEITED_VIOLATION + net 0; approve from FORFEITED fails)
#   T12  batch2 APIs: password reset (request/confirm/login-rotation/single-use),
#        oauth providers/start guards, billing plans/orders guards, admin overview
#
# Tooling: curl (API + presigned PUT + HLS GET), psql (state assertions), redis-cli (queue depth).
# Does NOT require docker itself; the stack must already be up (see scripts/start-local.sh).
#
# Usage:
#   ./scripts/test-e2e.sh
#   BASE_URL=http://localhost:8080 ./scripts/test-e2e.sh
#
# Env overrides: BASE_URL, DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD,
#   REDIS_HOST, REDIS_PORT, S3_PORT, S3_PUBLIC_ENDPOINT, E2E_KEEP_FIXTURES=1
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [ -f "$ROOT/.env" ]; then
  set -a
  # shellcheck disable=SC1091
  . "$ROOT/.env"
  set +a
fi

BASE_URL="${BASE_URL:-http://localhost:${API_PORT:-8080}}"
API="$BASE_URL/api/v1"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-ai_music}"
DB_USER="${DB_USER:-ai_music}"
DB_PASSWORD="${DB_PASSWORD:-}"
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"
# Public S3 base reachable from this host (published LocalStack port). The API
# signs staging URLs with the in-cluster endpoint, which host curl cannot
# resolve; upload_presigned retargets those URLs here.
S3_PUBLIC_ENDPOINT="${S3_PUBLIC_ENDPOINT:-http://localhost:${S3_PORT:-4566}}"

TMP_DIR="$(mktemp -d -t aimusic-e2e.XXXXXX)"
BODY="$TMP_DIR/body.json"
touch "$BODY"
trap 'rm -rf "$TMP_DIR"' EXIT

PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0

c_green=""; c_red=""; c_yellow=""; c_reset=""
if [ -t 1 ]; then
  c_green="$(printf '\033[32m')"; c_red="$(printf '\033[31m')"
  c_yellow="$(printf '\033[33m')"; c_reset="$(printf '\033[0m')"
fi

section() { printf '\n%s=== %s ===%s\n' "$c_yellow" "$1" "$c_reset"; }
info()  { printf '[INFO] %s\n' "$1"; }
pass()  { PASS_COUNT=$((PASS_COUNT+1)); printf '%sPASS%s %s\n' "$c_green" "$c_reset" "$1"; }
fail()  { FAIL_COUNT=$((FAIL_COUNT+1)); printf '%sFAIL%s %s\n' "$c_red" "$c_reset" "$1"; }
skip()  { SKIP_COUNT=$((SKIP_COUNT+1)); printf '%sSKIP%s %s\n' "$c_yellow" "$c_reset" "$1"; }

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    fail "missing required command: $1"
    echo "Install $1 and re-run." >&2
    exit 1
  fi
}
need_cmd curl
need_cmd psql
need_cmd redis-cli
need_cmd python3

# --- HTTP helper: api METHOD PATH TOKEN DATA -> prints status code, body in $BODY ---
api() {
  local method="$1"; local path="$2"; local token="${3:-}"; local data="${4:-}"
  local -a args=(-s -o "$BODY" -w "%{http_code}" -X "$method" "$API$path"
    -H "Content-Type: application/json" --max-time 20)
  if [ -n "$token" ]; then args+=(-H "Authorization: Bearer $token"); fi
  if [ -n "$data" ]; then args+=(-d "$data"); fi
  local code=""
  code="$(curl "${args[@]}" 2>"$TMP_DIR/curl.err" || echo "000")"
  touch "$BODY"
  printf '%s' "$code"
}

# --- JSON field helper: jget "a.b.c" reads $BODY, prints scalar or compact JSON ---
jget() {
  local path="$1"
  BODY_PATH="$BODY" FIELD_PATH="$path" python3 - <<'PY' 2>/dev/null || echo ""
import json, os
body = os.environ.get("BODY_PATH", "")
path = os.environ.get("FIELD_PATH", "")
try:
  with open(body) as f:
    d = json.load(f)
except Exception:
  print("")
  raise SystemExit(0)
cur = d
for part in path.split("."):
  if part == "":
    continue
  if isinstance(cur, dict) and part in cur:
    cur = cur[part]
  else:
    print("")
    raise SystemExit(0)
if cur is None:
  print("")
elif isinstance(cur, (dict, list)):
  print(json.dumps(cur, separators=(",", ":")))
else:
  print(cur)
PY
}

body_head() { head -c 600 "$BODY" 2>/dev/null | tr '\n' ' ' || true; }

assert_status() {
  local name="$1"; local want="$2"; local got="$3"
  if [ "$want" = "$got" ]; then pass "$name (status $got)";
  else fail "$name (want $want, got $got; body: $(body_head))"; fi
}

assert_contains() {
  local name="$1"; local needle="$2"; local haystack="$3"
  if [[ "$haystack" == *"$needle"* ]]; then pass "$name (contains $needle)";
  else fail "$name (missing '$needle' in: $(printf '%s' "$haystack" | head -c 600))"; fi
}

assert_eq() {
  local name="$1"; local want="$2"; local got="$3"
  if [ "$want" = "$got" ]; then pass "$name ($got)";
  else fail "$name (want '$want', got '$got')"; fi
}

psql_q() {
  local sql="$1"
  PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
    -t -A -c "$sql" 2>"$TMP_DIR/psql.err" || echo "__PSQL_ERROR__"
}

redis_q() {
  redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" "$@" 2>"$TMP_DIR/redis.err" || echo "__REDIS_ERROR__"
}

# Wait until psql query output equals want (timeout secs). Returns 0 on match.
wait_psql_eq() {
  local sql="$1"; local want="$2"; local timeout="${3:-120}"
  local start now out
  start="$(date +%s)"
  while true; do
    out="$(psql_q "$sql" | tr -d ' \r\n')"
    if [ "$out" = "$want" ]; then return 0; fi
    now="$(date +%s)"
    if [ $((now - start)) -ge "$timeout" ]; then return 1; fi
    sleep 3
  done
}

rand_hex() { python3 -c "import secrets; print(secrets.token_hex($1))"; }
now_rfc3339() { python3 -c "import datetime; print(datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00','Z'))"; }

# ============ T01 health ============
section "T01 healthz + infra sanity"
code="$(curl -s -o "$BODY" -w "%{http_code}" --max-time 10 "$BASE_URL/healthz" 2>/dev/null || echo "000")"
assert_status "GET /healthz" "200" "$code"
assert_contains "healthz body ok" '"ok"' "$(cat "$BODY" 2>/dev/null || echo "")"
pg_ok="$(psql_q "SELECT 1;" | tr -d ' \r\n')"
assert_eq "psql SELECT 1" "1" "$pg_ok"
redis_pong="$(redis_q ping 2>/dev/null | tr -d '\r\n')"
assert_eq "redis-cli ping" "PONG" "$redis_pong"

# ============ T02 auth bootstrap ============
section "T02 auth bootstrap (admin promote, partners, listener)"
STAMP="$(date +%s)-$(rand_hex 3)"
ADMIN_EMAIL="e2e-admin-$STAMP@example.com"
PARTNER_FREE_EMAIL="e2e-pfree-$STAMP@example.com"
PARTNER_A_EMAIL="e2e-pa-$STAMP@example.com"
PARTNER_B_EMAIL="e2e-pb-$STAMP@example.com"
USER_EMAIL="e2e-user-$STAMP@example.com"
PW="E2ePass123!"

code="$(api POST "/auth/register" "" "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$PW\",\"displayName\":\"e2e-admin\"}")"
assert_status "register admin-candidate" "201" "$code"
ADMIN_USER_ID="$(jget "data.id")"
if [ -z "$ADMIN_USER_ID" ]; then fail "admin user id empty"; else pass "admin user id $ADMIN_USER_ID"; fi
psql_q "UPDATE users SET role='ADMIN' WHERE id='$ADMIN_USER_ID'::uuid;" >/dev/null
role_after="$(psql_q "SELECT role FROM users WHERE id='$ADMIN_USER_ID'::uuid;" | tr -d ' \r\n')"
assert_eq "promote to ADMIN via psql" "ADMIN" "$role_after"

code="$(api POST "/auth/login" "" "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$PW\"}")"
assert_status "login admin" "200" "$code"
ADMIN_TOKEN="$(jget "data.tokens.accessToken")"
if [ -z "$ADMIN_TOKEN" ]; then fail "admin token empty"; else pass "admin token issued"; fi

register_partner() {
  local email="$1"; local company="$2"
  api POST "/auth/register/partner" "" "{\"email\":\"$email\",\"password\":\"$PW\",\"displayName\":\"$company\",\"company\":\"$company\"}"
}
code="$(register_partner "$PARTNER_FREE_EMAIL" "E2EFreeMusic")"
assert_status "register partnerFree" "201" "$code"
PFREE_USER_ID="$(jget "data.user.id")"
PFREE_PARTNER_ID="$(jget "data.partner.id")"
code="$(register_partner "$PARTNER_A_EMAIL" "E2EA-Music")"
assert_status "register partnerA" "201" "$code"
PA_USER_ID="$(jget "data.user.id")"
PA_PARTNER_ID="$(jget "data.partner.id")"
code="$(register_partner "$PARTNER_B_EMAIL" "E2EB-Music")"
assert_status "register partnerB" "201" "$code"
PB_USER_ID="$(jget "data.user.id")"
PB_PARTNER_ID="$(jget "data.partner.id")"
code="$(api POST "/auth/register" "" "{\"email\":\"$USER_EMAIL\",\"password\":\"$PW\",\"displayName\":\"e2e-listener\"}")"
assert_status "register listener" "201" "$code"
USER_ID="$(jget "data.id")"

code="$(api POST "/auth/login" "" "{\"email\":\"$PARTNER_A_EMAIL\",\"password\":\"$PW\"}")"
assert_status "login partnerA" "200" "$code"
PA_TOKEN="$(jget "data.tokens.accessToken")"
code="$(api POST "/auth/login" "" "{\"email\":\"$PARTNER_B_EMAIL\",\"password\":\"$PW\"}")"
assert_status "login partnerB" "200" "$code"
PB_TOKEN="$(jget "data.tokens.accessToken")"
code="$(api POST "/auth/login" "" "{\"email\":\"$PARTNER_FREE_EMAIL\",\"password\":\"$PW\"}")"
assert_status "login partnerFree" "200" "$code"
PFREE_TOKEN="$(jget "data.tokens.accessToken")"
code="$(api POST "/auth/login" "" "{\"email\":\"$USER_EMAIL\",\"password\":\"$PW\"}")"
assert_status "login listener" "200" "$code"
USER_TOKEN="$(jget "data.tokens.accessToken")"

for pid in "$PFREE_PARTNER_ID" "$PA_PARTNER_ID" "$PB_PARTNER_ID"; do
  code="$(api POST "/admin/review/partner" "$ADMIN_TOKEN" "{\"partnerId\":\"$pid\",\"approve\":true}")"
  assert_status "admin approve partner $pid" "200" "$code"
  assert_contains "partner APPROVED $pid" "APPROVED" "$(cat "$BODY")"
done
pa_status="$(psql_q "SELECT status FROM partner_profiles WHERE id='$PA_PARTNER_ID'::uuid;" | tr -d ' \r\n')"
assert_eq "partnerA APPROVED via psql" "APPROVED" "$pa_status"

# ---- subscriptions: Free stays PENDING, A/B Pro become VERIFIED ----
section "T02b subscriptions (Free PENDING, Pro VERIFIED)"
code="$(api POST "/partner/subscriptions" "$PFREE_TOKEN" '{"provider":"SUNO","planName":"Free"}')"
assert_status "partnerFree create SUNO Free" "201" "$code"
PFREE_SUB_ID="$(jget "data.id")"
code="$(api POST "/partner/subscriptions" "$PA_TOKEN" '{"provider":"SUNO","planName":"Pro"}')"
assert_status "partnerA create SUNO Pro" "201" "$code"
PA_SUB_ID="$(jget "data.id")"
code="$(api POST "/partner/subscriptions" "$PB_TOKEN" '{"provider":"SUNO","planName":"Pro"}')"
assert_status "partnerB create SUNO Pro" "201" "$code"
PB_SUB_ID="$(jget "data.id")"
EXPIRES_FUTURE="$(python3 -c "import datetime; print((datetime.datetime.now(datetime.timezone.utc)+datetime.timedelta(days=30)).isoformat().replace('+00:00','Z'))")"
code="$(api POST "/admin/review/subscription" "$ADMIN_TOKEN" "{\"subscriptionId\":\"$PA_SUB_ID\",\"approve\":true,\"expiresAt\":\"$EXPIRES_FUTURE\"}")"
assert_status "admin verify partnerA SUNO Pro" "200" "$code"
assert_contains "partnerA sub VERIFIED" "VERIFIED" "$(cat "$BODY")"
code="$(api POST "/admin/review/subscription" "$ADMIN_TOKEN" "{\"subscriptionId\":\"$PB_SUB_ID\",\"approve\":true,\"expiresAt\":\"$EXPIRES_FUTURE\"}")"
assert_status "admin verify partnerB SUNO Pro" "200" "$code"
pfree_status="$(psql_q "SELECT status FROM partner_ai_subscriptions WHERE id='$PFREE_SUB_ID'::uuid;" | tr -d ' \r\n')"
assert_eq "partnerFree SUNO Free stays PENDING" "PENDING" "$pfree_status"

GEN_NOW="$(now_rfc3339)"
GEN_OK="$(python3 -c "import datetime; print((datetime.datetime.now(datetime.timezone.utc)+datetime.timedelta(minutes=1)).isoformat().replace('+00:00','Z'))")"

section "T03 pre-check 403 on Suno Free (ERR_NON_COMMERCIAL_TIER)"
SHA_FREE="$(rand_hex 32)"
code="$(api POST "/partner/pre-check" "$PFREE_TOKEN" "{\"provider\":\"SUNO\",\"audioSha256\":\"$SHA_FREE\",\"fileName\":\"free-demo.mp3\",\"generatedAt\":\"$GEN_OK\",\"subscriptionId\":\"$PFREE_SUB_ID\",\"requiresActiveSub\":true}")"
assert_status "pre-check Suno Free -> 403" "403" "$code"
assert_contains "Suno Free gate code" "ERR_NON_COMMERCIAL_TIER" "$(cat "$BODY")"

section "T04 pre-check 403 on out-of-window generated date"
GEN_RETRO="2020-01-01T00:00:00Z"
SHA_RETRO="$(rand_hex 32)"
code="$(api POST "/partner/pre-check" "$PA_TOKEN" "{\"provider\":\"SUNO\",\"audioSha256\":\"$SHA_RETRO\",\"fileName\":\"retro.mp3\",\"generatedAt\":\"$GEN_RETRO\",\"subscriptionId\":\"$PA_SUB_ID\",\"requiresActiveSub\":true}")"
assert_status "pre-check retroactive date -> 403" "403" "$code"
assert_contains "retroactive gate code" "ERR_RETROACTIVE_GENERATION_DATE" "$(cat "$BODY")"
GEN_AFTER_EXPIRY="$(python3 -c "import datetime; print((datetime.datetime.now(datetime.timezone.utc)+datetime.timedelta(days=31)).isoformat().replace('+00:00','Z'))")"
SHA_AFTER="$(rand_hex 32)"
code="$(api POST "/partner/pre-check" "$PA_TOKEN" "{\"provider\":\"SUNO\",\"audioSha256\":\"$SHA_AFTER\",\"fileName\":\"after-expiry.mp3\",\"generatedAt\":\"$GEN_AFTER_EXPIRY\",\"subscriptionId\":\"$PA_SUB_ID\",\"requiresActiveSub\":true}")"
assert_status "pre-check after expiry -> 403" "403" "$code"
assert_contains "after-expiry gate code" "ERR_NON_COMMERCIAL_TIER" "$(cat "$BODY")"

section "T05 pre-check 403 on SYNC-only plan (ERR_SYNC_ONLY_LICENSE)"
SHA_SYNC="$(rand_hex 32)"
code="$(api POST "/partner/pre-check" "$PA_TOKEN" "{\"provider\":\"SOUNDRAW\",\"audioSha256\":\"$SHA_SYNC\",\"fileName\":\"sync-only.mp3\",\"generatedAt\":\"$GEN_OK\",\"requiresActiveSub\":false}")"
assert_status "pre-check SOUNDRAW SYNC-only -> 403" "403" "$code"
assert_contains "sync-only gate code" "ERR_SYNC_ONLY_LICENSE" "$(cat "$BODY")"
SHA_AIVA_STD="$(rand_hex 32)"
code="$(api POST "/partner/pre-check" "$PA_TOKEN" "{\"provider\":\"AIVA\",\"planTierUsed\":\"Standard\",\"audioSha256\":\"$SHA_AIVA_STD\",\"fileName\":\"aiva-std.mp3\",\"generatedAt\":\"$GEN_OK\",\"requiresActiveSub\":false}")"
assert_status "pre-check AIVA Standard SYNC-only -> 403" "403" "$code"
assert_contains "AIVA Standard sync-only gate code" "ERR_SYNC_ONLY_LICENSE" "$(cat "$BODY")"
policy_allows="$(psql_q "SELECT allows_standalone_streaming FROM ai_provider_policies WHERE provider='SOUNDRAW';" | tr -d ' \r\n')"
assert_eq "SOUNDRAW allows_standalone_streaming=false via psql" "f" "$policy_allows"

# ---- fixture helpers (ffmpeg sine fixtures; WAV bypasses bitrate gate, MP3 enforces it) ----
FIX_DIR="$TMP_DIR/fixtures"
mkdir -p "$FIX_DIR"
HAVE_FFMPEG=1
command -v ffmpeg >/dev/null 2>&1 || HAVE_FFMPEG=0
command -v ffprobe >/dev/null 2>&1 || HAVE_FFMPEG=0

make_fixture() {
  local out="$1"; local meta_args="${2:-}"; local secs="${3:-5}"
  if [ "$HAVE_FFMPEG" -eq 0 ]; then return 1; fi
  # Re-parse honoring embedded quotes: metadata values contain spaces.
  # (Callers are hardcoded below; no untrusted input reaches this eval.)
  local -a extra=()
  if [ -n "$meta_args" ]; then
    # shellcheck disable=SC2206
    eval "extra=($meta_args)"
  fi
  ffmpeg -hide_banner -loglevel error -y -f lavfi -i "sine=frequency=440:duration=$secs" \
    -ar 44100 -ac 2 "${extra[@]}" "$out"
}

upload_presigned() {
  local url="$1"; local file="$2"
  # Retarget cluster-local signed hosts to the host-reachable S3 endpoint.
  # SigV4 signs the host header, so a retargeted URL must drop the query
  # signature (LocalStack accepts the unsigned PUT; the signature path itself
  # is covered by backend storage unit tests). URLs whose host is already
  # reachable keep their signature untouched.
  url="$(SIGNED_URL="$url" PUBLIC_EP="$S3_PUBLIC_ENDPOINT" python3 -c "
import os
from urllib.parse import urlsplit, urlunsplit
raw = os.environ.get('SIGNED_URL', '')
try:
  u = urlsplit(raw)
  p = urlsplit(os.environ.get('PUBLIC_EP', ''))
  if u.scheme and p.scheme and u.netloc != p.netloc:
    print(urlunsplit((p.scheme, p.netloc, u.path, '', '')))
  else:
    print(raw)
except Exception:
  print(raw)
" 2>/dev/null || printf '%s' "$url")"
  curl -s -o /dev/null -w "%{http_code}" --max-time 60 -X PUT --data-binary "@$file" "$url" 2>/dev/null || echo "000"
}

# precheck_and_upload: args token provider sha filename generatedAt subId requiresActiveSub fixture
# Prints pre-check status to stdout; writes stagingKey/uploadUrl to
# $TMP_DIR/precheck.key and $TMP_DIR/precheck.url (safe under command substitution).
precheck_and_upload() {
  local token="$1" provider="$2" sha="$3" fname="$4" gen="$5" sub="$6" reqsub="$7" fixture="$8"
  local payload
  payload="$(python3 -c "import json; print(json.dumps({'provider':'$provider','audioSha256':'$sha','fileName':'$fname','generatedAt':'$gen','subscriptionId':'$sub','requiresActiveSub': True if '$reqsub'=='true' else False}))")"
  local code key url
  code="$(api POST "/partner/pre-check" "$token" "$payload")"
  key="$(jget "data.stagingKey")"
  url="$(jget "data.uploadUrl")"
  printf '%s' "$key" >"$TMP_DIR/precheck.key"
  printf '%s' "$url" >"$TMP_DIR/precheck.url"
  if [ "$code" = "200" ] && [ -n "$url" ] && [ -f "$fixture" ]; then
    local up
    up="$(upload_presigned "$url" "$fixture")"
    if [ "$up" != "200" ] && [ "$up" != "201" ] && [ "$up" != "204" ]; then
      echo "UPLOAD:$up" >&2
    fi
  fi
  printf '%s' "$code"
}

finalize_track() {
  local token="$1" key="$2" title="$3" provider="$4" sub="$5" sha="$6" generated="$7"
  local payload
  payload="$(python3 -c "import json; print(json.dumps({'stagingKey':'$key','title':'$title','provider':'$provider','audioSha256':'$sha','generatedAt':'$generated','subscriptionId':'$sub','disclosure':'AI-generated with $provider (commercial tier)','rights':{'declaresOwnership':True,'declaresNoVoiceClone':True,'declaresTrainingConsent':True,'subscriptionId':'$sub'}}))")"
  api POST "/partner/finalize" "$token" "$payload"
}

section "T06 duplicate SHA-256 block (409 ERR_DUPLICATE_AUDIO_FILE)"
if [ "$HAVE_FFMPEG" -eq 0 ]; then
  skip "T06 needs ffmpeg/ffprobe for upload fixtures"
else
  DUP_SHA="$(rand_hex 32)"
  DUP_FIX="$FIX_DIR/dup.wav"
  make_fixture "$DUP_FIX" "" 3
  code="$(precheck_and_upload "$PA_TOKEN" "SUNO" "$DUP_SHA" "dup.wav" "$GEN_OK" "$PA_SUB_ID" "true" "$DUP_FIX")"
  assert_status "dup first pre-check pass" "200" "$code"
  DUP_KEY="$(cat "$TMP_DIR/precheck.key" 2>/dev/null || echo "")"
  if [ -z "$DUP_KEY" ]; then fail "dup stagingKey empty";
  else
    code="$(finalize_track "$PA_TOKEN" "$DUP_KEY" "E2E Dup Track" "SUNO" "$PA_SUB_ID" "$DUP_SHA" "$GEN_OK")"
    assert_status "dup finalize -> 201" "201" "$code"
    DUP_TRACK_ID="$(jget "data.trackId")"
    DUP_JOB_ID="$(jget "data.jobId")"
    if [ -z "$DUP_TRACK_ID" ]; then fail "dup trackId empty"; else pass "dup track $DUP_TRACK_ID job $DUP_JOB_ID"; fi
  fi
  code="$(api POST "/partner/pre-check" "$PA_TOKEN" "{\"provider\":\"SUNO\",\"audioSha256\":\"$DUP_SHA\",\"fileName\":\"dup-again.wav\",\"generatedAt\":\"$GEN_OK\",\"subscriptionId\":\"$PA_SUB_ID\",\"requiresActiveSub\":true}")"
  assert_status "dup second pre-check -> 409" "409" "$code"
  assert_contains "dup gate code" "ERR_DUPLICATE_AUDIO_FILE" "$(cat "$BODY")"
  dup_count="$(psql_q "SELECT COUNT(*) FROM tracks WHERE original_object_key LIKE '%$DUP_SHA%';" | tr -d ' \r\n')"
  assert_eq "dup sha present once via psql" "1" "$dup_count"
  if [ -n "${DUP_KEY:-}" ]; then
    WRONG_SHA="ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
    code="$(finalize_track "$PA_TOKEN" "$DUP_KEY" "E2E Bypass Attempt" "SUNO" "$PA_SUB_ID" "$WRONG_SHA" "$GEN_OK")"
    assert_status "finalize sha mismatch -> 403" "403" "$code"
    assert_contains "finalize bypass code" "ERR_FORBIDDEN" "$(cat "$BODY")"
  fi
fi

section "T07 worker FFprobe block on Free-Trial-tagged fixture"
if [ "$HAVE_FFMPEG" -eq 0 ]; then
  skip "T07 needs ffmpeg/ffprobe"
else
  code="$(api GET "/admin/policies" "$ADMIN_TOKEN" "")"
  assert_status "admin list policies" "200" "$code"
  ORIG_BLOCKED="$(psql_q "SELECT COALESCE(array_to_string(blocked_metadata_keywords, '|'),'') FROM ai_provider_policies WHERE provider='SUNO';" | tr -d '\r\n')"
  info "SUNO blocked before: $ORIG_BLOCKED"
  # Add free-trial (and keep existing) via real admin API; min 192, allows true.
  NEW_BLOCKED_JSON="$(python3 -c "
import json
orig='''$ORIG_BLOCKED'''.split('|') if '''$ORIG_BLOCKED''' else []
seen=[]
for k in orig+['free-trial','Free Trial']:
  k=k.strip()
  if k and k not in seen: seen.append(k)
print(json.dumps(seen))
")"
  UPDATE_PAYLOAD="$(python3 -c "
import json
print(json.dumps({'allowsStandaloneStreaming':True,'minBitrateKbps':192,'blockedMetadataKeywords':json.loads('''$NEW_BLOCKED_JSON'''),'rejectionMessageKo':'Suno 무료 플랜 음원은 단독 스트리밍이 불가합니다.'}))
")"
  code="$(api PUT "/admin/policies/SUNO" "$ADMIN_TOKEN" "$UPDATE_PAYLOAD")"
  assert_status "admin add free-trial keyword" "200" "$code"
  assert_contains "policy has free-trial" "free-trial" "$(cat "$BODY")"
  BLOCK_SHA="$(rand_hex 32)"
  BLOCK_FIX="$FIX_DIR/blocked.mp3"
  # MP3 with comment tag containing free-trial triggers CheckBlockedMetadata.
  make_fixture "$BLOCK_FIX" '-metadata comment="free-trial demo render" -codec:a libmp3lame -b:a 128k' 4
  code="$(precheck_and_upload "$PA_TOKEN" "SUNO" "$BLOCK_SHA" "blocked.mp3" "$GEN_OK" "$PA_SUB_ID" "true" "$BLOCK_FIX")"
  assert_status "blocked pre-check passes (gate has no metadata yet)" "200" "$code"
  BLOCK_KEY="$(cat "$TMP_DIR/precheck.key" 2>/dev/null || echo "")"
  code="$(finalize_track "$PA_TOKEN" "$BLOCK_KEY" "E2E Blocked FreeTrial" "SUNO" "$PA_SUB_ID" "$BLOCK_SHA" "$GEN_OK")"
  assert_status "blocked finalize -> 201" "201" "$code"
  BLOCK_TRACK_ID="$(jget "data.trackId")"
  BLOCK_JOB_ID="$(jget "data.jobId")"
  if wait_psql_eq "SELECT status FROM tracks WHERE id='$BLOCK_TRACK_ID'::uuid;" "LICENSE_GATE_REJECTED" 150; then
    pass "worker rejected Free-Trial fixture (LICENSE_GATE_REJECTED)"
  else
    got="$(psql_q "SELECT status FROM tracks WHERE id='$BLOCK_TRACK_ID'::uuid;" | tr -d ' \r\n')"
    fail "worker reject timeout (track status=$got)"
  fi
  job_status="$(psql_q "SELECT status FROM transcoding_jobs WHERE id='$BLOCK_JOB_ID'::uuid;" | tr -d ' \r\n')"
  assert_eq "blocked job FAILED" "FAILED" "$job_status"
  rej_reason="$(psql_q "SELECT COALESCE(rejection_reason_ko,'') FROM tracks WHERE id='$BLOCK_TRACK_ID'::uuid;")"
  assert_contains "rejection reason Korean" "차단된 메타데이터" "$rej_reason"
  # Restore original policy so happy path is unaffected.
  RESTORE_JSON="$(python3 -c "
import json
orig='''$ORIG_BLOCKED'''.split('|') if '''$ORIG_BLOCKED''' else []
print(json.dumps([k for k in orig if k.strip()]))
")"
  RESTORE_PAYLOAD="$(python3 -c "
import json
print(json.dumps({'allowsStandaloneStreaming':True,'minBitrateKbps':192,'blockedMetadataKeywords':json.loads('''$RESTORE_JSON'''),'rejectionMessageKo':'Suno 무료 플랜 음원은 단독 스트리밍이 불가합니다.'}))
")"
  code="$(api PUT "/admin/policies/SUNO" "$ADMIN_TOKEN" "$RESTORE_PAYLOAD")"
  assert_status "restore SUNO policy" "200" "$code"
fi

section "T08 happy path: HLS + APPROVED + playlist + stream"
if [ "$HAVE_FFMPEG" -eq 0 ]; then
  skip "T08 needs ffmpeg/ffprobe"
else
  HAPPY_SHA="$(rand_hex 32)"
  HAPPY_FIX="$FIX_DIR/happy.wav"
  make_fixture "$HAPPY_FIX" '-metadata title="E2E Happy" -metadata artist="E2E"' 6
  code="$(precheck_and_upload "$PA_TOKEN" "SUNO" "$HAPPY_SHA" "happy.wav" "$GEN_OK" "$PA_SUB_ID" "true" "$HAPPY_FIX")"
  assert_status "happy pre-check pass" "200" "$code"
  HAPPY_KEY="$(cat "$TMP_DIR/precheck.key" 2>/dev/null || echo "")"
  code="$(finalize_track "$PA_TOKEN" "$HAPPY_KEY" "E2E Happy Track" "SUNO" "$PA_SUB_ID" "$HAPPY_SHA" "$GEN_OK")"
  assert_status "happy finalize -> 201" "201" "$code"
  HAPPY_TRACK_ID="$(jget "data.trackId")"
  HAPPY_JOB_ID="$(jget "data.jobId")"
  if [ -z "$HAPPY_TRACK_ID" ]; then fail "happy trackId empty"; else pass "happy track $HAPPY_TRACK_ID job $HAPPY_JOB_ID"; fi
  if wait_psql_eq "SELECT status FROM tracks WHERE id='$HAPPY_TRACK_ID'::uuid;" "LICENSE_GATE_PENDING" 240; then
    pass "happy track LICENSE_GATE_PENDING (worker HLS done, awaiting Gate 4 admin review)"
  else
    got="$(psql_q "SELECT status FROM tracks WHERE id='$HAPPY_TRACK_ID'::uuid;" | tr -d ' \r\n')"
    fail "happy LICENSE_GATE_PENDING timeout (status=$got)"
  fi
  job_ok="$(psql_q "SELECT status FROM transcoding_jobs WHERE id='$HAPPY_JOB_ID'::uuid;" | tr -d ' \r\n')"
  assert_eq "happy job SUCCEEDED" "SUCCEEDED" "$job_ok"
  hls_prefix="$(psql_q "SELECT COALESCE(hls_prefix,'') FROM tracks WHERE id='$HAPPY_TRACK_ID'::uuid;" | tr -d '\r\n')"
  if [ -z "$hls_prefix" ]; then fail "happy hls_prefix empty"; else pass "happy hls_prefix $hls_prefix"; fi
  qdepth="$(redis_q LLEN transcode:queue 2>/dev/null | tr -d '\r\n')"
  assert_eq "redis transcode:queue drained" "0" "$qdepth"
  # Gate 4 pre-approval check: user cannot stream before admin approval.
  code="$(api GET "/tracks/$HAPPY_TRACK_ID/stream" "$USER_TOKEN" "")"
  assert_status "stream before Gate 4 approval -> 403" "403" "$code"
  # APPROVED: admin review approve transitions LICENSE_GATE_PENDING -> READY.
  code="$(api POST "/admin/review" "$ADMIN_TOKEN" "{\"trackId\":\"$HAPPY_TRACK_ID\",\"approve\":true}")"
  assert_status "admin review approve happy" "200" "$code"
  assert_contains "happy now READY" "READY" "$(cat "$BODY")"
  code="$(api GET "/tracks?limit=50" "" "")"
  assert_status "public list tracks" "200" "$code"
  assert_contains "happy in public list" "$HAPPY_TRACK_ID" "$(cat "$BODY")"
  assert_contains "list envelope carries total" '"total"' "$(cat "$BODY")"
  # Server-side search + sort: q matches the happy-path title, unknown sort 400s.
  code="$(api GET "/tracks?q=e2e&sort=recent&limit=10" "" "")"
  assert_status "track search q+sort" "200" "$code"
  assert_contains "search finds happy track" "$HAPPY_TRACK_ID" "$(cat "$BODY")"
  code="$(api GET "/tracks?sort=bogus" "" "")"
  assert_status "unknown sort rejected" "400" "$code"
  # Playlist: create + add + list.
  code="$(api POST "/playlists" "$USER_TOKEN" '{"title":"E2E Mix","isPublic":false}')"
  assert_status "playlist create" "201" "$code"
  PLAYLIST_ID="$(jget "data.id")"
  code="$(api POST "/playlists/$PLAYLIST_ID/tracks" "$USER_TOKEN" "{\"trackId\":\"$HAPPY_TRACK_ID\"}")"
  assert_status "playlist add happy track" "201" "$code"
  code="$(api GET "/playlists/$PLAYLIST_ID/tracks" "$USER_TOKEN" "")"
  assert_status "playlist list tracks" "200" "$code"
  assert_contains "playlist contains happy" "$HAPPY_TRACK_ID" "$(cat "$BODY")"
  # Stream: signed HLS grant + master playlist fetch.
  code="$(api GET "/tracks/$HAPPY_TRACK_ID/stream" "$USER_TOKEN" "")"
  assert_status "stream grant" "200" "$code"
  STREAM_URL="$(jget "data.url")"
  STREAM_JTI="$(jget "data.jti")"
  if [[ "$STREAM_URL" == *"master.m3u8"* && "$STREAM_URL" == *"sig="* && "$STREAM_URL" == *"exp="* ]]; then
    pass "stream URL signed (master.m3u8 + exp/jti/sig)"
  else
    fail "stream URL not signed: $(printf '%s' "$STREAM_URL" | head -c 300)"
  fi
  if [ -n "$STREAM_URL" ]; then
    hls_code="$(curl -s -o "$TMP_DIR/master.m3u8" -w "%{http_code}" --max-time 20 "$STREAM_URL" 2>/dev/null || echo "000")"
    if [ "$hls_code" = "200" ] && grep -q "#EXTM3U" "$TMP_DIR/master.m3u8" 2>/dev/null; then
      pass "HLS master.m3u8 fetchable (#EXTM3U)"
    else
      fail "HLS master fetch failed (http=$hls_code)"
    fi
  fi
  # Heartbeat: 30s+ counts, short play does not; resume state persists.
  code="$(api POST "/playback/heartbeat" "$USER_TOKEN" "{\"trackId\":\"$HAPPY_TRACK_ID\",\"positionMs\":31000,\"playMs\":35000,\"jti\":\"$STREAM_JTI\"}")"
  assert_status "heartbeat 35s" "200" "$code"
  assert_contains "heartbeat counted" '"countedAsPlay":true' "$(cat "$BODY")"
  code="$(api POST "/playback/heartbeat" "$USER_TOKEN" "{\"trackId\":\"$HAPPY_TRACK_ID\",\"positionMs\":5000,\"playMs\":5000}")"
  assert_status "heartbeat 5s (short)" "200" "$code"
  assert_contains "short play not counted" '"countedAsPlay":false' "$(cat "$BODY")"
  code="$(api GET "/playback/state/$HAPPY_TRACK_ID" "$USER_TOKEN" "")"
  assert_status "playback get state" "200" "$code"
  plays_logged="$(psql_q "SELECT COUNT(*) FROM streaming_logs WHERE track_id='$HAPPY_TRACK_ID'::uuid AND counted_as_play=TRUE;" | tr -d ' \r\n')"
  if [ "$plays_logged" -ge 1 ] 2>/dev/null; then pass "streaming_logs counted>=1 ($plays_logged)"; else fail "streaming_logs counted=$plays_logged"; fi
fi

# Fallback seeder when ffmpeg is unavailable: insert READY rows directly.
seed_ready_track() {
  local partner_id="$1"; local title="$2"
  local tid
  tid="$(psql_q "INSERT INTO tracks (partner_id,title,original_object_key,ai_provider,ai_generated,status,play_count) VALUES ('$partner_id'::uuid,'$title','staging/$partner_id/e2e-fallback/dummy.wav','SUNO',TRUE,'READY',0) RETURNING id::text;" | tr -d ' \r\n')"
  psql_q "UPDATE tracks SET hls_prefix='hls/$tid' WHERE id='$tid'::uuid;" >/dev/null
  printf '%s' "$tid"
}

section "T09 dispute TAKEDOWN_TEMP + skip + counter + restore/permanent"
# Resolve track ids for both ffmpeg and fallback paths.
if [ "$HAVE_FFMPEG" -eq 1 ] && [ -n "${HAPPY_TRACK_ID:-}" ]; then
  DISPUTE_TRACK_A="$HAPPY_TRACK_ID"
else
  DISPUTE_TRACK_A="$(seed_ready_track "$PA_PARTNER_ID" "E2E Dispute A")"
  HAPPY_TRACK_ID="$DISPUTE_TRACK_A"
  PLAYLIST_ID="$(psql_q "INSERT INTO playlists (owner_user_id,title,is_public) VALUES ('$USER_ID'::uuid,'E2E Mix',FALSE) RETURNING id::text;" | tr -d ' \r\n')"
  psql_q "INSERT INTO playlist_tracks (playlist_id,track_id,position) VALUES ('$PLAYLIST_ID'::uuid,'$DISPUTE_TRACK_A'::uuid,0) ON CONFLICT DO NOTHING;" >/dev/null
  pass "fallback seeded dispute track A $DISPUTE_TRACK_A"
fi
if [ "$HAVE_FFMPEG" -eq 1 ]; then
  PB_SHA="$(rand_hex 32)"
  PB_FIX="$FIX_DIR/pb.wav"
  make_fixture "$PB_FIX" "" 4
  code="$(precheck_and_upload "$PB_TOKEN" "SUNO" "$PB_SHA" "pb.wav" "$GEN_OK" "$PB_SUB_ID" "true" "$PB_FIX")"
  assert_status "partnerB pre-check pass" "200" "$code"
  PB_KEY="$(cat "$TMP_DIR/precheck.key" 2>/dev/null || echo "")"
  code="$(finalize_track "$PB_TOKEN" "$PB_KEY" "E2E PartnerB Track" "SUNO" "$PB_SUB_ID" "$PB_SHA" "$GEN_OK")"
  assert_status "partnerB finalize" "201" "$code"
  DISPUTE_TRACK_B="$(jget "data.trackId")"
  if wait_psql_eq "SELECT status FROM tracks WHERE id='$DISPUTE_TRACK_B'::uuid;" "LICENSE_GATE_PENDING" 240; then
    pass "partnerB track LICENSE_GATE_PENDING $DISPUTE_TRACK_B"
  else fail "partnerB LICENSE_GATE_PENDING timeout"; fi
  code="$(api POST "/admin/review" "$ADMIN_TOKEN" "{\"trackId\":\"$DISPUTE_TRACK_B\",\"approve\":true}")"
  assert_status "admin approve partnerB track" "200" "$code"
else
  DISPUTE_TRACK_B="$(seed_ready_track "$PB_PARTNER_ID" "E2E Dispute B")"
  pass "fallback seeded dispute track B $DISPUTE_TRACK_B"
fi

# --- A: report -> TAKEDOWN_TEMP -> skip -> counter -> restore (dismiss) ---
code="$(api POST "/disputes/report" "$USER_TOKEN" "{\"trackId\":\"$DISPUTE_TRACK_A\",\"reason\":\"AI_POLICY_VIOLATION\",\"description\":\"e2e report A\"}")"
assert_status "report dispute A" "201" "$code"
DISPUTE_A_ID="$(jget "data.id")"
assert_contains "dispute A OPEN" "OPEN" "$(cat "$BODY")"
code="$(api POST "/admin/curation/takedown" "$ADMIN_TOKEN" "{\"trackId\":\"$DISPUTE_TRACK_A\",\"reason\":\"e2e temp takedown\"}")"
assert_status "admin takedown TEMP (A)" "200" "$code"
assert_contains "track A TAKEN_DOWN" "TAKEN_DOWN" "$(cat "$BODY")"
# skip: stream + heartbeat + playlist-add blocked, public list excludes.
code="$(api GET "/tracks/$DISPUTE_TRACK_A/stream" "$USER_TOKEN" "")"
assert_status "taken-down stream -> 403" "403" "$code"
assert_contains "stream blocked code" "ERR_NOT_STREAMABLE" "$(cat "$BODY")"
code="$(api POST "/playback/heartbeat" "$USER_TOKEN" "{\"trackId\":\"$DISPUTE_TRACK_A\",\"positionMs\":31000,\"playMs\":35000}")"
assert_status "taken-down heartbeat -> 403" "403" "$code"
code="$(api POST "/playlists/$PLAYLIST_ID/tracks" "$USER_TOKEN" "{\"trackId\":\"$DISPUTE_TRACK_A\"}")"
if [ "$code" = "403" ] || [ "$code" = "400" ]; then pass "taken-down playlist-add blocked ($code)"; else fail "playlist-add taken-down want 403/400 got $code"; fi
code="$(api GET "/tracks?limit=100" "" "")"
if [[ "$(cat "$BODY")" == *"$DISPUTE_TRACK_A"* ]]; then fail "taken-down still in public list"; else pass "taken-down excluded from public list"; fi
# counter by owner.
code="$(api POST "/disputes/$DISPUTE_A_ID/counter" "$PA_TOKEN" '{"note":"e2e counter: commercial Pro license attached"}')"
assert_status "partner counter A" "200" "$code"
assert_contains "counter -> UNDER_REVIEW" "UNDER_REVIEW" "$(cat "$BODY")"
desc_a="$(psql_q "SELECT description FROM copyright_disputes WHERE id='$DISPUTE_A_ID'::uuid;")"
assert_contains "counter note persisted" "[COUNTER]" "$desc_a"
# restore: dismiss + re-approve.
code="$(api POST "/admin/disputes/$DISPUTE_A_ID/resolve" "$ADMIN_TOKEN" '{"uphold":false}')"
assert_status "resolve dismiss A" "200" "$code"
assert_contains "dismiss status" "DISMISSED" "$(cat "$BODY")"
code="$(api POST "/admin/review" "$ADMIN_TOKEN" "{\"trackId\":\"$DISPUTE_TRACK_A\",\"approve\":true}")"
assert_status "restore track A to READY" "200" "$code"
code="$(api GET "/tracks/$DISPUTE_TRACK_A/stream" "$USER_TOKEN" "")"
assert_status "restored stream works" "200" "$code"

# --- B: report -> permanent (uphold) ---
code="$(api POST "/disputes/report" "$USER_TOKEN" "{\"trackId\":\"$DISPUTE_TRACK_B\",\"reason\":\"VOICE_CLONE\",\"description\":\"e2e report B permanent\"}")"
assert_status "report dispute B" "201" "$code"
DISPUTE_B_ID="$(jget "data.id")"
code="$(api POST "/admin/disputes/$DISPUTE_B_ID/resolve" "$ADMIN_TOKEN" '{"uphold":true}')"
assert_status "resolve uphold B (permanent)" "200" "$code"
assert_contains "upheld status" "UPHELD_TAKEDOWN" "$(cat "$BODY")"
b_status="$(psql_q "SELECT status FROM tracks WHERE id='$DISPUTE_TRACK_B'::uuid;" | tr -d ' \r\n')"
assert_eq "track B permanent TAKEN_DOWN" "TAKEN_DOWN" "$b_status"
code="$(api GET "/tracks/$DISPUTE_TRACK_B/stream" "$USER_TOKEN" "")"
assert_status "permanent stream still 403" "403" "$code"
strike1="$(psql_q "SELECT COUNT(*) FROM user_blocks WHERE user_id='$PB_USER_ID'::uuid;" | tr -d ' \r\n')"
assert_eq "partnerB strikes after 1 uphold" "1" "$strike1"

section "T10 strike_count 3 suspension"
# PartnerB already has 1 upheld strike; add 2 more on the same track (Report allows any track state).
for i in 2 3; do
  code="$(api POST "/disputes/report" "$USER_TOKEN" "{\"trackId\":\"$DISPUTE_TRACK_B\",\"reason\":\"UNAUTHORIZED_SAMPLE\",\"description\":\"e2e strike $i\"}")"
  assert_status "report strike $i" "201" "$code"
  did="$(jget "data.id")"
  code="$(api POST "/admin/disputes/$did/resolve" "$ADMIN_TOKEN" '{"uphold":true}')"
  assert_status "resolve uphold strike $i" "200" "$code"
done
strikes="$(psql_q "SELECT COUNT(*) FROM user_blocks WHERE user_id='$PB_USER_ID'::uuid;" | tr -d ' \r\n')"
assert_eq "partnerB strike_count=3" "3" "$strikes"
max_strike="$(psql_q "SELECT MAX(strike_number) FROM user_blocks WHERE user_id='$PB_USER_ID'::uuid;" | tr -d ' \r\n')"
assert_eq "max strike_number=3" "3" "$max_strike"
exp3="$(psql_q "SELECT COALESCE((SELECT expires_at::text FROM user_blocks WHERE user_id='$PB_USER_ID'::uuid AND strike_number=3 ORDER BY created_at DESC LIMIT 1),'NULL');" | tr -d '\r\n')"
if [ "$exp3" = "NULL" ] || [ -z "$exp3" ]; then pass "strike 3 permanent (expires_at NULL)"; else fail "strike 3 expires_at should be NULL, got $exp3"; fi
pb_status="$(psql_q "SELECT status FROM partner_profiles WHERE id='$PB_PARTNER_ID'::uuid;" | tr -d ' \r\n')"
assert_eq "partnerB SUSPENDED after 3 strikes" "SUSPENDED" "$pb_status"
code="$(api GET "/disputes/blocks" "$PB_TOKEN" "")"
assert_status "partnerB my blocks" "200" "$code"
assert_contains "blocks list has strike 3" '"strikeNumber":3' "$(cat "$BODY")"

section "T11 settlement FORFEITED_VIOLATION + 3.3% check"
PERIOD_DATES="$(python3 -c "
import datetime
today=datetime.datetime.now(datetime.timezone.utc).date()
start=today.replace(day=1)
# next month first day
if start.month==12: end=start.replace(year=start.year+1,month=1,day=1)
else: end=start.replace(month=start.month+1,day=1)
print(start.isoformat()+' '+end.isoformat())
")"
PERIOD_START="$(printf '%s' "$PERIOD_DATES" | cut -d' ' -f1)"
PERIOD_END="$(printf '%s' "$PERIOD_DATES" | cut -d' ' -f2)"
info "settlement period $PERIOD_START .. $PERIOD_END"
RATE_KRW=100
# Deterministic counted plays: wipe period rows for our partners, then insert exact counts.
psql_q "DELETE FROM streaming_logs WHERE partner_id IN ('$PA_PARTNER_ID'::uuid,'$PB_PARTNER_ID'::uuid) AND played_at >= '$PERIOD_START'::date AND played_at < '$PERIOD_END'::date;" >/dev/null
psql_q "INSERT INTO streaming_logs (user_id,track_id,partner_id,played_at,play_ms,counted_as_play) SELECT '$USER_ID'::uuid,'$DISPUTE_TRACK_A'::uuid,'$PA_PARTNER_ID'::uuid,NOW(),35000,TRUE FROM generate_series(1,10);" >/dev/null
psql_q "INSERT INTO streaming_logs (user_id,track_id,partner_id,played_at,play_ms,counted_as_play) SELECT '$USER_ID'::uuid,'$DISPUTE_TRACK_B'::uuid,'$PB_PARTNER_ID'::uuid,NOW(),35000,TRUE FROM generate_series(1,10);" >/dev/null
pa_plays="$(psql_q "SELECT COUNT(*) FROM streaming_logs WHERE partner_id='$PA_PARTNER_ID'::uuid AND counted_as_play=TRUE AND played_at >= '$PERIOD_START'::date AND played_at < '$PERIOD_END'::date;" | tr -d ' \r\n')"
assert_eq "partnerA counted plays=10" "10" "$pa_plays"
# Clean partnerA: PENDING with 3.3% math (gross 1000, withholding 33, net 967).
code="$(api POST "/admin/settlements/calculate" "$ADMIN_TOKEN" "{\"partnerId\":\"$PA_PARTNER_ID\",\"periodStart\":\"$PERIOD_START\",\"periodEnd\":\"$PERIOD_END\",\"ratePerPlayKrw\":$RATE_KRW}")"
assert_status "calculate settlement A" "200" "$code"
gross_a="$(jget "data.grossAmountKrw")"
with_a="$(jget "data.withholdingKrw")"
net_a="$(jget "data.netAmountKrw")"
status_a="$(jget "data.status")"
exp_gross=$((10 * RATE_KRW))
exp_with=$((exp_gross * 33 / 1000))
exp_net=$((exp_gross - exp_with))
assert_eq "settlement A gross" "$exp_gross" "$gross_a"
assert_eq "settlement A withholding 3.3%" "$exp_with" "$with_a"
assert_eq "settlement A net" "$exp_net" "$net_a"
assert_eq "settlement A PENDING" "PENDING" "$status_a"
code="$(api POST "/admin/settlements/approve" "$ADMIN_TOKEN" "{\"partnerId\":\"$PA_PARTNER_ID\",\"periodStart\":\"$PERIOD_START\",\"periodEnd\":\"$PERIOD_END\"}")"
assert_status "approve settlement A" "200" "$code"
assert_contains "settlement A APPROVED" "APPROVED" "$(cat "$BODY")"
# Violated partnerB (SUSPENDED + upheld): FORFEITED + FORFEITED_VIOLATION + net 0.
code="$(api POST "/admin/settlements/calculate" "$ADMIN_TOKEN" "{\"partnerId\":\"$PB_PARTNER_ID\",\"periodStart\":\"$PERIOD_START\",\"periodEnd\":\"$PERIOD_END\",\"ratePerPlayKrw\":$RATE_KRW}")"
assert_status "calculate settlement B" "200" "$code"
gross_b="$(jget "data.grossAmountKrw")"
with_b="$(jget "data.withholdingKrw")"
net_b="$(jget "data.netAmountKrw")"
status_b="$(jget "data.status")"
reason_b="$(jget "data.forfeitureReason")"
assert_eq "settlement B gross" "$exp_gross" "$gross_b"
assert_eq "settlement B withholding 3.3%" "$exp_with" "$with_b"
assert_eq "settlement B net zero" "0" "$net_b"
assert_eq "settlement B FORFEITED" "FORFEITED" "$status_b"
assert_eq "settlement B reason" "FORFEITED_VIOLATION" "$reason_b"
code="$(api POST "/admin/settlements/approve" "$ADMIN_TOKEN" "{\"partnerId\":\"$PB_PARTNER_ID\",\"periodStart\":\"$PERIOD_START\",\"periodEnd\":\"$PERIOD_END\"}")"
assert_status "approve FORFEITED fails 400" "400" "$code"
code="$(api GET "/admin/settlements/$PA_PARTNER_ID" "$ADMIN_TOKEN" "")"
assert_status "list settlements A" "200" "$code"
# Settlement account AES-256-GCM: ciphertext at rest, masked preview over API.
E2E_ACCT="98765432109876"
code="$(api PUT "/partner/settlement" "$PA_TOKEN" "{\"bank\":\"E2E Bank\",\"accountNumber\":\"$E2E_ACCT\"}")"
assert_status "settlement account update -> 200" "200" "$code"
stored_acct="$(psql_q "SELECT settlement_account_encrypted FROM partner_profiles WHERE id='$PA_PARTNER_ID'::uuid;" | tr -d ' \r\n')"
if [ "$stored_acct" == "$E2E_ACCT" ] || [ -z "$stored_acct" ]; then fail "settlement account stored in plaintext"; else pass "settlement account ciphertext at rest"; fi
code="$(api GET "/admin/partners/$PA_PARTNER_ID/settlement" "$ADMIN_TOKEN" "")"
assert_status "settlement preview -> 200" "200" "$code"
assert_contains "settlement preview masked" "**********9876" "$(cat "$BODY")"

section "T12 batch2: reset, oauth, billing, overview"
# Migration proof: batch2 tables exist.
for t in password_reset_tokens oauth_accounts billing_plans billing_orders billing_subscriptions billing_webhook_events; do
  present="$(psql_q "SELECT to_regclass('public.$t') IS NOT NULL;" | tr -d ' \r\n')"
  assert_eq "batch2 table $t present" "t" "$present"
done
# Password reset: unknown email still 202s and stores nothing (no enumeration).
code="$(api POST "/auth/password-reset/request" "" '{"email":"nobody-e2e@example.com"}')"
assert_status "reset request unknown -> 202" "202" "$code"
ghost_rows="$(psql_q "SELECT COUNT(*) FROM password_reset_tokens WHERE user_id NOT IN (SELECT id FROM users);" | tr -d ' \r\n')"
assert_eq "reset request unknown stores nothing" "0" "$ghost_rows"
# Password reset: known email 202s and stores a hashed token row.
code="$(api POST "/auth/password-reset/request" "" "{\"email\":\"$USER_EMAIL\"}")"
assert_status "reset request known -> 202" "202" "$code"
reset_rows="$(psql_q "SELECT COUNT(*) FROM password_reset_tokens WHERE user_id='$USER_ID'::uuid AND used_at IS NULL;" | tr -d ' \r\n')"
if [ "$reset_rows" -ge 1 ] 2>/dev/null; then pass "reset token row stored"; else fail "reset token row missing"; fi
# Password reset: garbage token 401s, short password 400s.
code="$(api POST "/auth/password-reset/confirm" "" '{"token":"e2e-garbage-token","newPassword":"E2eNewPass1!"}')"
assert_status "reset confirm garbage -> 401" "401" "$code"
code="$(api POST "/auth/password-reset/confirm" "" '{"token":"e2e-garbage-token","newPassword":"short"}')"
assert_status "reset confirm short pw -> 400" "400" "$code"
# Password reset: full cycle with a known token (simulates the emailed link).
E2E_RAW="e2e-reset-$STAMP"
E2E_HASH="$(python3 -c "import hashlib;print(hashlib.sha256('$E2E_RAW'.encode()).hexdigest())")"
psql_q "DELETE FROM password_reset_tokens WHERE user_id='$USER_ID'::uuid;" >/dev/null
psql_q "INSERT INTO password_reset_tokens (user_id, token_hash, expires_at) VALUES ('$USER_ID'::uuid, '$E2E_HASH', NOW() + interval '30 minutes');" >/dev/null
E2E_NEW_PW="E2eRotated1!"
code="$(api POST "/auth/password-reset/confirm" "" "{\"token\":\"$E2E_RAW\",\"newPassword\":\"$E2E_NEW_PW\"}")"
assert_status "reset confirm known -> 200" "200" "$code"
assert_contains "reset flag" '"reset":true' "$(cat "$BODY")"
code="$(api POST "/auth/login" "" "{\"email\":\"$USER_EMAIL\",\"password\":\"$E2E_NEW_PW\"}")"
assert_status "login with rotated password" "200" "$code"
code="$(api POST "/auth/login" "" "{\"email\":\"$USER_EMAIL\",\"password\":\"$PW\"}")"
assert_status "old password rejected" "401" "$code"
code="$(api POST "/auth/password-reset/confirm" "" "{\"token\":\"$E2E_RAW\",\"newPassword\":\"E2eAgain12!\"}")"
assert_status "reset token single-use replay -> 401" "401" "$code"
# OAuth: provider catalog + start guards.
code="$(api GET "/auth/oauth/providers" "" "")"
assert_status "oauth providers -> 200" "200" "$code"
assert_contains "providers list google" '"provider":"GOOGLE"' "$(cat "$BODY")"
assert_contains "providers list kakao" '"provider":"KAKAO"' "$(cat "$BODY")"
code="$(api GET "/auth/oauth/apple/start" "" "")"
assert_status "oauth unknown provider -> 400" "400" "$code"
code="$(curl -s -o /dev/null -w "%{http_code}" --max-time 20 "$API/auth/oauth/google/start")"
if [ "$code" = "302" ]; then
  loc="$(curl -s -o /dev/null -w "%{redirect_url}" --max-time 20 "$API/auth/oauth/google/start")"
  case "$loc" in
    https://accounts.google.com/*) pass "oauth google start redirects to Google" ;;
    *) fail "oauth google redirect target: $loc" ;;
  esac
elif [ "$code" = "400" ]; then
  pass "oauth google start 400 without keys (fail-closed)"
else
  fail "oauth google start want 302/400 got $code"
fi
# Billing: seeded catalog always lists; orders fail closed without TOSS keys.
code="$(api GET "/billing/plans" "" "")"
assert_status "billing plans -> 200" "200" "$code"
assert_contains "plan catalog seeded" "LISTENER_PLUS_MONTHLY" "$(cat "$BODY")"
code="$(api POST "/billing/orders" "$USER_TOKEN" '{"planCode":"LISTENER_PLUS_MONTHLY"}')"
if [ "$code" = "503" ]; then
  assert_contains "orders fail closed without keys" "ERR_PAYMENT_UNAVAILABLE" "$(cat "$BODY")"
elif [ "$code" = "201" ]; then
  assert_contains "order merchant id" "AIVIBE-" "$(cat "$BODY")"
else
  fail "billing orders want 503/201 got $code"
fi
code="$(api POST "/billing/webhooks/toss" "" "")"
if [ "$code" = "400" ]; then pass "webhook empty body -> 400"; else fail "webhook empty body want 400 got $code"; fi
code="$(api GET "/billing/subscriptions/me" "$USER_TOKEN" "")"
assert_status "my subscription -> 200" "200" "$code"
# Admin overview: shape for admins, 403 for listeners.
code="$(api GET "/admin/overview" "$ADMIN_TOKEN" "")"
assert_status "admin overview -> 200" "200" "$code"
for field in '"users"' '"tracks"' '"reviews"' '"disputesOpen"' '"plays"' '"queueDepth"' '"jobsFailed24h"' '"settlementsPending"'; do
  assert_contains "overview has $field" "$field" "$(cat "$BODY")"
done
code="$(api GET "/admin/overview" "$USER_TOKEN" "")"
assert_status "overview listener -> 403" "403" "$code"

section "Summary"
qdepth_final="$(redis_q LLEN transcode:queue 2>/dev/null | tr -d '\r\n')"
info "final redis transcode:queue depth=$qdepth_final"
printf '\nE2E result: %sPASS=%d%s %sFAIL=%d%s SKIP=%d\n' "$c_green" "$PASS_COUNT" "$c_reset" "$c_red" "$FAIL_COUNT" "$c_reset" "$SKIP_COUNT"
if [ "$FAIL_COUNT" -gt 0 ]; then
  printf 'E2E FAILED\n' >&2
  exit 1
fi
printf 'E2E PASSED\n'






