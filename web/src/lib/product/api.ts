// 商品接口（docs/api-list.md 2.4）。
import { api } from "@/lib/api/client";
import type { DeliveryType } from "@/lib/storefront/types";

import type { EditableProduct, MerchantProductStatus, Page, ProductListItem, ProductSavePayload } from "./types";

const path = (publicId: string) => `/products/${encodeURIComponent(publicId)}`;

export const productApi = {
  list: (params: { status?: MerchantProductStatus | ""; q?: string; page?: number; pageSize?: number }) => {
    const qs = new URLSearchParams();
    if (params.status) qs.set("status", params.status);
    if (params.q) qs.set("q", params.q);
    qs.set("page", String(params.page ?? 1));
    qs.set("pageSize", String(params.pageSize ?? 20));
    return api<Page<ProductListItem>>(`/products?${qs}`);
  },

  create: (body: { name: string; deliveryType: DeliveryType; price: number }) =>
    api<EditableProduct>("/products", { method: "POST", body }),

  get: (publicId: string) => api<EditableProduct>(path(publicId)),

  save: (publicId: string, body: ProductSavePayload) => api<EditableProduct>(path(publicId), { method: "PUT", body }),

  publish: (publicId: string) => api<EditableProduct>(`${path(publicId)}/publish`, { method: "POST" }),

  unpublish: (publicId: string) => api<EditableProduct>(`${path(publicId)}/unpublish`, { method: "POST" }),

  remove: (publicId: string) => api<null>(path(publicId), { method: "DELETE" }),
};
