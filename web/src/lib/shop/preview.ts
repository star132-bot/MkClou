import type { ProductSummary, PublicShop } from "@/lib/storefront/types";

// 店铺设置页与预览 iframe（/preview/shop）之间的消息协议
export const PREVIEW_READY = "mk:shop-preview-ready";
export const PREVIEW_DATA = "mk:shop-preview-data";

export interface PreviewMessage {
  type: typeof PREVIEW_DATA;
  shop: PublicShop;
  products: ProductSummary[];
}

/** 还没有商品时，预览用占位商品展示卡片比例和布局的效果。 */
export const SAMPLE_PRODUCTS: ProductSummary[] = [
  { publicId: "sample-1", name: "示例商品：设计素材包", price: 2990, originalPrice: 3990, soldOut: false, isNew: true },
  { publicId: "sample-2", name: "示例商品：入门教程", price: 0, originalPrice: null, soldOut: false, isNew: false },
  { publicId: "sample-3", name: "示例商品：会员兑换码", price: 990, originalPrice: null, soldOut: true, isNew: false },
].map((p) => ({ ...p, favoriteCount: 0, cover: { url: "", width: 4, height: 3, alt: "" } }));
