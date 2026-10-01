import { ArrowRight, Download, ShieldCheck, Wallet } from "lucide-react";

import { Button } from "@/components/ui/button";

// 临时首页：验证设计 Token、字体与按钮组件。正式官网在营销页阶段实现（PRD 总览 4.1）。
const features = [
  {
    icon: Wallet,
    title: "钱直接进你的账户",
    description: "买家付款直达你自己的支付宝，平台不经手、不抽成。",
  },
  {
    icon: Download,
    title: "付款后自动交付",
    description: "文件、卡密、链接、文本，秒级交付并发送邮件。",
  },
  {
    icon: ShieldCheck,
    title: "内容不被随意转卖",
    description: "限时签名链接与下载次数限制，保护你的作品。",
  },
];

export default function Home() {
  return (
    <div className="flex flex-1 flex-col">
      <header className="mx-auto flex w-full max-w-[1200px] items-center justify-between px-6 py-5">
        <span className="text-h3 font-semibold tracking-tight text-text-primary">MkClou</span>
        <Button variant="ghost" size="sm">
          登录
        </Button>
      </header>

      <main className="mx-auto flex w-full max-w-[1200px] flex-1 flex-col justify-center px-6 py-24">
        <p className="mb-6 inline-flex w-fit items-center gap-2 rounded-full border border-border px-3 py-1 text-caption text-text-secondary">
          <span className="size-1.5 rounded-full bg-brand" aria-hidden />
          正在开发中
        </p>

        <h1 className="text-h2 font-semibold text-text-primary sm:text-h1 md:text-display">
          <span className="block">把你的作品，</span>
          <span className="block">变成一家有品牌感的小店</span>
        </h1>
        <p className="mt-6 max-w-[560px] text-body-lg text-text-secondary">
          几分钟开店，出售模板、素材、软件授权码等虚拟商品。买家付款后自动交付，你只需要专心创作。
        </p>

        <div className="mt-10 flex flex-wrap gap-3">
          <Button size="lg">
            免费开店
            <ArrowRight data-icon="inline-end" />
          </Button>
          <Button size="lg" variant="outline">
            查看示例店铺
          </Button>
        </div>

        <ul className="mt-24 grid gap-8 border-t border-border pt-12 md:grid-cols-3">
          {features.map(({ icon: Icon, title, description }) => (
            <li key={title}>
              <Icon className="size-5 text-text-primary" strokeWidth={1.5} aria-hidden />
              <h2 className="mt-4 text-body-lg">{title}</h2>
              <p className="mt-2">{description}</p>
            </li>
          ))}
        </ul>
      </main>

      <footer className="mx-auto w-full max-w-[1200px] px-6 py-8 text-caption text-text-tertiary">
        © 2026 MkClou
      </footer>
    </div>
  );
}
