export const ENV_BASE_URL = "NEWSLETTER_BASE_URL";
export const ENV_API_KEY = "NEWSLETTER_API_KEY";
export const API_PREFIX = "/api/v1";

export interface FetchRequestInit {
  method?: string;
  headers?: Record<string, string>;
  body?: string;
  signal?: unknown;
}

export interface FetchResponse {
  ok: boolean;
  status: number;
  text(): Promise<string>;
}

export type FetchLike = (input: string, init?: FetchRequestInit) => Promise<FetchResponse>;

export interface ClientConfig {
  baseUrl?: string;
  apiKey?: string;
  fetch?: FetchLike;
  headers?: Record<string, string>;
}

export interface MessageResponse {
  message?: string;
  error?: string;
}

export interface NewsletterSendRequest {
  subject: string;
  body: string;
}

export interface NewsletterSendOptions {
  apiKey?: string;
  idempotencyKey?: string;
  signal?: unknown;
}

export interface RequestOptions {
  signal?: unknown;
}

export interface NewsletterSendResponse {
  message: string;
  total: number;
}

export interface HealthCheck {
  status: string;
  error?: string;
}

export interface HealthResponse {
  status: string;
  checks: Record<string, HealthCheck>;
}

export class APIError extends Error {
  readonly statusCode: number;
  readonly body: string;
  readonly data?: unknown;

  constructor(statusCode: number, message: string, body: string, data?: unknown) {
    super(`newsletter sdk request failed with status ${statusCode}: ${message}`);
    this.name = "APIError";
    this.statusCode = statusCode;
    this.body = body;
    this.data = data;
  }
}

export class NewsletterClient {
  private readonly baseUrl: string;
  private readonly fetchImpl: FetchLike;
  private readonly defaultHeaders: Record<string, string>;
  private apiKey?: string;

  constructor(config: ClientConfig = {}) {
    const runtimeFetch = config.fetch ?? readGlobalFetch();

    if (!runtimeFetch) {
      throw new Error(
        "A fetch implementation is required. Pass one in config.fetch or use a runtime with global fetch."
      );
    }

    this.baseUrl = resolveBaseUrl(config.baseUrl);
    this.apiKey = resolveApiKey(config.apiKey);
    this.fetchImpl = runtimeFetch;
    this.defaultHeaders = { ...(config.headers ?? {}) };
  }

  static fromEnv(config: Omit<ClientConfig, "baseUrl" | "apiKey"> = {}): NewsletterClient {
    return new NewsletterClient(config);
  }

  setApiKey(apiKey: string): void {
    this.apiKey = apiKey;
  }

  async subscribe(email: string, options: RequestOptions = {}): Promise<MessageResponse> {
    if (!email.trim()) {
      throw new Error("email is required");
    }

    return (await this.request<MessageResponse>("/api/v1/subscribe", {
      method: "POST",
      body: JSON.stringify({ email }),
      headers: { "Content-Type": "application/json" },
      signal: options.signal,
    })) as MessageResponse;
  }

  async confirm(token: string, options: RequestOptions = {}): Promise<MessageResponse> {
    if (!token.trim()) {
      throw new Error("token is required");
    }

    return (await this.request<MessageResponse>(
      `/api/v1/confirm?token=${encodeURIComponent(token)}`,
      {
        method: "GET",
        signal: options.signal,
      }
    )) as MessageResponse;
  }

  async sendNewsletter(
    input: NewsletterSendRequest,
    options: NewsletterSendOptions = {}
  ): Promise<NewsletterSendResponse> {
    if (!input.subject.trim()) {
      throw new Error("subject is required");
    }
    if (!input.body.trim()) {
      throw new Error("body is required");
    }

    const apiKey = options.apiKey ?? this.apiKey;
    if (!apiKey?.trim()) {
      throw new Error("apiKey is required for newsletter sends");
    }

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      "X-API-Key": apiKey,
    };

    if (options.idempotencyKey) {
      headers["Idempotency-Key"] = options.idempotencyKey;
    }

