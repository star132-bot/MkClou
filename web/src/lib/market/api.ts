// 收藏接口（PRD MKT-05、MKT-06）。
import { api } from "@/lib/api/client";
import type { Page } from "@/lib/product/types";
import type { MarketCard } from "@/lib/storefront/types";

export interface FavoriteItem extends MarketCard {
  available: boolean;
  /** 已失效 / 已售罄；正常可买时为空 */
  reason: string;
  favoritedAt: string;
}

const path = (publicId: string) => `/me/favorites/${encodeURIComponent(publicId)}`;

export const favoriteApi = {
  list: (page = 1) => api<Page<FavoriteItem>>(`/me/favorites?page=${page}`),
  status: (ids: string[]) => api<{ favorited: string[] }>(`/me/favorites/status?ids=${ids.map(encodeURIComponent).join(",")}`),
  add: (publicId: string) => api<{ favorited: true; favoriteCount: number }>(path(publicId), { method: "PUT" }),
  remove: (publicId: string) => api<{ favorited: false }>(path(publicId), { method: "DELETE" }),
};
