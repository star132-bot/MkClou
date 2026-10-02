import { STATUS_META } from "@/lib/product/meta";
import type { MerchantProductStatus } from "@/lib/product/types";
import { cn } from "@/lib/utils";

/** 商品状态标签（设计规范 8.6）。 */
export function ProductStatusBadge({ status, className }: { status: MerchantProductStatus; className?: string }) {
  const m = STATUS_META[status];
  return (
    <span className={cn("inline-flex h-[22px] shrink-0 items-center rounded-sm px-2 text-caption font-medium", m.className, className)}>
      {m.label}
    </span>
  );
}
