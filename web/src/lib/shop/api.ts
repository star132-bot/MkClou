// 店铺接口（docs/api-list.md 2.2）。
import { api } from "@/lib/api/client";
import type { ShopStatus } from "@/lib/storefront/types";

import type { MerchantShop, Onboarding, ShopPatch, SlugCheck } from "./types";

export const shopApi = {
  checkSlug: (slug: string) => api<SlugCheck>(`/shop/slug-availability?slug=${encodeURIComponent(slug)}`),

  create: (body: { name: string; slug: string }) => api<MerchantShop>("/shop", { method: "POST", body }),

  get: () => api<MerchantShop>("/shop"),

  update: (body: ShopPatch) => api<MerchantShop>("/shop", { method: "PATCH", body }),

  setStatus: (status: Exclude<ShopStatus, "BANNED">, pauseNote = "") =>
    api<MerchantShop>("/shop/status", { method: "PUT", body: { status, pauseNote } }),

  onboarding: () => api<Onboarding>("/shop/onboarding"),

  markShared: () => api<null>("/shop/onboarding/shared", { method: "POST" }),
};
