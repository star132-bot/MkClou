"use client";

import { ArrowRight, Package, Palette, Wallet } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { ApiError, ErrorCode } from "@/lib/api/client";
import { shopApi } from "@/lib/shop/api";
import { useShopStore } from "@/lib/shop/store";

/** 卖家中心页面需要店铺：未开店时显示开店引导页（PRD MKT 4.8），不再强制跳转。 */
export function ShopGuard({ children }: { children: React.ReactNode }) {
  const shop = useShopStore((s) => s.shop);
  const setShop = useShopStore((s) => s.setShop);
  const [failed, setFailed] = useState(false);
  const [noShop, setNoShop] = useState(false);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    if (shop) return;
    let cancelled = false;
    shopApi
      .get()
      .then((s) => !cancelled && setShop(s))
      .catch((e) => {
        if (cancelled) return;
        if (e instanceof ApiError && e.code === ErrorCode.ShopNotCreated) setNoShop(true);
        else setFailed(true);
      });
    return () => {
      cancelled = true;
    };
  }, [shop, attempt, setShop]);

  const retry = () => {
    setFailed(false);
    setAttempt((n) => n + 1);
  };

  if (noShop) return <OpenShopIntro />;
  if (failed) {
    return (
      <div className="mx-auto flex max-w-[400px] flex-1 flex-col items-center justify-center gap-4 px-6 text-center">
        <h1 className="text-h3">店铺信息加载失败</h1>
        <p>可能是网络不稳定，请稍后重试。</p>
        <Button variant="outline" onClick={retry}>
          重新加载
        </Button>
      </div>
    );
  }
  if (!shop) {
    return (
      <div className="mx-auto flex w-full max-w-[1080px] flex-col gap-4 px-6 py-10" aria-busy="true">
        <span className="sr-only">正在加载</span>
        <div className="h-8 w-48 animate-pulse rounded-md bg-muted" />
        <div className="h-32 animate-pulse rounded-lg bg-muted" />
        <div className="h-32 animate-pulse rounded-lg bg-muted" />
      </div>
    );
  }
  return <>{children}</>;
}

const perks = [
  { icon: Wallet, title: "钱直接进你的账户", desc: "买家付款直达你自己的支付宝，平台不经手、不抽成" },
  { icon: Package, title: "付款后自动交付", desc: "文件、卡密、链接、文本，买家付款后秒级送达" },
  { icon: Palette, title: "有品牌感的店铺页", desc: "主题色、布局、深色模式，商品同时出现在 MkClou 商城" },
];

function OpenShopIntro() {
  return (
    <main className="mx-auto flex w-full max-w-[720px] flex-1 flex-col gap-8 px-4 py-12 md:py-20">
      <div className="flex flex-col gap-3">
        <h1 className="text-h1">开一家你的店</h1>
        <p className="text-body-lg">把你的模板、素材、教程或工具上架到 MkClou，几分钟就能开始售卖。</p>
      </div>
      <ul className="grid gap-4 sm:grid-cols-3">
        {perks.map((p) => (
          <li key={p.title} className="flex flex-col gap-2 rounded-lg border border-border bg-background p-5">
            <p.icon className="size-5 text-brand" strokeWidth={1.5} aria-hidden />
            <span className="font-medium text-text-primary">{p.title}</span>
            <span className="text-caption text-text-tertiary">{p.desc}</span>
          </li>
        ))}
      </ul>
      <Button size="lg" asChild className="self-start">
        <Link href="/onboarding">
          我要开店
          <ArrowRight strokeWidth={1.5} data-icon="inline-end" />
        </Link>
      </Button>
    </main>
  );
}

