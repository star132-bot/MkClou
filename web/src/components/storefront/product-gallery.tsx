"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";
import Image from "next/image";
import { useState } from "react";

import type { CardRatio, ProductImage } from "@/lib/storefront/types";
import { cn } from "@/lib/utils";

const ratioClass: Record<CardRatio, string> = {
  "4:3": "aspect-[4/3]",
  "16:9": "aspect-video",
  "1:1": "aspect-square",
};

interface ProductGalleryProps {
  images: ProductImage[];
  ratio: CardRatio;
}

/** 商品封面图集：主图 + 缩略图导航，支持键盘左右切换与手机端滑动。 */
export function ProductGallery({ images, ratio }: ProductGalleryProps) {
  const [active, setActive] = useState(0);
  const [touchStartX, setTouchStartX] = useState<number | null>(null);
  const count = images.length;
  const go = (next: number) => setActive((next + count) % count);

  return (
    <div className="flex flex-col gap-3">
      <div
        className={cn(
          "group relative overflow-hidden rounded-lg border border-border bg-muted",
          ratioClass[ratio],
        )}
        onKeyDown={(e) => {
          if (e.key === "ArrowLeft") go(active - 1);
          if (e.key === "ArrowRight") go(active + 1);
        }}
        onTouchStart={(e) => setTouchStartX(e.touches[0].clientX)}
        onTouchEnd={(e) => {
          if (touchStartX === null) return;
          const dx = e.changedTouches[0].clientX - touchStartX;
          if (Math.abs(dx) > 40) go(active + (dx < 0 ? 1 : -1));
          setTouchStartX(null);
        }}
        role="region"
        aria-roledescription="图片轮播"
        aria-label={`商品图片，第 ${active + 1} 张，共 ${count} 张`}
        tabIndex={0}
      >
        {images.map((img, i) => (
          <Image
            key={img.url}
            src={img.url}
            alt={img.alt}
            fill
            priority={i === 0}
            sizes="(min-width: 1024px) 620px, 100vw"
            unoptimized // 演示用 SVG；接入对象存储后改为优化后的图片
            className={cn(
              "object-cover transition-opacity duration-200 ease-out",
              i === active ? "opacity-100" : "opacity-0",
            )}
            aria-hidden={i !== active}
          />
        ))}

        {count > 1 && (
          <>
            <GalleryArrow direction="prev" onClick={() => go(active - 1)} />
            <GalleryArrow direction="next" onClick={() => go(active + 1)} />
            <span className="absolute right-3 bottom-3 rounded-full bg-black/50 px-2 py-0.5 text-caption text-white tabular-nums md:hidden">
              {active + 1} / {count}
            </span>
          </>
        )}
      </div>

      {count > 1 && (
        <div className="flex gap-3" role="tablist" aria-label="选择图片">
          {images.map((img, i) => (
            <button
              key={img.url}
              type="button"
              role="tab"
              aria-selected={i === active}
              aria-label={`查看第 ${i + 1} 张图片`}
              onClick={() => setActive(i)}
              className={cn(
                "relative w-20 overflow-hidden rounded-md border-2 bg-muted transition-[border-color,opacity] duration-150",
                ratioClass[ratio],
                i === active ? "border-text-primary" : "border-transparent opacity-70 hover:opacity-100",
              )}
            >
              <Image src={img.url} alt="" fill sizes="80px" unoptimized className="object-cover" />
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

function GalleryArrow({ direction, onClick }: { direction: "prev" | "next"; onClick: () => void }) {
  const Icon = direction === "prev" ? ChevronLeft : ChevronRight;
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={direction === "prev" ? "上一张" : "下一张"}
      className={cn(
        "absolute top-1/2 hidden size-9 -translate-y-1/2 items-center justify-center rounded-full bg-background/90 text-text-primary shadow-md",
        "opacity-0 transition-opacity duration-150 group-hover:opacity-100 focus-visible:opacity-100 md:flex",
        direction === "prev" ? "left-3" : "right-3",
      )}
    >
      <Icon className="size-5" strokeWidth={1.5} />
    </button>
  );
}
