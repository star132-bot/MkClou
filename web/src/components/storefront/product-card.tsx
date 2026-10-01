import Image from "next/image";
import Link from "next/link";

import type { CardRatio, ProductSummary } from "@/lib/storefront/types";
import { cn } from "@/lib/utils";

import { Price } from "./price";

const ratioClass: Record<CardRatio, string> = {
  "4:3": "aspect-[4/3]",
  "16:9": "aspect-video",
  "1:1": "aspect-square",
};

/** 店铺商品卡片（设计规范 8.4）：整卡可点击，不放购买按钮。 */
export function ProductCard({ shopSlug, product, ratio }: { shopSlug: string; product: ProductSummary; ratio: CardRatio }) {
  const badge = product.soldOut ? "已售罄" : product.price === 0 ? "免费" : product.isNew ? "新品" : null;

  return (
    <Link
      href={`/s/${shopSlug}/p/${product.publicId}`}
      className="group flex flex-col gap-3 rounded-lg outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-4"
    >
      <div className={cn("relative overflow-hidden rounded-lg border border-border bg-muted", ratioClass[ratio])}>
        <Image
          src={product.cover.url}
          alt={product.cover.alt}
          fill
          sizes="(min-width: 1024px) 340px, (min-width: 640px) 50vw, 100vw"
          unoptimized
          className={cn(
            "object-cover transition-transform duration-200 ease-out group-hover:scale-[1.03]",
            product.soldOut && "opacity-60",
          )}
        />
        {badge && (
          <span className="absolute top-3 left-3 rounded-sm bg-background/90 px-2 py-0.5 text-caption font-medium text-text-primary">
            {badge}
          </span>
        )}
      </div>
      <div className="flex flex-col gap-1">
        <h3 className="line-clamp-2 text-body-lg font-medium">{product.name}</h3>
        <Price price={product.price} originalPrice={product.originalPrice} size="sm" />
      </div>
    </Link>
  );
}
