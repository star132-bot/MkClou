import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = { title: "隐私政策" };

// 占位页面：正式条款在上线前撰写（PRD 总览 4.1、立项说明第 9 节合规风险）
export default function Page() {
  return (
    <main className="mx-auto flex w-full max-w-[720px] flex-1 flex-col gap-4 px-6 py-16">
      <Link href="/" className="text-h3 font-semibold tracking-tight text-text-primary">
        MkClou
      </Link>
      <h1 className="mt-8 text-h1">隐私政策</h1>
      <p className="text-body-lg">正式内容正在撰写中，将在平台上线前发布。</p>
    </main>
  );
}
