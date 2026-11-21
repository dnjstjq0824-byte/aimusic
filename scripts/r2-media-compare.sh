#!/usr/bin/env bash
# WI-N10: compare the pre-cutover LocalStack media buckets against R2.
# READ-ONLY: lists keys + sizes + ETags on both sides and reports
# match / missing / mismatch. Writes nothing anywhere.
# Runs on the VPS in /opt/avibe (reads prod only) or against the local
# stack for rehearsal. Actual deletion needs PO approval (G-R2/P7).
#
# Local rehearsal:
#   SRC_ENDPOINT=http://localstack:4566 SRC_STAGING_BUCKET=src-staging \
#   SRC_HLS_BUCKET=src-hls DST_ENDPOINT=http://localstack:4566 \
#   R2_STAGING_BUCKET=dst-staging R2_PUBLIC_HLS_BUCKET=dst-hls \
#   AWS_DOCKER_NETWORK=avibe-local_local bash scripts/r2-media-compare.sh
set -euo pipefail

SRC_ENDPOINT="${SRC_ENDPOINT:-http://localstack:4566}"
SRC_STAGING_BUCKET="${SRC_STAGING_BUCKET:-${S3_STAGING_BUCKET:-local-staging}}"
SRC_HLS_BUCKET="${SRC_HLS_BUCKET:-${S3_PUBLIC_HLS_BUCKET:-local-hls}}"
DST_ENDPOINT="${DST_ENDPOINT:-${R2_ENDPOINT:-}}"
DST_STAGING="${R2_STAGING_BUCKET:?R2_STAGING_BUCKET missing}"
DST_HLS="${R2_PUBLIC_HLS_BUCKET:?R2_PUBLIC_HLS_BUCKET missing}"
DST_KEY_ID="${R2_ACCESS_KEY_ID:?R2_ACCESS_KEY_ID missing}"
DST_KEY_SECRET="${R2_SECRET_ACCESS_KEY:?R2_SECRET_ACCESS_KEY missing}"
SRC_KEY_ID="${SRC_ACCESS_KEY_ID:-dummy}"
SRC_KEY_SECRET="${SRC_SECRET_ACCESS_KEY:-dummy}"
NET="${AWS_DOCKER_NETWORK:-avibe-prod_avibe}"
IMG=amazon/aws-cli:2.0.9
# Tolerate a Windows-edited .env: trailing CR breaks endpoint/auth values.
DST_ENDPOINT="${DST_ENDPOINT%$'\r'}"
DST_STAGING="${DST_STAGING%$'\r'}"
DST_HLS="${DST_HLS%$'\r'}"
DST_KEY_ID="${DST_KEY_ID%$'\r'}"
DST_KEY_SECRET="${DST_KEY_SECRET%$'\r'}"

[ -n "$DST_ENDPOINT" ] || { echo "DST_ENDPOINT (or R2_ENDPOINT) missing"; exit 2; }

WORK=$(mktemp -d "${TMPDIR:-/tmp}/avibe-mediacmp-XXXXXXXX")
chmod 700 "$WORK"
cleanup() { rm -rf "$WORK"; }
trap cleanup EXIT

# Credentials via mounted files (mode 600), never on a command line.
mkdir -p "$WORK/src" "$WORK/dst"
printf '[default]\nregion = us-east-1\noutput = text\n' >"$WORK/src/config"
printf '[default]\naws_access_key_id = %s\naws_secret_access_key = %s\n' \
  "$SRC_KEY_ID" "$SRC_KEY_SECRET" >"$WORK/src/credentials"
printf '[default]\nregion = auto\noutput = text\n' >"$WORK/dst/config"
printf '[default]\naws_access_key_id = %s\naws_secret_access_key = %s\n' \
  "$DST_KEY_ID" "$DST_KEY_SECRET" >"$WORK/dst/credentials"
chmod 600 "$WORK"/src/credentials "$WORK"/dst/credentials

aws_keys() { # <profile-dir> <endpoint> <bucket> <outfile> (key \t size \t etag)
  docker run --rm --network "$NET" \
    -v "$WORK/$1:/root/.aws:ro" "$IMG" \
    --endpoint-url "$2" s3api list-objects-v2 --bucket "$3" \
    --query 'Contents[].[Key, Size, ETag]' --output text 2>/dev/null | sort >"$4"
}

compare_pair() { # <src-bucket> <dst-bucket> <label>
  local src="$1" dst="$2" label="$3"
  echo "== $label: $src -> $dst =="
  aws_keys src "$SRC_ENDPOINT" "$src" "$WORK/$label.src"
  aws_keys dst "$DST_ENDPOINT" "$dst" "$WORK/$label.dst"
  python3 - "$WORK/$label.src" "$WORK/$label.dst" <<'PY'
import sys
def load(p):
    d = {}
    with open(p, encoding="utf-8") as fh:
        for line in fh:
            parts = line.rstrip("\n").split("\t")
            if len(parts) >= 3 and parts[0]:
                d[parts[0]] = (parts[1], parts[2].strip('"'))
    return d
src, dst = load(sys.argv[1]), load(sys.argv[2])
match, missing, mismatch = [], [], []
for k, (size, etag) in sorted(src.items()):
    if k not in dst:
        missing.append(k)
    elif dst[k] != (size, etag):
        mismatch.append(f"{k} src=({size},{etag}) dst=({dst[k][0]},{dst[k][1]})")
    else:
        match.append(k)
extra = sorted(set(dst) - set(src))
print(f"match={len(match)} missing={len(missing)} mismatch={len(mismatch)} extra={len(extra)}")
for k in missing[:50]:
    print(f"MISSING: {k}")
if len(missing) > 50:
    print(f"... and {len(missing) - 50} more missing")
for m in mismatch[:50]:
    print(f"MISMATCH: {m}")
if len(mismatch) > 50:
    print(f"... and {len(mismatch) - 50} more mismatched")
for k in extra[:20]:
    print(f"EXTRA (dst-only): {k}")
if len(extra) > 20:
    print(f"... and {len(extra) - 20} more dst-only")
sys.exit(1 if (missing or mismatch) else 0)
PY
}

fail=0
compare_pair "$SRC_STAGING_BUCKET" "$DST_STAGING" staging || fail=1
compare_pair "$SRC_HLS_BUCKET" "$DST_HLS" hls || fail=1
if [ "$fail" -ne 0 ]; then
  echo "COMPARE FAILED -- missing or mismatched objects (do not delete, G-R2)"
  exit 1
fi
echo "COMPARE OK -- every source key exists in R2 with same size+ETag"
echo "NOTE: ETag equality proves same bytes only for single-part uploads;"
echo "multipart ETags (with -N suffix) compare as opaque version stamps."
