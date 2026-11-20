// Web-vitals RUM (WI-S5, client only): LCP/INP/CLS samples ride the
// cookieless analytics endpoint as web_vital events. Page-hide flush keeps
// the last samples on unload.
// web-vitals loads lazily (bundle diet): its observers use buffered entries,
// so registering after the initial render still sees the page's LCP/CLS.
import { flushOnHide, trackEvent } from '@/utils/analytics';

export default defineNuxtPlugin(() => {
  flushOnHide();
  const send = (metric: string, value: number, rating: string) => {
    trackEvent({
      name: 'web_vital',
      props: { metric, value: String(Math.round(value * 100) / 100), rating },
    });
  };
  import('web-vitals')
    .then(({ onCLS, onINP, onLCP }) => {
      onLCP((m) => send('LCP', m.value, m.rating));
      onINP((m) => send('INP', m.value, m.rating));
      onCLS((m) => send('CLS', m.value, m.rating));
    })
    .catch(() => {
      // Metrics must never break the app (old browsers, blocked chunk).
    });
});
