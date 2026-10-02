import { create } from "zustand";

import { useAuthStore } from "@/lib/auth/store";

import { favoriteApi } from "./api";

/**
 * 当前账号的收藏状态。页面上的收藏按钮各自登记商品 ID，同一时刻登记的 ID 合并为一次查询，
 * 避免一页 24 个卡片发出 24 个请求。
 */
interface FavoriteState {
  favorited: Record<string, boolean>;
  setFavorited: (publicId: string, value: boolean) => void;
  reset: () => void;
}

export const useFavoriteStore = create<FavoriteState>((set) => ({
  favorited: {},
  setFavorited: (publicId, value) => set((s) => ({ favorited: { ...s.favorited, [publicId]: value } })),
  reset: () => set({ favorited: {} }),
}));

const pending = new Set<string>();
const checked = new Set<string>();
let timer: ReturnType<typeof setTimeout> | null = null;

/** 登记需要显示收藏状态的商品，下一个事件循环批量查询。未登录时不查询。 */
export function requestFavoriteStatus(publicId: string) {
  if (checked.has(publicId)) return;
  pending.add(publicId);
  timer ??= setTimeout(flush, 0);
}

function flush() {
  timer = null;
  if (useAuthStore.getState().status !== "authenticated") return; // 登录后由 FavoriteSync 重新触发
  const ids = [...pending].slice(0, 100);
  ids.forEach((id) => {
    pending.delete(id);
    checked.add(id);
  });
  if (ids.length === 0) return;
  favoriteApi
    .status(ids)
    .then(({ favorited }) => {
      const on = new Set(favorited);
      const { setFavorited } = useFavoriteStore.getState();
      ids.forEach((id) => setFavorited(id, on.has(id)));
    })
    .catch(() => ids.forEach((id) => checked.delete(id)));
  if (pending.size > 0) timer = setTimeout(flush, 0);
}

/** 登录状态变化时调用：登录后查询已登记的商品，退出后清空。 */
export function onAuthChange(status: string) {
  if (status === "authenticated") {
    checked.forEach((id) => pending.add(id));
    checked.clear();
    timer ??= setTimeout(flush, 0);
  } else if (status === "anonymous") {
    useFavoriteStore.getState().reset();
  }
}
