import { SearchX } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";

import { FavoriteButton } from "@/components/market/favorite-button";
import { MarketCard, MarketGrid } from "@/components/market/market-card";
import { SiteFooter } from "@/components/site/site-footer";
import { SiteHeader } from "@/components/site/site-header";
import { CATEGORIES, categoryLabel } from "@/lib/product/meta";
import { fetchMarketHome, fetchMarketSearch } from "@/lib/storefront/server";
import { cn } from "@/lib/utils";

type Props = PageProps<"/search">;

const PRICES = [
  { value: "", label: "全部价格" },
  { value: "free", label: "免费" },
  { value: "0-50", label: "¥50 以内" },
  { value: "50-200", label: "¥50～200" },
  { value: "200+", label: "¥200 以上" },
];

const SORTS = [
  { value: "", label: "综合" },
  { value: "sales", label: "销量" },
  { value: "latest", label: "最新" },
  { value: "price_asc", label: "价格从低到高" },
  { value: "price_desc", label: "价格从高到低" },
];

type Params = { q: string; category: string; price: string; sort: string; page: string };

async function readParams(props: Props): Promise<Params> {
  const sp = await props.searchParams;
  const one = (k: string) => {
    const v = sp[k];
    return (Array.isArray(v) ? v[0] : v)?.trim() ?? "";
  };
  return { q: one("q").slice(0, 50), category: one("category"), price: one("price"), sort: one("sort"), page: one("page") };
}

/** 生成保留其他条件、只修改部分条件的链接；修改筛选条件时回到第 1 页。 */
function hrefWith(p: Params, patch: Partial<Params>) {
  const next = { ...p, page: "", ...patch };
  const qs = new URLSearchParams(Object.entries(next).filter(([, v]) => v !== "") as [string, string][]);
  const s = qs.toString();
  return s ? `/search?${s}` : "/search";
}

export async function generateMetadata(props: Props): Promise<Metadata> {
  const p = await readParams(props);
  if (p.q) return { title: `“${p.q}”的搜索结果` };
  return { title: p.category ? categoryLabel(p.category) : "全部商品" };
}

// 搜索结果（PRD MKT-02）
export default async function SearchPage(props: Props) {
  const p = await readParams(props);
  const result = await fetchMarketSearch(Object.fromEntries(Object.entries(p).filter(([, v]) => v !== "")));
  const totalPages = Math.max(1, Math.ceil(result.total / result.pageSize));
  const title = p.q ? `“${p.q}”` : p.category ? categoryLabel(p.category) : "全部商品";
  const filtered = p.category !== "" || p.price !== "";

  // 没有结果时推荐热门商品
  const recommend = result.items.length === 0 ? (await fetchMarketHome()).hot : [];

  return (
    <div className="flex flex-1 flex-col">
      <SiteHeader query={p.q} />
      <main className="mx-auto flex w-full max-w-[1200px] flex-1 flex-col gap-6 px-4 pt-8 pb-20 md:px-6">
        <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
          <h1 className="text-h2">{title}</h1>
          <span className="text-text-tertiary tabular-nums">共 {result.total} 个商品</span>
        </div>

        <div className="flex flex-col gap-3 border-b border-border pb-4">
          <FilterRow label="分类">
            <Chip href={hrefWith(p, { category: "" })} active={p.category === ""}>
              全部
            </Chip>
            {CATEGORIES.map((c) => (
              <Chip key={c.value} href={hrefWith(p, { category: c.value })} active={p.category === c.value}>
                {c.label}
              </Chip>
            ))}
          </FilterRow>
          <FilterRow label="价格">
            {PRICES.map((o) => (
              <Chip key={o.value} href={hrefWith(p, { price: o.value })} active={p.price === o.value}>
                {o.label}
              </Chip>
            ))}
          </FilterRow>
          <FilterRow label="排序">
            {SORTS.map((o) => (
              <Chip key={o.value} href={hrefWith(p, { sort: o.value })} active={p.sort === o.value || (o.value === "" && !SORTS.some((s) => s.value === p.sort))}>
                {o.label}
              </Chip>
            ))}
          </FilterRow>
        </div>

        {result.items.length === 0 ? (
          <div className="flex flex-col gap-10">
            <div className="flex flex-col items-center gap-3 rounded-lg border border-dashed border-border px-6 py-16 text-center">
              <SearchX className="size-8 text-text-tertiary" strokeWidth={1.5} aria-hidden />
              <p className="text-body-lg">{p.q ? `没有找到“${p.q}”相关的商品，换个关键词试试` : "这里还没有商品"}</p>
              {filtered && (
                <Link href={hrefWith(p, { category: "", price: "" })} className="font-medium text-brand underline-offset-4 hover:underline">
                  清除筛选条件
                </Link>
              )}
            </div>
            {recommend.length > 0 && (
              <section className="flex flex-col gap-6">
                <h2 className="text-h3">看看大家在买什么</h2>
                <MarketGrid>
                  {recommend.map((c) => (
                    <MarketCard key={c.publicId} product={c} action={<FavoriteButton publicId={c.publicId} />} />
                  ))}
                </MarketGrid>
              </section>
            )}
          </div>
        ) : (
          <>
            <MarketGrid>
              {result.items.map((c) => (
                <MarketCard key={c.publicId} product={c} action={<FavoriteButton publicId={c.publicId} />} />
              ))}
            </MarketGrid>
            {totalPages > 1 && (
              <nav aria-label="分页" className="flex items-center justify-center gap-2 pt-4">
                {result.page > 1 && <PageLink href={hrefWith(p, { page: String(result.page - 1) })}>上一页</PageLink>}
                <span className="px-3 text-text-tertiary tabular-nums">
                  {result.page} / {totalPages}
                </span>
                {result.page < totalPages && <PageLink href={hrefWith(p, { page: String(result.page + 1) })}>下一页</PageLink>}
              </nav>
            )}
          </>
        )}
      </main>
      <SiteFooter />
    </div>
  );
}

function FilterRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-start gap-3">
      <span className="w-8 shrink-0 pt-1.5 text-caption text-text-tertiary">{label}</span>
      <div className="flex min-w-0 flex-1 flex-wrap gap-1.5">{children}</div>
    </div>
  );
}

function Chip({ href, active, children }: { href: string; active: boolean; children: React.ReactNode }) {
  return (
    <Link
      href={href}
      aria-current={active ? "true" : undefined}
      scroll={false}
      className={cn(
        "inline-flex h-8 items-center rounded-full px-3 font-medium transition-colors focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none",
        active ? "bg-text-primary text-background" : "text-text-secondary hover:bg-muted hover:text-text-primary",
      )}
    >
      {children}
    </Link>
  );
}

function PageLink({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <Link href={href} className="inline-flex h-9 items-center rounded-md border border-border px-4 font-medium text-text-primary transition-colors hover:bg-muted">
      {children}
    </Link>
  );
}
