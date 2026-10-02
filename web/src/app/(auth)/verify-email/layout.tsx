import type { Metadata } from "next";

// page.tsx 是客户端组件，不能导出 metadata，页面标题在这里设置
export const metadata: Metadata = { title: "验证邮箱" };

export default function Layout({ children }: { children: React.ReactNode }) {
  return children;
}
