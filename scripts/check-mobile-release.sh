#!/usr/bin/env bash
# WI-D6 (F-07): release API-host gate for the mobile app.
# Fails a release whose API host did not come from --dart-define.
#
# Two halves, both required:
#   1. No Dart source may hardcode the prod host: it enters only via
#      --dart-define=API_BASE_URL=... (keeps the artifact check meaningful).
#   2. The built APK's libapp.so must contain the expected host, proving
#      the define took effect (a forgotten flag bakes in localhost instead).
#
# Usage: scripts/check-mobile-release.sh <apk-path> <expected-host>
#   scripts/check-mobile-release.sh mobile/build/app/outputs/flutter-apk/app-release.apk api.avibemusic.com
set -euo pipefail

APK="${1:?apk path required}"
HOST="${2:?expected host required}"

if grep -rn --include='*.dart' -e "$HOST" mobile/lib 2>/dev/null; then
  echo "FAIL: prod host hardcoded in Dart source (pass it via --dart-define instead)"
  exit 1
fi
echo "source check: no hardcoded $HOST"

FOUND="$(python3 -c "
import sys, zipfile
z = zipfile.ZipFile(sys.argv[1])
sos = [n for n in z.namelist() if n.endswith('libapp.so')]
hits = sum(1 for n in sos if sys.argv[2].encode() in z.read(n))
print(f'{hits}/{len(sos)}')
" "$APK" "$HOST")"
echo "artifact check: $HOST in libapp.so ($FOUND)"
[ "${FOUND%%/*}" -ge 1 ] && [ "${FOUND%%/*}" = "${FOUND##*/}" ]
