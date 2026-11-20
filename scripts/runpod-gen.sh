#!/usr/bin/env bash
# AVIBE generator on Runpod: bring a GPU pod up on the network volume that
# already holds ACE-Step 1.5 + weights + generator/server.py, or tear it down.
# Nothing is reinstalled: the volume keeps everything (docs/generation/poc-runpod-20261121.md).
#
#   bash scripts/runpod-gen.sh up       # create pod (cheapest GPU in stock), start server, print access
#   bash scripts/runpod-gen.sh test     # one 30s generation over SSH, prints timing
#   bash scripts/runpod-gen.sh status   # pod state, price, SSH command
#   bash scripts/runpod-gen.sh down     # terminate the pod (GPU billing stops; the volume stays)
#   bash scripts/runpod-gen.sh sync     # copy this repo's generator/server.py onto the volume
#
# Needs: Runpod CLI logged in (`runpod login`), SSH key ~/.ssh/runpod_ed25519.
# The pod is public on https://<podId>-8000.proxy.runpod.net only with PUBLIC=1;
# the server then requires the bearer token in /workspace/gen/.token.
set -euo pipefail

RP="${RP:-$HOME/.local/bin/runpod}"; [ -x "$RP" ] || RP="$HOME/.local/bin/runpod.exe"
VOLUME_ID="${VOLUME_ID:-y7tzkyd0g5}"          # network volume avibe-gen, EU-RO-1, 40GB
DATA_CENTER="${DATA_CENTER:-EU-RO-1}"
POD_NAME="${POD_NAME:-avibe-gen}"
KEY="${KEY:-$HOME/.ssh/runpod_ed25519}"
# Cheapest first; each needs >=16GB VRAM (ACE-Step 1.5 turbo) and CUDA >= 12.8.
GPUS="${GPUS:-rtx 2000 ada|rtx 4000 ada|rtx pro 4000|l4|rtx 4090|a40|rtx a6000}"

pod_id() { "$RP" pod list --json 2>/dev/null | node -e '
  let d="";process.stdin.on("data",c=>d+=c).on("end",()=>{const j=JSON.parse(d||"[]");
  const l=Array.isArray(j)?j:(j.pods||[]);const p=l.find(x=>x.name===process.argv[1]);console.log(p?p.id:"")})' "$POD_NAME"; }

ssh_cmd() { "$RP" pod get "$1" --json | node -e '
  let d="";process.stdin.on("data",c=>d+=c).on("end",()=>{const x=JSON.parse(d);const p=x.pod||x;
  const s=p.ssh&&p.ssh.direct;console.log(s?`${s.username}@${s.host} ${s.port}`:"")})'; }

on_pod() { local id="$1"; shift; read -r host port <<<"$(ssh_cmd "$id")"
  ssh -i "$KEY" -o StrictHostKeyChecking=accept-new -o BatchMode=yes -o ConnectTimeout=30 -p "$port" "$host" "$@"; }

case "${1:-}" in
up)
  id=$(pod_id)
  if [ -z "$id" ]; then
    pk=$(cat "$KEY.pub")
    IFS='|' read -ra list <<<"$GPUS"
    for g in "${list[@]}"; do
      echo "== trying $g"
      if out=$("$RP" pod create "$POD_NAME" --gpu "$g" --cloud secure --data-center "$DATA_CENTER" --cuda '>=12.8' \
          --template runpod-torch-v280 --disk 20 --volume "$VOLUME_ID:/workspace" \
          -p 8000/http -p 22/tcp -e "PUBLIC_KEY=$pk" --wait --json 2>&1); then
        id=$(pod_id); echo "created $id ($g)"; break
      fi
      echo "$out" | grep -v "ssh keys" | tail -1
    done
    [ -n "$id" ] || { echo "no GPU in stock in $DATA_CENTER right now; retry later or set GPUS=..."; exit 1; }
  fi
  for i in $(seq 1 20); do on_pod "$id" true 2>/dev/null && break; sleep 6; done
  if [ "${PUBLIC:-0}" = 1 ]; then on_pod "$id" 'HOST=0.0.0.0 /workspace/gen/start.sh'; else on_pod "$id" '/workspace/gen/start.sh'; fi
  "$0" status
  ;;
test)
  id=$(pod_id); [ -n "$id" ] || { echo "no pod; run up"; exit 1; }
  on_pod "$id" 'for i in $(seq 1 60); do curl -sf 127.0.0.1:8000/healthz >/dev/null && break; sleep 3; done; T=$(cat /workspace/gen/.token);
    curl -s -o /tmp/t.json -w "generate 30s: http=%{http_code} time=%{time_total}s\n" -X POST 127.0.0.1:8000/generate \
      -H "Authorization: Bearer $T" -H "Content-Type: application/json" \
      -d "{\"jobId\":\"smoke\",\"prompt\":\"lofi chill hop, warm piano\",\"params\":{\"durationSec\":30,\"seed\":42}}"'
  ;;
status)
  id=$(pod_id); [ -n "$id" ] || { echo "no pod named $POD_NAME (volume $VOLUME_ID is kept)"; exit 0; }
  "$RP" pod get "$id" --json | node -e '
    let d="";process.stdin.on("data",c=>d+=c).on("end",()=>{const x=JSON.parse(d);const p=x.pod||x;
    console.log(`pod ${p.id} ${p.status} | ${p.gpu&&p.gpu.id} | $${p.cost}/hr | ${p.dataCenterId}`);
    if(p.ssh&&p.ssh.direct)console.log("ssh: "+p.ssh.direct.command+" -i ~/.ssh/runpod_ed25519");
    console.log(`proxy: https://${p.id}-8000.proxy.runpod.net (only with PUBLIC=1)`)})'
  ;;
down)
  id=$(pod_id); [ -n "$id" ] || { echo "no pod to terminate"; exit 0; }
  "$RP" pod delete "$id" --yes >/dev/null && echo "terminated $id (volume $VOLUME_ID kept)"
  ;;
sync)
  id=$(pod_id); [ -n "$id" ] || { echo "no pod; run up"; exit 1; }
  read -r host port <<<"$(ssh_cmd "$id")"
  scp -i "$KEY" -P "$port" "$(dirname "$0")/../generator/server.py" "$host:/workspace/gen/server.py" && echo "synced server.py"
  ;;
*) echo "usage: $0 up|test|status|down|sync"; exit 2 ;;
esac
