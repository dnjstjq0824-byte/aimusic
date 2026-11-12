// Cloudflare Worker: serves the SPA from static assets, reverse-proxies
// /api/* to the external Go backend, and reads /local-hls/* from the private
// R2 bucket bound as HLS_BUCKET (WI-D1: no origin fetch, no Access tokens).
//
// HLS is signature-gated: every playlist/segment request must carry the
// exp+jti+q+sig query issued by GET /tracks/:id/stream (HMAC-SHA256 hex over
// "trackId|jti|exp|q", same construction as backend SignPlaybackToken).
// q is the quality tier: SQ grants only the 128k variant (stream_0), HQ all.
// The R2 bucket itself MUST be private; this Worker is the only reader.
// Secrets: wrangler secret put HLS_SIGNING_KEY  (== backend HLS_SIGNING_KEY)
const TEXT_ENCODER = new TextEncoder();

function timingSafeEqualHex(a, b) {
  if (typeof a !== 'string' || typeof b !== 'string' || a.length !== b.length) return false;
  let diff = 0;
  for (let i = 0; i < a.length; i++) diff |= a.charCodeAt(i) ^ b.charCodeAt(i);
  return diff === 0;
}

// SQ grants: the 128k variant playlist, its segments, and waveform peaks.
const SQ_FILE = /^(stream_0\.m3u8|seg_0_\d+\.ts|peaks\.json)$/;

// fileAllowed reports whether a grant of tier q may fetch file (B-04).
export function fileAllowed(q, file) {
  if (q === 'HQ') return true;
  if (q === 'SQ') return SQ_FILE.test(file);
  return false;
}

// parseHlsPath splits '/local-hls/hls/<trackId>/<file>' for the R2 key.
export function parseHlsPath(pathname) {
  if (!pathname.startsWith('/local-hls/')) return null;
  const key = pathname.slice('/local-hls/'.length);
  const parts = key.split('/');
  if (parts.length < 3 || parts[0] !== 'hls' || !parts[1]) return null;
  return { key, trackId: parts[1], file: parts[parts.length - 1] || '' };
}

