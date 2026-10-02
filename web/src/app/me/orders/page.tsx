import { Package } from "lucide-react";
import Link from "next/link";

import { Button } from "@/components/ui/button";

// 我的订单（PRD MKT-07）：依赖订单模块（PRD 04），完成后展示用本账号邮箱下单的全部订单
export default function OrdersPage() {
  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-h2">我的订单</h1>
      <div className="flex flex-col items-center gap-4 rounded-lg border border-dashed border-border px-6 py-16 text-center">
        <Package className="size-8 text-text-tertiary" strokeWidth={1.5} aria-hidden />
        <p>结账与订单功能即将开放。开放后，用你的账号邮箱下的订单都会显示在这里。</p>
        <Button variant="outline" asChild>
          <Link href="/">去逛逛</Link>
        </Button>
      </div>
    </div>
  );
}
