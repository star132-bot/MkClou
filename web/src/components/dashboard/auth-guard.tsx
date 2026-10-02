"use client";

import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";

import { refreshSession } from "@/lib/api/client";
import { useAuthStore } from "@/lib/auth/store";

/**
 * 后台页面守卫：首次进入时用 HttpOnly Cookie 恢复登录状态；
 * 未登录则跳转登录页，并带上当前路径，登录后回到原页面（PRD AUTH-03）。
 */
export function AuthGuard({ children }: { children: React.ReactNode }) {
  const status = useAuthStore((s) => s.status);
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (status === "unknown") void refreshSession();
  }, [status]);

  useEffect(() => {
    if (status === "anonymous") router.replace(`/login?redirect=${encodeURIComponent(pathname)}`);
  }, [status, router, pathname]);

  // 其他标签页登出后，本页在下次请求时会收到 401 并被跳转；这里同时监听页面重新可见时校验一次
  useEffect(() => {
    const onVisible = () => {
      if (document.visibilityState === "visible" && useAuthStore.getState().status === "authenticated") {
        void refreshSession();
      }
    };
    document.addEventListener("visibilitychange", onVisible);
    return () => document.removeEventListener("visibilitychange", onVisible);
  }, []);

  if (status !== "authenticated") {
    return (
      <div className="flex flex-1 items-center justify-center" aria-busy="true" aria-live="polite">
        <span className="sr-only">正在加载</span>
        <div className="flex w-full max-w-[640px] flex-col gap-4 px-6">
          <div className="h-8 w-48 animate-pulse rounded-md bg-muted" />
          <div className="h-24 animate-pulse rounded-lg bg-muted" />
          <div className="h-24 animate-pulse rounded-lg bg-muted" />
        </div>
      </div>
    );
  }
  return <>{children}</>;
}
