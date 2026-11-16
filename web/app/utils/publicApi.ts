// SSR-safe public API fetch (WI-S2): server-side calls go straight to the
// engine origin (runtimeConfig.apiOrigin), client-side calls use the /api
// proxy. Responses are unwrapped from the {data} envelope.
export function publicApiUrl(path: string): string {
  const p = path.startsWith('/') ? path : `/${path}`;
  if (import.meta.server) {
    const config = useRuntimeConfig();
    const origin = String(config.apiOrigin || 'http://localhost:18081').replace(/\/$/, '');
    return `${origin}/api/v1${p}`;
  }
  return `/api/v1${p}`;
}

interface Envelope<T> {
  data: T;
}

export async function fetchPublic<T>(path: string, query?: Record<string, string | number | boolean | undefined>): Promise<T> {
  const res = await $fetch<Envelope<T>>(publicApiUrl(path), { query: query as Record<string, string> });
  return (res as Envelope<T>).data as T;
}
