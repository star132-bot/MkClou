// 商品接口的数据类型，对应 docs/api-list.md #34 ～ #42（后端 internal/product/model.go）。
import type { DeliveryType, ProductFaq, ProductStatus } from "@/lib/storefront/types";

export type CategoryValue = "design" | "software" | "course" | "template" | "membership" | "other";

/** 商家端的商品状态比买家端多“待审核”。 */
export type MerchantProductStatus = ProductStatus | "PENDING_REVIEW";

export interface ProductImageRef {
  key: string;
  url: string;
  width: number;
  height: number;
}

export interface ProductLink {
  name: string;
  url: string;
  code: string;
}

export interface ProductDetail {
  includes: string[];
  audience: string;
  faqs: ProductFaq[];
  notice: string;
}

export interface DeliveryConfig {
  links: ProductLink[];
  text: string;
  cardInstructions: string;
  note: string;
}

/** 编辑页使用的完整商品（GET /products/{publicId}）。 */
export interface EditableProduct {
  publicId: string;
  name: string;
  tagline: string;
  category: CategoryValue | null;
  price: number;
  originalPrice: number | null;
  deliveryType: DeliveryType;
  status: MerchantProductStatus;
  descriptionMd: string;
  detail: ProductDetail;
  deliveryConfig: DeliveryConfig;
  maxPerOrder: number;
  images: ProductImageRef[];
  salesCount: number;
  banReason: string | null;
  version: number;
  deliveryTypeLocked: boolean;
  publishedAt: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface ProductListItem {
  publicId: string;
  name: string;
  cover: ProductImageRef | null;
  category: CategoryValue | null;
  deliveryType: DeliveryType;
  price: number;
  originalPrice: number | null;
  status: MerchantProductStatus;
  salesCount: number;
  updatedAt: string;
}

export interface Page<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

/** 上架检查未通过的一项（错误码 40001 的 data.issues）。 */
export interface PublishIssue {
  field: string;
  message: string;
}

export interface ProductSavePayload {
  version: number;
  name: string;
  tagline: string;
  category: CategoryValue | null;
  price: number;
  originalPrice: number | null;
  deliveryType: DeliveryType;
  descriptionMd: string;
  detail: ProductDetail;
  deliveryConfig: DeliveryConfig;
  maxPerOrder: number;
  images: { key: string; width: number; height: number }[];
}