    return (await this.request<NewsletterSendResponse>("/api/v1/newsletter/send", {
      method: "POST",
      body: JSON.stringify(input),
      headers,
      signal: options.signal,
    })) as NewsletterSendResponse;
  }

  async health(options: RequestOptions = {}): Promise<HealthResponse> {
    return (await this.request<HealthResponse>("/api/v1/health", {
      method: "GET",
      signal: options.signal,
    })) as HealthResponse;
  }

  async metrics(options: RequestOptions = {}): Promise<string> {
    return (await this.request<string>(
      "/api/v1/metrics",
      {
        method: "GET",
        signal: options.signal,
      },
      "text"
    )) as string;
  }

  private async request<T>(
    path: string,
    init: FetchRequestInit,
    responseType: "json" | "text" = "json"
  ): Promise<T | string> {
    const response = await this.fetchImpl(`${this.baseUrl}${path}`, {
      ...init,
      headers: {
        ...this.defaultHeaders,
        ...(init.headers ?? {}),
      },
    });

    const rawBody = await response.text();
    if (!response.ok) {
      throw buildApiError(response.status, rawBody);
    }

    if (responseType === "text") {
      return rawBody;
    }

    if (!rawBody.trim()) {
      return {} as T;
    }

    return JSON.parse(rawBody) as T;
  }
}

function buildApiError(statusCode: number, rawBody: string): APIError {
  let message = `request failed with status ${statusCode}`;
  let data: unknown;

  if (rawBody.trim()) {
    try {
      data = JSON.parse(rawBody);
      if (isRecord(data)) {
        message = readMessage(data) ?? message;
      }
    } catch {
      message = rawBody.trim();
    }
  }

  return new APIError(statusCode, message, rawBody, data);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function readMessage(data: Record<string, unknown>): string | undefined {
  const error = data["error"];
  if (typeof error === "string" && error.length > 0) {
    return error;
  }

  const message = data["message"];
  if (typeof message === "string" && message.length > 0) {
    return message;
  }

  return undefined;
}

function resolveBaseUrl(baseUrl?: string): string {
  const explicit = baseUrl?.trim();
  if (explicit) {
    return normalizeBaseUrl(explicit);
  }

  const envBaseUrl = readEnv(ENV_BASE_URL);
  if (envBaseUrl) {
    return normalizeBaseUrl(envBaseUrl);
  }

  const locationCandidate = readGlobalLocation();
  if (isLocationLike(locationCandidate)) {
    return normalizeBaseUrl(locationCandidate.origin);
  }

  throw new Error(
    "baseUrl is required. Pass it explicitly, set NEWSLETTER_BASE_URL, or run the SDK in a browser on the same origin as the API."
  );
}

function resolveApiKey(apiKey?: string): string | undefined {
  const explicit = apiKey?.trim();
  if (explicit) {
    return explicit;
  }

  const envApiKey = readEnv(ENV_API_KEY);
  return envApiKey || undefined;
}

function readEnv(name: string): string | undefined {
  const candidate = (globalThis as {
    process?: { env?: Record<string, string | undefined> };
  }).process?.env?.[name];

  if (typeof candidate === "string") {
    const trimmed = candidate.trim();
    return trimmed || undefined;
  }

  return undefined;
}

function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, "");
}

function normalizeBaseUrl(value: string): string {
  let normalized = trimTrailingSlash(value.trim());
  if (normalized.endsWith(API_PREFIX)) {
    normalized = trimTrailingSlash(normalized.slice(0, -API_PREFIX.length));
  }
  return normalized;
}

function readGlobalFetch(): FetchLike | undefined {
  const candidate = (globalThis as {
    fetch?: unknown;
  }).fetch;

  if (typeof candidate === "function") {
    return candidate.bind(globalThis) as FetchLike;
  }

  return undefined;
}

function readGlobalLocation(): unknown {
  return (globalThis as {
    location?: unknown;
  }).location;
}

function isLocationLike(value: unknown): value is { origin: string } {
  return (
    typeof value === "object" &&
    value !== null &&
    "origin" in value &&
    typeof (value as { origin: unknown }).origin === "string" &&
    (value as { origin: string }).origin.trim().length > 0
  );
}

export default NewsletterClient;