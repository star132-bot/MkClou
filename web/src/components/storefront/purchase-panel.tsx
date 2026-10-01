"use client";

import { ArrowRight, CircleCheck, FileArchive, FileText, KeyRound, Link2, Minus, Plus } from "lucide-react";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { formatFileSize } from "@/lib/format";
import type { PurchaseState } from "@/lib/storefront/purchase-state";
import type { PublicProduct } from "@/lib/storefront/types";
import { cn } from "@/lib/utils";

import { Price } from "./price";

interface PurchasePanelProps {
  product: PublicProduct;
  state: PurchaseState;
}

/**
 * 购买面板：价格、交付方式、数量、购买按钮，以及滚动后出现的吸顶 / 底部购买条（PRD SF-02）。
 */
export function PurchasePanel({ product, state }: PurchasePanelProps) {
  const buyRef = useRef<HTMLDivElement>(null);
  const [barMode, setBarMode] = useState<"hidden" | "above" | "below">("hidden");

  const isCard = product.delivery.type === "CARD";
  const maxQty = Math.max(1, Math.min(product.maxPerOrder, product.stockHint ?? Infinity));
  const [qty, setQty] = useState(1);

  // 主购买按钮离开视口时显示购买条：滚过按钮（above）在桌面端显示吸顶条，手机端两种情况都显示底部栏
  useEffect(() => {
    const el = buyRef.current;
    if (!el) return;
    const observer = new IntersectionObserver(([entry]) => {
      if (entry.isIntersecting) setBarMode("hidden");
      else setBarMode(entry.boundingClientRect.top < 0 ? "above" : "below");
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  const handleBuy = () => {
    toast.info("演示模式：结账页将在订单模块完成后接入", {
      description: `${product.name} × ${qty}`,
    });
  };

  const buyButton = (className?: string) => (
    <Button size="lg" className={className} disabled={state.disabled} onClick={handleBuy}>
      {state.label}
      {!state.disabled && <ArrowRight data-icon="inline-end" />}
    </Button>
  );

  return (
    <>
      <div className="flex flex-col gap-6">
        <Price price={product.price} originalPrice={product.originalPrice} size="lg" />

        <dl className="flex flex-col gap-2 border-y border-border py-4">
          <DeliveryMeta product={product} />
          {isCard && product.stockHint !== null && (
            <div className="flex items-center gap-2 text-warning">
              <dt className="sr-only">库存</dt>
              <dd>仅剩 {product.stockHint} 件</dd>
            </div>
          )}
        </dl>

        {isCard && !state.disabled && (
          <QuantityStepper value={qty} max={maxQty} onChange={setQty} />
        )}

        <div ref={buyRef} className="flex flex-col gap-3">
          {buyButton("w-full")}
          {state.note && <p className="text-text-secondary">{state.note}</p>}
          <Link
            href="/orders/lookup"
            className="inline-flex w-fit items-center gap-1 text-text-secondary underline-offset-4 transition-colors hover:text-text-primary hover:underline"
          >
            已经买过？获取你的商品
            <ArrowRight className="size-4" strokeWidth={1.5} aria-hidden />
          </Link>
        </div>

        <ul className="flex flex-col gap-2 text-text-secondary">
          <TrustItem>付款后自动交付，同时发送到你的邮箱</TrustItem>
          <TrustItem>款项直接支付给店主，平台不经手</TrustItem>
        </ul>
      </div>

      {/* 桌面端：吸顶购买条 */}
      <div
        className={cn(
          "fixed inset-x-0 top-0 z-40 hidden border-b border-border bg-background/95 backdrop-blur-sm transition-transform duration-200 ease-out md:block",
          barMode === "above" ? "translate-y-0" : "-translate-y-full",
        )}
        aria-hidden={barMode !== "above"}
        inert={barMode !== "above"}
      >
        <div className="mx-auto flex h-16 max-w-[1080px] items-center gap-6 px-6">
          <p className="min-w-0 flex-1 truncate font-medium text-text-primary">{product.name}</p>
          <Price price={product.price} originalPrice={product.originalPrice} size="md" />
          {buyButton("h-9 px-4 text-body")}
        </div>
      </div>

      {/* 手机端：底部固定购买栏 */}
      <div
        className={cn(
          "fixed inset-x-0 bottom-0 z-40 border-t border-border bg-background/95 pb-[env(safe-area-inset-bottom)] backdrop-blur-sm transition-transform duration-200 ease-out md:hidden",
          barMode === "hidden" ? "translate-y-full" : "translate-y-0",
        )}
        aria-hidden={barMode === "hidden"}
        inert={barMode === "hidden"}
      >
        <div className="flex items-center gap-4 px-4 py-3">
          <Price price={product.price} originalPrice={product.originalPrice} size="md" className="flex-1" />
          {buyButton("min-w-36")}
        </div>
      </div>
    </>
  );
}

function DeliveryMeta({ product }: { product: PublicProduct }) {
  const { type, fileCount, totalSize } = product.delivery;
  const meta = {
    FILE: {
      icon: FileArchive,
      text: ["文件", fileCount && `${fileCount} 个`, totalSize && formatFileSize(totalSize)].filter(Boolean).join(" · "),
    },
    CARD: { icon: KeyRound, text: "卡密 · 付款后自动发放" },
    LINK: { icon: Link2, text: "链接 · 付款后查看" },
    TEXT: { icon: FileText, text: "文本 · 付款后查看" },
  }[type];
  const Icon = meta.icon;

  return (
    <div className="flex items-center gap-2">
      <dt className="sr-only">交付方式</dt>
      <Icon className="size-4 text-text-tertiary" strokeWidth={1.5} aria-hidden />
      <dd className="text-text-secondary">{meta.text}</dd>
    </div>
  );
}

function QuantityStepper({ value, max, onChange }: { value: number; max: number; onChange: (v: number) => void }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-text-secondary" id="qty-label">
        数量
      </span>
      <div className="flex items-center rounded-md border border-border" role="group" aria-labelledby="qty-label">
        <Button
          variant="ghost"
          size="icon"
          className="rounded-r-none"
          onClick={() => onChange(Math.max(1, value - 1))}
          disabled={value <= 1}
          aria-label="减少数量"
        >
          <Minus strokeWidth={1.5} />
        </Button>
        <output className="w-10 text-center font-medium text-text-primary tabular-nums" aria-live="polite">
          {value}
        </output>
        <Button
          variant="ghost"
          size="icon"
          className="rounded-l-none"
          onClick={() => onChange(Math.min(max, value + 1))}
          disabled={value >= max}
          aria-label="增加数量"
        >
          <Plus strokeWidth={1.5} />
        </Button>
      </div>
    </div>
  );
}

function TrustItem({ children }: { children: React.ReactNode }) {
  return (
    <li className="flex items-center gap-2">
      <CircleCheck className="size-4 shrink-0 text-success" strokeWidth={1.5} aria-hidden />
      {children}
    </li>
  );
}
