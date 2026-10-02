"use client";

import { Check, ChevronDown, Copy, ExternalLink, Settings } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";

import { GuardedLink } from "@/components/dashboard/dashboard-header";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { ShopAvatar } from "@/components/storefront/shop-chrome";
import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api/client";
import { authApi } from "@/lib/auth/api";
import { copyText } from "@/lib/clipboard";
import { useAuthStore } from "@/lib/auth/store";
import { shopApi } from "@/lib/shop/api";
import { useShopStore } from "@/lib/shop/store";
import type { Onboarding } from "@/lib/shop/types";
import { cn } from "@/lib/utils";

const COLLAPSE_KEY = "mk:onboarding-collapsed";

function greeting() {
  const h = new Date().getHours();
  if (h < 6) return "夜深了";
  if (h < 12) return "早上好";
  if (h < 18) return "下午好";
  return "晚上好";
}

/** 复制店铺链接，同时完成新手清单的“分享店铺”一步（PRD SHOP-05：点击过复制链接即算完成）。 */
async function copyShopLink(slug: string) {
  const url = `${window.location.origin}/s/${slug}`;
  if (await copyText(url)) toast.success("链接已复制");
  else toast.error("复制失败，请手动复制链接", { description: url });
  await shopApi.markShared().catch(() => undefined); // 标记失败不影响复制
}

// 后台概览页（PRD DSB-01）。销售指标、趋势图与最近订单在订单与数据模块完成后加入。
export default function DashboardPage() {
  const merchant = useAuthStore((s) => s.merchant);
  const shop = useShopStore((s) => s.shop)!;
  const [onboarding, setOnboarding] = useState<Onboarding | null>(null);
  const [loadFailed, setLoadFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let cancelled = false;
    shopApi
      .onboarding()
      .then((o) => !cancelled && setOnboarding(o))
      .catch(() => !cancelled && setLoadFailed(true));
    return () => {
      cancelled = true;
    };
  }, [attempt]);

  const retry = () => {
    setLoadFailed(false);
    setAttempt((n) => n + 1);
  };

  const copy = async () => {
    await copyShopLink(shop.slug);
    setOnboarding((o) => (o ? { ...o, shared: true } : o));
  };

  return (
    <main className="mx-auto flex w-full max-w-[1080px] flex-col gap-6 px-4 py-8 md:px-8">
      <h1 className="text-h1">
        {greeting()}
        {merchant?.nickname ? `，${merchant.nickname}` : ""}
      </h1>

      {loadFailed ? (
        <div className="flex items-center justify-between gap-4 rounded-lg border border-border bg-background p-6">
          <p>新手清单加载失败。</p>
          <Button variant="outline" size="sm" onClick={retry}>
            重试
          </Button>
        </div>
      ) : onboarding ? (
        <OnboardingChecklist
          data={{ ...onboarding, emailVerified: onboarding.emailVerified || !!merchant?.emailVerified }}
          onCopy={copy}
        />
      ) : (
        <div className="h-64 animate-pulse rounded-lg bg-muted" aria-busy="true" />
      )}

      <section className="flex flex-col gap-4 rounded-lg border border-border bg-background p-6 sm:flex-row sm:items-center">
        <ShopAvatar shop={shop} size="lg" />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <h2 className="truncate text-h3">{shop.name}</h2>
            <StatusBadge status={shop.status} />
          </div>
          <p className="mt-1 truncate text-text-tertiary">/s/{shop.slug}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" size="sm" onClick={copy}>
            <Copy strokeWidth={1.5} data-icon="inline-start" />
            复制链接
          </Button>
          <Button variant="outline" size="sm" asChild>
            <a href={`/s/${shop.slug}`} target="_blank" rel="noopener noreferrer">
              <ExternalLink strokeWidth={1.5} data-icon="inline-start" />
              查看店铺
            </a>
          </Button>
          <Button variant="outline" size="sm" asChild>
            <GuardedLink href="/dashboard/shop">
              <Settings strokeWidth={1.5} data-icon="inline-start" />
              店铺设置
            </GuardedLink>
          </Button>
        </div>
      </section>
    </main>
  );
}

