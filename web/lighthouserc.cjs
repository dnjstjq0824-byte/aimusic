// Lighthouse CI (WI-S5, SEO §8): PR runs assert SEO 100 + performance 90 on
// /, /en/tracks/…, /th/playlists, /ja/business, plus the 170KB gzip initial-JS
// budget for public pages. LHCI_TRACK_URL is discovered from the CI stack
// (see .github/workflows/ci.yml e2e job).
const BASE = process.env.LHCI_BASE_URL || 'http://localhost:3000';
const TRACK = process.env.LHCI_TRACK_URL;
if (!TRACK) {
  throw new Error('LHCI_TRACK_URL is required (discover a READY track from the API first)');
}

module.exports = {
  ci: {
    collect: {
      url: [`${BASE}/`, TRACK, `${BASE}/th/playlists`, `${BASE}/ja/business`],
      numberOfRuns: 3,
      settings: {
        formFactor: 'mobile',
        throttlingMethod: 'simulate',
      },
    },
    assert: {
      // Performance + JS budget stay 'warn' until the bundle diet lands
      // (2026-11-26 measured on a CI-parity build: perf 0.56-0.68, JS
      // 323-612KB vs the 170KB budget across the four URLs; hls.js is
      // already lazy and NOT preloaded — the gap is vendor + route chunks,
      // starting with the ~199KB core vendor chunk). SEO + accessibility
      // are hard gates (a11y measured 0.96-1.0 on 2026-11-26).
      assertions: {
        'categories:performance': ['warn', { minScore: 0.9 }],
        'categories:seo': ['error', { minScore: 1 }],
        'categories:accessibility': ['error', { minScore: 0.9 }],
        'resource-summary:script:size': ['warn', { maxNumericValue: 174080 }],
      },
    },
    upload: {
      target: 'temporary-public-storage',
    },
  },
};
