#!/bin/sh
# MinIO first-boot init: buckets, anonymous HLS reads, staging CORS.
# Runs once via the minio-init compose service (idempotent).
set -eu

mc alias set avibe http://minio:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" >/dev/null
mc mb --ignore-existing avibe/local-staging avibe/local-hls

# Worker origin-fetch is unauthenticated; object keys (hls/<uuid>/...) are
# unguessable and the Worker enforces the playback signature in front.
mc anonymous set download avibe/local-hls

# Browser XHR PUT of presigned staging URLs (cross-origin).
mc cors set avibe/local-staging /init/cors.json

echo "minio-init done:"
mc ls avibe/ 
mc anonymous get avibe/local-hls
