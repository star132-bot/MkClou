"use client";

import { Heart, Trash2 } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";
import { toast } from "sonner";

import { FavoriteButton } from "@/components/market/favorite-button";
import { MarketCard, MarketGrid } from "@/components/market/market-card";
import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api/client";
import { favoriteApi, type FavoriteItem } from "@/lib/market/api";
import { useFavoriteStore } from "@/lib/market/favorites";
import type { Page } from "@/lib/product/types";

// 我的收藏（PRD MKT-06）
export default function FavoritesPage() {
  const [page, setPage] = useState(1);
  const [data, setData] = useState<Page<FavoriteItem> | null>(null);
  const [failed, setFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const setFavorited = useFavoriteStore((s) => s.setFavorited);

  useEffect(() => {
    let cancelled = false;
    favoriteApi
      .list(page)
      .then((d) => {
        if (cancelled) return;
        d.items.forEach((it) => setFavorited(it.publicId, true));
        setData(d);
        setFailed(false);
      })
      .catch(() => !cancelled && setFailed(true));
    return () => {
      cancelled = true;
    };
  }, [page, attempt, setFavorited]);

  const drop = (publicId: string) =>
    setData((d) => (d ? { ...d, items: d.items.filter((i) => i.publicId !== publicId), total: d.total - 1 } : d));

  const remove = async (publicId: string) => {
    try {
      await favoriteApi.remove(publicId);
      setFavorited(publicId, false);
      drop(publicId);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : "移除失败，请稍后重试");
    }
  };

  const totalPages = data ? Math.max(1, Math.ceil(data.total / data.pageSize)) : 1;

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-baseline gap-3">
        <h1 className="text-h2">我的收藏</h1>
        {data && <span className="text-text-tertiary tabular-nums">{data.total} 个</span>}
      </div>

      {failed ? (
        <div className="flex flex-col items-center gap-3 rounded-lg border border-border px-6 py-16 text-center">
          <p>收藏加载失败，可能是网络不稳定。</p>
          <Button variant="outline" onClick={() => setAttempt((n) => n + 1)}>
            重试
          </Button>
        </div>
      ) : !data ? (
        <div className="grid grid-cols-2 gap-4 md:grid-cols-3" aria-busy="true">
          {[0, 1, 2].map((i) => (
            <div key={i} className="aspect-[4/3] animate-pulse rounded-lg bg-muted" />
          ))}
        </div>
      ) : data.items.length === 0 ? (
        <div className="flex flex-col items-center gap-4 rounded-lg border border-dashed border-border px-6 py-16 text-center">
          <Heart className="size-8 text-text-tertiary" strokeWidth={1.5} aria-hidden />
          <p>还没有收藏，看到喜欢的作品点一下爱心就能收藏</p>
          <Button variant="outline" asChild>
            <Link href="/">去逛逛</Link>
          </Button>
        </div>
      ) : (
        <>
          <MarketGrid>
            {data.items.map((it) =>
              it.available ? (
                <MarketCard
                  key={it.publicId}
                  product={it}
                  action={<FavoriteButton publicId={it.publicId} onChange={(on) => !on && drop(it.publicId)} />}
                />
              ) : (
                <div key={it.publicId} className="relative flex flex-col gap-3 opacity-60">
                  <div className="flex aspect-[4/3] items-center justify-center rounded-lg border border-border bg-muted">
                    <span className="rounded-sm bg-background px-2 py-0.5 text-caption font-medium text-text-primary">{it.reason}</span>
                  </div>
                  <p className="line-clamp-2 font-medium text-text-primary">{it.name}</p>
                  <p className="truncate text-caption text-text-tertiary">{it.shop.name}</p>
                  <Button variant="outline" size="sm" className="self-start" onClick={() => remove(it.publicId)}>
                    <Trash2 strokeWidth={1.5} data-icon="inline-start" />
                    移除
                  </Button>
                </div>
              ),
            )}
          </MarketGrid>
          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-3">
              <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage(page - 1)}>
                上一页
              </Button>
              <span className="text-text-tertiary tabular-nums">
                {page} / {totalPages}
              </span>
              <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>
                下一页
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
