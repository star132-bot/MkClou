"use client";

import { useEffect, useState } from "react";

import { ShopHome } from "@/components/storefront/shop-home";
import { ShopThemeScope } from "@/components/storefront/shop-theme-scope";
import { PREVIEW_DATA, PREVIEW_READY, type PreviewMessage } from "@/lib/shop/preview";

/**
 * 店铺设置页右侧的实时预览，在 iframe 中渲染，使桌面 / 手机的响应式断点与真实页面一致。
 * 数据由父页面通过 postMessage 传入（只接受同源消息），不请求接口、不保存任何内容。
 */
export default function ShopPreviewPage() {
  const [data, setData] = useState<PreviewMessage | null>(null);

  useEffect(() => {
    const onMessage = (e: MessageEvent) => {
      if (e.origin !== window.location.origin || e.data?.type !== PREVIEW_DATA) return;
      setData(e.data as PreviewMessage);
    };
    window.addEventListener("message", onMessage);
    window.parent.postMessage({ type: PREVIEW_READY }, window.location.origin);
    return () => window.removeEventListener("message", onMessage);
  }, []);

  if (!data) return <div className="flex-1 bg-background" />;
  return (
    // inert：预览中的链接和按钮不可点击，避免在 iframe 中跳转
    <div inert className="flex min-h-full flex-1 flex-col">
      <ShopThemeScope theme={data.shop.theme}>
        <ShopHome shop={data.shop} products={data.products} />
      </ShopThemeScope>
    </div>
  );
}
