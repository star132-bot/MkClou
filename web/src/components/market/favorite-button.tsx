"use client";

import { Heart } from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect } from "react";
import { toast } from "sonner";

import { ApiError, refreshSession } from "@/lib/api/client";
import { useAuthStore } from "@/lib/auth/store";
import { favoriteApi } from "@/lib/market/api";
import { onAuthChange, requestFavoriteStatus, useFavoriteStore } from "@/lib/market/favorites";
import { cn } from "@/lib/utils";

/** 登录后回到页面时，自动完成登录前想收藏的商品（PRD MKT 4.6）。 */
const RESUME_PARAM = "favorite";
// 同一商品可能同时出现在“热门”和“最新”两个区块，只由第一个按钮处理
const resumedIds = new Set<string>();

/**
 * 收藏按钮（PRD MKT-05）。未登录时跳转登录页，登录后回到原页面并自动收藏；
 * 点击后立即切换图标（乐观更新），请求失败时回滚并提示。
 */
interface FavoriteButtonProps {
  publicId: string;
  /** overlay：卡片封面右上角的圆形按钮；inline：带文字的按钮 */
  variant?: "overlay" | "inline";
  onChange?: (favorited: boolean) => void;
}

// 读取地址参数需要 Suspense 边界，按钮自带边界，页面可以直接使用
export function FavoriteButton(props: FavoriteButtonProps) {
  return (
    <Suspense fallback={null}>
      <FavoriteButtonInner {...props} />
    </Suspense>
  );
}

function FavoriteButtonInner({ publicId, variant = "overlay", onChange }: FavoriteButtonProps) {
  const status = useAuthStore((s) => s.status);
  const favorited = useFavoriteStore((s) => s.favorited[publicId] ?? false);
  const setFavorited = useFavoriteStore((s) => s.setFavorited);
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();

  useEffect(() => {
    if (status === "unknown") void refreshSession();
    onAuthChange(status);
  }, [status]);

  useEffect(() => requestFavoriteStatus(publicId), [publicId]);

  const toggle = async (next: boolean) => {
    setFavorited(publicId, next);
    onChange?.(next);
    try {
      if (next) await favoriteApi.add(publicId);
      else await favoriteApi.remove(publicId);
    } catch (e) {
      setFavorited(publicId, !next);
      onChange?.(!next);
      toast.error(e instanceof ApiError ? e.message : "操作失败，请稍后重试");
    }
  };

  // 登录回来后自动收藏一次，并去掉地址中的参数
  useEffect(() => {
    if (resumedIds.has(publicId) || status !== "authenticated" || params.get(RESUME_PARAM) !== publicId) return;
    resumedIds.add(publicId);
    const rest = new URLSearchParams(params);
    rest.delete(RESUME_PARAM);
    router.replace(rest.size ? `${pathname}?${rest}` : pathname, { scroll: false });
    void toggle(true).then(() => toast.success("已收藏"));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- 只在登录状态确定后执行一次
  }, [status, publicId]);

  const onClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (status !== "authenticated") {
      const back = new URLSearchParams(params);
      back.set(RESUME_PARAM, publicId);
      router.push(`/login?redirect=${encodeURIComponent(`${pathname}?${back}`)}`);
      return;
    }
    void toggle(!favorited);
  };

  const label = favorited ? "取消收藏" : "收藏";
  if (variant === "inline") {
    return (
      <button
        type="button"
        onClick={onClick}
        aria-pressed={favorited}
        className="inline-flex h-9 items-center gap-2 rounded-md border border-border bg-background px-3 font-medium text-text-primary transition-colors outline-none hover:bg-muted focus-visible:ring-2 focus-visible:ring-ring"
      >
        <Heart className={cn("size-4", favorited ? "fill-danger text-danger" : "text-text-tertiary")} strokeWidth={1.5} aria-hidden />
        {favorited ? "已收藏" : "收藏"}
      </button>
    );
  }
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={favorited}
      aria-label={label}
      title={label}
      className="flex size-9 items-center justify-center rounded-full bg-background/90 shadow-sm backdrop-blur-sm transition-transform outline-none hover:scale-105 focus-visible:ring-2 focus-visible:ring-ring active:scale-95"
    >
      <Heart className={cn("size-4", favorited ? "fill-danger text-danger" : "text-text-secondary")} strokeWidth={1.5} aria-hidden />
    </button>
  );
}
