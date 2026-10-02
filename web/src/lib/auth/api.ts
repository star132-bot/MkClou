// 账号相关接口（docs/api-list.md 2.1）。
import { api, type AuthResponse, type MerchantView } from "@/lib/api/client";

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

  me: () => api<{ merchant: MerchantView; shop: null }>("/me"),
};
