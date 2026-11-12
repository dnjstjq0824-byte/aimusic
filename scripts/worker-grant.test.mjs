// node --test scripts/worker-grant.test.mjs  (B-04: quality-bound HLS grants;
// WI-D1: R2 binding reads with the same signature gate)
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createHmac } from 'node:crypto';
import handler, { fileAllowed, parseHlsPath, rewritePlaylistUris } from '../worker.js';

test('SQ grant only reaches the 128k variant', () => {
  assert.equal(fileAllowed('SQ', 'stream_0.m3u8'), true);
  assert.equal(fileAllowed('SQ', 'seg_0_001.ts'), true);
  assert.equal(fileAllowed('SQ', 'peaks.json'), true);
  assert.equal(fileAllowed('SQ', 'master.m3u8'), false);
  assert.equal(fileAllowed('SQ', 'stream_1.m3u8'), false);
  assert.equal(fileAllowed('SQ', 'seg_1_001.ts'), false);
});

test('HQ grant reaches every file; unknown tier reaches none', () => {
  assert.equal(fileAllowed('HQ', 'master.m3u8'), true);
  assert.equal(fileAllowed('HQ', 'seg_1_004.ts'), true);
  assert.equal(fileAllowed('', 'stream_0.m3u8'), false);
  assert.equal(fileAllowed('XQ', 'stream_0.m3u8'), false);
});

// ---- WI-D1: R2 binding reads ----

const SIGNING_KEY = 'test-hls-signing-key-64-chars-minimum-00000000000000000000';
const TRACK = 'track-0001';
const JTI = 'jti-0001';

function sign(trackId, jti, exp, q) {
  return createHmac('sha256', SIGNING_KEY).update(`${trackId}|${jti}|${exp}|${q}`).digest('hex');
}

function grantUrl(file, q, overrides = {}) {
  const exp = overrides.exp ?? Math.floor(Date.now() / 1000) + 300;
  const sig = overrides.sig ?? sign(TRACK, JTI, exp, q);
  return `https://avibemusic.com/local-hls/hls/${TRACK}/${file}?exp=${exp}&jti=${JTI}&q=${q}&sig=${sig}`;
}

function toStream(bytes) {
  return new ReadableStream({
    start(c) {
      c.enqueue(bytes);
      c.close();
    },
  });
}

// Minimal R2Bucket double: get/head with range support.
function mockBucket(entries) {
  const store = new Map(entries);
  const wrap = (key, o, sliced, range) => ({
    key,
    size: o.bytes.length,
    etag: o.etag,
    httpEtag: `"${o.etag}"`,
    uploaded: o.uploaded,
    httpMetadata: { contentType: o.contentType },
    range,
    body: toStream(sliced),
    async arrayBuffer() {
      return sliced.buffer.slice(sliced.byteOffset, sliced.byteOffset + sliced.byteLength);
    },
    async text() {
      return new TextDecoder().decode(sliced);
    },
  });
  return {
    async get(key, opts = {}) {
      const o = store.get(key);
      if (!o) return null;
      if (opts.range) {
        const offset = opts.range.offset ?? 0;
        const length = opts.range.length ?? o.bytes.length - offset;
        const sliced = o.bytes.slice(offset, offset + length);
        return wrap(key, o, sliced, { offset, length: sliced.length });
      }
      return wrap(key, o, o.bytes, null);
    },
    async head(key) {
      const o = store.get(key);
      if (!o) return null;
      return {
        key,
        size: o.bytes.length,
        etag: o.etag,
        httpEtag: `"${o.etag}"`,
        uploaded: o.uploaded,
        httpMetadata: { contentType: o.contentType },
      };
    },
  };
}

const SEG0 = new Uint8Array(1024).map((_, i) => i % 251);
const MASTER = '#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=128000\nstream_0.m3u8\n#EXT-X-STREAM-INF:BANDWIDTH=256000\nstream_1.m3u8\n';
const UPLOADED = new Date('2026-11-12T00:00:00.000Z');

function testEnv() {
  return {
    HLS_SIGNING_KEY: SIGNING_KEY,
    ALLOW_LEGACY_GRANTS: 'false',
    HLS_BUCKET: mockBucket([
      [`hls/${TRACK}/master.m3u8`, { bytes: new TextEncoder().encode(MASTER), etag: 'etag-master', uploaded: UPLOADED, contentType: 'application/vnd.apple.mpegurl' }],
      [`hls/${TRACK}/seg_0_001.ts`, { bytes: SEG0, etag: 'etag-seg0', uploaded: UPLOADED, contentType: 'video/mp2t' }],
    ]),
  };
}

test('parseHlsPath accepts hls/<track>/<file> only', () => {
  assert.deepEqual(parseHlsPath('/local-hls/hls/abc/master.m3u8'), { key: 'hls/abc/master.m3u8', trackId: 'abc', file: 'master.m3u8' });
  assert.equal(parseHlsPath('/local-hls/master.m3u8'), null);
  assert.equal(parseHlsPath('/local-hls/other/abc/master.m3u8'), null);
  assert.equal(parseHlsPath('/api/v1/tracks'), null);
});

