import type { Metadata } from "next";
import Link from "next/link";
import { notFound, permanentRedirect } from "next/navigation";

import { FavoriteButton } from "@/components/market/favorite-button";
import { ProductCard } from "@/components/storefront/product-card";
import { ProductContent } from "@/components/storefront/product-content";
import { ProductGallery } from "@/components/storefront/product-gallery";
import { PurchasePanel } from "@/components/storefront/purchase-panel";
import { SellerCard, ShopTopBar, StoreFooter, TestModeBanner } from "@/components/storefront/shop-chrome";
import { ShopThemeScope } from "@/components/storefront/shop-theme-scope";
import { Button } from "@/components/ui/button";
import { getPurchaseState } from "@/lib/storefront/purchase-state";
import { fetchProductPage } from "@/lib/storefront/server";

type Props = PageProps<"/s/[slug]/p/[productId]">;

async function load(props: Props) {
  const { productId } = await props.params;
  return fetchProductPage(productId);
}

export async function generateMetadata(props: Props): Promise<Metadata> {
  const data = await load(props);
  if (!data) return { title: "商品不存在" };
  const { shop, product } = data;
  return {
    title: `${product.name} - ${shop.name}`,
    description: product.tagline,
    openGraph: {
      title: product.name,
      description: product.tagline,
      images: product.cover?.url ? [{ url: product.cover.url, width: product.cover.width, height: product.cover.height }] : undefined,
    },
  };
}

export default async function ProductPage(props: Props) {
  const data = await load(props);
  if (!data) notFound();

  const { shop, product, moreFromShop } = data;
  // 店铺修改过链接，或链接与商品所属店铺不一致时，跳转到正确的地址
  const { slug } = await props.params;
  if (slug !== shop.slug) permanentRedirect(`/s/${shop.slug}/p/${product.publicId}`);

  if (product.status !== "ON_SALE") {
    return (
      <ShopThemeScope theme={shop.theme}>
        <ShopTopBar shop={shop} />
        <main className="mx-auto flex w-full max-w-[480px] flex-1 flex-col items-center justify-center gap-4 px-6 py-24 text-center">
          <h1 className="text-h2">商品已下架</h1>
          <p>这个商品暂时无法购买，看看店铺里的其他商品吧。</p>
          <Button variant="outline" asChild className="mt-2">
            <Link href={`/s/${shop.slug}`}>返回店铺</Link>
          </Button>
        </main>
        <StoreFooter reportHref={`/report?productId=${product.publicId}`} />
      </ShopThemeScope>
    );
  }

  const purchaseState = getPurchaseState(shop, product);

  return (
    <ShopThemeScope theme={shop.theme}>
      {shop.isTestMode && <TestModeBanner />}
      <ShopTopBar shop={shop} />

      <main className="mx-auto w-full max-w-[1080px] flex-1 px-4 pt-6 pb-24 md:px-6 md:pt-10">
        {/* 手机端按 DOM 顺序：图片 → 购买面板 → 详情；桌面端购买面板固定在右列并跟随滚动 */}
        <div className="grid grid-cols-[minmax(0,1fr)] gap-8 md:grid-cols-[minmax(0,3fr)_minmax(0,2fr)] md:gap-x-12 md:gap-y-16">
          <div className="md:col-start-1 md:row-start-1">
            <ProductGallery images={product.images} ratio={shop.theme.cardRatio} />
          </div>

          <div className="flex flex-col gap-6 md:sticky md:top-8 md:col-start-2 md:row-span-2 md:row-start-1 md:self-start">
            <div className="flex flex-col gap-2">
              <h1 className="text-h2">{product.name}</h1>
              {product.tagline && <p className="text-body-lg">{product.tagline}</p>}
              <div className="mt-1 flex items-center gap-3">
                <FavoriteButton publicId={product.publicId} variant="inline" />
                {product.favoriteCount > 0 && (
                  <span className="text-caption text-text-tertiary tabular-nums">{product.favoriteCount} 人收藏</span>
                )}
              </div>
            </div>
            <PurchasePanel product={product} state={purchaseState} />
          </div>

          <div className="mt-8 flex flex-col gap-12 md:col-start-1 md:row-start-2 md:mt-0">
            <ProductContent product={product} />
            <SellerCard shop={shop} />
          </div>
        </div>

        {moreFromShop.length > 0 && (
          <section className="mt-20 border-t border-border pt-12">
            <h2 className="mb-6 text-h3">店铺里的其他商品</h2>
            <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
              {moreFromShop.map((p) => (
                <ProductCard key={p.publicId} shopSlug={shop.slug} product={p} ratio={shop.theme.cardRatio} />
              ))}
            </div>
          </section>
        )}
      </main>

      <StoreFooter reportHref={`/report?productId=${product.publicId}`} />
    </ShopThemeScope>
  );
}