// rewritePlaylistUris embeds the verified grant into relative media URIs.
// HLS clients resolve segment URIs without forwarding the playlist query.
export function rewritePlaylistUris(text, grant) {
  return text
    .split('\n')
    .map((line) => {
      const t = line.trim();
      if (t === '' || t.startsWith('#')) return line;
      if (/^https?:\/\//i.test(t)) return line;
      const hash = t.indexOf('#');
      const path = hash === -1 ? t : t.slice(0, hash);
      const frag = hash === -1 ? '' : t.slice(hash);
      return `${path}${path.includes('?') ? '&' : '?'}${grant}${frag}`;
    })
    .join('\n');
}

function contentTypeFor(file) {
  if (file.endsWith('.m3u8')) return 'application/vnd.apple.mpegurl';
  if (file.endsWith('.ts')) return 'video/mp2t';
  if (file.endsWith('.json')) return 'application/json';
  return 'application/octet-stream';
}

// parseRange maps a single 'bytes=start-end' range onto size.
export function parseRange(header, size) {
  const m = /^bytes=(\d*)-(\d*)$/.exec((header || '').trim());
  if (!m || (m[1] === '' && m[2] === '')) return null;
  let start;
  let end;
  if (m[1] === '') {
    const suffix = Number(m[2]);
    if (!suffix || size === 0) return null;
    start = Math.max(0, size - suffix);
    end = size - 1;
  } else {
    start = Number(m[1]);
    end = m[2] === '' ? size - 1 : Number(m[2]);
  }
  if (start >= size || end < start) return null;
  return { offset: start, length: Math.min(end, size - 1) - start + 1 };
}

// r2ResponseHeaders sets playback headers from R2 metadata.
function r2ResponseHeaders(obj, file, bodyLength, extra = {}) {
  const headers = new Headers();
  headers.set('Access-Control-Allow-Origin', '*');
  headers.set('Access-Control-Allow-Methods', 'GET, HEAD, OPTIONS');
  headers.set('Content-Type', (obj.httpMetadata && obj.httpMetadata.contentType) || contentTypeFor(file));
  const etag = obj.httpEtag || (obj.etag ? `"${obj.etag}"` : '');
  if (etag) headers.set('ETag', etag);
  headers.set('Accept-Ranges', 'bytes');
  headers.set('Content-Length', String(bodyLength));
  if (file.endsWith('.ts')) {
    headers.set('Cache-Control', 'public, max-age=31536000, immutable');
  } else if (file.endsWith('.m3u8')) {
    headers.set('Cache-Control', 'private, max-age=60');
  } else {
    headers.set('Cache-Control', 'private, max-age=3600');
  }
  if (extra.contentRange) headers.set('Content-Range', extra.contentRange);
  return headers;
}

async function hmacHex(key, message) {
  const cryptoKey = await crypto.subtle.importKey(
    'raw',
    TEXT_ENCODER.encode(key),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign'],
  );
  const sig = await crypto.subtle.sign('HMAC', cryptoKey, TEXT_ENCODER.encode(message));
  return [...new Uint8Array(sig)].map((b) => b.toString(16).padStart(2, '0')).join('');
}

export default {
  async fetch(request, env) {
    const url = new URL(request.url);

    // 1. Audio stream and segment delivery (HLS .m3u8 and .ts)
    if (url.pathname.startsWith('/local-hls/')) {
      if (request.method === 'OPTIONS') {
        return new Response(null, {
          status: 204,
          headers: {
            'Access-Control-Allow-Origin': '*',
            'Access-Control-Allow-Methods': 'GET, HEAD, OPTIONS',
            'Access-Control-Allow-Headers': '*',
            'Access-Control-Max-Age': '86400',
          },
        });
      }
      if (request.method !== 'GET' && request.method !== 'HEAD') {
        return Response.json({ error: 'Method Not Allowed' }, { status: 405 });
      }
      // Object keys look like hls/<trackId>/<file>; only media files served.
      const parsed = parseHlsPath(url.pathname);
      const key = parsed ? parsed.key : '';
      const file = parsed ? parsed.file : '';
      const trackId = parsed ? parsed.trackId : '';
      const extOk = file.endsWith('.m3u8') || file.endsWith('.ts') || file === 'peaks.json';
      const signingKey = env.HLS_SIGNING_KEY || '';
      if (!signingKey) {
        return Response.json(
          { error: 'Bad Gateway', detail: 'HLS_SIGNING_KEY is not configured on this Worker.' },
          { status: 502 },
        );
      }
      const exp = url.searchParams.get('exp') || '';
      const jti = url.searchParams.get('jti') || '';
      const q = url.searchParams.get('q') || '';
      const sig = url.searchParams.get('sig') || '';
      const expUnix = Number.parseInt(exp, 10);
      const fresh = Number.isFinite(expUnix) && Date.now() / 1000 <= expUnix;
      let valid = false;
      if (trackId && extOk && fresh && jti && sig) {
        if (q) {
          if (fileAllowed(q, file)) {
            const expected = await hmacHex(signingKey, `${trackId}|${jti}|${expUnix}|${q}`);
            valid = timingSafeEqualHex(expected, sig.toLowerCase());
          }
        } else if (env.ALLOW_LEGACY_GRANTS === 'true') {
          // Rollout bridge only: grants minted by a backend that predates the
          // q tier. Set ALLOW_LEGACY_GRANTS while backend and Worker deploy,
          // then remove it (legacy grants carry no tier, see B-04).
          const expected = await hmacHex(signingKey, `${trackId}|${jti}|${expUnix}`);
          valid = timingSafeEqualHex(expected, sig.toLowerCase());
        }
      }
      if (!valid) {
        return Response.json(
          { error: 'Forbidden', detail: 'Missing, expired, or invalid playback signature.' },
          { status: 403 },
        );
      }
      const bucket = env.HLS_BUCKET;
      if (!bucket) {
        return Response.json(
          { error: 'Bad Gateway', detail: 'HLS_BUCKET is not bound to this Worker.' },
          { status: 502 }
        );
      }
      // R2 is private to this Worker: read the binding, never the network.
      if (request.method === 'HEAD') {
        const meta = await bucket.head(key);
        if (!meta) {
          return Response.json({ error: 'Not Found' }, { status: 404 });
        }
        return new Response(null, { status: 200, headers: r2ResponseHeaders(meta, file, meta.size) });
      }
      const rangeHeader = request.headers.get('range');
      let obj;
      let status = 200;
      let contentRange = '';
      if (rangeHeader) {
        const meta = await bucket.head(key);
        if (!meta) {
          return Response.json({ error: 'Not Found' }, { status: 404 });
        }
        const range = parseRange(rangeHeader, meta.size);
        if (!range) {
          return Response.json({ error: 'Range Not Satisfiable' }, { status: 416 });
        }
        obj = await bucket.get(key, { range });
        if (!obj) {
          return Response.json({ error: 'Not Found' }, { status: 404 });
        }
        status = 206;
        contentRange = `bytes ${range.offset}-${range.offset + range.length - 1}/${meta.size}`;
      } else {
        obj = await bucket.get(key);
        if (!obj) {
          return Response.json({ error: 'Not Found' }, { status: 404 });
        }
      }
      const etag = obj.httpEtag || (obj.etag ? `"${obj.etag}"` : '');
      const inm = request.headers.get('if-none-match');
      if (inm && etag && inm.split(',').some((t) => t.trim() === etag || t.trim() === '*')) {
        const notModified = new Headers();
        notModified.set('ETag', etag);
        return new Response(null, { status: 304, headers: notModified });
      }
      const ims = request.headers.get('if-modified-since');
      if (ims && obj.uploaded && !Number.isNaN(Date.parse(ims)) && Date.parse(ims) >= obj.uploaded.getTime()) {
        const notModified = new Headers();
        if (etag) notModified.set('ETag', etag);
        return new Response(null, { status: 304, headers: notModified });
      }
      const bodyLength = obj.range ? obj.range.length : obj.size;
      const headers = r2ResponseHeaders(obj, file, bodyLength, { contentRange });
      if (file.endsWith('.m3u8') && status === 200) {
        // Embed the already-verified grant into each media URI (see
        // rewritePlaylistUris). Private: one grant must never be served
        // to another client.
        const grant = `exp=${encodeURIComponent(exp)}&jti=${encodeURIComponent(jti)}&q=${encodeURIComponent(q)}&sig=${encodeURIComponent(sig)}`;
        const rewritten = rewritePlaylistUris(await obj.text(), grant);
        headers.delete('content-length');
        return new Response(rewritten, { status, headers });
      }
      return new Response(obj.body, { status, headers });
    }

    // 2. API reverse proxy
    if (url.pathname.startsWith('/api/')) {
      const origin = (env.API_ORIGIN || '').replace(/\/+$/, '');
      if (!origin) {
        return Response.json(
          { error: 'Bad Gateway', detail: 'API_ORIGIN is not configured on this Worker.' },
          { status: 502 }
        );
      }
      const target = origin + url.pathname + url.search;
      const init = {
        method: request.method,
        headers: request.headers,
        redirect: 'manual',
      };
      if (request.method !== 'GET' && request.method !== 'HEAD') {
        init.body = request.body;
        init.duplex = 'half';
      }
      return fetch(target, init);
    }

    // 3. Static SPA assets with hardening headers (tokens live in
    // localStorage, so XSS/clickjacking defenses on the app shell matter).
    const asset = await env.ASSETS.fetch(request);
    const headers = new Headers(asset.headers);
    headers.set('X-Content-Type-Options', 'nosniff');
    headers.set('Referrer-Policy', 'strict-origin-when-cross-origin');
    headers.set('X-Frame-Options', 'DENY');
    headers.set('Strict-Transport-Security', 'max-age=31536000; includeSubDomains');
    headers.set('Permissions-Policy', 'camera=(), microphone=(), geolocation=()');
    headers.set(
      'Content-Security-Policy',
      "default-src 'self'; " +
        "script-src 'self' https://static.cloudflareinsights.com; " +
        "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdn.jsdelivr.net; " +
        'font-src \'self\' https://fonts.gstatic.com https://cdn.jsdelivr.net data:; ' +
        'img-src \'self\' data: blob:; ' +
        'media-src \'self\' blob: https:; ' +
        'connect-src \'self\' https:; ' +
        'frame-src https://pay.tosspayments.com https://*.tosspayments.com; ' +
        "frame-ancestors 'none'; base-uri 'self'; form-action 'self'",
    );
    return new Response(asset.body, {
      status: asset.status,
      statusText: asset.statusText,
      headers,
    });
  },
};