test('rewritePlaylistUris embeds the grant in media lines only', () => {
  const out = rewritePlaylistUris(MASTER, 'exp=1&jti=2&q=HQ&sig=3').split('\n');
  assert.equal(out[0], '#EXTM3U');
  assert.equal(out[2], 'stream_0.m3u8?exp=1&jti=2&q=HQ&sig=3');
  assert.equal(out[4], 'stream_1.m3u8?exp=1&jti=2&q=HQ&sig=3');
  const abs = rewritePlaylistUris('https://cdn.example.com/a.ts\n', 'exp=1');
  assert.equal(abs, 'https://cdn.example.com/a.ts\n');
});

test('HQ grant fetches master.m3u8 from R2 with embedded grant', async () => {
  const res = await handler.fetch(new Request(grantUrl('master.m3u8', 'HQ')), testEnv());
  assert.equal(res.status, 200);
  const text = await res.text();
  assert.match(text, /stream_0\.m3u8\?exp=\d+&jti=jti-0001&q=HQ&sig=[0-9a-f]{64}/);
  assert.match(text, /stream_1\.m3u8\?exp=\d+&jti=jti-0001&q=HQ&sig=[0-9a-f]{64}/);
  assert.equal(res.headers.get('cache-control'), 'private, max-age=60');
});

test('SQ grant fetches SQ segment bytes with immutable cache', async () => {
  const res = await handler.fetch(new Request(grantUrl('seg_0_001.ts', 'SQ')), testEnv());
  assert.equal(res.status, 200);
  assert.equal(res.headers.get('cache-control'), 'public, max-age=31536000, immutable');
  const buf = new Uint8Array(await res.arrayBuffer());
  assert.equal(buf.length, 1024);
  assert.deepEqual(buf.slice(0, 4), SEG0.slice(0, 4));
});

test('SQ grant for master.m3u8 is rejected', async () => {
  const res = await handler.fetch(new Request(grantUrl('master.m3u8', 'SQ')), testEnv());
  assert.equal(res.status, 403);
});

test('bad signature, expired grant, and missing tier are rejected', async () => {
  const bad = await handler.fetch(new Request(grantUrl('seg_0_001.ts', 'SQ', { sig: '0'.repeat(64) })), testEnv());
  assert.equal(bad.status, 403);
  const old = await handler.fetch(new Request(grantUrl('seg_0_001.ts', 'SQ', { exp: 1000 })), testEnv());
  assert.equal(old.status, 403);
  const noq = await handler.fetch(new Request(`https://avibemusic.com/local-hls/hls/${TRACK}/seg_0_001.ts?exp=9999999999&jti=${JTI}&sig=${'0'.repeat(64)}`), testEnv());
  assert.equal(noq.status, 403);
});

test('missing R2 object is 404', async () => {
  const res = await handler.fetch(new Request(grantUrl('seg_0_009.ts', 'SQ')), testEnv());
  assert.equal(res.status, 404);
});

test('missing HLS_BUCKET binding is 502 without touching R2', async () => {
  const env = testEnv();
  delete env.HLS_BUCKET;
  const res = await handler.fetch(new Request(grantUrl('seg_0_001.ts', 'SQ')), env);
  assert.equal(res.status, 502);
  assert.match(await res.text(), /HLS_BUCKET/);
});

test('range request returns 206 with content-range', async () => {
  const req = new Request(grantUrl('seg_0_001.ts', 'SQ'), { headers: { range: 'bytes=0-99' } });
  const res = await handler.fetch(req, testEnv());
  assert.equal(res.status, 206);
  assert.equal(res.headers.get('content-range'), 'bytes 0-99/1024');
  assert.equal((await res.arrayBuffer()).byteLength, 100);
});

test('HEAD returns headers with an empty body', async () => {
  const res = await handler.fetch(new Request(grantUrl('seg_0_001.ts', 'SQ'), { method: 'HEAD' }), testEnv());
  assert.equal(res.status, 200);
  assert.equal(res.headers.get('content-length'), '1024');
  assert.equal((await res.arrayBuffer()).byteLength, 0);
});

test('unsatisfiable range is 416', async () => {
  const req = new Request(grantUrl('seg_0_001.ts', 'SQ'), { headers: { range: 'bytes=9999-' } });
  const res = await handler.fetch(req, testEnv());
  assert.equal(res.status, 416);
});

test('matching if-none-match is 304 without a body', async () => {
  const req = new Request(grantUrl('seg_0_001.ts', 'SQ'), { headers: { 'if-none-match': '"etag-seg0"' } });
  const res = await handler.fetch(req, testEnv());
  assert.equal(res.status, 304);
  assert.equal((await res.arrayBuffer()).byteLength, 0);
});
