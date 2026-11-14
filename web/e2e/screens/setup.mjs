// Provisions 5 harness roles on the LOCAL stack and resolves dynamic route ids.
// Writes web/e2e/screens/.auth.json (gitignored, local-only tokens). Never run against prod.
import fs from 'node:fs';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { API } from './config.mjs';

const DIR = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(DIR, '../../..');
const OUT = path.join(DIR, '.auth.json');
const PW = 'Harness123!x';
const STAMP = Date.now().toString(36);

async function api(method, p, token, body) {
  const res = await fetch(API + p, {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  const json = await res.json().catch(() => ({}));
  return { status: res.status, json };
}

function psql(sql) {
  return execFileSync(
    'docker',
    ['compose', '-f', 'docker-compose.local.yml', 'exec', '-T', 'postgres', 'psql', '-U', 'avibe', '-d', 'avibe', '-At', '-c', sql],
    { cwd: ROOT, encoding: 'utf8' },
  ).trim();
}

async function clearGates(token) {
  await api('POST', '/auth/age-confirm', token, { ageConfirmed: true, birthYear: 1990 });
  const pend = await api('GET', '/legal/consents/pending', token);
  const items = pend.json?.data ?? [];
  for (const t of items) await api('POST', '/legal/consents', token, { termsId: t.id });
}

async function registerListener(tag) {
  const email = `harness-${STAMP}-${tag}@local.test`;
  const r = await api('POST', '/auth/register', null, {
    email, password: PW, displayName: `harness-${tag}`, ageConfirmed: true,
  });
  if (r.status !== 201) throw new Error(`register ${tag}: ${r.status} ${JSON.stringify(r.json)}`);
  const l = await api('POST', '/auth/login', null, { email, password: PW });
  if (l.status !== 200) throw new Error(`login ${tag}: ${l.status}`);
  await clearGates(l.json.data.tokens.accessToken);
  return { email, id: l.json.data.user?.id ?? r.json.data?.id, tokens: l.json.data.tokens };
}

async function registerPartner(tag) {
  const email = `harness-${STAMP}-${tag}@local.test`;
  const r = await api('POST', '/auth/register/partner', null, {
    email, password: PW, displayName: `harness-${tag}`, company: `Harness ${tag}`, ageConfirmed: true,
  });
  if (r.status !== 201) throw new Error(`register ${tag}: ${r.status} ${JSON.stringify(r.json)}`);
  const l = await api('POST', '/auth/login', null, { email, password: PW });
  if (l.status !== 200) throw new Error(`login ${tag}: ${l.status}`);
  await clearGates(l.json.data.tokens.accessToken);
  return { email, id: l.json.data.user?.id ?? r.json.data?.id, tokens: l.json.data.tokens };
}

function genBizno() {
  const w = [1, 3, 7, 1, 3, 7, 1, 3];
  const d = [1, 2, ...Array.from({ length: 7 }, () => Math.floor(Math.random() * 10))];
  let s = d.slice(0, 8).reduce((acc, v, i) => acc + v * w[i], 0);
  const p = d[8] * 5;
  s += Math.floor(p / 10) + (p % 10);
  d.push((10 - (s % 10)) % 10);
  return d.join('');
}

const listener = await registerListener('listener');
const creator = await registerPartner('creator');
const storePartner = await registerPartner('store');
const st = await api('POST', '/partner/stores', storePartner.tokens.accessToken, {
  businessNumber: genBizno(), storeName: 'Harness Store',
});
if (![200, 201].includes(st.status)) throw new Error(`store create: ${st.status} ${JSON.stringify(st.json)}`);
const admin = await registerListener('admin');
psql(`UPDATE users SET role='ADMIN' WHERE email='${admin.email}';`);
const relogin = await api('POST', '/auth/login', null, { email: admin.email, password: PW });
admin.tokens = relogin.json.data.tokens;

// Resolve dynamic ids from real local data.
const tracks = await api('GET', '/tracks?limit=50', listener.tokens.accessToken);
const items = tracks.json?.data?.items ?? tracks.json?.data ?? [];
const ready = items.find((t) => t.status === 'READY') ?? items[0];
if (!ready) throw new Error('no tracks in local DB');
const pl = await api('POST', '/playlists', listener.tokens.accessToken, { title: `Harness Pub ${STAMP}`, isPublic: true });
if (![200, 201].includes(pl.status)) throw new Error(`playlist create: ${pl.status}`);
const cert = psql('SELECT cert_number FROM track_certificates LIMIT 1;');

const auth = {
  createdAt: new Date().toISOString(),
  roles: {
    listener: { email: listener.email, ...listener.tokens },
    creator: { email: creator.email, ...creator.tokens },
    storePartner: { email: storePartner.email, ...storePartner.tokens },
    admin: { email: admin.email, ...admin.tokens },
  },
  ids: { track: ready.id, playlist: pl.json.data.id, cert: cert || 'AVB-0000' },
};
fs.writeFileSync(OUT, JSON.stringify(auth, null, 1));
console.log(`auth written: 4 roles, track=${auth.ids.track.slice(0, 8)} playlist=${auth.ids.playlist.slice(0, 8)} cert=${auth.ids.cert}`);
