import type { Metadata } from "next";

import { AuthGuard } from "@/components/dashboard/auth-guard";

export const metadata: Metadata = {
  title: "创建店铺",
  robots: { index: false, follow: false },
};

export default function OnboardingLayout({ children }: LayoutProps<"/onboarding">) {
  return <AuthGuard>{children}</AuthGuard>;
}
