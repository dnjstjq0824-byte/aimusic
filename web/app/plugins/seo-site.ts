// Site-wide head extras (WI-S3/S5): search-console verification metas and
// the Cloudflare Web Analytics beacon. Tokens come from public runtime
// config (PO-provided); empty tokens render nothing.
export default defineNuxtPlugin(() => {
  const config = useRuntimeConfig();
  const metas: { name: string; content: string }[] = [];
  const push = (name: string, value: unknown) => {
    if (typeof value === 'string' && value) metas.push({ name, content: value });
  };
  push('google-site-verification', config.public.verifyGoogle);
  push('msvalidate.01', config.public.verifyBing);
  push('naver-site-verification', config.public.verifyNaver);
  // Daum verifies via meta or file; the file slot is public/daum-www-verification.html (PO drops it in).
  push('daum-site-verification', config.public.verifyDaum);
  if (metas.length > 0) useHead({ meta: metas });

  const beacon = config.public.cfWebAnalyticsToken;
  if (typeof beacon === 'string' && beacon) {
    useHead({
      script: [
        {
          src: 'https://static.cloudflareinsights.com/beacon.min.js',
          defer: true,
          'data-cf-beacon': JSON.stringify({ token: beacon }),
        },
      ],
    });
  }
});
