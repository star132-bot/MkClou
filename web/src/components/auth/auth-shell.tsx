import Link from "next/link";

/** 认证页外框：居中单列卡片，宽 400px（PRD 01 4.1）。 */
export function AuthShell({
  title,
  description,
  children,
  footer,
}: {
  title: string;
  description?: React.ReactNode;
  children: React.ReactNode;
  footer?: React.ReactNode;
}) {
  return (
    <div className="flex min-h-full flex-1 flex-col bg-surface">
      <header className="px-6 py-5">
        <Link href="/" className="text-h3 font-semibold tracking-tight text-text-primary">
          MkClou
        </Link>
      </header>
      <main className="flex flex-1 items-start justify-center px-4 pt-8 pb-16 sm:items-center sm:pt-0">
        <div className="w-full max-w-[400px]">
          <div className="rounded-lg border border-border bg-background p-6 sm:p-8">
            <h1 className="text-h2">{title}</h1>
            {description && <p className="mt-2 text-text-secondary">{description}</p>}
            <div className="mt-8">{children}</div>
          </div>
          {footer && <div className="mt-6 text-center text-text-secondary">{footer}</div>}
        </div>
      </main>
    </div>
  );
}

/** 页面级提示条，用于表单整体错误或成功提示。 */
export function FormAlert({ tone = "danger", children }: { tone?: "danger" | "success" | "info"; children: React.ReactNode }) {
  const tones = {
    danger: "bg-danger/10 text-danger",
    success: "bg-success/10 text-success",
    info: "bg-info/10 text-info",
  };
  return (
    <div role={tone === "danger" ? "alert" : "status"} className={`rounded-md px-3 py-2.5 ${tones[tone]}`}>
      {children}
    </div>
  );
}
