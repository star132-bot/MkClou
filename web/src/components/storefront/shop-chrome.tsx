import { ChevronLeft, Mail } from "lucide-react";
import Link from "next/link";

import { Button } from "@/components/ui/button";
import type { PublicShop } from "@/lib/storefront/types";
import { cn } from "@/lib/utils";

/** 店铺头像：无头像时用店铺名首字（PRD SHOP-01）。尺寸与字号对应设计规范的字号档位。 */
const avatarSizes = {
  sm: "size-6 text-caption",
  md: "size-8 text-body",
  lg: "size-12 text-h3",
};

export function ShopAvatar({
  shop,
  size = "md",
}: {
  shop: Pick<PublicShop, "name" | "avatarUrl">;
  size?: keyof typeof avatarSizes | "xl";
}) {
  if (shop.avatarUrl) {
    return (
      // eslint-disable-next-line @next/next/no-img-element -- 对象存储中的头像，尺寸很小，无需 next/image 优化
      <img
        src={shop.avatarUrl}
        alt=""
        className={cn("shrink-0 rounded-full border border-border object-cover", size === "xl" ? "size-20" : avatarSizes[size])}
      />
    );
  }
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center justify-center rounded-full bg-brand font-semibold text-brand-foreground",
        size === "xl" ? "size-20 text-h1" : avatarSizes[size],
      )}
      aria-hidden
    >
      {shop.name.slice(0, 1)}
    </span>
  );
}

/** 沙箱模式提示条（PRD 06 全局要求）。 */
export function TestModeBanner() {
  return (
    <div className="bg-warning/10 px-4 py-2 text-center text-caption text-warning" role="status">
      测试模式：付款不会产生真实扣费
    </div>
  );
}

/** 买家页顶栏：返回店铺。 */
export function ShopTopBar({ shop }: { shop: PublicShop }) {
  return (
    <header className="border-b border-border">
      <div className="mx-auto flex h-14 max-w-[1080px] items-center px-4 md:px-6">
        <Link
          href={`/s/${shop.slug}`}
          className="-ml-2 inline-flex items-center gap-2 rounded-md px-2 py-1.5 text-text-primary transition-colors hover:bg-muted"
        >
          <ChevronLeft className="size-4 text-text-tertiary" strokeWidth={1.5} aria-hidden />
          <ShopAvatar shop={shop} size="sm" />
          <span className="font-medium">{shop.name}</span>
        </Link>
      </div>
    </header>
  );
}

/** 商品页底部的店主信息卡片。 */
export function SellerCard({ shop }: { shop: PublicShop }) {
  return (
    <section className="flex flex-col gap-4 rounded-lg border border-border p-6 sm:flex-row sm:items-center">
      <ShopAvatar shop={shop} size="lg" />
      <div className="min-w-0 flex-1">
        <p className="text-body-lg font-medium text-text-primary">{shop.name}</p>
        <p className="mt-1">{shop.description}</p>
      </div>
      <div className="flex gap-2">
        <Button variant="outline" asChild>
          <a href={`mailto:${shop.contactEmail}`}>
            <Mail strokeWidth={1.5} data-icon="inline-start" />
            联系店主
          </a>
        </Button>
        <Button variant="outline" asChild>
          <Link href={`/s/${shop.slug}`}>查看店铺</Link>
        </Button>
      </div>
    </section>
  );
}

/** 买家页页脚（PRD SF-07）。 */
export function StoreFooter({ reportHref }: { reportHref: string }) {
  return (
    <footer className="border-t border-border">
      <div className="mx-auto flex max-w-[1080px] flex-wrap items-center justify-between gap-4 px-4 py-8 text-caption text-text-tertiary md:px-6">
        <Link href="/" className="transition-colors hover:text-text-primary">
          Powered by <span className="font-medium">MkClou</span>
        </Link>
        <nav className="flex gap-4">
          <Link href={reportHref} className="transition-colors hover:text-text-primary">
            举报
          </Link>
          <Link href="/terms" className="transition-colors hover:text-text-primary">
            用户协议
          </Link>
          <Link href="/privacy" className="transition-colors hover:text-text-primary">
            隐私政策
          </Link>
        </nav>
      </div>
    </footer>
  );
}
