"use client";

import { Heart, Package, Settings, Store } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { authApi } from "@/lib/auth/api";
import { cn } from "@/lib/utils";

const items = [
  { href: "/me/favorites", label: "我的收藏", icon: Heart },
  { href: "/me/orders", label: "我的订单", icon: Package },
  { href: "/me/account", label: "账号设置", icon: Settings },
];

export function MeNav() {
  const pathname = usePathname();
  const [hasShop, setHasShop] = useState<boolean | null>(null);

  useEffect(() => {
    let cancelled = false;
    authApi
      .me()
      .then((r) => !cancelled && setHasShop(r.shop !== null))
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <aside className="flex shrink-0 flex-col gap-6 md:w-[200px]">
      <nav aria-label="个人中心" className="-mx-4 flex gap-1 overflow-x-auto border-b border-border px-4 md:mx-0 md:flex-col md:border-0 md:px-0">
        {items.map((it) => {
          const active = pathname.startsWith(it.href);
          return (
            <Link
              key={it.href}
              href={it.href}
              aria-current={active ? "page" : undefined}
              className={cn(
                "flex h-10 shrink-0 items-center gap-2 border-b-2 px-3 font-medium transition-colors md:rounded-md md:border-0",
                active ? "border-text-primary text-text-primary md:bg-muted" : "border-transparent text-text-secondary hover:text-text-primary md:hover:bg-muted",
              )}
            >
              <it.icon className="size-4" strokeWidth={1.5} aria-hidden />
              {it.label}
            </Link>
          );
        })}
      </nav>
      {hasShop !== null && (
        <div className="hidden flex-col gap-3 rounded-lg border border-border p-4 md:flex">
          <Store className="size-5 text-brand" strokeWidth={1.5} aria-hidden />
          <p className="text-caption">{hasShop ? "管理你的商品、订单和店铺设置" : "有作品想卖？几分钟开一家店，平台不抽成"}</p>
          <Button size="sm" variant={hasShop ? "outline" : "default"} asChild>
            <Link href={hasShop ? "/dashboard" : "/onboarding"}>{hasShop ? "进入卖家中心" : "我要开店"}</Link>
          </Button>
        </div>
      )}
    </aside>
  );
}
