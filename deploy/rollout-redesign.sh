#!/usr/bin/env bash
# Rollout of the 2026-11-14 redesign (Nuxt 4 SSR web + backend 000020~000027)
# to production. Run from the repo root on the operator PC (Git Bash) with
# Docker running and the release commit checked out (clean tree).
#
#   bash deploy/rollout-redesign.sh vps      # backend + migrations + web container
#   bash deploy/rollout-redesign.sh worker   # Worker (assets from the same web image)
#   bash deploy/rollout-redesign.sh smoke    # public smoke checks
#   bash deploy/rollout-redesign.sh rollback # previous backend + configs, web stopped
#
# Prerequisites done once by the operator:
#   - DNS: web.avibemusic.com -> tunnel `avibe` (cloudflared tunnel route dns ...)
#   - npx wrangler login with the avibemusic.com account
set -euo pipefail

VPS=root@158.247.219.159
KEY="$HOME/.ssh/avibe_vultr"
SSH=(ssh -i "$KEY" -o BatchMode=yes "$VPS")
CF_ACCOUNT=f5c363135efe9b22ca66eb39f2f8e1da   # avibemusic.com account
TS=$(date -u +%Y%m%d-%H%M%S)
REV=$(git rev-parse --short HEAD)
WEB_IMAGE=avibe-prod-web:latest
COMPOSE="docker compose -f docker-compose.prod.yml"

say() { printf '\n== %s\n' "$*"; }

build_web_image() {
  # Build from the committed tree only (no local node_modules/.output leak).
  local ctx; ctx=$(mktemp -d)
  git -c core.autocrlf=false archive HEAD web | tar -x -C "$ctx" --strip-components=1
  # The lockfile was produced with legacy peer resolution (vite 5 vs unhead's
  # optional vite>=6 peer); npm ci in the image needs the same setting.
  echo 'legacy-peer-deps=true' > "$ctx/.npmrc"
  sed -i 's#^COPY package.json package-lock.json ./#COPY package.json package-lock.json .npmrc ./#' "$ctx/Dockerfile"
  docker build --platform linux/amd64 -t "$WEB_IMAGE" -t "avibe-prod-web:$REV" "$ctx"
  rm -rf "$ctx"
}

case "${1:-}" in
vps)
  say "0. Preconditions"
  git diff --quiet || { echo "working tree has uncommitted changes"; exit 1; }
  "${SSH[@]}" 'docker ps --format "{{.Names}}" | grep -q avibe-prod-api'
  docker image inspect "avibe-prod-web:$REV" >/dev/null 2>&1 || build_web_image

  say "1. DB backup now"
  "${SSH[@]}" '/opt/avibe/backup-db.sh && ls -lt /opt/avibe/backups | sed -n 2p'

  say "2. Ship backend source (previous kept as backend.prev-$TS)"
  git -c core.autocrlf=false archive HEAD backend | "${SSH[@]}" "set -e; cd /opt/avibe; rm -rf backend.new; mkdir backend.new; tar -x -C backend.new --strip-components=1; mv backend backend.prev-$TS; mv backend.new backend"

  say "3. Compose + tunnel config (backups *.bak-rollout-$TS)"
  "${SSH[@]}" "set -e; cd /opt/avibe; cp -p .env .env.bak-rollout-$TS; cp -p docker-compose.prod.yml docker-compose.prod.yml.bak-rollout-$TS; cp -p cloudflared/config.yml cloudflared/config.yml.bak-rollout-$TS"
  git -c core.autocrlf=false show HEAD:docker-compose.prod.yml | "${SSH[@]}" 'cat > /opt/avibe/docker-compose.prod.yml'
  git -c core.autocrlf=false show HEAD:deploy/cloudflared-config.yml | "${SSH[@]}" 'cat > /opt/avibe/cloudflared/config.yml'
  "${SSH[@]}" 'docker run --rm -v /opt/avibe/cloudflared:/etc/cloudflared:ro cloudflare/cloudflared:2026.10.7 --config /etc/cloudflared/config.yml tunnel ingress validate'

  say "4. Ship web image ($REV)"
  docker save "$WEB_IMAGE" "avibe-prod-web:$REV" | gzip -1 | "${SSH[@]}" 'gunzip | docker load'

  say "5. Build api, migrate (000020~000027), restart api/worker/web/cloudflared"
  "${SSH[@]}" "set -e; cd /opt/avibe
    $COMPOSE build api
    $COMPOSE run --rm migrate
    $COMPOSE up -d --no-deps --force-recreate api worker web
    $COMPOSE up -d --no-deps --force-recreate cloudflared
    sleep 15
    docker exec avibe-prod-postgres psql -U avibe -d avibe -At -c 'SELECT max(version) FROM schema_migrations'
    docker ps --format '{{.Names}}\t{{.Image}}\t{{.Status}}'
    docker run --rm --network avibe-prod_avibe curlimages/curl:8.11.1 -s -o /dev/null -w 'web ssr %{http_code}\n' http://web:3000/
    docker run --rm --network avibe-prod_avibe curlimages/curl:8.11.1 -s -o /dev/null -w 'api ready %{http_code}\n' http://api:8080/readyz"
  ;;