function OnboardingChecklist({ data, onCopy }: { data: Onboarding; onCopy: () => void }) {
  const [sending, setSending] = useState(false);

  const steps = [
    { done: data.emailVerified, title: "验证邮箱", desc: "验证后才能上架商品" },
    { done: data.paymentReady, title: "配置收款", desc: "绑定支付宝，买家付款直接进入你的账户" },
    { done: data.hasProduct, title: "上架第一个商品", desc: "文件、卡密、链接或文本都可以" },
    { done: data.shared, title: "分享你的店铺", desc: "把店铺链接发给你的读者和粉丝" },
  ];
  const doneCount = steps.filter((s) => s.done).length;
  const allDone = doneCount === steps.length;
  // 全部完成后可收起，收起状态记忆在本地
  const [collapsed, setCollapsed] = useState(() => {
    try {
      return allDone && localStorage.getItem(COLLAPSE_KEY) === "1";
    } catch {
      return false; // 存储不可用时保持展开
    }
  });

  const toggle = () => {
    setCollapsed((c) => {
      try {
        localStorage.setItem(COLLAPSE_KEY, c ? "0" : "1");
      } catch {
        // 忽略
      }
      return !c;
    });
  };

  const resend = async () => {
    setSending(true);
    try {
      await authApi.resendVerification();
      toast.success("验证邮件已发送", { description: "请检查收件箱和垃圾邮件文件夹" });
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : "发送失败，请稍后重试");
    } finally {
      setSending(false);
    }
  };

  const actions = [
    <Button key="email" variant="outline" size="sm" onClick={resend} disabled={sending}>
      {sending ? "发送中…" : "重新发送邮件"}
    </Button>,
    <Button key="pay" variant="outline" size="sm" disabled title="收款设置即将开放">
      去配置
    </Button>,
    <Button key="product" variant="outline" size="sm" asChild>
      <GuardedLink href="/dashboard/products">去上架</GuardedLink>
    </Button>,
    <Button key="share" variant="outline" size="sm" onClick={onCopy}>
      复制链接
    </Button>,
  ];

  return (
    <section className="rounded-lg border border-border bg-background" aria-labelledby="onboarding-heading">
      <div className="flex items-center gap-4 p-6">
        <div className="min-w-0 flex-1">
          <h2 id="onboarding-heading" className="text-h3">
            新手清单
          </h2>
          <p className="mt-1">{allDone ? "全部完成，你的店铺已准备就绪" : "完成以下步骤，开始你的第一笔销售"}</p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex gap-1" aria-hidden>
            {steps.map((s, i) => (
              <span key={i} className={cn("size-2 rounded-full", s.done ? "bg-success" : "bg-muted")} />
            ))}
          </div>
          <span className="text-body font-medium text-text-primary tabular-nums">
            {doneCount}/{steps.length}
          </span>
          {allDone && (
            <Button variant="ghost" size="icon-sm" onClick={toggle} aria-expanded={!collapsed} aria-label={collapsed ? "展开新手清单" : "收起新手清单"}>
              <ChevronDown className={cn("transition-transform duration-150", !collapsed && "rotate-180")} strokeWidth={1.5} />
            </Button>
          )}
        </div>
      </div>
      {!collapsed && (
        <ol className="divide-y divide-border border-t border-border">
          {steps.map((s, i) => (
            <li key={s.title} className="flex items-center gap-4 px-6 py-4">
              <span
                className={cn(
                  "flex size-6 shrink-0 items-center justify-center rounded-full text-caption font-medium",
                  s.done ? "bg-success/10 text-success" : "border border-border text-text-tertiary",
                )}
              >
                {s.done ? <Check className="size-3.5" strokeWidth={2} aria-label="已完成" /> : i + 1}
              </span>
              <div className="min-w-0 flex-1">
                <p className={cn("font-medium", s.done ? "text-text-tertiary line-through" : "text-text-primary")}>{s.title}</p>
                {!s.done && <p className="text-caption text-text-tertiary">{s.desc}</p>}
              </div>
              {!s.done && actions[i]}
            </li>
          ))}
        </ol>
      )}
    </section>
  );
}
