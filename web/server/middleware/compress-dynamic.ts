// CI-only dynamic compression (WI-N1, item 1).
//
// `nuxt preview` (node-server) serves HTML and `/_i18n/.../messages.json`
// uncompressed, while prod is compressed by Cloudflare — so Lighthouse
// measured ~105KB raw that prod never ships. This middleware gzips dynamic
// responses when NUXT_COMPRESS_PUBLIC_ASSETS=1 (the CI Lighthouse build
// only); the shipped image is unchanged (the flag defaults to off).
//
// h3 v1 has no onResponse hook, so this wraps node's res.write/end for
// compressible content types. Pre-compressed static assets (already have
// content-encoding) pass through untouched.
import { createBrotliCompress, createGzip } from 'node:zlib';

const COMPRESSIBLE = /^(text\/html|text\/css|text\/javascript|application\/javascript|application\/json|image\/svg\+xml)/i;

export default defineEventHandler((event) => {
  // Build-time flag (nuxt.config runtimeConfig.compressDynamic): on for the
  // CI Lighthouse build only, so `preview` needs no extra env.
  if (!useRuntimeConfig(event).compressDynamic) return;
  const res = event.node.res;
  const req = event.node.req;
  const accept = req.headers['accept-encoding'] ?? '';
  const useBr = /\bbr\b/.test(accept);
  const useGzip = !useBr && /\bgzip\b/.test(accept);
  if (!useBr && !useGzip) return;

  const origWrite = res.write.bind(res);
  const origEnd = res.end.bind(res);
  let compressor: ReturnType<typeof createGzip> | ReturnType<typeof createBrotliCompress> | null = null;
  let started = false;

  const maybeStart = (chunk: unknown): boolean => {
    if (started) return true;
    // Static assets already carry content-encoding (precompressed .gz/.br).
    if (res.getHeader('content-encoding')) return false;
    const type = String(res.getHeader('content-type') ?? '');
    if (!COMPRESSIBLE.test(type)) return false;
    // HTML streams without content-type until first write; sniff it.
    started = true;
    res.removeHeader('content-length');
    if (useBr) {
      res.setHeader('content-encoding', 'br');
      compressor = createBrotliCompress();
    } else {
      res.setHeader('content-encoding', 'gzip');
      compressor = createGzip();
    }
    compressor.on('data', (buf: Buffer) => origWrite(buf));
    compressor.on('end', () => origEnd());
    void chunk;
    return true;
  };

  res.write = ((chunk: unknown, ...args: unknown[]) => {
    if (!maybeStart(chunk)) return origWrite(chunk as never, ...(args as []));
    (compressor as NonNullable<typeof compressor>).write(chunk as never);
    const cb = args.find((a) => typeof a === 'function') as (() => void) | undefined;
    cb?.();
    return true;
  }) as typeof res.write;

  res.end = ((chunk?: unknown, ...args: unknown[]) => {
    if (!started) {
      // Buffer tiny bodies to sniff content-type set just before end().
      if (!maybeStart(chunk)) return origEnd(chunk as never, ...(args as []));
    }
    if (chunk !== undefined && chunk !== null) {
      (compressor as NonNullable<typeof compressor>).write(chunk as never);
    }
    (compressor as NonNullable<typeof compressor>).end();
    const cb = args.find((a) => typeof a === 'function') as (() => void) | undefined;
    cb?.();
    return res;
  }) as typeof res.end;
});
