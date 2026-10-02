"use client";

import { ImageIcon, PackageOpen, Search } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";
import { toast } from "sonner";

import { CreateProductDialog } from "@/components/product/create-product-dialog";
import { ProductStatusBadge } from "@/components/product/status-badge";
import { Price } from "@/components/storefront/price";
import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { ApiError, ErrorCode } from "@/lib/api/client";
import { formatTime } from "@/lib/format";
import { productApi } from "@/lib/product/api";
import { categoryLabel, deliveryLabel } from "@/lib/product/meta";
import type { MerchantProductStatus, Page, ProductListItem } from "@/lib/product/types";
import { cn } from "@/lib/utils";

const TABS: { value: MerchantProductStatus | ""; label: string }[] = [
  { value: "", label: "全部" },
  { value: "ON_SALE", label: "已上架" },
  { value: "DRAFT", label: "草稿" },
  { value: "OFF_SALE", label: "已下架" },
  { value: "PENDING_REVIEW", label: "待审核" },
];

const PAGE_SIZE = 20;

// 商品列表（PRD-01）
export default function ProductsPage() {
  const [status, setStatus] = useState<MerchantProductStatus | "">("");
  const [input, setInput] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [data, setData] = useState<Page<ProductListItem> | null>(null);
  const [failed, setFailed] = useState(false);
  const [reload, setReload] = useState(0);
  const [deleting, setDeleting] = useState<ProductListItem | null>(null);

  // 搜索防抖 300ms
  useEffect(() => {
    const t = setTimeout(() => {
      setQuery(input.trim());
      setPage(1);
    }, 300);
    return () => clearTimeout(t);
  }, [input]);

  useEffect(() => {
    let cancelled = false;
    productApi
      .list({ status, q: query, page, pageSize: PAGE_SIZE })
      .then((d) => {
        if (cancelled) return;
        setData(d);
        setFailed(false);
      })
      .catch(() => !cancelled && setFailed(true));
    return () => {
      cancelled = true;
    };
  }, [status, query, page, reload]);

  const refresh = () => setReload((n) => n + 1);

  const toggle = async (p: ProductListItem) => {
    try {
      if (p.status === "ON_SALE" || p.status === "PENDING_REVIEW") {
        await productApi.unpublish(p.publicId);
        toast.success("商品已下架");
      } else {
        const r = await productApi.publish(p.publicId);
        toast.success(r.status === "PENDING_REVIEW" ? "已提交审核，通过后自动上架" : "商品已上架");
      }
      refresh();
    } catch (e) {
      if (e instanceof ApiError && e.code === ErrorCode.PublishCheckFailed) {
        toast.error("还有内容需要完善", { description: "请进入编辑页查看未通过的检查项" });
      } else {
        toast.error(e instanceof ApiError ? e.message : "操作失败，请稍后重试");
      }
    }
  };

  const filtered = query !== "" || status !== "";
  const totalPages = data ? Math.max(1, Math.ceil(data.total / PAGE_SIZE)) : 1;

  return (
    <main className="mx-auto flex w-full max-w-[1280px] flex-col gap-6 px-4 py-8 md:px-8">
      <div className="flex items-center justify-between gap-4">
        <h1 className="text-h1">商品</h1>
        <CreateProductDialog />
      </div>

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex gap-1 overflow-x-auto rounded-md bg-muted p-1" role="tablist" aria-label="商品状态">
          {TABS.map((t) => (
            <button
              key={t.value}
              type="button"
              role="tab"
              aria-selected={status === t.value}
              onClick={() => {
                setStatus(t.value);
                setPage(1);
              }}
              className={cn(
                "h-7 shrink-0 rounded-sm px-3 font-medium transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring",
                status === t.value ? "bg-background text-text-primary shadow-sm" : "text-text-secondary hover:text-text-primary",
              )}
            >
              {t.label}
            </button>
          ))}
        </div>
        <div className="relative sm:w-64">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-text-tertiary" strokeWidth={1.5} aria-hidden />
          <Input value={input} onChange={(e) => setInput(e.target.value)} placeholder="搜索商品名称" aria-label="搜索商品名称" className="pl-9" />
        </div>
      </div>

      {failed ? (
        <div className="flex flex-col items-center gap-3 rounded-lg border border-border bg-background px-6 py-16 text-center">
          <p>商品列表加载失败，可能是网络不稳定。</p>
          <Button variant="outline" onClick={refresh}>
            重试
          </Button>
        </div>
      ) : !data ? (
        <div className="flex flex-col gap-2" aria-busy="true">
          <span className="sr-only">正在加载</span>
          {[0, 1, 2, 3].map((i) => (
            <div key={i} className="h-16 animate-pulse rounded-lg bg-muted" />
          ))}
        </div>
      ) : data.items.length === 0 ? (
        <div className="flex flex-col items-center gap-4 rounded-lg border border-dashed border-border bg-background px-6 py-16 text-center">
          <PackageOpen className="size-8 text-text-tertiary" strokeWidth={1.5} aria-hidden />
          {filtered ? (
            <>
              <p>没有找到匹配的商品</p>
              <Button
                variant="outline"
                onClick={() => {
                  setInput("");
                  setStatus("");
                }}
              >
                清除筛选
              </Button>
            </>
          ) : (
            <>
              <p>还没有商品，上架你的第一个作品吧</p>
              <CreateProductDialog />
            </>
          )}
        </div>
      ) : (
        <>
          <div className="overflow-x-auto rounded-lg border border-border bg-background">
            <table className="w-full min-w-[760px] text-left">
              <thead>
                <tr className="border-b border-border bg-surface text-caption font-medium text-text-tertiary">
                  <th className="px-4 py-3 font-medium">商品</th>
                  <th className="px-4 py-3 text-right font-medium">价格</th>
                  <th className="px-4 py-3 text-right font-medium">销量</th>
                  <th className="px-4 py-3 font-medium">状态</th>
                  <th className="px-4 py-3 font-medium">更新时间</th>
                  <th className="px-4 py-3 text-right font-medium">操作</th>
                </tr>
              </thead>
              <tbody>
                {data.items.map((p) => (
                  <tr key={p.publicId} className="border-b border-border transition-colors last:border-0 hover:bg-muted/50">
                    <td className="px-4 py-3">
                      <Link href={`/dashboard/products/${p.publicId}`} className="flex items-center gap-3 outline-none focus-visible:underline">
                        <span className="flex size-12 shrink-0 items-center justify-center overflow-hidden rounded-md border border-border bg-muted">
                          {p.cover ? (
                            // eslint-disable-next-line @next/next/no-img-element -- 48px 缩略图
                            <img src={p.cover.url} alt="" className="size-full object-cover" />
                          ) : (
                            <ImageIcon className="size-4 text-text-tertiary" strokeWidth={1.5} aria-hidden />
                          )}
                        </span>
                        <span className="flex min-w-0 flex-col gap-0.5">
                          <span className="truncate font-medium text-text-primary">{p.name}</span>
                          <span className="text-caption text-text-tertiary">
                            {deliveryLabel(p.deliveryType)} · {categoryLabel(p.category)}
                          </span>
                        </span>
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-right">
                      <Price price={p.price} originalPrice={p.originalPrice} size="sm" />
                    </td>
                    <td className="px-4 py-3 text-right tabular-nums">{p.salesCount}</td>
                    <td className="px-4 py-3">
                      <ProductStatusBadge status={p.status} />
                    </td>
                    <td className="px-4 py-3 text-caption text-text-tertiary">{formatTime(p.updatedAt)}</td>
                    <td className="px-4 py-3">
                      <div className="flex justify-end gap-1">
                        <Button variant="ghost" size="sm" asChild>
                          <Link href={`/dashboard/products/${p.publicId}`}>编辑</Link>
                        </Button>
                        {p.status !== "BANNED" && (
                          <Button variant="ghost" size="sm" onClick={() => toggle(p)}>
                            {p.status === "ON_SALE" || p.status === "PENDING_REVIEW" ? "下架" : "上架"}
                          </Button>
                        )}
                        {(p.status === "DRAFT" || p.status === "OFF_SALE") && (
                          <Button variant="ghost" size="sm" className="text-danger hover:text-danger" onClick={() => setDeleting(p)}>
                            删除
                          </Button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {totalPages > 1 && (
            <div className="flex items-center justify-end gap-3">
              <span className="text-caption text-text-tertiary tabular-nums">
                第 {page} / {totalPages} 页，共 {data.total} 个
              </span>
              <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage(page - 1)}>
                上一页
              </Button>
              <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>
                下一页
              </Button>
            </div>
          )}
        </>
      )}

      <DeleteDialog product={deleting} onClose={() => setDeleting(null)} onDeleted={refresh} />
    </main>
  );
}

function DeleteDialog({ product, onClose, onDeleted }: { product: ProductListItem | null; onClose: () => void; onDeleted: () => void }) {
  const [pending, setPending] = useState(false);
  const remove = async () => {
    if (!product) return;
    setPending(true);
    try {
      await productApi.remove(product.publicId);
      toast.success("商品已删除");
      onDeleted();
      onClose();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : "删除失败，请稍后重试");
    } finally {
      setPending(false);
    }
  };
  return (
    <Dialog open={product !== null} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogTitle>删除“{product?.name}”？</DialogTitle>
        <DialogDescription>删除后商品将从后台和店铺中移除，无法恢复。已有订单的商品无法删除，只能下架。</DialogDescription>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">取消</Button>
          </DialogClose>
          <Button variant="destructive" onClick={remove} disabled={pending}>
            {pending ? "删除中…" : "删除商品"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
