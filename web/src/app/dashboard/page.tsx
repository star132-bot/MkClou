"use client";

import { LogOut } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { authApi } from "@/lib/auth/api";
import { useAuthStore } from "@/lib/auth/store";

// 临时后台首页：验证登录流程。正式的概览页见 PRD DSB-01。
export default function DashboardPage() {
  const merchant = useAuthStore((s) => s.merchant);
  const signOut = useAuthStore((s) => s.signOut);
  const router = useRouter();
  const [leaving, setLeaving] = useState(false);

  const logout = async () => {
    setLeaving(true);
    try {
      await authApi.logout();
    } finally {
      signOut();
      router.replace("/login");
    }
  };

  return (
    <div className="mx-auto flex w-full max-w-[1080px] flex-col gap-8 px-6 py-10">
      <header className="flex items-center justify-between">
        <span className="text-h3 font-semibold tracking-tight text-text-primary">MkClou</span>
        <Button variant="ghost" size="sm" onClick={logout} disabled={leaving}>
          <LogOut strokeWidth={1.5} data-icon="inline-start" />
          {leaving ? "退出中…" : "退出登录"}
        </Button>
      </header>

      <section className="flex flex-col gap-2">
        <h1 className="text-h1">欢迎回来{merchant?.nickname ? `，${merchant.nickname}` : ""}</h1>
        <p>当前账号：{merchant?.email}</p>
      </section>

      <section className="rounded-lg border border-border p-6">
        <h2 className="text-h3">接下来</h2>
        <p className="mt-2">店铺、商品、订单等功能正在开发中。</p>
      </section>
    </div>
  );
}
