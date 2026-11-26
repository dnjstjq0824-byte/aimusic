#!/usr/bin/env bash
# Web-only zero-downtime rollout (WI-N4). Run from the repo root on the
# operator PC (Git Bash) with Docker running and the release commit checked
# out (clean tree).
#
#   bash deploy/rollout-web.sh web        # union static assets -> worker -> ship image -> recreate web -> verify
#   bash deploy/rollout-web.sh rollback   # previous web image + wrangler rollback
#   bash deploy/rollout-web.sh smoke      # public checks incl. live /_nuxt/* assets
#   bash deploy/rollout-web.sh --dry-run web|rollback|smoke   # print the plan only
#
# Unlike rollout-redesign.sh vps (backend rebuild + API restart), this path
# touches the web container only. There is no window where Worker assets
# and web HTML disagree on /_nuxt hashes: the union of the old and new
# /app/public ships to the Worker BEFORE the web container is swapped.
#
# Prerequisites done once by the operator:
#   - npx wrangler login with the avibemusic.com account
#
# NEVER runs against prod without PO approval (G-2). --dry-run prints
# every remote command without executing it.
set -euo pipefail

# shellcheck source=common.sh
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"
COMPOSE="docker compose -f docker-compose.prod.yml"
# Web image tag = last commit touching web/ (unrelated commits reuse it).
REV=$(git log -1 --format=%h -- web)
WEB_IMAGE=avibe-prod-web:latest
PREV_FILE=deploy/.web-prev-image   # VPS image ID before the last web rollout (git-ignored)
DRY=0

if [ "${1:-}" = "--dry-run" ]; then DRY=1; shift; fi

preconditions() {
  say "0. Preconditions (target avibe-prod-web:$REV)"
  git diff --quiet || precheck "working tree has uncommitted changes"
  command -v docker >/dev/null || precheck "docker is required"
  docker info >/dev/null 2>&1 || precheck "docker daemon is not running"
  npx wrangler whoami 2>/dev/null | grep -q "$CF_ACCOUNT" || precheck "run: npx wrangler login  (avibemusic.com account)"
  docker image inspect "avibe-prod-web:$REV" >/dev/null 2>&1 || precheck "missing local image avibe-prod-web:$REV (build it first)"
  ssh_run 'docker ps --format "{{.Names}}" | grep -q avibe-prod-web'
}

build_union() {
  say "1. Union old+new /app/public -> deploy/web-public"
  if [ "$DRY" -eq 1 ]; then
    echo "[dry-run] extract /app/public from previous + avibe-prod-web:$REV into deploy/web-public"
    return 0
  fi
  rm -rf deploy/web-public
  mkdir -p deploy/web-public
  # Old assets first (if the previous image is still local), then the new
  # build overwrites same-name files and adds new hashes.
  if [ -f "$PREV_FILE" ]; then
    prev_img=$(cat "$PREV_FILE")
    if docker image inspect "$prev_img" >/dev/null 2>&1; then
      cid=$(docker create "$prev_img")
      docker cp "$cid:/app/public/." deploy/web-public/
      docker rm "$cid" >/dev/null
      echo "union base: $prev_img"
    else
      echo "WARN: previous image $prev_img not local; union is new-build only"
    fi
  else
    echo "WARN: $PREV_FILE missing (first run); union is new-build only"
  fi
  cid=$(docker create "avibe-prod-web:$REV")
  docker cp "$cid:/app/public/." deploy/web-public/
  docker rm "$cid" >/dev/null
  echo "union top: avibe-prod-web:$REV ($(find deploy/web-public -type f | wc -l | tr -d ' ') files)"
}

