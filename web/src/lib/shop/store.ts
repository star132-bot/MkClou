import { create } from "zustand";

import type { MerchantShop } from "./types";

/**
 * 当前商家的店铺。由后台的 ShopGuard 加载；保存设置后更新，供导航栏、概览页等共用。
 * dirty 表示店铺设置页有未保存的修改，导航离开前需要确认（PRD 02 第 5 节）。
 */
interface ShopState {
  shop: MerchantShop | null;
  dirty: boolean;
  /** 有未保存修改时点击了站内链接，等待商家确认是否离开 */
  pendingHref: string | null;
  setShop: (shop: MerchantShop | null) => void;
  setDirty: (dirty: boolean) => void;
  setPendingHref: (href: string | null) => void;
}

export const useShopStore = create<ShopState>((set) => ({
  shop: null,
  dirty: false,
  pendingHref: null,
  setShop: (shop) => set({ shop }),
  setDirty: (dirty) => set({ dirty }),
  setPendingHref: (pendingHref) => set({ pendingHref }),
}));
