"use client";

import type { CSSProperties } from "react";
import { useSyncExternalStore } from "react";

import type { ShopTheme } from "@/lib/storefront/types";
import { cn } from "@/lib/utils";

const query = "(prefers-color-scheme: dark)";

function subscribe(onChange: () => void) {
  const mql = window.matchMedia(query);
  mql.addEventListener("change", onChange);
  return () => mql.removeEventListener("change", onChange);
}

/**
 * 应用店铺装修中的主题色与明暗模式（PRD SHOP-03）。
 * 主题色只覆盖 --brand（链接、焦点环、强调），主按钮保持近黑色（设计规范 3.2）；
 * 深色模式通过 .dark 类切换整套中性色 Token。“跟随买家系统”在服务端按浅色渲染，客户端再按系统设置切换。
 */
export function ShopThemeScope({
  theme,
  className,
  children,
}: {
  theme: ShopTheme;
  className?: string;
  children: React.ReactNode;
}) {
  const prefersDark = useSyncExternalStore(
    subscribe,
    () => window.matchMedia(query).matches,
    () => false,
  );
  const dark = theme.mode === "dark" || (theme.mode === "system" && prefersDark);
  return (
    <div
      style={{ "--brand": theme.color } as CSSProperties}
      className={cn("flex flex-1 flex-col bg-background text-text-secondary", dark && "dark", className)}
    >
      {children}
    </div>
  );
}