worker)
  say "Worker: assets from avibe-prod-web:$REV"
  npx wrangler whoami 2>/dev/null | grep -q "$CF_ACCOUNT" || { echo "run: npx wrangler login  (avibemusic.com account)"; exit 1; }
  docker image inspect "avibe-prod-web:$REV" >/dev/null || { echo "build/ship the web image first (vps step)"; exit 1; }
  rm -rf deploy/web-public
  cid=$(docker create "avibe-prod-web:$REV")
  docker cp "$cid:/app/public" deploy/web-public
  docker rm "$cid" >/dev/null
  npx wrangler deploy
  ;;

smoke)
  check() { local want="$1" url="$2"; local got; got=$(curl -s -o /dev/null -m 20 -w '%{http_code}' "$url"); printf '%s (want %s)  %s\n' "$got" "$want" "$url"; [[ "$got" == "$want" ]]; }
  check 200 https://api.avibemusic.com/readyz
  check 200 "https://avibemusic.com/api/v1/tracks?limit=1"
  check 200 https://avibemusic.com/
  check 200 https://avibemusic.com/en
  check 200 https://avibemusic.com/explore
  check 200 https://avibemusic.com/pricing
  check 200 https://avibemusic.com/login
  check 200 https://avibemusic.com/robots.txt
  check 200 https://avibemusic.com/sitemap_index.xml
  check 404 https://avibemusic.com/this-page-does-not-exist
  check 403 "https://avibemusic.com/local-hls/hls/x/stream_0.m3u8"
  curl -s https://avibemusic.com/ | grep -q '__NUXT' && echo "SSR html OK" || { echo "FAIL: not Nuxt SSR html"; exit 1; }
  ;;

rollback)
  say "Rollback: previous backend, .env, compose, tunnel config; web stopped"
  "${SSH[@]}" 'set -e; cd /opt/avibe
    PREV=$(ls -dt backend.prev-* | head -1); test -n "$PREV"
    rm -rf backend && mv "$PREV" backend
    cp -p "$(ls -t .env.bak-rollout-* | head -1)" .env
    docker stop avibe-prod-web 2>/dev/null || true
    cp -p "$(ls -t docker-compose.prod.yml.bak-rollout-* | head -1)" docker-compose.prod.yml
    cp -p "$(ls -t cloudflared/config.yml.bak-rollout-* | head -1)" cloudflared/config.yml
    docker compose -f docker-compose.prod.yml build api
    docker compose -f docker-compose.prod.yml up -d --no-deps --force-recreate api worker cloudflared'
  echo "Migrations 000020~000027 are additive and stay (down files in backend/migrations)."
  echo "Worker: npx wrangler rollback"
  ;;

*)
  echo "usage: $0 vps|worker|smoke|rollback"; exit 2 ;;
esac
