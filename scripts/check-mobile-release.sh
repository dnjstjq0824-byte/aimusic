#!/usr/bin/env bash
# WI-D6 (F-07): release API-host gate for the mobile app.
# Fails a release whose API host did not come from --dart-define.
#
# Three checks, all required:
#   1. No Dart source may hardcode the prod host: it enters only via
#      --dart-define=API_BASE_URL=... (keeps the artifact check meaningful).
#   2. The 5 ARB files (WI-R12) carry identical key sets (ko is canonical).
#   3. The built APK's libapp.so must contain the expected host, proving
#      the define took effect (a forgotten flag bakes in localhost instead).
#
# Usage: scripts/check-mobile-release.sh <apk-path> <expected-host>
#   scripts/check-mobile-release.sh mobile/build/app/outputs/flutter-apk/app-release.apk api.avibemusic.com
# Usage (ARB only): scripts/check-mobile-release.sh --arb-only
set -euo pipefail

if [ "${1:-}" = "--arb-only" ]; then
  node - <<'EOF'
const fs = require('node:fs');
const files = ['ko', 'en', 'ja', 'zh', 'th'];
const keys = {};
for (const loc of files) {
  const raw = fs.readFileSync(`mobile/lib/l10n/app_${loc}.arb`, 'utf8');
  keys[loc] = new Set(Object.keys(JSON.parse(raw)).filter((k) => !k.startsWith('@')));
}
const base = keys.ko;
let fail = 0;
for (const loc of files.slice(1)) {
  for (const k of [...base].filter((k) => !keys[loc].has(k)).sort()) {
    console.log(`ARB-PARITY missing ${loc}: ${k}`);
    fail++;
  }
  for (const k of [...keys[loc]].filter((k) => !base.has(k)).sort()) {
    console.log(`ARB-PARITY extra ${loc}: ${k}`);
    fail++;
  }
}
console.log(`arb check: 5 locales x ${base.size} keys ${fail === 0 ? 'OK' : 'FAIL'}`);
process.exit(fail === 0 ? 0 : 1);
EOF
  exit $?
fi

APK="${1:?apk path required}"
HOST="${2:?expected host required}"

if grep -rn --include='*.dart' -e "$HOST" mobile/lib 2>/dev/null; then
  echo "FAIL: prod host hardcoded in Dart source (pass it via --dart-define instead)"
  exit 1
fi
echo "source check: no hardcoded $HOST"

"$0" --arb-only

FOUND="$(python3 -c "
import sys, zipfile
z = zipfile.ZipFile(sys.argv[1])
sos = [n for n in z.namelist() if n.endswith('libapp.so')]
hits = sum(1 for n in sos if sys.argv[2].encode() in z.read(n))
print(f'{hits}/{len(sos)}')
" "$APK" "$HOST")"
echo "artifact check: $HOST in libapp.so ($FOUND)"
[ "${FOUND%%/*}" -ge 1 ] && [ "${FOUND%%/*}" = "${FOUND##*/}" ]
