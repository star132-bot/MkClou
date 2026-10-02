"use client";

import { ExternalLink, LogOut } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useState } from "react";

import { ShopAvatar } from "@/components/storefront/shop-chrome";
import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogTitle } from "@/components/ui/dialog";
import { authApi } from "@/lib/auth/api";
import { useAuthStore } from "@/lib/auth/store";
import { useShopStore } from "@/lib/shop/store";
import { cn } from "@/lib/utils";

const nav = [
  { href: "/dashboard", label: "概览" },
  { href: "/dashboard/products", label: "商品" },
  { href: "/dashboard/shop", label: "店铺设置" },
] as const;

/** 站内链接：有未保存的修改时先弹窗确认（PRD 02 第 5 节）。 */
export function GuardedLink({ href, className, children }: { href: string; className?: string; children: React.ReactNode }) {
  const dirty = useShopStore((s) => s.dirty);
  const setPendingHref = useShopStore((s) => s.setPendingHref);
  return (
    <Link
      href={href}
      className={className}
      onClick={(e) => {
        if (!dirty) return;
        e.preventDefault();
        setPendingHref(href);
      }}
    >
      {children}
    </Link>
  );
}

/**
 * 卖家中心顶栏：品牌（回到商城）、导航（开店后显示）、查看店铺、退出登录。
 * 完整的侧边栏后台框架在商家后台模块实现（PRD DSB-02）。
 */
export function DashboardHeader() {
  const shop = useShopStore((s) => s.shop);
  const pathname = usePathname();
  const router = useRouter();
  const signOut = useAuthStore((s) => s.signOut);
  const [leaving, setLeaving] = useState(false);

  const logout = async () => {
    setLeaving(true);
    try {
      await authApi.logout();
    } finally {
      signOut();
      useShopStore.setState({ shop: null, dirty: false });
      router.replace("/login");
    }
  };

  return (
    <header className="border-b border-border bg-background">
      <div className="mx-auto flex h-14 w-full max-w-[1280px] items-center gap-6 px-4 md:px-6">
        <div className="flex items-baseline gap-2">
          <GuardedLink href="/" className="text-h3 font-semibold tracking-tight text-text-primary">
            MkClou
          </GuardedLink>
          <span className="hidden text-caption text-text-tertiary sm:inline">卖家中心</span>
        </div>
        {shop && (
          <nav className="flex items-center gap-1" aria-label="卖家中心导航">
            {nav.map((item) => {
              const active = item.href === "/dashboard" ? pathname === item.href : pathname.startsWith(item.href);
              return (
                <GuardedLink
                  key={item.href}
                  href={item.href}
                  className={cn(
                    "rounded-md px-3 py-1.5 font-medium transition-colors hover:bg-muted hover:text-text-primary",
                    active ? "text-text-primary" : "text-text-tertiary",
                  )}
                >
                  <span aria-current={active ? "page" : undefined}>{item.label}</span>
                </GuardedLink>
              );
            })}
          </nav>
        )}
        <div className="ml-auto flex items-center gap-2">
          {shop && (
            <Button variant="ghost" size="sm" asChild className="hidden sm:inline-flex">
              <a href={`/s/${shop.slug}`} target="_blank" rel="noopener noreferrer">
                <ShopAvatar shop={shop} size="sm" />
                查看店铺
                <ExternalLink strokeWidth={1.5} data-icon="inline-end" />
              </a>
            </Button>
          )}
          <Button variant="ghost" size="sm" onClick={logout} disabled={leaving} aria-label="退出登录">
            <LogOut strokeWidth={1.5} />
            <span className="hidden sm:inline">{leaving ? "退出中…" : "退出登录"}</span>
          </Button>
        </div>
      </div>
      <UnsavedChangesDialog />
    </header>
  );
}

function UnsavedChangesDialog() {
  const pendingHref = useShopStore((s) => s.pendingHref);
  const setPendingHref = useShopStore((s) => s.setPendingHref);
  const setDirty = useShopStore((s) => s.setDirty);
  const router = useRouter();

  return (
    <Dialog open={pendingHref !== null} onOpenChange={(open) => !open && setPendingHref(null)}>
      <DialogContent>
        <DialogTitle>放弃未保存的修改？</DialogTitle>
        <DialogDescription>离开后，你对店铺设置所做的修改将不会保存。</DialogDescription>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">继续编辑</Button>
          </DialogClose>
          <Button
            variant="destructive"
            onClick={() => {
              const href = pendingHref;
              setDirty(false);
              setPendingHref(null);
              if (href) router.push(href);
            }}
          >
            放弃修改
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
