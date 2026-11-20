// Minimal axios-shaped HTTP client on top of fetch (bundle diet, 2026-11-21).
// axios cost ~25KB gzip of the initial JS on every page; the app only uses
// get/post/put/patch/delete with JSON, query params, blob responses, a
// timeout and two interceptors (auth header, 401 refresh). Responses keep the
// axios shape ({ data, status, headers }) and errors keep `isAxiosError` +
// `response.{status,data}`, so endpoints.ts and error helpers are unchanged.

export type ResponseType = 'json' | 'blob' | 'text' | 'arraybuffer';

export interface HttpRequestConfig {
  url?: string;
  method?: string;
  baseURL?: string;
  params?: Record<string, unknown>;
  headers?: Record<string, string>;
  data?: unknown;
  responseType?: ResponseType;
  timeout?: number;
  signal?: AbortSignal;
  /** Set by the 401 refresh flow so a request is retried at most once. */
  _retried?: boolean;
}

export interface HttpResponse<T = any> {
  data: T;
  status: number;
  headers: Record<string, string>;
  config: HttpRequestConfig;
}

export class HttpError<T = unknown> extends Error {
  /** Kept for compatibility with code (and tests) that check axios errors. */
  readonly isAxiosError = true;
  code?: string;
  config: HttpRequestConfig;
  response?: { status: number; data: T; headers: Record<string, string> };

  constructor(message: string, config: HttpRequestConfig, code?: string, response?: HttpError<T>['response']) {
    super(message);
    this.name = 'HttpError';
    this.config = config;
    this.code = code;
    this.response = response;
  }
}

export function isHttpError<T = unknown>(err: unknown): err is HttpError<T> {
  return typeof err === 'object' && err !== null && (err as { isAxiosError?: unknown }).isAxiosError === true;
}

type RequestHook = (config: HttpRequestConfig) => HttpRequestConfig | void;
type ErrorHook = (err: HttpError, client: HttpClient) => Promise<HttpResponse>;

export interface HttpClient {
  request<T = any>(config: HttpRequestConfig): Promise<HttpResponse<T>>;
  get<T = any>(url: string, config?: HttpRequestConfig): Promise<HttpResponse<T>>;
  delete<T = any>(url: string, config?: HttpRequestConfig): Promise<HttpResponse<T>>;
  post<T = any>(url: string, data?: unknown, config?: HttpRequestConfig): Promise<HttpResponse<T>>;
  put<T = any>(url: string, data?: unknown, config?: HttpRequestConfig): Promise<HttpResponse<T>>;
  patch<T = any>(url: string, data?: unknown, config?: HttpRequestConfig): Promise<HttpResponse<T>>;
}

export interface CreateHttpClientOptions {
  baseURL?: string;
  timeout?: number;
  /** Runs before every request (e.g. attach the Authorization header). */
  onRequest?: RequestHook;
  /** Runs on every failed request; resolve to recover, rethrow to fail. */
  onError?: ErrorHook;
}

export function buildUrl(baseURL: string, url: string, params?: Record<string, unknown>): string {
  const full = /^https?:\/\//i.test(url) ? url : `${baseURL.replace(/\/$/, '')}/${url.replace(/^\//, '')}`;
  if (!params) return full;
  const qs = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === null) continue;
    if (Array.isArray(v)) v.forEach((item) => qs.append(`${k}[]`, String(item)));
    else qs.append(k, v instanceof Date ? v.toISOString() : String(v));
  }
  const q = qs.toString();
  return q ? `${full}${full.includes('?') ? '&' : '?'}${q}` : full;
}

function isRawBody(data: unknown): data is BodyInit {
  return (
    (typeof FormData !== 'undefined' && data instanceof FormData) ||
    (typeof Blob !== 'undefined' && data instanceof Blob) ||
    (typeof URLSearchParams !== 'undefined' && data instanceof URLSearchParams) ||
    data instanceof ArrayBuffer ||
    typeof data === 'string'
  );
}

