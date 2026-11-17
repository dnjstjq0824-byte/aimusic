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
      numberOfRuns: 1,
      settings: {
        formFactor: 'mobile',
        throttlingMethod: 'simulate',
      },
    },
    assert: {
      assertions: {
        'categories:performance': ['error', { minScore: 0.9 }],
        'categories:seo': ['error', { minScore: 1 }],
        'categories:accessibility': ['warn', { minScore: 0.9 }],
        'resource-summary:script:size': ['error', { maxNumericValue: 174080 }],
      },
    },
    upload: {
      target: 'temporary-public-storage',
    },
  },
};
