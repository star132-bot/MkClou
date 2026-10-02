import type { Metadata } from "next";

import { AuthGuard } from "@/components/dashboard/auth-guard";
import { DashboardHeader } from "@/components/dashboard/dashboard-header";
import { ShopGuard } from "@/components/dashboard/shop-guard";
import { VerifyEmailBanner } from "@/components/dashboard/verify-email-banner";

export const metadata: Metadata = {
  title: "商家后台",
  robots: { index: false, follow: false },
};

// 卖家中心。完整的侧边栏框架在商家后台模块中实现（PRD DSB-02）；未开店时 ShopGuard 显示开店引导页
export default function DashboardLayout({ children }: LayoutProps<"/dashboard">) {
  return (
    <AuthGuard>
      <div className="flex flex-1 flex-col bg-surface">
        <DashboardHeader />
        <VerifyEmailBanner />
        <ShopGuard>{children}</ShopGuard>
      </div>
    </AuthGuard>
  );
}
