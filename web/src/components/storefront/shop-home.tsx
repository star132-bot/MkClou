import { ExternalLink, ImageIcon, Package } from "lucide-react";
import Link from "next/link";

import { SOCIAL_TYPES } from "@/lib/shop/validation";
import type { CardRatio, ProductSummary, PublicShop } from "@/lib/storefront/types";
import { cn } from "@/lib/utils";

import { Price } from "./price";
import { ProductCard } from "./product-card";
import { ShareButton } from "./share-button";
import { ShopAvatar, StoreFooter, TestModeBanner } from "./shop-chrome";

const socialLabel = Object.fromEntries(SOCIAL_TYPES.map((s) => [s.value, s.label]));

const thumbRatio: Record<CardRatio, string> = {
  "4:3": "aspect-[4/3]",
  "16:9": "aspect-video",
  "1:1": "aspect-square",
};

/**
 * 店铺首页内容（PRD SF-01）：封面 → 店铺信息 → 商品。
 * 买家端 /s/{slug} 与后台店铺设置的实时预览共用此组件，保证预览与实际效果一致。
 */
export function ShopHome({ shop, products }: { shop: PublicShop; products: ProductSummary[] }) {
  return (
    <>
      {shop.isTestMode && <TestModeBanner />}
      {shop.status === "PAUSED" && (
        <div className="bg-warning/10 px-4 py-2.5 text-center text-warning" role="status">
          <span className="font-medium">店铺暂停营业中</span>
          {shop.pauseNote && <span>：{shop.pauseNote}</span>}
          <span className="block text-caption sm:ml-2 sm:inline">商品可浏览，暂时无法购买</span>
        </div>
      )}

      <main className="mx-auto w-full max-w-[1080px] flex-1 px-4 pb-24 md:px-6">
        {shop.coverUrl && (
          <div className="mt-4 aspect-[3/1] overflow-hidden rounded-lg border border-border bg-muted md:mt-6">
            {/* eslint-disable-next-line @next/next/no-img-element -- 商家上传的封面，比例固定 3:1 */}
            <img src={shop.coverUrl} alt="" className="size-full object-cover" />
          </div>
        )}

        <header
          className={cn(
            "flex flex-col items-center gap-4 text-center md:flex-row md:items-start md:text-left",
            shop.coverUrl ? "pt-6" : "pt-10 md:pt-14",
          )}
        >
          <ShopAvatar shop={shop} size="xl" />
          <div className="flex min-w-0 flex-1 flex-col items-center gap-2 md:items-start">
            <h1 className="text-h1 break-all">{shop.name}</h1>
            {shop.description && <p className="max-w-[640px] text-left text-body-lg whitespace-pre-line">{shop.description}</p>}
            {shop.socialLinks.length > 0 && (
              <ul className="mt-1 flex flex-wrap justify-center gap-2 md:justify-start">
                {shop.socialLinks.map((l, i) => (
                  <li key={`${l.type}-${i}`}>
                    <a
                      href={l.url}
                      target="_blank"
                      rel="noopener noreferrer nofollow"
                      className="inline-flex h-7 items-center gap-1 rounded-sm bg-muted px-2.5 text-caption font-medium text-text-primary transition-colors hover:text-brand"
                    >
                      {socialLabel[l.type] ?? "链接"}
                      <ExternalLink className="size-3" strokeWidth={1.5} aria-hidden />
                    </a>
                  </li>
                ))}
              </ul>
            )}
          </div>
          <ShareButton path={`/s/${shop.slug}`} />
        </header>

        <section className="mt-12" aria-labelledby="products-heading">
          <h2 id="products-heading" className="mb-6 text-h3">
            全部商品{products.length > 0 && <span className="ml-1.5 text-text-tertiary tabular-nums">({products.length})</span>}
          </h2>
          {products.length === 0 ? (
            <div className="flex flex-col items-center gap-3 rounded-lg border border-dashed border-border px-6 py-16 text-center">
              <Package className="size-8 text-text-tertiary" strokeWidth={1.5} aria-hidden />
              <p>店主正在准备商品，敬请期待</p>
            </div>
          ) : shop.theme.layout === "list" ? (
            <ul className="flex flex-col divide-y divide-border rounded-lg border border-border">
              {products.map((p) => (
                <li key={p.publicId}>
                  <Link
                    href={`/s/${shop.slug}/p/${p.publicId}`}
                    className="flex items-center gap-4 p-4 transition-colors hover:bg-muted focus-visible:bg-muted focus-visible:outline-none"
                  >
                    <div className={cn("relative w-24 shrink-0 overflow-hidden rounded-md border border-border bg-muted sm:w-32", thumbRatio[shop.theme.cardRatio])}>
                      {p.cover.url ? (
                        // eslint-disable-next-line @next/next/no-img-element -- 列表缩略图
                        <img src={p.cover.url} alt={p.cover.alt} className="size-full object-cover" />
                      ) : (
                        <ImageIcon className="absolute inset-0 m-auto size-5 text-text-tertiary" strokeWidth={1.5} aria-hidden />
                      )}
                    </div>
                    <div className="flex min-w-0 flex-1 flex-col gap-1">
                      <h3 className="line-clamp-2 text-body-lg font-medium">{p.name}</h3>
                      {p.soldOut && <span className="text-caption text-text-tertiary">已售罄</span>}
                    </div>
                    <Price price={p.price} originalPrice={p.originalPrice} size="sm" />
                  </Link>
                </li>
              ))}
            </ul>
          ) : (
            <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
              {products.map((p) => (
                <ProductCard key={p.publicId} shopSlug={shop.slug} product={p} ratio={shop.theme.cardRatio} />
              ))}
            </div>
          )}
        </section>
      </main>

      <StoreFooter reportHref={`/report?shop=${shop.slug}`} />
    </>
  );
}

/** 已封禁店铺（PRD 02 3.2）。 */
export function ShopClosed() {
  return (
    <main className="mx-auto flex w-full max-w-[480px] flex-1 flex-col items-center justify-center gap-4 px-6 py-24 text-center">
      <h1 className="text-h2">该店铺已关闭</h1>
      <p>店铺因违反平台规则已被关闭，无法继续浏览和购买。</p>
      <Link href="/" className="font-medium text-brand underline-offset-4 hover:underline">
        返回 MkClou 首页
      </Link>
    </main>
  );
}
