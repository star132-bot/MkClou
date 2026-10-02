import { ArrowRight, Flame, Sparkles } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";

import { FavoriteButton } from "@/components/market/favorite-button";
import { MarketCard, MarketGrid } from "@/components/market/market-card";
import { SiteFooter } from "@/components/site/site-footer";
import { SiteHeader } from "@/components/site/site-header";
import { Button } from "@/components/ui/button";
import { CATEGORIES } from "@/lib/product/meta";
import { fetchMarketHome } from "@/lib/storefront/server";
import type { MarketCard as MarketCardData } from "@/lib/storefront/types";

export const metadata: Metadata = {
  title: { absolute: "MkClou · 发现独立创作者的数字作品" },
};

// 商城首页（PRD MKT-01）：未登录也可访问，服务端渲染
export default async function HomePage() {
  const { hot, latest } = await fetchMarketHome();
  const empty = hot.length === 0 && latest.length === 0;

  return (
    <div className="flex flex-1 flex-col">
      <SiteHeader />
      <main className="mx-auto flex w-full max-w-[1200px] flex-1 flex-col gap-14 px-4 pt-10 pb-20 md:px-6 md:pt-14">
        <section className="flex flex-col gap-6">
          <div className="flex flex-col gap-3">
            <h1 className="text-h1 md:text-display">发现独立创作者的数字作品</h1>
            <p className="max-w-[560px] text-body-lg">模板、素材、教程、小工具……付款后自动交付，钱直接给到创作者本人。</p>
          </div>
          <nav aria-label="商品分类" className="-mx-4 flex gap-2 overflow-x-auto px-4 pb-1 [scrollbar-width:none] md:mx-0 md:flex-wrap md:px-0">
            <CategoryChip href="/search" label="全部" />
            {CATEGORIES.map((c) => (
              <CategoryChip key={c.value} href={`/search?category=${c.value}`} label={c.label} />
            ))}
          </nav>
        </section>

        {empty ? (
          <div className="flex flex-col items-center gap-4 rounded-lg border border-dashed border-border px-6 py-20 text-center">
            <Sparkles className="size-8 text-text-tertiary" strokeWidth={1.5} aria-hidden />
            <p className="text-body-lg">商城正在上新，成为第一批卖家</p>
            <Button asChild>
              <Link href="/onboarding">我要开店</Link>
            </Button>
          </div>
        ) : (
          <>
            <ProductSection title="热门商品" icon={<Flame className="size-5 text-danger" strokeWidth={1.5} aria-hidden />} href="/search?sort=sales" items={hot} eager />
            <ProductSection title="最新上架" icon={<Sparkles className="size-5 text-brand" strokeWidth={1.5} aria-hidden />} href="/search?sort=latest" items={latest} />
          </>
        )}

        <section className="flex flex-col items-start gap-4 rounded-lg border border-border bg-surface p-6 md:flex-row md:items-center md:justify-between md:p-8">
          <div className="flex flex-col gap-1">
            <h2 className="text-h3">你也有作品可以卖？</h2>
            <p>几分钟开一家有品牌感的店。买家付款直接进你的支付宝，平台不抽成。</p>
          </div>
          <Button asChild>
            <Link href="/onboarding">
              我要开店
              <ArrowRight strokeWidth={1.5} data-icon="inline-end" />
            </Link>
          </Button>
        </section>
      </main>
      <SiteFooter />
    </div>
  );
}

function CategoryChip({ href, label }: { href: string; label: string }) {
  return (
    <Link
      href={href}
      className="inline-flex h-9 shrink-0 items-center rounded-full border border-border px-4 font-medium text-text-primary transition-colors hover:bg-muted focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
    >
      {label}
    </Link>
  );
}

function ProductSection({
  title,
  icon,
  href,
  items,
  eager = false,
}: {
  title: string;
  icon: React.ReactNode;
  href: string;
  items: MarketCardData[];
  /** 首屏区块：前 4 张封面立即加载（最大内容绘制） */
  eager?: boolean;
}) {
  if (items.length === 0) return null;
  return (
    <section className="flex flex-col gap-6" aria-label={title}>
      <div className="flex items-center justify-between">
        <h2 className="flex items-center gap-2 text-h2">
          {icon}
          {title}
        </h2>
        <Link href={href} className="inline-flex items-center gap-1 font-medium text-text-secondary transition-colors hover:text-text-primary">
          查看更多
          <ArrowRight className="size-4" strokeWidth={1.5} aria-hidden />
        </Link>
      </div>
      <MarketGrid>
        {items.map((p, i) => (
          <MarketCard key={p.publicId} product={p} eager={eager && i < 4} action={<FavoriteButton publicId={p.publicId} />} />
        ))}
      </MarketGrid>
    </section>
  );
}
