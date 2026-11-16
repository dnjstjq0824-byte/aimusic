// Web-vitals RUM (WI-S5, client only): LCP/INP/CLS samples ride the
// cookieless analytics endpoint as web_vital events. Page-hide flush keeps
// the last samples on unload.
import { onCLS, onINP, onLCP } from 'web-vitals';
import { flushOnHide, trackEvent } from '@/utils/analytics';

export default defineNuxtPlugin(() => {
  flushOnHide();
  const send = (metric: string, value: number, rating: string) => {
    trackEvent({
      name: 'web_vital',
      props: { metric, value: String(Math.round(value * 100) / 100), rating },
    });
  };
  try {
    onLCP((m) => send('LCP', m.value, m.rating));
    onINP((m) => send('INP', m.value, m.rating));
    onCLS((m) => send('CLS', m.value, m.rating));
  } catch {
    // Metrics must never break the app (old browsers).
  }
});
