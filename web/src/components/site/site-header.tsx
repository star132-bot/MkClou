"use client";

import { Heart, LogOut, Package, Search, Settings, Store, UserRound } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { refreshSession } from "@/lib/api/client";
import { authApi } from "@/lib/auth/api";
import { useAuthStore } from "@/lib/auth/store";
import type { ShopSummary } from "@/lib/shop/types";

/**
 * 商城统一顶栏（PRD MKT 4.1）：品牌、搜索框、登录入口或账号菜单。
 * 搜索框是普通的 GET 表单，脚本加载前也能使用。
 */
export function SiteHeader({ query = "" }: { query?: string }) {
  const status = useAuthStore((s) => s.status);
  const merchant = useAuthStore((s) => s.merchant);
  const signOut = useAuthStore((s) => s.signOut);
  const pathname = usePathname();
  const router = useRouter();
  // undefined：尚未查询；null：没有店铺
  const [shop, setShop] = useState<ShopSummary | null | undefined>(undefined);

  // 公开页面静默恢复登录状态：没有登录 Cookie 时续期失败，状态变为未登录，不跳转
  useEffect(() => {
    if (status === "unknown") void refreshSession();
  }, [status]);

  useEffect(() => {
    if (status !== "authenticated") return;
    let cancelled = false;
    authApi
      .me()
      .then((r) => !cancelled && setShop(r.shop))
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [status]);

  const logout = async () => {
    try {
      await authApi.logout();
    } finally {
      signOut();
      router.refresh();
    }
  };

  const loginHref = `/login?redirect=${encodeURIComponent(pathname)}`;
  const name = merchant?.nickname || merchant?.email.split("@")[0] || "";

  return (
    <header className="sticky top-0 z-40 border-b border-border bg-background/95 backdrop-blur-sm">
      <div className="mx-auto flex h-16 w-full max-w-[1200px] items-center gap-3 px-4 md:gap-6 md:px-6">
        <Link href="/" className="shrink-0 text-h3 font-semibold tracking-tight text-text-primary">
          MkClou
        </Link>

        <form action="/search" role="search" className="relative min-w-0 flex-1 md:max-w-[480px]">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-text-tertiary" strokeWidth={1.5} aria-hidden />
          <input
            name="q"
            type="search"
            defaultValue={query}
            maxLength={50}
            placeholder="搜索商品、店铺……"
            aria-label="搜索商品和店铺"
            className="h-9 w-full rounded-full border border-input bg-surface pr-4 pl-9 text-body-lg text-text-primary transition-[border-color,box-shadow] duration-150 outline-none placeholder:text-text-tertiary focus-visible:border-brand focus-visible:ring-3 focus-visible:ring-brand/15 md:text-body"
          />
        </form>

        <div className="ml-auto flex shrink-0 items-center gap-1">
          {status === "authenticated" ? (
            <>
              <Button variant="ghost" size="sm" asChild className="hidden sm:inline-flex">
                <Link href="/me/favorites">
                  <Heart strokeWidth={1.5} data-icon="inline-start" />
                  收藏
                </Link>
              </Button>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <button
                    type="button"
                    aria-label="账号菜单"
                    className="flex size-9 items-center justify-center rounded-full bg-muted text-body font-semibold text-text-primary uppercase transition-colors outline-none hover:bg-border focus-visible:ring-2 focus-visible:ring-ring"
                  >
                    {name.slice(0, 1) || <UserRound className="size-4" strokeWidth={1.5} />}
                  </button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuLabel className="truncate">{merchant?.email}</DropdownMenuLabel>
                  <DropdownMenuItem asChild>
                    <Link href="/me/favorites">
                      <Heart strokeWidth={1.5} />
                      我的收藏
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild>
                    <Link href="/me/orders">
                      <Package strokeWidth={1.5} />
                      我的订单
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild>
                    <Link href={shop ? "/dashboard" : "/onboarding"}>
                      <Store strokeWidth={1.5} />
                      {shop === undefined ? "卖家中心" : shop ? "卖家中心" : "我要开店"}
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild>
                    <Link href="/me/account">
                      <Settings strokeWidth={1.5} />
                      账号设置
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onSelect={logout}>
                    <LogOut strokeWidth={1.5} />
                    退出登录
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </>
          ) : status === "anonymous" ? (
            <>
              <Button variant="ghost" size="sm" asChild>
                <Link href={loginHref}>登录</Link>
              </Button>
              <Button size="sm" asChild className="hidden sm:inline-flex">
                <Link href={`/register?redirect=${encodeURIComponent(pathname)}`}>注册</Link>
              </Button>
            </>
          ) : (
            <span className="h-8 w-16 animate-pulse rounded-md bg-muted" aria-hidden />
          )}
        </div>
      </div>
    </header>
  );
}
