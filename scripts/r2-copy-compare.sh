#!/usr/bin/env bash
# WI-D1: copy LocalStack buckets to R2 and compare key lists + sizes.
# Runs on the VPS in /opt/avibe. Reads prod only; writes only to R2.
# Prints bucket names, object counts, and sizes -- never credentials.
# Re-running copies just the delta (used for the pre-switch re-sync).
set -euo pipefail

cd /opt/avibe
set -a
# shellcheck disable=SC1091
. ./.env
set +a

: "${R2_ENDPOINT:?R2_ENDPOINT missing in .env (PO scope)}"
: "${R2_ACCESS_KEY_ID:?R2_ACCESS_KEY_ID missing in .env (PO scope)}"
: "${R2_SECRET_ACCESS_KEY:?R2_SECRET_ACCESS_KEY missing in .env (PO scope)}"
: "${R2_STAGING_BUCKET:?R2_STAGING_BUCKET missing in .env}"
: "${R2_PUBLIC_HLS_BUCKET:?R2_PUBLIC_HLS_BUCKET missing in .env}"
SRC_STAGING="${S3_STAGING_BUCKET:-local-staging}"
SRC_HLS="${S3_PUBLIC_HLS_BUCKET:-local-hls}"
# Tolerate a Windows-edited .env: trailing CR breaks endpoint/auth values.
R2_ENDPOINT="${R2_ENDPOINT%$'\r'}"
R2_ACCESS_KEY_ID="${R2_ACCESS_KEY_ID%$'\r'}"
R2_SECRET_ACCESS_KEY="${R2_SECRET_ACCESS_KEY%$'\r'}"
R2_STAGING_BUCKET="${R2_STAGING_BUCKET%$'\r'}"
R2_PUBLIC_HLS_BUCKET="${R2_PUBLIC_HLS_BUCKET%$'\r'}"
SRC_STAGING="${SRC_STAGING%$'\r'}"
SRC_HLS="${SRC_HLS%$'\r'}"

NET=avibe-prod_avibe
IMG=amazon/aws-cli:2.0.9
WORK=$(mktemp -d /tmp/avibe-r2copy-XXXXXXXX)
chmod 700 "$WORK"
cleanup() { rm -rf "$WORK"; }
trap cleanup EXIT

# Credentials via mounted files (mode 600), never on a command line.
mkdir -p "$WORK/r2" "$WORK/ls"
printf '[default]\nregion = auto\noutput = text\n' >"$WORK/r2/config"
printf '[default]\naws_access_key_id = %s\naws_secret_access_key = %s\n' \
  "$R2_ACCESS_KEY_ID" "$R2_SECRET_ACCESS_KEY" >"$WORK/r2/credentials"
printf '[default]\nregion = us-east-1\noutput = text\n' >"$WORK/ls/config"
printf '[default]\naws_access_key_id = dummy\naws_secret_access_key = dummy\n' >"$WORK/ls/credentials"
chmod 600 "$WORK"/r2/credentials "$WORK"/ls/credentials

aws_ls() { # <profile-dir> <endpoint> <bucket> -> "objects size"
  docker run --rm --network "$NET" \
    -v "$WORK/$1:/root/.aws:ro" "$IMG" \
    --endpoint-url "$2" s3 ls "s3://$3" --recursive --summarize 2>/dev/null |
    awk '/Total Objects:/{o=$3} /Total Size:/{s=$3} END{print o+0, s+0}'
}

aws_keys() { # <profile-dir> <endpoint> <bucket> <outfile> (key + size per line)
  docker run --rm --network "$NET" \
    -v "$WORK/$1:/root/.aws:ro" "$IMG" \
    --endpoint-url "$2" s3api list-objects-v2 --bucket "$3" \
    --query 'Contents[].[Key, Size]' --output text 2>/dev/null | sort >"$4"
}

copy_pair() { # <src-bucket> <dst-bucket> <label>
  local src="$1" dst="$2" label="$3"
  echo "== $label: $src -> $dst =="
  read -r src_n src_s <<<"$(aws_ls ls http://localstack:4566 "$src")"
  echo "source: objects=$src_n size=$src_s"
  mkdir -p "$WORK/data/$label"
  docker run --rm --network "$NET" -v "$WORK/ls:/root/.aws:ro" \
    -v "$WORK/data/$label:/data" "$IMG" \
    --endpoint-url http://localstack:4566 s3 sync "s3://$src" /data --quiet
  docker run --rm -v "$WORK/r2:/root/.aws:ro" \
    -v "$WORK/data/$label:/data:ro" "$IMG" \
    --endpoint-url "$R2_ENDPOINT" s3 sync /data "s3://$dst" --quiet
  read -r dst_n dst_s <<<"$(aws_ls r2 "$R2_ENDPOINT" "$dst")"
  echo "dest:   objects=$dst_n size=$dst_s"
  aws_keys ls http://localstack:4566 "$src" "$WORK/$label.src"
  aws_keys r2 "$R2_ENDPOINT" "$dst" "$WORK/$label.dst"
  if [ "$src_n" -eq "$dst_n" ] && [ "$src_s" -eq "$dst_s" ] &&
    cmp -s "$WORK/$label.src" "$WORK/$label.dst"; then
    echo "MATCH: key lists and sizes identical"
  else
    echo "MISMATCH: key/size diff:"
    diff "$WORK/$label.src" "$WORK/$label.dst" || true
    return 1
  fi
}

fail=0
copy_pair "$SRC_STAGING" "$R2_STAGING_BUCKET" staging || fail=1
copy_pair "$SRC_HLS" "$R2_PUBLIC_HLS_BUCKET" hls || fail=1
if [ "$fail" -ne 0 ]; then
  echo "COMPARE FAILED -- do not switch (G-R2)"
  exit 1
fi
echo "COMPARE OK -- ready for G-R2 review"
