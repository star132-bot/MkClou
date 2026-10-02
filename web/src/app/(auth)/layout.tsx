import type { Metadata } from "next";

export const metadata: Metadata = {
  // 认证页面不需要被搜索引擎收录
  robots: { index: false, follow: false },
};

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return children;
}
