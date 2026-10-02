"use client";

import { Share2 } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { copyText } from "@/lib/clipboard";

/** 复制店铺链接（PRD SHOP-07）。 */
export function ShareButton({ path }: { path: string }) {
  const copy = async () => {
    const url = new URL(path, window.location.origin).toString();
    if (await copyText(url)) toast.success("链接已复制");
    else toast.error("复制失败，请手动复制链接", { description: url });
  };
  return (
    <Button variant="outline" size="sm" onClick={copy}>
      <Share2 strokeWidth={1.5} data-icon="inline-start" />
      分享
    </Button>
  );
}
