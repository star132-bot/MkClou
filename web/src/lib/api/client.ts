// API 客户端：统一响应解析、Access Token 注入、401 自动续期（PRD AUTH-06、总览第 7 节）。
import { useAuthStore } from "@/lib/auth/store";

const BASE = "/api/v1";
const TIMEOUT_MS = 15_000;

/** 后端统一响应：{ code, message, data, traceId } */
interface Envelope<T> {
  code: number;
  message: string;
  data: T;
  traceId: string;
}

/** 业务错误。fields 为字段级错误，供表单逐项显示。 */
export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: number,
    message: string,
    readonly data: unknown = null,
    readonly traceId = "",
  ) {
    super(message);
  }

  get fields(): Record<string, string> {
    const d = this.data as { fields?: Record<string, string> } | null;
    return d?.fields ?? {};
  }
}

export const ErrorCode = {
  InvalidParams: 10001,
  TooManyRequests: 10002,
  Unauthorized: 20000,
  BadCredentials: 20001,
  CaptchaRequired: 20002,
  AccountLocked: 20003,
  EmailNotVerified: 20004,
  TokenInvalid: 20005,
  AccountDisabled: 20006,
  EmailRegistered: 20007,
} as const;

interface RequestOptions {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  /** 是否携带 Access Token 并在 401 时尝试续期，默认 true */
  auth?: boolean;
}

async function send<T>(path: string, { method = "GET", body, auth = true }: RequestOptions): Promise<T> {
  const headers: Record<string, string> = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const token = useAuthStore.getState().accessToken;
  if (auth && token) headers.Authorization = `Bearer ${token}`;

  let res: Response;
  try {
    res = await fetch(BASE + path, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      credentials: "same-origin",
      signal: AbortSignal.timeout(TIMEOUT_MS),
    });
  } catch {
    throw new ApiError(0, -1, "网络连接异常，请检查网络后重试");
  }

  let env: Envelope<T> | null = null;
  try {
    env = (await res.json()) as Envelope<T>;
  } catch {
    // 非 JSON 响应（如网关错误页）
  }
  if (!res.ok || !env || env.code !== 0) {
    throw new ApiError(
      res.status,
      env?.code ?? -1,
      env?.message ?? "服务异常，请稍后重试",
      env?.data ?? null,
      env?.traceId ?? "",
    );
  }
  return env.data;
}

/**
 * 发起请求。需要登录的接口在 401 时自动续期一次并重试；续期失败则清空登录状态，由页面守卫跳转登录页。
 */
export async function api<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  try {
    return await send<T>(path, opts);
  } catch (e) {
    const canRetry = opts.auth !== false && e instanceof ApiError && e.status === 401 && !path.startsWith("/auth/");
    if (!canRetry) throw e;
    const ok = await refreshSession();
    if (!ok) throw e;
    return send<T>(path, opts);
  }
}

export interface MerchantView {
  email: string;
  nickname: string;
  emailVerified: boolean;
  createdAt: string;
}

export interface AuthResponse {
  accessToken: string;
  expiresAt: string;
  merchant: MerchantView;
}

let inflight: Promise<boolean> | null = null;

/**
 * 用 Refresh Token（HttpOnly Cookie）换取新的 Access Token。
 * - 同一标签页内的并发调用合并为一次请求；
 * - 多个标签页之间用 Web Locks 串行化：后拿到锁的标签页会带着已轮换的新 Cookie 续期，
 *   不会触发后端的“旧 Token 重放”检测。
 */
export function refreshSession(): Promise<boolean> {
  inflight ??= (async () => {
    const run = async () => {
      try {
        const r = await send<AuthResponse>("/auth/refresh", { method: "POST", auth: false });
        useAuthStore.getState().signIn(r.accessToken, r.merchant);
        return true;
      } catch {
        useAuthStore.getState().signOut();
        return false;
      }
    };
    try {
      return typeof navigator !== "undefined" && navigator.locks
        ? await navigator.locks.request("mkclou-refresh", run)
        : await run();
    } finally {
      inflight = null;
    }
  })();
  return inflight;
}
