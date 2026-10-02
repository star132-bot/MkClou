import type { ShopStatus } from "@/lib/storefront/types";
import { cn } from "@/lib/utils";

// 设计规范 8.6：高 22px，功能色 10% 底 + 原色字
const statusMeta: Record<ShopStatus, { label: string; className: string }> = {
  OPEN: { label: "营业中", className: "bg-success/10 text-success" },
  PAUSED: { label: "暂停营业", className: "bg-warning/10 text-warning" },
  BANNED: { label: "已封禁", className: "bg-danger/10 text-danger" },
};

export function StatusBadge({ status }: { status: ShopStatus }) {
  const m = statusMeta[status];
  return (
    <span className={cn("inline-flex h-[22px] shrink-0 items-center rounded-sm px-2 text-caption font-medium", m.className)}>
      {m.label}
    </span>
  );
}
