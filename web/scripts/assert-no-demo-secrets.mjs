// Post-build guard (P1): the dev-only demo password must never reach
// production output. login.vue keeps it behind import.meta.env.DEV so the
// bundler drops it; this fails the build if it ever leaks back in.
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = fileURLToPath(new URL('../.output/', import.meta.url));
const NEEDLE = 'Admin1234!';

function* walk(dir) {
  for (const e of readdirSync(dir)) {
    const p = join(dir, e);
    if (statSync(p).isDirectory()) yield* walk(p);
    else yield p;
  }
}

const hits = [];
for (const f of walk(ROOT)) {
  let s;
  try {
    s = readFileSync(f, 'utf8');
  } catch {
    continue;
  }
  if (s.includes(NEEDLE)) hits.push(f);
}
if (hits.length > 0) {
  console.error(`demo password leaked into ${hits.length} build file(s):`);
  for (const h of hits.slice(0, 10)) console.error(`  ${h}`);
  process.exit(1);
}
console.log('assert-no-demo-secrets: clean');
