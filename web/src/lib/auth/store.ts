import { create } from "zustand";

import type { MerchantView } from "@/lib/api/client";

/**
 * 登录状态。Access Token 只保存在内存中（不写 localStorage，降低 XSS 窃取风险，见 PRD SEC-04）；
 * 刷新页面后通过 HttpOnly Cookie 中的 Refresh Token 恢复。
 */
interface AuthState {
  status: "unknown" | "authenticated" | "anonymous";
  accessToken: string | null;
  merchant: MerchantView | null;
  signIn: (accessToken: string, merchant: MerchantView) => void;
  signOut: () => void;
  updateMerchant: (patch: Partial<MerchantView>) => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  status: "unknown",
  accessToken: null,
  merchant: null,
  signIn: (accessToken, merchant) => set({ status: "authenticated", accessToken, merchant }),
  signOut: () => set({ status: "anonymous", accessToken: null, merchant: null }),
  updateMerchant: (patch) => set((s) => (s.merchant ? { merchant: { ...s.merchant, ...patch } } : s)),
}));
