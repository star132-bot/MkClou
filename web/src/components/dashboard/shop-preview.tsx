"use client";

import { Monitor, Smartphone } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";

import { Segmented } from "@/components/ui/segmented";
import { PREVIEW_DATA, PREVIEW_READY, SAMPLE_PRODUCTS } from "@/lib/shop/preview";
import type { PublicShop } from "@/lib/storefront/types";

const devices = {
  desktop: { width: 1280, height: 900 },
  mobile: { width: 375, height: 760 },
} as const;

type Device = keyof typeof devices;

/** 店铺页实时预览（PRD 02 4.2）：按真实宽度渲染 iframe 后等比缩小，可切换桌面 / 手机。 */
export function ShopPreview({ shop }: { shop: PublicShop }) {
  const [device, setDevice] = useState<Device>("desktop");
  const [width, setWidth] = useState(0);
  const boxRef = useRef<HTMLDivElement>(null);
  const frameRef = useRef<HTMLIFrameElement>(null);
  const shopRef = useRef(shop);

  const post = useCallback(() => {
    frameRef.current?.contentWindow?.postMessage(
      { type: PREVIEW_DATA, shop: shopRef.current, products: SAMPLE_PRODUCTS },
      window.location.origin,
    );
  }, []);

  useEffect(() => {
    shopRef.current = shop;
    post();
  }, [shop, post]);

  // iframe 加载完成后会发出 ready 消息，此时再发送一次数据
  useEffect(() => {
    const onMessage = (e: MessageEvent) => {
      if (e.origin === window.location.origin && e.data?.type === PREVIEW_READY) post();
    };
    window.addEventListener("message", onMessage);
    return () => window.removeEventListener("message", onMessage);
  }, [post]);

  useEffect(() => {
    const el = boxRef.current;
    if (!el) return;
    const ro = new ResizeObserver(([entry]) => setWidth(entry.contentRect.width));
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const size = devices[device];
  const scale = width ? Math.min(1, width / size.width) : 0;

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <span className="text-body font-medium text-text-primary">实时预览</span>
        <Segmented
          aria-label="预览设备"
          value={device}
          onValueChange={setDevice}
          options={[
            { value: "desktop", label: <Monitor className="size-4" strokeWidth={1.5} aria-label="桌面端" /> },
            { value: "mobile", label: <Smartphone className="size-4" strokeWidth={1.5} aria-label="手机端" /> },
          ]}
        />
      </div>
      <div ref={boxRef} className="w-full">
        <div
          className="mx-auto overflow-hidden rounded-lg border border-border bg-background"
          style={{ width: size.width * scale, height: size.height * scale }}
        >
          {scale > 0 && (
            <iframe
              ref={frameRef}
              src="/preview/shop"
              title="店铺页预览"
              tabIndex={-1}
              style={{ width: size.width, height: size.height, transform: `scale(${scale})`, transformOrigin: "0 0" }}
              className="border-0"
            />
          )}
        </div>
      </div>
      <p className="text-caption text-text-tertiary">预览中的商品为示例，买家看到的是你已上架的商品。</p>
    </div>
  );
}
