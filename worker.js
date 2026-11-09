// Cloudflare Worker: serves the SPA from static assets and reverse-proxies
// /api/* to the external Go backend and /local-hls/* to the S3 audio storage.
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
      const s3Origin = (env.S3_ORIGIN || '').replace(/\/+$/, '');
      if (!s3Origin) {
        return Response.json(
          { error: 'Bad Gateway', detail: 'S3_ORIGIN is not configured on this Worker.' },
          { status: 502 }
        );
      }
      const target = s3Origin + url.pathname + url.search;
      const init = {
        method: request.method,
        headers: request.headers,
        redirect: 'follow',
      };
      const res = await fetch(target, init);
      const headers = new Headers(res.headers);
      headers.set('Access-Control-Allow-Origin', '*');
      headers.set('Access-Control-Allow-Methods', 'GET, HEAD, OPTIONS');
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

    // 3. Static SPA assets
    return env.ASSETS.fetch(request);
  },
};
