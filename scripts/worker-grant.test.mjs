// node --test scripts/worker-grant.test.mjs  (B-04: quality-bound HLS grants)
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { fileAllowed } from '../worker.js';

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
