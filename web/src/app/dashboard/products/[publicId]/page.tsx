"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";

import { ProductEditor } from "@/components/product/product-editor";
import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api/client";
import { productApi } from "@/lib/product/api";
import type { EditableProduct } from "@/lib/product/types";

// 编辑商品（PRD-02）
export default function EditProductPage() {
  const { publicId } = useParams<{ publicId: string }>();
  const [product, setProduct] = useState<EditableProduct | null>(null);
  const [error, setError] = useState<"notFound" | "failed" | null>(null);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let cancelled = false;
    productApi
      .get(publicId)
      .then((p) => !cancelled && setProduct(p))
      .catch((e) => !cancelled && setError(e instanceof ApiError && e.status === 404 ? "notFound" : "failed"));
    return () => {
      cancelled = true;
    };
  }, [publicId, attempt]);

  if (error) {
    return (
      <main className="mx-auto flex max-w-[400px] flex-1 flex-col items-center justify-center gap-4 px-6 py-24 text-center">
        <h1 className="text-h3">{error === "notFound" ? "商品不存在或已删除" : "商品加载失败"}</h1>
        {error === "failed" ? (
          <Button
            variant="outline"
            onClick={() => {
              setError(null);
              setAttempt((n) => n + 1);
            }}
          >
            重新加载
          </Button>
        ) : (
          <Button variant="outline" asChild>
            <Link href="/dashboard/products">返回商品列表</Link>
          </Button>
        )}
      </main>
    );
  }
  if (!product) {
    return (
      <main className="mx-auto flex w-full max-w-[880px] flex-col gap-6 px-4 py-8 md:px-8" aria-busy="true">
        <span className="sr-only">正在加载</span>
        <div className="h-10 animate-pulse rounded-md bg-muted" />
        <div className="h-48 animate-pulse rounded-lg bg-muted" />
        <div className="h-48 animate-pulse rounded-lg bg-muted" />
      </main>
    );
  }
  // key：切换到其他商品时重新初始化表单
  return <ProductEditor key={product.publicId} initial={product} />;
}
