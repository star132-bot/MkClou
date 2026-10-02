import type { Metadata } from "next";
import { notFound, permanentRedirect } from "next/navigation";

import { ShopClosed, ShopHome } from "@/components/storefront/shop-home";
import { ShopThemeScope } from "@/components/storefront/shop-theme-scope";
import { fetchPublicShop, fetchShopProducts } from "@/lib/storefront/server";

type Props = PageProps<"/s/[slug]">;

export async function generateMetadata(props: Props): Promise<Metadata> {
  const { slug } = await props.params;
  const shop = (await fetchPublicShop(slug))?.shop;
  if (!shop) return { title: "店铺不存在" };
  if (shop.status === "BANNED") return { title: "店铺已关闭", robots: { index: false } };
  return {
    title: { absolute: `${shop.name} · MkClou` },
    description: shop.description || `${shop.name}的小店`,
    openGraph: {
      title: shop.name,
      description: shop.description || undefined,
      images: shop.coverUrl ? [{ url: shop.coverUrl }] : shop.avatarUrl ? [{ url: shop.avatarUrl }] : undefined,
    },
  };
}

// 店铺首页（PRD SF-01，对外可见规则见 PRD 02 3.2）
export default async function ShopPage(props: Props) {
  const { slug } = await props.params;
  const result = await fetchPublicShop(slug);
  if (!result) notFound();
  // 修改链接后 90 天内旧链接永久跳转到新链接
  if (result.redirectTo) permanentRedirect(`/s/${result.redirectTo}`);

  const shop = result.shop;
  if (!shop) notFound();
  if (shop.status === "BANNED") return <ShopClosed />;

  const products = await fetchShopProducts(shop.slug);
  return (
    <ShopThemeScope theme={shop.theme}>
      <ShopHome shop={shop} products={products} />
    </ShopThemeScope>
  );
}
