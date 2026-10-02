// 店铺接口的数据类型，对应 docs/api-list.md #18 ～ #24（后端 internal/shop/model.go）。
import type { ShopStatus, ShopTheme, SocialLink } from "@/lib/storefront/types";

/** 商家后台看到的店铺信息（GET /shop）。 */
export interface MerchantShop {
  slug: string;
  name: string;
  description: string;
  avatarUrl: string | null;
  coverUrl: string | null;
  contactEmail: string;
  socialLinks: SocialLink[];
  theme: ShopTheme;
  status: ShopStatus;
  pauseNote: string | null;
  slugChangeAllowed: boolean;
  /** 下一次可以修改链接的时间；当前可修改时为 null */
  nextSlugChangeAt: string | null;
  createdAt: string;
}

/** /me 中的店铺概要。 */
export interface ShopSummary {
  slug: string;
  name: string;
  status: ShopStatus;
}

export interface SlugCheck {
  available: boolean;
  reason: string;
  suggestions: string[];
}

/** 新手清单完成情况（PRD SHOP-05）。 */
export interface Onboarding {
  emailVerified: boolean;
  paymentReady: boolean;
  hasProduct: boolean;
  shared: boolean;
}

export type ShopPatch = Partial<Pick<MerchantShop, "name" | "slug" | "description" | "contactEmail" | "socialLinks" | "theme">>;
