// 账号相关接口（docs/api-list.md 2.1）。
import { api, type AuthResponse, type MerchantView } from "@/lib/api/client";
import type { ShopSummary } from "@/lib/shop/types";

export const authApi = {
  register: (body: { email: string; password: string; agreeTerms: boolean }) =>
    api<AuthResponse>("/auth/register", { method: "POST", body, auth: false }),

  login: (body: { email: string; password: string; remember: boolean; captchaId?: string; captchaCode?: string }) =>
    api<AuthResponse>("/auth/login", { method: "POST", body, auth: false }),

  logout: () => api<null>("/auth/logout", { method: "POST", auth: false }),

  captcha: () => api<{ captchaId: string; image: string }>("/auth/captcha", { auth: false }),

  verifyEmail: (token: string) =>
    api<{ alreadyVerified: boolean }>("/auth/email/verify", { method: "POST", body: { token }, auth: false }),

  resendVerification: () => api<null>("/auth/email/resend", { method: "POST" }),

  forgotPassword: (email: string) =>
    api<null>("/auth/password/forgot", { method: "POST", body: { email }, auth: false }),

  resetPassword: (token: string, password: string) =>
    api<null>("/auth/password/reset", { method: "POST", body: { token, password }, auth: false }),

  me: () => api<{ merchant: MerchantView; shop: ShopSummary | null }>("/me"),

  updateNickname: (nickname: string) => api<null>("/me", { method: "PATCH", body: { nickname } }),

  changePassword: (currentPassword: string, newPassword: string) =>
    api<null>("/me/password", { method: "PUT", body: { currentPassword, newPassword } }),

  sessions: () => api<{ items: SessionView[] }>("/me/sessions"),

  revokeSession: (id: string) => api<null>(`/me/sessions/${encodeURIComponent(id)}`, { method: "DELETE" }),

  revokeOtherSessions: () => api<null>("/me/sessions", { method: "DELETE" }),
};

/** 登录设备（AUTH-07）。 */
export interface SessionView {
  id: string;
  userAgent: string;
  ip: string;
  lastActiveAt: string;
  createdAt: string;
  current: boolean;
}
