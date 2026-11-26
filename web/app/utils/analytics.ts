// Cookieless product analytics (WI-S5, product spec §1.4).
// Events batch in memory and flush to POST /api/v1/events every 5s (or on
// page hide via sendBeacon). No cookies, no personal identifiers — the
// session hash is a per-tab random id kept in memory only.
import type { AnalyticsEventName } from './analyticsEvents';

export interface AnalyticsPayload {
  name: AnalyticsEventName | string;
  locale?: string;
  path?: string;
  props?: Record<string, string>;
}

const queue: AnalyticsPayload[] = [];
let timer: ReturnType<typeof setTimeout> | null = null;
let sessionHash = '';

function session(): string {
  if (!sessionHash) {
    const bytes = new Uint8Array(16);
    crypto.getRandomValues(bytes);
    sessionHash = Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
  }
  return sessionHash;
}

function scheduleFlush(): void {
  if (timer || queue.length === 0) return;
  timer = setTimeout(() => {
    timer = null;
    void flush();
  }, 5000);
}

/** Queue one event (client only; server calls are no-ops). */
export function trackEvent(input: AnalyticsPayload): void {
  if (import.meta.server) return;
  queue.push({
    name: input.name,
    locale: input.locale || document.documentElement.lang || 'ko',
    path: (input.path || location.pathname).slice(0, 256),
    props: input.props ?? {},
  });
  if (queue.length >= 20) {
    void flush();
    return;
  }
  scheduleFlush();
}

export async function flush(): Promise<void> {
  if (queue.length === 0) return;
  const batch = queue.splice(0, 50).map((e) => ({ ...e, sessionHash: session() }));
  try {
    await fetch('/api/v1/events', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ events: batch }),
      keepalive: true,
    });
  } catch {
    // Analytics never breaks the product; drop the batch.
  }
}

/** Flush on tab hide (registered once by the RUM plugin). */
export function flushOnHide(): void {
  if (import.meta.server || typeof document === 'undefined') return;
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'hidden') void flush();
  });
}
