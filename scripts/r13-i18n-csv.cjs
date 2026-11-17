// R13: builds docs/handoff/20261114/i18n-review/{ja,zh-CN,th}.csv
// Columns: key, ko (source), translation, screenshot (harness shot path).
// ja/zh-CN/th are machine-draft (Beta, G-L) until native reviewers sign off.
const fs = require('node:fs');
const path = require('node:path');

const ROOT = path.join(__dirname, '..');
const LOCALES = path.join(ROOT, 'web', 'i18n', 'locales');
const OUT = path.join(ROOT, 'docs', 'handoff', '20261114', 'i18n-review');

// namespace (2 levels max) -> [role, route] owning the primary screenshot.
const SHOT = [
  [/^account\./, ['listener', 'account']],
  [/^auth\.ageGate/, ['guest', 'login']],
  [/^auth\./, ['guest', 'login']],
  [/^business\./, ['storePartner', 'console']],
  [/^checkout\./, ['listener', 'checkout']],
  [/^common\.httpError/, ['guest', 'server-error']],
  [/^common\.paginator/, ['guest', 'explore']],
  [/^common\./, ['guest', 'home']],
  [/^errors\./, ['guest', 'login']],
  [/^explore\./, ['guest', 'explore']],
  [/^generate\./, ['creator', 'studio-create']],
  [/^home\./, ['guest', 'home']],
  [/^legal\.cert/, ['guest', 'cert']],
  [/^legal\.precheck/, ['creator', 'studio-upload']],
  [/^legal\.aiBadge/, ['listener', 'track']],
  [/^legal\.consentGate/, ['listener', 'account']],
  [/^legal\.rc/, ['listener', 'rights-center']],
  [/^legal\.md/, ['listener', 'track']],
  [/^legal\./, ['guest', 'legal']],
  [/^library\./, ['listener', 'library']],
  [/^player\./, ['listener', 'track']],
  [/^playlist\.picker/, ['listener', 'track']],
  [/^playlist\./, ['listener', 'me-playlists']],
  [/^pricing\./, ['guest', 'pricing']],
  [/^seo\./, ['guest', 'home']],
  [/^studio\.upload/, ['creator', 'studio-upload']],
  [/^studio\.create/, ['creator', 'studio-create']],
  [/^studio\.catalog/, ['creator', 'studio-catalog']],
  [/^studio\.(status|genStatus|genType)/, ['creator', 'studio-catalog']],
  [/^studio\./, ['creator', 'studio']],
  [/^tags\./, ['guest', 'explore']],
  [/^track\./, ['guest', 'track']],
];

const flat = (o, p = '', acc = {}) => {
  for (const [k, v] of Object.entries(o)) {
    const key = p ? `${p}.${k}` : k;
    if (v && typeof v === 'object' && !Array.isArray(v)) flat(v, key, acc);
    else acc[key] = v;
  }
  return acc;
};
const deepMerge = (a, b) => {
  const o = { ...a };
  for (const [k, v] of Object.entries(b)) {
    o[k] = v && typeof v === 'object' && !Array.isArray(v) && a[k] && typeof a[k] === 'object' ? deepMerge(a[k], v) : v;
  }
  return o;
};
const merged = (loc) => {
  let m = {};
  for (const f of fs.readdirSync(path.join(LOCALES, loc)).filter((x) => x.endsWith('.json'))) {
    m = deepMerge(m, JSON.parse(fs.readFileSync(path.join(LOCALES, loc, f), 'utf8')));
  }
  return flat(m);
};
const shotFor = (key, lang) => {
  for (const [re, [role, route]] of SHOT) {
    if (re.test(key)) return `evidence/r13/harness/d1440-${role}-${lang}-${route}.png`;
  }
  return `evidence/r13/harness/d1440-guest-${lang}-home.png`;
};
const csvCell = (v) => {
  const s = String(v ?? '');
  return /[",\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
};

fs.mkdirSync(OUT, { recursive: true });
const ko = merged('ko');
const targets = [
  ['ja', 'ja'],
  ['zh-CN', 'zh-cn'],
  ['th', 'th'],
];
for (const [dir, harnessLang] of targets) {
  const tr = merged(dir);
  const keys = [...new Set([...Object.keys(ko), ...Object.keys(tr)])]
    .filter((k) => !(k.startsWith('admin.') && dir !== 'ko' && dir !== 'en'))
    .sort();
  const rows = ['key,ko,translation,screenshot'];
  for (const k of keys) {
    rows.push([k, csvCell(ko[k]), csvCell(tr[k]), shotFor(k, harnessLang)].join(','));
  }
  fs.writeFileSync(path.join(OUT, `${dir}.csv`), rows.join('\n') + '\n');
  console.log(`${dir}.csv: ${keys.length} rows`);
}
