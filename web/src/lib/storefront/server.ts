// 买家端页面在服务端请求后端公开接口（docs/api-list.md 2.7）。
import { headers } from "next/headers";

import type { MarketHome, MarketSearchResult, ProductPageData, ProductSummary, PublicShopResult } from "./types";

const apiOrigin = process.env.API_ORIGIN ?? "http://127.0.0.1:18080";

/**
 * 获取店铺信息；店铺不存在时返回 null。
 * 后端已有 5 分钟 Redis 缓存，这里不再使用 Next.js 的数据缓存，保证商家修改后买家立即可见。
 * 转发买家 IP，避免所有服务端渲染请求共用 Next 服务器的 IP 而触发后端限流。
 */
export function fetchPublicShop(slug: string): Promise<PublicShopResult | null> {
  return getPublic<PublicShopResult>(`/public/shops/${encodeURIComponent(slug)}`);
}

/** 店铺已上架的商品（#73），第一页。 */
export async function fetchShopProducts(slug: string): Promise<ProductSummary[]> {
  const r = await getPublic<{ items: ProductSummary[]; nextCursor: string | null }>(
    `/public/shops/${encodeURIComponent(slug)}/products?limit=48`,
  );
  return r?.items ?? [];
}

/** 商品详情页数据（#74）；商品不存在时返回 null。 */
export function fetchProductPage(publicId: string): Promise<ProductPageData | null> {
  return getPublic<ProductPageData>(`/public/products/${encodeURIComponent(publicId)}`);
}

/** 商城首页（MKT-01）。 */
export async function fetchMarketHome(): Promise<MarketHome> {
  return (await getPublic<MarketHome>("/public/market/home")) ?? { hot: [], latest: [] };
}

/** 搜索商品（MKT-02），参数原样转发。 */
export async function fetchMarketSearch(params: Record<string, string>): Promise<MarketSearchResult> {
  const qs = new URLSearchParams(params);
  return (await getPublic<MarketSearchResult>(`/public/market/search?${qs}`)) ?? { items: [], total: 0, page: 1, pageSize: 24 };
}

/** 请求买家端公开接口；404 返回 null，其他错误抛出（由 Next 错误页处理）。 */
export async function getPublic<T>(path: string): Promise<T | null> {
  const forwardedFor = (await headers()).get("x-forwarded-for");
  const res = await fetch(`${apiOrigin}/api/v1${path}`, {
    cache: "no-store",
    headers: forwardedFor ? { "X-Forwarded-For": forwardedFor } : undefined,
  });
  if (res.status === 404) return null;
  if (!res.ok) throw new Error(`GET ${path}: HTTP ${res.status}`);
  const body = (await res.json()) as { data: T };
  return body.data;
}
