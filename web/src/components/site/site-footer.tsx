import Link from "next/link";

/** 商城页脚。 */
export function SiteFooter() {
  return (
    <footer className="border-t border-border">
      <div className="mx-auto flex max-w-[1200px] flex-wrap items-center justify-between gap-4 px-4 py-8 text-caption text-text-tertiary md:px-6">
        <span>© 2026 MkClou · 钱直接给创作者，平台不抽成</span>
        <nav className="flex gap-4">
          <Link href="/onboarding" className="transition-colors hover:text-text-primary">
            我要开店
          </Link>
          <Link href="/terms" className="transition-colors hover:text-text-primary">
            用户协议
          </Link>
          <Link href="/privacy" className="transition-colors hover:text-text-primary">
            隐私政策
          </Link>
        </nav>
      </div>
    </footer>
  );
}