async function parseBody(res: Response, type: ResponseType): Promise<unknown> {
  if (type === 'blob') return res.blob();
  if (type === 'arraybuffer') return res.arrayBuffer();
  const text = await res.text();
  if (type === 'text') return text;
  if (!text) return '';
  try {
    return JSON.parse(text);
  } catch {
    return text; // axios also falls back to the raw string
  }
}

export function createHttpClient(opts: CreateHttpClientOptions = {}): HttpClient {
  async function send<T>(input: HttpRequestConfig): Promise<HttpResponse<T>> {
    const config: HttpRequestConfig = { ...input, headers: { ...(input.headers ?? {}) } };
    opts.onRequest?.(config);

    const method = (config.method ?? 'GET').toUpperCase();
    const url = buildUrl(config.baseURL ?? opts.baseURL ?? '', config.url ?? '', config.params);
    const headers = config.headers as Record<string, string>;
    let body: BodyInit | undefined;
    if (config.data !== undefined && method !== 'GET' && method !== 'HEAD') {
      if (isRawBody(config.data)) {
        body = config.data;
      } else {
        body = JSON.stringify(config.data);
        if (!Object.keys(headers).some((h) => h.toLowerCase() === 'content-type')) {
          headers['Content-Type'] = 'application/json';
        }
      }
    }
    if (!Object.keys(headers).some((h) => h.toLowerCase() === 'accept')) {
      headers.Accept = 'application/json, text/plain, */*';
    }

    const ctrl = new AbortController();
    const timeout = config.timeout ?? opts.timeout ?? 0;
    let timedOut = false;
    const timer = timeout > 0 ? setTimeout(() => { timedOut = true; ctrl.abort(); }, timeout) : undefined;
    const onAbort = () => ctrl.abort();
    config.signal?.addEventListener('abort', onAbort, { once: true });

    let res: Response;
    try {
      res = await fetch(url, { method, headers, body, signal: ctrl.signal });
    } catch (e) {
      const code = timedOut ? 'ECONNABORTED' : config.signal?.aborted ? 'ERR_CANCELED' : 'ERR_NETWORK';
      throw new HttpError(timedOut ? `timeout of ${timeout}ms exceeded` : (e as Error)?.message || 'Network Error', config, code);
    } finally {
      if (timer) clearTimeout(timer);
      config.signal?.removeEventListener('abort', onAbort);
    }

    const resHeaders: Record<string, string> = {};
    res.headers.forEach((v, k) => { resHeaders[k.toLowerCase()] = v; });
    const ok = res.status >= 200 && res.status < 300;
    // Error bodies are always read as JSON/text so error codes stay readable
    // even for blob downloads.
    const data = await parseBody(res, ok ? (config.responseType ?? 'json') : 'json');
    if (!ok) {
      throw new HttpError(`Request failed with status code ${res.status}`, config,
        res.status >= 500 ? 'ERR_BAD_RESPONSE' : 'ERR_BAD_REQUEST',
        { status: res.status, data, headers: resHeaders });
    }
    return { data: data as T, status: res.status, headers: resHeaders, config };
  }

  const client: HttpClient = {
    async request<T>(config: HttpRequestConfig) {
      try {
        return await send<T>(config);
      } catch (e) {
        if (opts.onError && isHttpError(e)) return (await opts.onError(e, client)) as HttpResponse<T>;
        throw e;
      }
    },
    get: (url, config) => client.request({ ...config, url, method: 'GET' }),
    delete: (url, config) => client.request({ ...config, url, method: 'DELETE' }),
    post: (url, data, config) => client.request({ ...config, url, data, method: 'POST' }),
    put: (url, data, config) => client.request({ ...config, url, data, method: 'PUT' }),
    patch: (url, data, config) => client.request({ ...config, url, data, method: 'PATCH' }),
  };
  return client;
}
