import { Check } from "lucide-react";
import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";

import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";
import type { PublicProduct } from "@/lib/storefront/types";

/**
 * 商品详情的结构化区块，由平台统一排版（PRD 03 4.2 分组 3）：
 * 商品介绍（Markdown）→ 包含内容 → 适合谁 → 常见问题 → 购买须知。
 */
/** 按商品详情的版式渲染 Markdown（白名单：不渲染原始 HTML，见 SEC-05）。编辑页预览也使用它。 */
export function MarkdownBody({ children }: { children: string }) {
  return (
    <ReactMarkdown remarkPlugins={[remarkGfm]} components={markdownComponents}>
      {children}
    </ReactMarkdown>
  );
}

export function ProductContent({ product }: { product: PublicProduct }) {
  const { includes, audience, faqs, notice } = product.detail;

  return (
    <div className="flex flex-col gap-12">
      <Section title="商品介绍">
        <MarkdownBody>{product.descriptionMd}</MarkdownBody>
      </Section>

      {includes.length > 0 && (
        <Section title="包含内容">
          <ul className="grid gap-3 sm:grid-cols-2">
            {includes.map((item) => (
              <li key={item} className="flex items-start gap-3 rounded-lg border border-border p-4 text-body-lg text-text-primary">
                <Check className="mt-1 size-4 shrink-0 text-brand" strokeWidth={2} aria-hidden />
                {item}
              </li>
            ))}
          </ul>
        </Section>
      )}

      {audience && (
        <Section title="适合谁">
          <p className="text-body-lg">{audience}</p>
        </Section>
      )}

      {faqs.length > 0 && (
        <Section title="常见问题">
          <Accordion type="multiple" className="border-y border-border">
            {faqs.map((faq) => (
              <AccordionItem key={faq.q} value={faq.q} className="border-border">
                <AccordionTrigger className="py-4 text-body-lg font-medium text-text-primary hover:no-underline">
                  {faq.q}
                </AccordionTrigger>
                <AccordionContent className="pb-4 text-body-lg text-text-secondary">{faq.a}</AccordionContent>
              </AccordionItem>
            ))}
          </Accordion>
        </Section>
      )}

      {notice && (
        <Section title="购买须知">
          <p className="rounded-lg bg-surface p-4 text-text-secondary">{notice}</p>
        </Section>
      )}
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section>
      <h2 className="mb-4 text-h3">{title}</h2>
      {children}
    </section>
  );
}

// 商家 Markdown 的渲染样式。react-markdown 默认不渲染原始 HTML，并会过滤 javascript: 等危险链接（PRD SEC-05）。
const markdownComponents: Components = {
  p: ({ children }) => <p className="mb-4 text-body-lg last:mb-0">{children}</p>,
  h1: ({ children }) => <h3 className="mt-8 mb-3 text-h3">{children}</h3>,
  h2: ({ children }) => <h3 className="mt-8 mb-3 text-h3">{children}</h3>,
  h3: ({ children }) => <h3 className="mt-8 mb-3 text-h3">{children}</h3>,
  strong: ({ children }) => <strong className="font-semibold text-text-primary">{children}</strong>,
  ul: ({ children }) => <ul className="mb-4 list-disc space-y-2 pl-5 text-body-lg">{children}</ul>,
  ol: ({ children }) => <ol className="mb-4 list-decimal space-y-2 pl-5 text-body-lg">{children}</ol>,
  a: ({ href, children }) => (
    <a href={href} target="_blank" rel="noopener noreferrer nofollow" className="text-brand underline underline-offset-4">
      {children}
    </a>
  ),
  pre: ({ children }) => (
    <pre className="mb-4 overflow-x-auto rounded-lg border border-border bg-surface p-4 font-mono text-body text-text-primary">
      {children}
    </pre>
  ),
  code: ({ className, children }) =>
    className ? (
      <code className={className}>{children}</code>
    ) : (
      <code className="rounded-sm bg-muted px-1.5 py-0.5 font-mono text-text-primary">{children}</code>
    ),
};
