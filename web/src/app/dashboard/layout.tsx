import type { Metadata } from "next";

import { AuthGuard } from "@/components/dashboard/auth-guard";
import { VerifyEmailBanner } from "@/components/dashboard/verify-email-banner";

export const metadata: Metadata = {
  title: "商家后台",
  robots: { index: false, follow: false },
};

// 完整的后台框架（侧边导航、顶栏）在商家后台模块中实现（PRD DSB-02）
export default function DashboardLayout({ children }: LayoutProps<"/dashboard">) {
  return (
    <AuthGuard>
      <div className="flex flex-1 flex-col">
        <VerifyEmailBanner />
        {children}
      </div>
    </AuthGuard>
  );
}
