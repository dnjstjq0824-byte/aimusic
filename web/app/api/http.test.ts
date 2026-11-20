import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { buildUrl, createHttpClient, HttpError, isHttpError } from './http';

function jsonResponse(status: number, body: unknown, headers: Record<string, string> = {}): Response {
  return new Response(body === undefined ? '' : JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json', ...headers },
  });
}

describe('buildUrl', () => {
  it('joins base and path and drops empty params', () => {
    expect(buildUrl('/api/v1', '/tracks', { limit: 10, q: undefined, genre: null, certified: false }))
      .toBe('/api/v1/tracks?limit=10&certified=false');
  });

  it('keeps absolute URLs and appends to an existing query', () => {
    expect(buildUrl('/api/v1', 'https://x.test/a?b=1', { c: 2 })).toBe('https://x.test/a?b=1&c=2');
  });

  it('serialises arrays like axios (key[]=a&key[]=b)', () => {
    expect(buildUrl('', '/a', { ids: ['x', 'y'] })).toBe('/a?ids%5B%5D=x&ids%5B%5D=y');
  });
});

describe('createHttpClient', () => {
  const fetchMock = vi.fn();
  beforeEach(() => {
    fetchMock.mockReset();
    vi.stubGlobal('fetch', fetchMock);
  });
  afterEach(() => vi.unstubAllGlobals());

  it('sends JSON bodies and returns the axios-shaped response', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { data: { ok: true } }, { 'X-Trace': 't1' }));
    const client = createHttpClient({ baseURL: '/api/v1' });
    const res = await client.post('/auth/login', { email: 'a@b.c' });

    const [url, init] = fetchMock.mock.calls[0]!;
    expect(url).toBe('/api/v1/auth/login');
    expect(init.method).toBe('POST');
    expect(init.body).toBe('{"email":"a@b.c"}');
    expect(init.headers['Content-Type']).toBe('application/json');
    expect(res.status).toBe(200);
    expect(res.data).toEqual({ data: { ok: true } });
    expect(res.headers['x-trace']).toBe('t1');
  });

  it('passes FormData through untouched (browser sets the boundary)', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, {}));
    const fd = new FormData();
    fd.append('f', 'v');
    await createHttpClient().post('/upload', fd);
    const [, init] = fetchMock.mock.calls[0]!;
    expect(init.body).toBe(fd);
    expect(init.headers['Content-Type']).toBeUndefined();
  });

  it('returns blobs for responseType blob', async () => {
    fetchMock.mockResolvedValue(new Response('csv,data', { status: 200, headers: { 'Content-Disposition': 'attachment; filename="k.csv"' } }));
    const res = await createHttpClient().get('/kit', { responseType: 'blob' });
    expect(res.data).toBeInstanceOf(Blob);
    expect(await (res.data as Blob).text()).toBe('csv,data');
    expect(res.headers['content-disposition']).toContain('k.csv');
  });

  it('throws an axios-compatible error carrying the backend body', async () => {
    fetchMock.mockResolvedValue(jsonResponse(400, { error: { code: 'ERR_INVALID_INPUT', message: 'bad' } }));
    const err = await createHttpClient().get('/x').catch((e: unknown) => e);
    expect(isHttpError(err)).toBe(true);
    expect((err as HttpError).isAxiosError).toBe(true);
    expect((err as HttpError).response?.status).toBe(400);
    expect((err as HttpError<{ error: { code: string } }>).response?.data.error.code).toBe('ERR_INVALID_INPUT');
  });

  it('reports network failures without a response', async () => {
    fetchMock.mockRejectedValue(new TypeError('Failed to fetch'));
    const err = (await createHttpClient().get('/x').catch((e: unknown) => e)) as HttpError;
    expect(err.code).toBe('ERR_NETWORK');
    expect(err.response).toBeUndefined();
  });

  it('aborts after the timeout', async () => {
    fetchMock.mockImplementation((_url: string, init: RequestInit) => new Promise((_, reject) => {
      init.signal?.addEventListener('abort', () => reject(new DOMException('aborted', 'AbortError')));
    }));
    const err = (await createHttpClient({ timeout: 10 }).get('/slow').catch((e: unknown) => e)) as HttpError;
    expect(err.code).toBe('ECONNABORTED');
  });

  it('runs onRequest before every attempt and lets onError retry once', async () => {
    let token = 'old';
    fetchMock
      .mockResolvedValueOnce(jsonResponse(401, { error: { code: 'ERR_UNAUTHORIZED' } }))
      .mockResolvedValueOnce(jsonResponse(200, { data: 'ok' }));
    const client = createHttpClient({
      onRequest(config) {
        (config.headers ??= {}).Authorization = `Bearer ${token}`;
      },
      async onError(err, c) {
        if (err.response?.status === 401 && !err.config._retried) {
          err.config._retried = true;
          token = 'new';
          return c.request(err.config);
        }
        throw err;
      },
    });
    const res = await client.get('/me');
    expect(res.data).toEqual({ data: 'ok' });
    expect(fetchMock.mock.calls[0]![1].headers.Authorization).toBe('Bearer old');
    expect(fetchMock.mock.calls[1]![1].headers.Authorization).toBe('Bearer new');
  });
});
