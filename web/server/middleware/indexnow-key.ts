// Serves the IndexNow key file /{key}.txt (WI-S3).
// The key lives in INDEXNOW_KEY (engine env); the web tier mirrors it so the
// ownership proof URL and the submission key can never drift apart.
export default defineEventHandler((event) => {
  const path = event.path || '';
  const m = /^\/([A-Za-z0-9_-]{8,64})\.txt$/.exec(path);
  if (!m) return;
  const { indexnowKey } = useRuntimeConfig(event);
  if (!indexnowKey || m[1] !== String(indexnowKey)) return;
  setHeader(event, 'Content-Type', 'text/plain; charset=utf-8');
  setHeader(event, 'Cache-Control', 'public, max-age=86400');
  return String(indexnowKey);
});
