import type { PublicProduct, PublicShop } from "./types";

export interface PurchaseState {
  disabled: boolean;
  label: string;
  /** 按钮下方的说明文字 */
  note: string | null;
}

/** 购买按钮状态，规则见 PRD 06-storefront 3.2。按优先级依次判断。 */
export function getPurchaseState(shop: PublicShop, product: PublicProduct): PurchaseState {
  if (shop.status === "PAUSED") {
    return { disabled: true, label: "店铺暂停营业", note: shop.pauseNote };
  }
  if (!product.purchasable) {
    return { disabled: true, label: "暂时无法购买", note: "店铺收款暂不可用，请稍后再来或联系店主。" };
  }
  if (product.soldOut) {
    return { disabled: true, label: "已售罄", note: null };
  }
  return { disabled: false, label: product.price === 0 ? "免费领取" : "立即购买", note: null };
}
