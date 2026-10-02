import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "店铺预览",
  robots: { index: false, follow: false },
};

export default function PreviewLayout({ children }: LayoutProps<"/preview/shop">) {
  return children;
}
