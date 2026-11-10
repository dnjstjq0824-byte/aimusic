// Cloudflare Worker: serves the SPA from static assets and reverse-proxies
// /api/* to the external Go backend and /local-hls/* to the S3 audio storage.
//
// HLS is signature-gated: every playlist/segment request must carry the
// exp+jti+sig query issued by GET /tracks/:id/stream (HMAC-SHA256 hex over
// "trackId|jti|exp", same construction as backend SignPlaybackToken).
// The R2 bucket itself MUST be private; this Worker is the only reader.
// Secrets: wrangler secret put HLS_SIGNING_KEY  (== backend HLS_SIGNING_KEY)
const TEXT_ENCODER = new TextEncoder();

function timingSafeEqualHex(a, b) {
  if (typeof a !== 'string' || typeof b !== 'string' || a.length !== b.length) return false;
  let diff = 0;
  for (let i = 0; i < a.length; i++) diff |= a.charCodeAt(i) ^ b.charCodeAt(i);
  return diff === 0;
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
      const key = url.pathname.slice('/local-hls/'.length);
      const parts = key.split('/');
      const file = parts[parts.length - 1] || '';
      const trackId = parts.length >= 3 && parts[0] === 'hls' ? parts[1] : '';
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
      const sig = url.searchParams.get('sig') || '';
      const expUnix = Number.parseInt(exp, 10);
      const fresh = Number.isFinite(expUnix) && Date.now() / 1000 <= expUnix;
      let valid = false;
      if (trackId && extOk && fresh && jti && sig) {
        const expected = await hmacHex(signingKey, `${trackId}|${jti}|${expUnix}`);
        valid = timingSafeEqualHex(expected, sig.toLowerCase());
      }
      if (!valid) {
        return Response.json(
          { error: 'Forbidden', detail: 'Missing, expired, or invalid playback signature.' },
          { status: 403 },
        );
      }
      const s3Origin = (env.S3_ORIGIN || '').replace(/\/+$/, '');
      if (!s3Origin) {
        return Response.json(
          { error: 'Bad Gateway', detail: 'S3_ORIGIN is not configured on this Worker.' },
          { status: 502 }
        );
      }
      // Strip the signature before hitting origin (R2 is private to Workers).
      const clean = new URL(url.pathname, 'http://origin');
      const target = s3Origin + clean.pathname;
      const init = {
        method: request.method,
        headers: request.headers,
        redirect: 'follow',
      };
      const res = await fetch(target, init);
      const headers = new Headers(res.headers);
      headers.set('Access-Control-Allow-Origin', '*');
      headers.set('Access-Control-Allow-Methods', 'GET, HEAD, OPTIONS');
      // VOD segments are immutable once packaged: cache hard.
      if (file.endsWith('.ts')) {
        headers.set('Cache-Control', 'public, max-age=31536000, immutable');
      } else if (file.endsWith('.m3u8')) {
        // HLS clients (hls.js, Safari, just_audio) resolve segment URIs
        // relative to the playlist WITHOUT forwarding its query string,
        // so every segment would 403 here. Embed the already-verified
        // grant into each media URI instead of patching every player.
        // Private: one grant must never be served to another client.
        if (res.status === 200 && !headers.get('content-encoding')) {
          const grant = `exp=${encodeURIComponent(exp)}&jti=${encodeURIComponent(jti)}&sig=${encodeURIComponent(sig)}`;
          const text = await res.text();
          const rewritten = text
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
          headers.set('Cache-Control', 'private, max-age=60');
          headers.delete('content-length');
          return new Response(rewritten, {
            status: res.status,
            statusText: res.statusText,
            headers,
          });
        }
        headers.set('Cache-Control', 'private, max-age=60');
      }
      return new Response(res.body, {
        status: res.status,
        statusText: res.statusText,
        headers,
      });
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
