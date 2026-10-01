import { createCn } from "cn/config";

/**
 * 合并 Tailwind 类名。
 *
 * 必须注册设计规范中的自定义字号（docs/design-system.md 4.2）：否则 `text-body` 这类类名
 * 会被误判为文字颜色，与 `text-primary-foreground` 等颜色类冲突而被删除。
 * 所有组件都要从这里导入 cn，不要直接从 "cn" 包导入（ESLint 已限制）。
 */
export const cn = createCn({
  extend: {
    classGroups: {
      "font-size": [{ text: ["display", "h1", "h2", "h3", "body-lg", "body", "caption", "price"] }],
    },
  },
});
