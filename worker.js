// Cloudflare Worker: serves the SPA from static assets and reverse-proxies
// /api/* to the external Go backend (Workers cannot run Go/Postgres/Redis).
// Set API_ORIGIN in the Worker vars (e.g. quick tunnel or server domain).
export default {
  async fetch(request, env) {
    const url = new URL(request.url);

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

    return env.ASSETS.fetch(request);
  },
};
