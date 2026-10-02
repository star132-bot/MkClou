import { SLUG_MAX, sanitizeSlug } from "./validation";

/**
 * 根据店铺名生成链接建议（PRD SHOP-01）：中文转为不带声调的拼音，英文和数字保留，
 * 不同片段之间用连字符分隔。例：“阿杰的工具铺” → ajiegongjupu，“AI 编程 Lab” → ai-biancheng-lab。
 * 拼音库约 300KB，按需加载，只在开店向导中使用。
 */
export async function slugFromName(name: string): Promise<string> {
  const text = name.replace(/的/g, "").trim();
  if (!text) return "";
  const { pinyin } = await import("pinyin-pro");
  const tokens = pinyin(text, { toneType: "none", type: "array", nonZh: "consecutive", v: true });
  // 连续的拼音音节合并为一个片段，非中文片段前后加分隔符
  const raw = tokens.map((t) => (/^[a-z]+$/.test(t) ? t : `-${t}-`)).join("");
  return sanitizeSlug(raw).replace(/-+$/, "").slice(0, SLUG_MAX).replace(/-+$/, "");
}