ship_and_swap() {
  confirm "ship web $REV"
  say "2. Worker deploy (union assets)"
  run npx wrangler deploy

  say "3. Record current prod web image, ship $REV"
  if [ "$DRY" -eq 0 ]; then
    "${SSH[@]}" 'docker inspect avibe-prod-web --format "{{.Image}}"' >"$PREV_FILE"
    echo "previous prod image: $(cat "$PREV_FILE") -> $PREV_FILE"
  else
    echo "[dry-run] ssh $VPS docker inspect avibe-prod-web --format '{{.Image}}' >$PREV_FILE"
  fi
  if [ "$DRY" -eq 1 ]; then
    echo "[dry-run] docker save $WEB_IMAGE avibe-prod-web:$REV | gzip | ssh $VPS 'gunzip | docker load'"
  else
    docker save "$WEB_IMAGE" "avibe-prod-web:$REV" | gzip -1 | "${SSH[@]}" 'gunzip | docker load'
  fi

  say "4. Recreate web only"
  ssh_run "set -e; cd /opt/avibe; $COMPOSE up -d --no-deps --force-recreate web; docker ps --format '{{.Names}}\t{{.Image}}\t{{.Status}}' | grep avibe-prod-web"
  wait_healthy avibe-prod-web 240
}

smoke() {
  say "5. Smoke (public)"
  check() {
    local want="$1" url="$2" got
    if [ "$DRY" -eq 1 ]; then printf '[dry-run] want %s  %s\n' "$want" "$url"; return 0; fi
    got=$(curl -s -o /dev/null -m 20 -w '%{http_code}' "$url")
    printf '%s (want %s)  %s\n' "$got" "$want" "$url"
    [[ "$got" == "$want" ]]
  }
  check 200 https://avibemusic.com/
  check 200 https://avibemusic.com/en
  check 200 https://avibemusic.com/explore
  check 200 https://avibemusic.com/pricing
  check 404 https://avibemusic.com/this-page-does-not-exist
  if [ "$DRY" -eq 1 ]; then
    echo "[dry-run] fetch live HTML, assert every /_nuxt/* reference returns 200"
    return 0
  fi
  if curl -s https://avibemusic.com/ | grep -q '__NUXT'; then echo "SSR html OK"; else echo "FAIL: not Nuxt SSR html"; exit 1; fi
  fail=0
  while read -r asset; do
    code=$(curl -s -o /dev/null -m 20 -w '%{http_code}' "https://avibemusic.com$asset")
    printf '%s  %s\n' "$code" "$asset"
    [ "$code" = "200" ] || fail=1
  done < <(curl -s https://avibemusic.com/ https://avibemusic.com/en https://avibemusic.com/explore https://avibemusic.com/pricing \
    | grep -o '/_nuxt/[^"'"'"' )]\+' | sort -u)   # files only (skips buildAssetsDir:"/_nuxt/")
  [ "$fail" -eq 0 ] || { echo "FAIL: some live /_nuxt assets are not 200"; exit 1; }
  echo "live /_nuxt assets all 200"
}

rollback() {
  say "Rollback: previous web image + wrangler rollback"
  [ -f "$PREV_FILE" ] || { echo "no $PREV_FILE (unknown previous image); refusing"; exit 1; }
  prev_img=$(cat "$PREV_FILE")
  [ -n "$prev_img" ] || { echo "empty $PREV_FILE; refusing"; exit 1; }
  # The ref lands in a remote shell: allow only image-ref characters so a
  # tampered $PREV_FILE cannot inject a remote command.
  [[ "$prev_img" =~ ^[A-Za-z0-9_.:/@-]+$ ]] || { echo "unsafe image ref in $PREV_FILE; refusing"; exit 1; }
  confirm "rollback to $prev_img"
  ssh_run "set -e; cd /opt/avibe; docker tag \"$prev_img\" avibe-prod-web:latest && $COMPOSE up -d --no-deps --force-recreate web && docker ps --format '{{.Names}}\t{{.Image}}\t{{.Status}}' | grep avibe-prod-web"
  wait_healthy avibe-prod-web 240
  run npx wrangler rollback
}

case "${1:-}" in
web) preconditions; build_union; ship_and_swap; smoke ;;
rollback) rollback ;;
smoke) smoke ;;
*) echo "usage: $0 [--dry-run] web|rollback|smoke"; exit 2 ;;
esac
