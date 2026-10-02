import { Heart, ImageIcon } from "lucide-react";
import Image from "next/image";
import Link from "next/link";

import { Price } from "@/components/storefront/price";
import type { MarketCard as MarketCardData } from "@/lib/storefront/types";
import { cn } from "@/lib/utils";

/**
 * 商城商品卡片（设计规范 8.4）：封面统一 4:3，整卡可点击进入商品详情；显示所属店铺与收藏人数。
 * action 用于放置收藏按钮等卡片右上角的操作。
 */
export function MarketCard({ product, action, eager = false }: { product: MarketCardData; action?: React.ReactNode; eager?: boolean }) {
  const badge = product.soldOut ? "已售罄" : product.price === 0 ? "免费" : product.isNew ? "新品" : null;
  return (
    <div className="group relative flex flex-col gap-3">
      <Link
        href={`/s/${product.shop.slug}/p/${product.publicId}`}
        className="flex flex-col gap-3 rounded-lg outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-4"
      >
        <div className="relative aspect-[4/3] overflow-hidden rounded-lg border border-border bg-muted transition-shadow duration-200 group-hover:shadow-sm">
          {product.cover.url ? (
            <Image
              src={product.cover.url}
              alt={product.cover.alt}
              fill
              sizes="(min-width: 1024px) 280px, (min-width: 768px) 33vw, 50vw"
              unoptimized
              loading={eager ? "eager" : "lazy"}
              className={cn("object-cover transition-transform duration-200 ease-out group-hover:scale-[1.03]", product.soldOut && "opacity-60")}
            />
          ) : (
            <ImageIcon className="absolute inset-0 m-auto size-6 text-text-tertiary" strokeWidth={1.5} aria-hidden />
          )}
          {badge && (
            <span className="absolute top-2.5 left-2.5 rounded-sm bg-background/90 px-2 py-0.5 text-caption font-medium text-text-primary">
              {badge}
            </span>
          )}
        </div>
        <div className="flex flex-col gap-1">
          <h3 className="line-clamp-2 text-body font-medium text-text-primary md:text-body-lg">{product.name}</h3>
          <p className="truncate text-caption text-text-tertiary">{product.shop.name}</p>
          <div className="flex items-center justify-between gap-2">
            <Price price={product.price} originalPrice={product.originalPrice} size="sm" />
            {product.favoriteCount > 0 && (
              <span className="inline-flex items-center gap-1 text-caption text-text-tertiary tabular-nums">
                <Heart className="size-3" strokeWidth={1.5} aria-hidden />
                <span className="sr-only">收藏人数</span>
                {product.favoriteCount}
              </span>
            )}
          </div>
        </div>
      </Link>
      {action && <div className="absolute top-2 right-2">{action}</div>}
    </div>
  );
}

/** 商城商品网格：桌面 4 列 → 平板 3 列 → 手机 2 列（PRD MKT 4.2）。 */
export function MarketGrid({ children }: { children: React.ReactNode }) {
  return <div className="grid grid-cols-2 gap-x-4 gap-y-8 md:grid-cols-3 lg:grid-cols-4 lg:gap-x-6">{children}</div>;
}
