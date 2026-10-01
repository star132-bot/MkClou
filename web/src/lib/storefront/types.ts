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

export interface PublicShop {
  slug: string;
  name: string;
  description: string;
  avatarUrl: string | null;
  contactEmail: string;
  socialLinks: { type: string; url: string }[];
  theme: ShopTheme;
  status: ShopStatus;
  pauseNote: string | null;
  isTestMode: boolean;
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
