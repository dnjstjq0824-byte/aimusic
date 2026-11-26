#!/usr/bin/env bash
# Shared rollout plumbing (sourced, never executed directly).
# Single home for the prod target identity plus the dry-run/confirm/health
# helpers used by every deploy/rollout-*.sh. Requires bash; callers set
# DRY=0/1 before invoking helpers (evaluated at call time).
# shellcheck disable=SC2034 # VPS/KEY/SSH/CF_ACCOUNT are consumed by callers.

# Canonical prod target (VPS + operator key + Cloudflare account).
# scripts/r2-env-append.ps1 defaults to the same host.
VPS="${VPS:-root@158.247.219.159}"
KEY="${KEY:-$HOME/.ssh/avibe_vultr}"
SSH=(ssh -i "$KEY" -o BatchMode=yes "$VPS")
CF_ACCOUNT="${CF_ACCOUNT:-f5c363135efe9b22ca66eb39f2f8e1da}" # avibemusic.com account

say() { printf '\n== %s\n' "$*"; }
run() {
  if [ "$DRY" -eq 1 ]; then printf '[dry-run] %s\n' "$*"; else "$@"; fi
}
ssh_run() {
  if [ "$DRY" -eq 1 ]; then printf '[dry-run] ssh %s %s\n' "$VPS" "$*"; else "${SSH[@]}" "$*"; fi
}
confirm() {
  # G-2: every prod mutation needs an explicit, typed confirmation per run.
  # Reads /dev/tty (not stdin): earlier pipeline stages may have consumed
  # stdin, and without a terminal there is no safe way to confirm.
  if [ "$DRY" -eq 1 ]; then echo "[dry-run] would ask: type YES to proceed ($1)"; return 0; fi
  printf 'Type YES to proceed against PROD (%s): ' "$1" >/dev/tty || { echo "no terminal; refusing without confirmation"; exit 1; }
  read -r ans </dev/tty || { echo "no confirmation received; aborting"; exit 1; }
  [ "$ans" = "YES" ] || { echo "aborted (confirmation required)"; exit 1; }
}
wait_healthy() {
  # $1 = container, $2 = timeout seconds. Replaces blind sleep with a gate.
  say "Waiting for $1 to report healthy (up to $2s)"
  ssh_run "end=\$((\$(date +%s) + $2)); while [ \"\$(date +%s)\" -lt \"\$end\" ]; do st=\$(docker inspect --format '{{.State.Health.Status}}' '$1' 2>/dev/null || echo missing); if [ \"\$st\" = healthy ]; then echo '$1 healthy'; exit 0; fi; sleep 5; done; echo '$1 never healthy (last: \$st)'; exit 1"
}
# precheck degrades a precondition failure to a warning under --dry-run so
# the full plan always prints; real runs stop.
precheck() {
  if [ "$DRY" -eq 1 ]; then echo "WARN: $1 (real run would stop here)";
  else echo "$1"; exit 1; fi
}
