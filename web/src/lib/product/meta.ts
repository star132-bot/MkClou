import type { DeliveryType } from "@/lib/storefront/types";

import type { CategoryValue, MerchantProductStatus } from "./types";

/** 平台一级分类（PRD MKT-04），与后端 product.Categories 一致。 */
export const CATEGORIES: { value: CategoryValue; label: string }[] = [
  { value: "design", label: "设计素材" },
  { value: "software", label: "软件工具" },
  { value: "course", label: "教程课程" },
  { value: "template", label: "效率模板" },
  { value: "membership", label: "会员兑换" },
  { value: "other", label: "其他" },
];

export const categoryLabel = (v: string | null | undefined) => CATEGORIES.find((c) => c.value === v)?.label ?? "未分类";

export const DELIVERY_TYPES: { value: DeliveryType; label: string; desc: string; available: boolean }[] = [
  { value: "LINK", label: "链接", desc: "网盘、在线文档等链接，可附提取码", available: true },
  { value: "TEXT", label: "文本", desc: "付款后显示一段文字，如激活方法、群号", available: true },
  { value: "FILE", label: "文件", desc: "上传文件，买家付款后下载", available: false },
  { value: "CARD", label: "卡密", desc: "每单自动分配一个兑换码", available: false },
];

export const deliveryLabel = (v: DeliveryType) => DELIVERY_TYPES.find((d) => d.value === v)?.label ?? v;

export const STATUS_META: Record<MerchantProductStatus, { label: string; className: string }> = {
  ON_SALE: { label: "已上架", className: "bg-success/10 text-success" },
  PENDING_REVIEW: { label: "待审核", className: "bg-info/10 text-info" },
  DRAFT: { label: "草稿", className: "bg-muted text-text-secondary" },
  OFF_SALE: { label: "已下架", className: "bg-muted text-text-tertiary" },
  BANNED: { label: "平台下架", className: "bg-danger/10 text-danger" },
};

/** “29.9” → 2990 分；格式不正确返回 null。 */
export function yuanToCents(input: string): number | null {
  const v = input.trim();
  if (!/^\d+(\.\d{1,2})?$/.test(v)) return null;
  const [int, dec = ""] = v.split(".");
  return Number(int) * 100 + Number(dec.padEnd(2, "0"));
}

export const centsToYuan = (cents: number) => (cents / 100).toFixed(2);
