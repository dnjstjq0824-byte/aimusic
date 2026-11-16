// R13: writes the static harness matrix (route x role x viewport x lang table).
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { VIEWPORTS, MATRIX, LANGS } from '../web/e2e/screens/config.mjs';

const DIR = path.dirname(fileURLToPath(import.meta.url));
const OUT = path.join(DIR, '..', 'docs', 'handoff', '20261114', 'evidence', 'r13', 'harness-matrix.txt');
const withLang = (p, lang) => {
  if (lang === 'ko') return p;
  const qi = p.indexOf('?');
  const pathOnly = qi === -1 ? p : p.slice(0, qi);
  const q = qi === -1 ? '' : p.slice(qi);
  return `/${lang}${pathOnly === '/' ? '' : pathOnly}${q}`;
};
const lines = ['viewport role lang route url screenshot'];
for (const vk of Object.keys(VIEWPORTS)) {
  for (const [role, routes] of Object.entries(MATRIX)) {
    for (const lang of LANGS) {
      for (const [name, raw] of routes) {
        lines.push(`${vk} ${role} ${lang} ${name} ${withLang(raw, lang)} ${vk}-${role}-${lang}-${name}.png`);
      }
    }
  }
}
fs.writeFileSync(OUT, lines.join('\n') + '\n');
console.log(`matrix: ${lines.length - 1} rows -> evidence/r13/harness-matrix.txt`);
