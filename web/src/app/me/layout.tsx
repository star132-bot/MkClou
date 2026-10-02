import type { Metadata } from "next";

import { AuthGuard } from "@/components/dashboard/auth-guard";
import { MeNav } from "@/components/site/me-nav";
import { SiteFooter } from "@/components/site/site-footer";
import { SiteHeader } from "@/components/site/site-header";

export const metadata: Metadata = {
  title: "个人中心",
  robots: { index: false, follow: false },
};

// 个人中心（PRD MKT 4.7）：左侧导航，手机端为顶部标签
export default function MeLayout({ children }: LayoutProps<"/me">) {
  return (
    <div className="flex flex-1 flex-col">
      <SiteHeader />
      <AuthGuard>
        <div className="mx-auto flex w-full max-w-[1200px] flex-1 flex-col gap-6 px-4 pt-6 pb-20 md:flex-row md:gap-10 md:px-6 md:pt-10">
          <MeNav />
          <div className="min-w-0 flex-1">{children}</div>
        </div>
      </AuthGuard>
      <SiteFooter />
    </div>
  );
}
