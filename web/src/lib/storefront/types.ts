// 买家端公开接口的数据类型，对应 docs/api-list.md #72 ～ #74。
// 金额单位为分；时间为 ISO 8601 字符串。

export type DeliveryType = "FILE" | "LINK" | "TEXT" | "CARD";

export type ShopStatus = "OPEN" | "PAUSED" | "BANNED";

export type ProductStatus = "ON_SALE" | "OFF_SALE" | "BANNED" | "DRAFT";

export type CardRatio = "4:3" | "16:9" | "1:1";

export interface ShopTheme {
  color: string;
  cardRatio: CardRatio;
  layout: "grid" | "list";
  mode: "light" | "dark" | "system";
}

export type SocialType = "website" | "github" | "bilibili" | "xiaohongshu" | "weibo" | "douyin" | "x" | "youtube";

export interface SocialLink {
  type: SocialType;
  url: string;
}

export interface PublicShop {
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
  isTestMode: boolean;
}

/** GET /public/shops/{slug}：店铺信息，或旧链接需要跳转到的新链接（PRD 02 3.2）。 */
export interface PublicShopResult {
  shop: PublicShop | null;
  redirectTo: string | null;
}

export interface ProductImage {
  url: string;
  width: number;
  height: number;
  alt: string;
}

export interface ProductFaq {
  q: string;
  a: string;
}

export interface ProductSummary {
  publicId: string;
  name: string;
  price: number;
  originalPrice: number | null;
  cover: ProductImage;
  soldOut: boolean;
  isNew: boolean;
  favoriteCount: number;
}

/** 商城中的商品卡片：商品摘要 + 所属店铺（PRD MKT-01）。 */
export interface MarketCard extends ProductSummary {
  shop: { slug: string; name: string };
}

export interface MarketHome {
  hot: MarketCard[];
  latest: MarketCard[];
}

export interface MarketSearchResult {
  items: MarketCard[];
  total: number;
  page: number;
  pageSize: number;
}

export interface PublicProduct extends ProductSummary {
  tagline: string;
  status: ProductStatus;
  images: ProductImage[];
  descriptionMd: string;
  detail: {
    includes: string[];
    audience: string | null;
    faqs: ProductFaq[];
    notice: string | null;
  };
  delivery: {
    type: DeliveryType;
    fileCount: number | null;
    totalSize: number | null; // 字节
  };
  maxPerOrder: number;
  /** 卡密商品的剩余库存；仅在 ≤ 10 时返回具体数字，其他情况为 null */
  stockHint: number | null;
  /** 店铺收款是否可用 */
  purchasable: boolean;
}

export interface ProductPageData {
  shop: PublicShop;
  product: PublicProduct;
  moreFromShop: ProductSummary[];
}
